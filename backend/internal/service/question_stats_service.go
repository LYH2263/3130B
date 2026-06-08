package service

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"gorm.io/gorm"

	"label3130/backend/internal/models"
)

type studentStat struct {
	userID  uint
	correct int
	total   int
}

type QuestionStatsService struct {
	db  *gorm.DB
	log *slog.Logger
}

type QuestionWithStats struct {
	models.Question
	Stats *models.QuestionStats `json:"stats,omitempty"`
}

type RecalcResult struct {
	UpdatedCount int   `json:"updatedCount"`
	FailedCount  int   `json:"failedCount"`
	DurationMs   int64 `json:"durationMs"`
}

func NewQuestionStatsService(db *gorm.DB, log *slog.Logger) *QuestionStatsService {
	return &QuestionStatsService{db: db, log: log}
}

func (s *QuestionStatsService) GetStatsByQuestionID(questionID uint) (*models.QuestionStats, error) {
	var stats models.QuestionStats
	err := s.db.Where("question_id = ?", questionID).First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get question stats: %w", err)
	}
	return &stats, nil
}

func (s *QuestionStatsService) ListQuestionsWithStats(filter QuestionStatsFilter) ([]QuestionWithStats, int64, error) {
	var questions []models.Question
	var total int64

	query := s.db.Model(&models.Question{}).Preload("Options")

	needsJoin := filter.DifficultyLevel != "" ||
		filter.SortBy == "difficulty_asc" ||
		filter.SortBy == "difficulty_desc"

	if needsJoin {
		query = query.Joins("LEFT JOIN question_stats ON question_stats.question_id = questions.id")
	}

	if filter.DifficultyLevel != "" {
		switch filter.DifficultyLevel {
		case models.DifficultyEasy:
			query = query.Where("question_stats.difficulty >= ? AND question_stats.has_enough_data = ?",
				models.DifficultyCalcEasyThreshold, true)
		case models.DifficultyHard:
			query = query.Where("question_stats.difficulty <= ? AND question_stats.has_enough_data = ?",
				models.DifficultyCalcHardThreshold, true)
		case models.DifficultyMedium:
			query = query.Where("question_stats.difficulty > ? AND question_stats.difficulty < ? AND question_stats.has_enough_data = ?",
				models.DifficultyCalcHardThreshold, models.DifficultyCalcEasyThreshold, true)
		case "no_data":
			query = query.Where("question_stats.has_enough_data = ? OR question_stats.id IS NULL", false)
		}
	}

	if filter.SortBy == "difficulty_asc" {
		query = query.Order("CASE WHEN question_stats.has_enough_data = true THEN question_stats.difficulty END DESC")
	} else if filter.SortBy == "difficulty_desc" {
		query = query.Order("CASE WHEN question_stats.has_enough_data = true THEN question_stats.difficulty END ASC")
	} else {
		query = query.Order("questions.id desc")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count questions: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Offset(offset).Limit(filter.PageSize).Find(&questions).Error; err != nil {
		return nil, 0, fmt.Errorf("list questions: %w", err)
	}

	questionIDs := make([]uint, len(questions))
	for i, q := range questions {
		questionIDs[i] = q.ID
	}

	var statsList []models.QuestionStats
	if len(questionIDs) > 0 {
		if err := s.db.Where("question_id IN ?", questionIDs).Find(&statsList).Error; err != nil {
			return nil, 0, fmt.Errorf("load stats: %w", err)
		}
	}

	statsMap := make(map[uint]*models.QuestionStats, len(statsList))
	for i := range statsList {
		statsMap[statsList[i].QuestionID] = &statsList[i]
	}

	result := make([]QuestionWithStats, len(questions))
	for i, q := range questions {
		result[i] = QuestionWithStats{
			Question: q,
			Stats:    statsMap[q.ID],
		}
	}

	return result, total, nil
}

func (s *QuestionStatsService) GetDifficultyDistribution() (*models.DifficultyDistribution, error) {
	var totalQuestions int64
	if err := s.db.Model(&models.Question{}).Count(&totalQuestions).Error; err != nil {
		return nil, fmt.Errorf("count total questions: %w", err)
	}

	type countResult struct {
		Count int64
	}

	var easyCount countResult
	err := s.db.Table("question_stats").
		Where("difficulty >= ? AND has_enough_data = ?", models.DifficultyCalcEasyThreshold, true).
		Select("COUNT(*) as count").
		Scan(&easyCount).Error
	if err != nil {
		return nil, fmt.Errorf("count easy: %w", err)
	}

	var hardCount countResult
	err = s.db.Table("question_stats").
		Where("difficulty <= ? AND has_enough_data = ?", models.DifficultyCalcHardThreshold, true).
		Select("COUNT(*) as count").
		Scan(&hardCount).Error
	if err != nil {
		return nil, fmt.Errorf("count hard: %w", err)
	}

	var mediumCount countResult
	err = s.db.Table("question_stats").
		Where("difficulty > ? AND difficulty < ? AND has_enough_data = ?",
			models.DifficultyCalcHardThreshold, models.DifficultyCalcEasyThreshold, true).
		Select("COUNT(*) as count").
		Scan(&mediumCount).Error
	if err != nil {
		return nil, fmt.Errorf("count medium: %w", err)
	}

	var hasDataCount countResult
	err = s.db.Table("question_stats").
		Where("has_enough_data = ?", true).
		Select("COUNT(*) as count").
		Scan(&hasDataCount).Error
	if err != nil {
		return nil, fmt.Errorf("count has data: %w", err)
	}

	noDataCount := totalQuestions - hasDataCount.Count

	return &models.DifficultyDistribution{
		EasyCount:   easyCount.Count,
		MediumCount: mediumCount.Count,
		HardCount:   hardCount.Count,
		NoDataCount: noDataCount,
		Total:       totalQuestions,
	}, nil
}

func (s *QuestionStatsService) GetAbnormalQuestions(abnormalType string) ([]models.AbnormalQuestion, error) {
	var statsList []models.QuestionStats
	query := s.db.Where("has_enough_data = ?", true)

	switch abnormalType {
	case "too_hard":
		query = query.Where("difficulty <= ?", models.DifficultyCalcHardThreshold)
	case "too_easy":
		query = query.Where("difficulty >= ?", models.DifficultyCalcEasyThreshold)
	case "poor_discrimination":
		query = query.Where("discrimination <= ?", models.DiscriminationPoorThreshold)
	default:
		query = query.Where("difficulty <= ? OR difficulty >= ? OR discrimination <= ?",
			models.DifficultyCalcHardThreshold, models.DifficultyCalcEasyThreshold,
			models.DiscriminationPoorThreshold)
	}

	if err := query.Order("difficulty asc").Find(&statsList).Error; err != nil {
		return nil, fmt.Errorf("load abnormal stats: %w", err)
	}

	if len(statsList) == 0 {
		return []models.AbnormalQuestion{}, nil
	}

	questionIDs := make([]uint, len(statsList))
	for i, s := range statsList {
		questionIDs[i] = s.QuestionID
	}

	var questions []models.Question
	if err := s.db.Where("id IN ?", questionIDs).Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("load questions: %w", err)
	}

	questionMap := make(map[uint]string, len(questions))
	for _, q := range questions {
		questionMap[q.ID] = q.Title
	}

	result := make([]models.AbnormalQuestion, 0, len(statsList))
	for _, stat := range statsList {
		if stat.Difficulty == nil || stat.Discrimination == nil {
			continue
		}
		abnormalTypes := []string{}
		if *stat.Difficulty <= models.DifficultyCalcHardThreshold {
			abnormalTypes = append(abnormalTypes, "过难")
		}
		if *stat.Difficulty >= models.DifficultyCalcEasyThreshold {
			abnormalTypes = append(abnormalTypes, "过易")
		}
		if *stat.Discrimination <= models.DiscriminationPoorThreshold {
			abnormalTypes = append(abnormalTypes, "区分度差")
		}

		if len(abnormalTypes) > 0 {
			result = append(result, models.AbnormalQuestion{
				QuestionID:     stat.QuestionID,
				Title:          questionMap[stat.QuestionID],
				Difficulty:     *stat.Difficulty,
				Discrimination: *stat.Discrimination,
				TotalAttempts:  stat.TotalAttempts,
				AbnormalType:   joinStrings(abnormalTypes, "/"),
			})
		}
	}

	return result, nil
}

func (s *QuestionStatsService) RecalculateAll() (*RecalcResult, error) {
	start := time.Now()
	s.log.Info("starting difficulty recalculation for all questions")

	var questions []models.Question
	if err := s.db.Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("load questions: %w", err)
	}

	if len(questions) == 0 {
		return &RecalcResult{UpdatedCount: 0, DurationMs: 0}, nil
	}

	updated := 0
	failed := 0

	for _, q := range questions {
		if err := s.recalculateOne(q.ID); err != nil {
			s.log.Warn("recalculate question failed", "questionID", q.ID, "error", err.Error())
			failed++
			continue
		}
		updated++
	}

	duration := time.Since(start).Milliseconds()
	s.log.Info("difficulty recalculation finished", "updated", updated, "failed", failed, "durationMs", duration)

	return &RecalcResult{
		UpdatedCount: updated,
		FailedCount:  failed,
		DurationMs:   duration,
	}, nil
}

func (s *QuestionStatsService) RecalculateSingle(questionID uint) error {
	return s.recalculateOne(questionID)
}

func (s *QuestionStatsService) recalculateOne(questionID uint) error {
	var answers []models.AttemptAnswer
	if err := s.db.Where("question_id = ?", questionID).Find(&answers).Error; err != nil {
		return fmt.Errorf("load answers: %w", err)
	}

	totalAttempts := int64(len(answers))
	correctCount := int64(0)
	for _, a := range answers {
		if a.IsCorrect {
			correctCount++
		}
	}

	hasEnoughData := totalAttempts >= int64(models.MinSampleSize)

	stats := models.QuestionStats{
		QuestionID:    questionID,
		TotalAttempts: totalAttempts,
		CorrectCount:  correctCount,
		HasEnoughData: hasEnoughData,
		UpdatedAt:     time.Now(),
	}

	if hasEnoughData {
		difficulty := float64(correctCount) / float64(totalAttempts)
		stats.Difficulty = &difficulty

		discrimination, err := s.calculateDiscrimination(questionID)
		if err == nil {
			stats.Discrimination = &discrimination
		}
	}

	var existing models.QuestionStats
	err := s.db.Where("question_id = ?", questionID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("check existing stats: %w", err)
	}

	if err == gorm.ErrRecordNotFound {
		if err := s.db.Create(&stats).Error; err != nil {
			return fmt.Errorf("create stats: %w", err)
		}
	} else {
		updates := map[string]interface{}{
			"total_attempts":  totalAttempts,
			"correct_count":   correctCount,
			"has_enough_data": hasEnoughData,
			"updated_at":      time.Now(),
		}

		if hasEnoughData {
			updates["difficulty"] = stats.Difficulty
			updates["discrimination"] = stats.Discrimination
		} else {
			updates["difficulty"] = nil
			updates["discrimination"] = nil
		}

		if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
			return fmt.Errorf("update stats: %w", err)
		}
	}

	return nil
}

func (s *QuestionStatsService) calculateDiscrimination(questionID uint) (float64, error) {
	type answerWithUser struct {
		UserID    uint
		IsCorrect bool
	}

	var answers []answerWithUser
	err := s.db.Table("attempt_answers").
		Select("attempt_answers.is_correct, attempts.user_id").
		Joins("JOIN attempts ON attempts.id = attempt_answers.attempt_id").
		Where("attempt_answers.question_id = ?", questionID).
		Scan(&answers).Error
	if err != nil {
		return 0, fmt.Errorf("load answers for discrimination: %w", err)
	}

	if len(answers) < models.MinSampleSize {
		return 0, fmt.Errorf("insufficient sample size")
	}

	studentMap := make(map[uint]*studentStat)
	for _, ans := range answers {
		ss, ok := studentMap[ans.UserID]
		if !ok {
			ss = &studentStat{userID: ans.UserID}
			studentMap[ans.UserID] = ss
		}
		ss.total++
		if ans.IsCorrect {
			ss.correct++
		}
	}

	if len(studentMap) < 10 {
		return 0, fmt.Errorf("insufficient student count")
	}

	students := make([]*studentStat, 0, len(studentMap))
	for _, ss := range studentMap {
		students = append(students, ss)
	}

	sort.Slice(students, func(i, j int) bool {
		rateI := float64(students[i].correct) / float64(students[i].total)
		rateJ := float64(students[j].correct) / float64(students[j].total)
		return rateI > rateJ
	})

	groupSize := int(math.Max(1, float64(len(students))*models.HighGroupRatio))

	highGroup := students[:groupSize]
	lowGroup := students[len(students)-groupSize:]

	highCorrectRate := calcGroupCorrectRate(highGroup)
	lowCorrectRate := calcGroupCorrectRate(lowGroup)

	discrimination := highCorrectRate - lowCorrectRate
	if discrimination < 0 {
		discrimination = 0
	}

	return discrimination, nil
}

func calcGroupCorrectRate(group []*studentStat) float64 {
	if len(group) == 0 {
		return 0
	}
	correct := 0
	total := 0
	for _, s := range group {
		correct += s.correct
		total += s.total
	}
	if total == 0 {
		return 0
	}
	return float64(correct) / float64(total)
}

func joinStrings(arr []string, sep string) string {
	if len(arr) == 0 {
		return ""
	}
	result := arr[0]
	for i := 1; i < len(arr); i++ {
		result += sep + arr[i]
	}
	return result
}

type QuestionStatsFilter struct {
	DifficultyLevel string `form:"difficultyLevel"`
	SortBy          string `form:"sortBy"`
	Page            int    `form:"page,default=1"`
	PageSize        int    `form:"pageSize,default=20"`
}
