package service

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

type ProctorService struct {
	db        *gorm.DB
	log       *slog.Logger
	rateLimit sync.Map
}

func NewProctorService(db *gorm.DB, log *slog.Logger) *ProctorService {
	return &ProctorService{db: db, log: log}
}

func (s *ProctorService) InitDefaultConfig() error {
	var count int64
	if err := s.db.Model(&models.ProctorConfig{}).Where("is_global = ?", true).Count(&count).Error; err != nil {
		return fmt.Errorf("check default config: %w", err)
	}
	if count > 0 {
		return nil
	}
	defaultConfig := models.DefaultProctorConfig()
	if err := s.db.Create(&defaultConfig).Error; err != nil {
		return fmt.Errorf("create default config: %w", err)
	}
	s.log.Info("default proctor config initialized")
	return nil
}

func (s *ProctorService) GetConfig(examID *uint) (*models.ProctorConfig, error) {
	var config models.ProctorConfig
	if examID != nil {
		if err := s.db.Where("exam_id = ?", *examID).First(&config).Error; err == nil {
			return &config, nil
		}
	}
	if err := s.db.Where("is_global = ?", true).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProctorConfigNotFound
		}
		return nil, fmt.Errorf("get proctor config: %w", err)
	}
	return &config, nil
}

func (s *ProctorService) SaveConfig(examID *uint, input dto.ProctorConfigInput) (*models.ProctorConfig, error) {
	var config models.ProctorConfig
	var exists bool

	if examID != nil {
		if err := s.db.Where("exam_id = ?", *examID).First(&config).Error; err == nil {
			exists = true
		}
	} else {
		if err := s.db.Where("is_global = ?", true).First(&config).Error; err == nil {
			exists = true
		}
	}

	if !exists {
		config = models.DefaultProctorConfig()
		if examID != nil {
			config.ExamID = examID
			config.IsGlobal = false
		}
	}

	if input.WarningThreshold > 0 {
		config.WarningThreshold = input.WarningThreshold
	}
	if input.ForceSubmitThreshold > 0 {
		config.ForceSubmitThreshold = input.ForceSubmitThreshold
	}
	config.TabSwitchWeight = input.TabSwitchWeight
	config.BlurWeight = input.BlurWeight
	config.CopyWeight = input.CopyWeight
	config.PasteWeight = input.PasteWeight
	config.FullscreenExitWeight = input.FullscreenExitWeight
	config.ReconnectWeight = input.ReconnectWeight
	config.AutoForceSubmit = input.AutoForceSubmit
	config.AutoMarkSuspicious = input.AutoMarkSuspicious
	config.Enabled = input.Enabled

	if err := s.db.Save(&config).Error; err != nil {
		return nil, fmt.Errorf("save proctor config: %w", err)
	}

	s.log.Info("proctor config saved", "examID", examID, "isGlobal", config.IsGlobal)
	return &config, nil
}

func (s *ProctorService) checkRateLimit(studentID, examID uint) bool {
	key := fmt.Sprintf("%d_%d", studentID, examID)
	now := time.Now().Unix()

	var timestamps []int64
	if val, ok := s.rateLimit.Load(key); ok {
		timestamps = val.([]int64)
	}

	cutoff := now - 60
	valid := make([]int64, 0, len(timestamps))
	for _, ts := range timestamps {
		if ts > cutoff {
			valid = append(valid, ts)
		}
	}

	if len(valid) >= 30 {
		return false
	}

	valid = append(valid, now)
	s.rateLimit.Store(key, valid)
	return true
}

func (s *ProctorService) getWeight(config *models.ProctorConfig, eventType string) int {
	switch eventType {
	case models.ProctorEventTypeTabSwitch:
		return config.TabSwitchWeight
	case models.ProctorEventTypeBlur:
		return config.BlurWeight
	case models.ProctorEventTypeCopy:
		return config.CopyWeight
	case models.ProctorEventTypePaste:
		return config.PasteWeight
	case models.ProctorEventTypeFullscreenExit:
		return config.FullscreenExitWeight
	case models.ProctorEventTypeReconnect:
		return config.ReconnectWeight
	default:
		return 1
	}
}

func (s *ProctorService) calculateViolationScore(events []models.ProctorEvent, config *models.ProctorConfig) int {
	score := 0
	for _, e := range events {
		score += s.getWeight(config, e.EventType)
	}
	return score
}

func (s *ProctorService) getStudentEvents(examID, studentID uint) ([]models.ProctorEvent, error) {
	var events []models.ProctorEvent
	if err := s.db.Where("exam_id = ? AND student_id = ?", examID, studentID).
		Order("event_time desc").
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("get student events: %w", err)
	}
	return events, nil
}

func (s *ProctorService) ReportEvents(studentID uint, req dto.ProctorReportRequest, clientIP, userAgent string) (*dto.ProctorReportResponse, error) {
	config, err := s.GetConfig(&req.ExamID)
	if err != nil {
		return nil, err
	}

	if !config.Enabled {
		return nil, ErrProctorDisabled
	}

	if !s.checkRateLimit(studentID, req.ExamID) {
		return nil, ErrProctorRateLimit
	}

	var participant models.ExamParticipant
	if err := s.db.Where("exam_id = ? AND student_id = ?", req.ExamID, studentID).
		First(&participant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrParticipantNotFound
		}
		return nil, fmt.Errorf("find participant: %w", err)
	}

	if participant.Status == models.ParticipantStatusSubmitted {
		return nil, ErrProctorAlreadySubmitted
	}

	validEvents := make([]models.ProctorEvent, 0, len(req.Events))
	validEventTypes := map[string]bool{
		models.ProctorEventTypeTabSwitch:      true,
		models.ProctorEventTypeBlur:           true,
		models.ProctorEventTypeCopy:           true,
		models.ProctorEventTypePaste:          true,
		models.ProctorEventTypeFullscreenExit: true,
		models.ProctorEventTypeReconnect:      true,
	}

	for _, item := range req.Events {
		if !validEventTypes[item.EventType] {
			continue
		}
		eventTime := time.UnixMilli(item.EventTime)
		if eventTime.IsZero() {
			eventTime = time.Now()
		}
		severity := models.ProctorEventSeverityMap[item.EventType]
		validEvents = append(validEvents, models.ProctorEvent{
			ExamID:    req.ExamID,
			StudentID: studentID,
			EventType: item.EventType,
			Severity:  severity,
			EventTime: eventTime,
			ExtraInfo: item.ExtraInfo,
			ClientIP:  clientIP,
			UserAgent: userAgent,
		})
	}

	if len(validEvents) > 0 {
		if err := s.db.Create(&validEvents).Error; err != nil {
			return nil, fmt.Errorf("create proctor events: %w", err)
		}
		s.log.Info("proctor events reported", "examID", req.ExamID, "studentID", studentID, "count", len(validEvents))
	}

	allEvents, err := s.getStudentEvents(req.ExamID, studentID)
	if err != nil {
		return nil, err
	}

	violationScore := s.calculateViolationScore(allEvents, config)
	status := models.ProctorStatusNormal
	remainingWarns := config.WarningThreshold - violationScore
	if remainingWarns < 0 {
		remainingWarns = 0
	}

	if violationScore >= config.ForceSubmitThreshold {
		status = models.ProctorStatusForceSubmitted
		if config.AutoForceSubmit && participant.Status == models.ParticipantStatusOngoing {
			s.forceSubmitExam(req.ExamID, studentID)
		}
	} else if violationScore >= config.WarningThreshold {
		status = models.ProctorStatusWarning
		if config.AutoMarkSuspicious {
			status = models.ProctorStatusSuspicious
		}
	}

	return &dto.ProctorReportResponse{
		ReportedCount:    len(validEvents),
		ViolationScore:   violationScore,
		WarningThreshold: config.WarningThreshold,
		ForceThreshold:   config.ForceSubmitThreshold,
		Status:           status,
		RemainingWarns:   remainingWarns,
	}, nil
}

func (s *ProctorService) forceSubmitExam(examID, studentID uint) {
	s.log.Warn("auto force submit exam due to proctor violation", "examID", examID, "studentID", studentID)

	var participant models.ExamParticipant
	if err := s.db.Where("exam_id = ? AND student_id = ?", examID, studentID).
		First(&participant).Error; err != nil {
		s.log.Error("force submit: participant not found", "error", err.Error())
		return
	}

	if participant.Status == models.ParticipantStatusSubmitted {
		return
	}

	now := time.Now()
	score := 0.0
	participant.Status = models.ParticipantStatusSubmitted
	participant.Score = &score
	participant.SubmittedAt = &now

	if err := s.db.Save(&participant).Error; err != nil {
		s.log.Error("force submit failed", "error", err.Error())
	}
}

func (s *ProctorService) GetStudentStatus(examID, studentID uint) (*dto.ProctorStudentStatusResponse, error) {
	config, err := s.GetConfig(&examID)
	if err != nil {
		return nil, err
	}

	events, err := s.getStudentEvents(examID, studentID)
	if err != nil {
		return nil, err
	}

	violationScore := s.calculateViolationScore(events, config)
	status := models.ProctorStatusNormal
	remainingWarns := config.WarningThreshold - violationScore
	if remainingWarns < 0 {
		remainingWarns = 0
	}

	if violationScore >= config.ForceSubmitThreshold {
		status = models.ProctorStatusForceSubmitted
	} else if violationScore >= config.WarningThreshold {
		status = models.ProctorStatusSuspicious
	}

	breakdown := make(map[string]int)
	for _, e := range events {
		breakdown[e.EventType]++
	}

	recentEvents := make([]dto.ProctorEventBrief, 0, 10)
	for i := 0; i < len(events) && i < 10; i++ {
		recentEvents = append(recentEvents, dto.ProctorEventBrief{
			ID:        events[i].ID,
			EventType: events[i].EventType,
			EventTime: events[i].EventTime.Format(time.RFC3339),
			Severity:  events[i].Severity,
		})
	}

	return &dto.ProctorStudentStatusResponse{
		ExamID:           examID,
		StudentID:        studentID,
		TotalEvents:      len(events),
		ViolationScore:   violationScore,
		WarningThreshold: config.WarningThreshold,
		ForceThreshold:   config.ForceSubmitThreshold,
		Status:           status,
		RemainingWarns:   remainingWarns,
		EventBreakdown:   breakdown,
		RecentEvents:     recentEvents,
	}, nil
}

func (s *ProctorService) GetExamStats(examID uint) (*dto.ProctorExamStatsResponse, error) {
	var exam models.Exam
	if err := s.db.First(&exam, examID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrExamNotFound
		}
		return nil, fmt.Errorf("get exam: %w", err)
	}

	config, err := s.GetConfig(&examID)
	if err != nil {
		return nil, err
	}

	var participants []models.ExamParticipant
	if err := s.db.Preload("Student").
		Where("exam_id = ?", examID).
		Find(&participants).Error; err != nil {
		return nil, fmt.Errorf("get participants: %w", err)
	}

	var allEvents []models.ProctorEvent
	if err := s.db.Where("exam_id = ?", examID).
		Order("event_time desc").
		Find(&allEvents).Error; err != nil {
		return nil, fmt.Errorf("get all events: %w", err)
	}

	eventsByStudent := make(map[uint][]models.ProctorEvent)
	for _, e := range allEvents {
		eventsByStudent[e.StudentID] = append(eventsByStudent[e.StudentID], e)
	}

	studentStats := make([]dto.ProctorStudentStat, 0, len(participants))
	suspiciousCount := 0
	warningCount := 0
	totalEvents := len(allEvents)

	for _, p := range participants {
		events := eventsByStudent[p.StudentID]
		violationScore := s.calculateViolationScore(events, config)

		status := models.ProctorStatusNormal
		if violationScore >= config.ForceSubmitThreshold {
			status = models.ProctorStatusForceSubmitted
			suspiciousCount++
		} else if violationScore >= config.WarningThreshold {
			status = models.ProctorStatusSuspicious
			suspiciousCount++
			warningCount++
		} else if violationScore > 0 {
			warningCount++
		}

		breakdown := make(map[string]int)
		lastEventTime := ""
		for _, e := range events {
			breakdown[e.EventType]++
		}
		if len(events) > 0 {
			lastEventTime = events[0].EventTime.Format(time.RFC3339)
		}

		studentName := ""
		if p.Student != nil {
			studentName = p.Student.Username
		}

		studentStats = append(studentStats, dto.ProctorStudentStat{
			StudentID:      p.StudentID,
			StudentName:    studentName,
			TotalEvents:    len(events),
			ViolationScore: violationScore,
			EventBreakdown: breakdown,
			Status:         status,
			LastEventTime:  lastEventTime,
		})
	}

	return &dto.ProctorExamStatsResponse{
		ExamID:          examID,
		ExamName:        exam.Name,
		TotalStudents:   len(participants),
		TotalEvents:     totalEvents,
		SuspiciousCount: suspiciousCount,
		WarningCount:    warningCount,
		StudentStats:    studentStats,
	}, nil
}

func (s *ProctorService) GetStudentEvents(examID, studentID uint, page, pageSize int) ([]models.ProctorEvent, int64, error) {
	var events []models.ProctorEvent
	var total int64

	query := s.db.Model(&models.ProctorEvent{}).
		Where("exam_id = ? AND student_id = ?", examID, studentID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	if err := query.Order("event_time desc").
		Limit(pageSize).Offset(offset).
		Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}

	return events, total, nil
}
