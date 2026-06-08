package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

const (
	exportDir          = "./exports"
	asyncThreshold     = 500
	expireHours        = 24
	streamBatchSize    = 100
	passScoreThreshold = 60
)

type ExportService struct {
	db      *gorm.DB
	log     *slog.Logger
	mu      sync.Mutex
	workers map[uint]chan struct{}
}

func NewExportService(db *gorm.DB, log *slog.Logger) *ExportService {
	svc := &ExportService{
		db:      db,
		log:     log,
		workers: make(map[uint]chan struct{}),
	}
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		log.Error("create export dir failed", "error", err.Error())
	}
	go svc.cleanupExpired()
	return svc
}

func (s *ExportService) CreateExportTask(userID uint, req dto.ExportRequest) (*dto.ExportTaskResponse, error) {
	classIDsJSON, _ := json.Marshal(req.ClassIDs)

	var startTime, endTime *time.Time
	if req.StartTime != nil {
		if t, err := time.Parse("2006-01-02", *req.StartTime); err == nil {
			startTime = &t
		}
	}
	if req.EndTime != nil {
		if t, err := time.Parse("2006-01-02", *req.EndTime); err == nil {
			endOfDay := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			endTime = &endOfDay
		}
	}

	task := &models.ExportTask{
		UserID:     userID,
		Format:     req.Format,
		Dimension:  req.Dimension,
		ClassIDs:   string(classIDsJSON),
		ExamID:     req.ExamID,
		StartTime:  startTime,
		EndTime:    endTime,
		Status:     models.ExportStatusProcessing,
		Progress:   0,
	}

	if err := s.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("create export task: %w", err)
	}

	estimatedCount, _ := s.estimateRecordCount(task)
	isAsync := estimatedCount > asyncThreshold

	if isAsync {
		go s.processTaskAsync(task.ID)
	} else {
		if err := s.generateExport(task); err != nil {
			s.log.Error("sync export failed", "taskID", task.ID, "error", err.Error())
		}
		s.db.First(task, task.ID)
	}

	return s.toResponse(task, isAsync), nil
}

func (s *ExportService) estimateRecordCount(task *models.ExportTask) (int64, error) {
	query := s.db.Model(&models.Attempt{})

	classIDs := s.parseClassIDs(task.ClassIDs)
	if len(classIDs) > 0 {
		query = query.Where("class_id IN ?", classIDs)
	}
	if task.ExamID != nil {
		query = query.Where("id IN (?)", s.db.Table("exam_participants").Select("attempt_id").Where("exam_id = ?", *task.ExamID))
	}
	if task.StartTime != nil {
		query = query.Where("created_at >= ?", *task.StartTime)
	}
	if task.EndTime != nil {
		query = query.Where("created_at <= ?", *task.EndTime)
	}

	var count int64
	query.Count(&count)
	return count, nil
}

func (s *ExportService) GetTask(taskID uint, userID uint) (*dto.ExportTaskResponse, error) {
	var task models.ExportTask
	if err := s.db.Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		if IsNotFound(err) {
			return nil, ErrExportTaskNotFound
		}
		return nil, fmt.Errorf("get export task: %w", err)
	}
	return s.toResponse(&task, false), nil
}

func (s *ExportService) ListTasks(userID uint, filter dto.ExportListFilter) ([]dto.ExportTaskResponse, int64, error) {
	query := s.db.Model(&models.ExportTask{}).Where("user_id = ?", userID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var total int64
	query.Count(&total)

	offset := (filter.Page - 1) * filter.PageSize
	var tasks []models.ExportTask
	if err := query.Order("created_at desc").Offset(offset).Limit(filter.PageSize).Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("list export tasks: %w", err)
	}

	result := make([]dto.ExportTaskResponse, len(tasks))
	for i, task := range tasks {
		result[i] = *s.toResponse(&task, false)
	}
	return result, total, nil
}

func (s *ExportService) DownloadFile(taskID uint, userID uint) (string, io.ReadCloser, int64, error) {
	var task models.ExportTask
	if err := s.db.Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		if IsNotFound(err) {
			return "", nil, 0, ErrExportTaskNotFound
		}
		return "", nil, 0, fmt.Errorf("get export task: %w", err)
	}

	if task.Status != models.ExportStatusCompleted {
		return "", nil, 0, ErrExportGenerateFailed
	}

	if task.ExpiresAt != nil && time.Now().After(*task.ExpiresAt) {
		return "", nil, 0, ErrExportExpired
	}

	filePath := filepath.Join(exportDir, task.FileName)
	file, err := os.Open(filePath)
	if err != nil {
		return "", nil, 0, ErrExportGenerateFailed
	}

	return task.FileName, file, task.FileSize, nil
}

func (s *ExportService) processTaskAsync(taskID uint) {
	s.mu.Lock()
	if _, exists := s.workers[taskID]; exists {
		s.mu.Unlock()
		return
	}
	done := make(chan struct{})
	s.workers[taskID] = done
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.workers, taskID)
		s.mu.Unlock()
		close(done)
	}()

	var task models.ExportTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		s.log.Error("async export: task not found", "taskID", taskID)
		return
	}

	if err := s.generateExport(&task); err != nil {
		s.log.Error("async export failed", "taskID", taskID, "error", err.Error())
		s.db.Model(&task).Updates(map[string]interface{}{
			"status":   models.ExportStatusFailed,
			"error_msg": err.Error(),
		})
	}
}

func (s *ExportService) generateExport(task *models.ExportTask) error {
	fileName := fmt.Sprintf("score_export_%d_%d.%s", task.ID, time.Now().Unix(), task.Format)
	filePath := filepath.Join(exportDir, fileName)

	var err error
	var totalRecords int

	switch task.Format {
	case models.ExportFormatExcel:
		totalRecords, err = s.generateExcel(task, filePath)
	case models.ExportFormatCSV:
		totalRecords, err = s.generateCSV(task, filePath)
	default:
		return ErrExportFormatInvalid
	}

	if err != nil {
		s.db.Model(task).Updates(map[string]interface{}{
			"status":    models.ExportStatusFailed,
			"error_msg": err.Error(),
			"progress":  100,
		})
		return err
	}

	fileInfo, statErr := os.Stat(filePath)
	fileSize := int64(0)
	if statErr == nil {
		fileSize = fileInfo.Size()
	}

	expiresAt := time.Now().Add(time.Duration(expireHours) * time.Hour)
	fileURL := fmt.Sprintf("/api/teacher/exports/%d/download", task.ID)

	s.db.Model(task).Updates(map[string]interface{}{
		"status":        models.ExportStatusCompleted,
		"file_name":     fileName,
		"file_url":      fileURL,
		"file_size":     fileSize,
		"total_records": totalRecords,
		"progress":      100,
		"expires_at":    expiresAt,
	})

	return nil
}

func (s *ExportService) generateExcel(task *models.ExportTask, filePath string) (int, error) {
	f := excelize.NewFile()
	defer f.Close()

	classStats, err := s.getClassOverviewStats(task)
	if err != nil {
		return 0, fmt.Errorf("get class stats: %w", err)
	}
	if len(classStats) == 0 {
		return 0, ErrExportNoData
	}

	overviewSheet := "总览"
	f.SetSheetName("Sheet1", overviewSheet)

	headers := []string{"班级", "人数", "平均分", "最高分", "最低分", "及格率", "总分"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(overviewSheet, cell, h)
		style, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#E8F4FD"}, Pattern: 1},
		})
		f.SetCellStyle(overviewSheet, cell, cell, style)
	}

	totalRecords := 0
	for i, stat := range classStats {
		row := i + 2
		f.SetCellValue(overviewSheet, fmt.Sprintf("A%d", row), stat.ClassName)
		f.SetCellValue(overviewSheet, fmt.Sprintf("B%d", row), stat.StudentCount)
		f.SetCellValue(overviewSheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%.2f", stat.AvgScore))
		f.SetCellValue(overviewSheet, fmt.Sprintf("D%d", row), stat.MaxScore)
		f.SetCellValue(overviewSheet, fmt.Sprintf("E%d", row), stat.MinScore)
		f.SetCellValue(overviewSheet, fmt.Sprintf("F%d", row), fmt.Sprintf("%.2f%%", stat.PassRate))
		f.SetCellValue(overviewSheet, fmt.Sprintf("G%d", row), stat.TotalScore)
		totalRecords++
	}

	colWidths := map[string]float64{"A": 20, "B": 10, "C": 12, "D": 10, "E": 10, "F": 12, "G": 10}
	for col, w := range colWidths {
		f.SetColWidth(overviewSheet, col, col, w)
	}

	for _, stat := range classStats {
		sheetName := s.sanitizeSheetName(fmt.Sprintf("%s明细", stat.ClassName))
		f.NewSheet(sheetName)

		detailHeaders := []string{"学号", "学生姓名", "班级", "答题记录ID", "得分", "总分", "正确率", "答题时间"}
		for j, h := range detailHeaders {
			cell := fmt.Sprintf("%c1", 'A'+j)
			f.SetCellValue(sheetName, cell, h)
			style, _ := f.NewStyle(&excelize.Style{
				Font: &excelize.Font{Bold: true},
				Fill: excelize.Fill{Type: "pattern", Color: []string{"#E8F4FD"}, Pattern: 1},
			})
			f.SetCellStyle(sheetName, cell, cell, style)
		}

		rowNum := 2
		err = s.streamStudentScores(task, stat.ClassID, func(item dto.StudentScoreItem) bool {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), item.StudentID)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), "'"+item.StudentName)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), item.ClassName)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), item.AttemptID)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), item.Score)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), item.Total)
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowNum), item.Rate)
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowNum), item.CreatedAt)
			rowNum++
			totalRecords++
			return true
		})
		if err != nil {
			return 0, fmt.Errorf("stream scores for class %d: %w", stat.ClassID, err)
		}

		detailWidths := map[string]float64{"A": 10, "B": 16, "C": 20, "D": 14, "E": 8, "F": 8, "G": 10, "H": 20}
		for col, w := range detailWidths {
			f.SetColWidth(sheetName, col, col, w)
		}
	}

	s.db.Model(task).Update("progress", 80)

	if err := f.SaveAs(filePath); err != nil {
		return 0, fmt.Errorf("save excel: %w", err)
	}

	return totalRecords, nil
}

func (s *ExportService) generateCSV(task *models.ExportTask, filePath string) (int, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return 0, fmt.Errorf("create csv file: %w", err)
	}
	defer file.Close()

	file.WriteString("\xEF\xBB\xBF")

	writer := csv.NewWriter(file)
	defer writer.Flush()

	classStats, err := s.getClassOverviewStats(task)
	if err != nil {
		return 0, fmt.Errorf("get class stats: %w", err)
	}
	if len(classStats) == 0 {
		return 0, ErrExportNoData
	}

	writer.Write([]string{"=== 成绩总览 ==="})
	writer.Write([]string{"班级", "人数", "平均分", "最高分", "最低分", "及格率", "总分"})

	totalRecords := 0
	for _, stat := range classStats {
		writer.Write([]string{
			stat.ClassName,
			strconv.Itoa(stat.StudentCount),
			fmt.Sprintf("%.2f", stat.AvgScore),
			strconv.Itoa(stat.MaxScore),
			strconv.Itoa(stat.MinScore),
			fmt.Sprintf("%.2f%%", stat.PassRate),
			strconv.Itoa(stat.TotalScore),
		})
		totalRecords++
	}

	writer.Write([]string{})
	writer.Write([]string{"=== 成绩明细 ==="})
	writer.Write([]string{"学号", "学生姓名", "班级", "答题记录ID", "得分", "总分", "正确率", "答题时间"})

	for _, stat := range classStats {
		err = s.streamStudentScores(task, stat.ClassID, func(item dto.StudentScoreItem) bool {
			writer.Write([]string{
				strconv.FormatUint(uint64(item.StudentID), 10),
				item.StudentName,
				item.ClassName,
				strconv.FormatUint(uint64(item.AttemptID), 10),
				strconv.Itoa(item.Score),
				strconv.Itoa(item.Total),
				item.Rate,
				item.CreatedAt,
			})
			totalRecords++
			return true
		})
		if err != nil {
			return 0, fmt.Errorf("stream scores: %w", err)
		}
	}

	s.db.Model(task).Update("progress", 80)

	return totalRecords, nil
}

func (s *ExportService) getClassOverviewStats(task *models.ExportTask) ([]dto.ClassOverviewStat, error) {
	classIDs := s.parseClassIDs(task.ClassIDs)

	query := s.db.Model(&models.Attempt{}).
		Select("class_id, COUNT(DISTINCT user_id) as student_count, " +
			"AVG(score) as avg_score, MAX(score) as max_score, MIN(score) as min_score, " +
			"SUM(score) as total_score, " +
			"SUM(CASE WHEN score * 100.0 / total >= ? THEN 1 ELSE 0 END) as pass_count", passScoreThreshold).
		Group("class_id")

	if len(classIDs) > 0 {
		query = query.Where("class_id IN ?", classIDs)
	}
	if task.ExamID != nil {
		query = query.Where("id IN (?)", s.db.Table("exam_participants").Select("attempt_id").Where("exam_id = ? AND attempt_id IS NOT NULL", *task.ExamID))
	}
	if task.StartTime != nil {
		query = query.Where("created_at >= ?", *task.StartTime)
	}
	if task.EndTime != nil {
		query = query.Where("created_at <= ?", *task.EndTime)
	}

	type rawStat struct {
		ClassID      uint
		StudentCount int
		AvgScore     float64
		MaxScore     int
		MinScore     int
		TotalScore   int
		PassCount    int
	}

	var rawStats []rawStat
	if err := query.Scan(&rawStats).Error; err != nil {
		return nil, fmt.Errorf("query class stats: %w", err)
	}

	classIDSet := make(map[uint]struct{})
	for _, r := range rawStats {
		classIDSet[r.ClassID] = struct{}{}
	}
	classIDList := make([]uint, 0, len(classIDSet))
	for id := range classIDSet {
		classIDList = append(classIDList, id)
	}

	var classes []models.ClassRoom
	if err := s.db.Where("id IN ?", classIDList).Find(&classes).Error; err != nil {
		return nil, fmt.Errorf("load classes: %w", err)
	}
	classMap := make(map[uint]string)
	for _, c := range classes {
		classMap[c.ID] = c.Name
	}

	result := make([]dto.ClassOverviewStat, 0, len(rawStats))
	for _, r := range rawStats {
		passRate := 0.0
		if r.StudentCount > 0 {
			passRate = float64(r.PassCount) / float64(r.StudentCount) * 100
		}
		result = append(result, dto.ClassOverviewStat{
			ClassID:      r.ClassID,
			ClassName:    classMap[r.ClassID],
			StudentCount: r.StudentCount,
			AvgScore:     math.Round(r.AvgScore*100) / 100,
			MaxScore:     r.MaxScore,
			MinScore:     r.MinScore,
			PassRate:     math.Round(passRate*100) / 100,
			TotalScore:   r.TotalScore,
		})
	}

	return result, nil
}

func (s *ExportService) streamStudentScores(task *models.ExportTask, classID uint, callback func(dto.StudentScoreItem) bool) error {
	query := s.db.Model(&models.Attempt{}).
		Preload("User").
		Preload("ClassRoom").
		Where("class_id = ?", classID).
		Order("created_at desc")

	if task.ExamID != nil {
		query = query.Where("id IN (?)", s.db.Table("exam_participants").Select("attempt_id").Where("exam_id = ? AND attempt_id IS NOT NULL", *task.ExamID))
	}
	if task.StartTime != nil {
		query = query.Where("created_at >= ?", *task.StartTime)
	}
	if task.EndTime != nil {
		query = query.Where("created_at <= ?", *task.EndTime)
	}

	var total int64
	query.Count(&total)

	offset := 0
	for {
		var batch []models.Attempt
		if err := query.Limit(streamBatchSize).Offset(offset).Find(&batch).Error; err != nil {
			return fmt.Errorf("query batch: %w", err)
		}
		if len(batch) == 0 {
			break
		}

		for _, attempt := range batch {
			rate := fmt.Sprintf("%.0f%%", float64(attempt.Score)/float64(attempt.Total)*100)
			className := ""
			if attempt.ClassID != 0 {
				className = attempt.ClassRoom.Name
			}
			item := dto.StudentScoreItem{
				StudentID:   attempt.UserID,
				StudentName: attempt.User.Username,
				ClassID:     attempt.ClassID,
				ClassName:   className,
				AttemptID:   attempt.ID,
				Score:       attempt.Score,
				Total:       attempt.Total,
				Rate:        rate,
				CreatedAt:   attempt.CreatedAt.Format("2006-01-02 15:04:05"),
			}
			if !callback(item) {
				return nil
			}
		}

		offset += len(batch)
		if int64(offset) >= total {
			break
		}
	}

	return nil
}

func (s *ExportService) parseClassIDs(classIDsJSON string) []uint {
	if classIDsJSON == "" {
		return nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(classIDsJSON), &ids); err != nil {
		return nil
	}
	return ids
}

func (s *ExportService) sanitizeSheetName(name string) string {
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if len(name) > 31 {
		name = name[:31]
	}
	return name
}

func (s *ExportService) toResponse(task *models.ExportTask, isAsync bool) *dto.ExportTaskResponse {
	expiresAt := ""
	if task.ExpiresAt != nil {
		expiresAt = task.ExpiresAt.Format("2006-01-02 15:04:05")
	}
	return &dto.ExportTaskResponse{
		ID:           task.ID,
		Format:       task.Format,
		Dimension:    task.Dimension,
		Status:       task.Status,
		FileName:     task.FileName,
		FileURL:      task.FileURL,
		FileSize:     task.FileSize,
		TotalRecords: task.TotalRecords,
		Progress:     task.Progress,
		ErrorMsg:     task.ErrorMsg,
		ExpiresAt:    expiresAt,
		CreatedAt:    task.CreatedAt.Format("2006-01-02 15:04:05"),
		IsAsync:      isAsync,
	}
}

func (s *ExportService) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		var tasks []models.ExportTask
		s.db.Where("status = ? AND expires_at < ?", models.ExportStatusCompleted, time.Now()).Find(&tasks)

		for _, task := range tasks {
			if task.FileName != "" {
				filePath := filepath.Join(exportDir, task.FileName)
				os.Remove(filePath)
			}
			s.db.Delete(&task)
		}

		if len(tasks) > 0 {
			s.log.Info("cleaned up expired exports", "count", len(tasks))
		}
	}
}
