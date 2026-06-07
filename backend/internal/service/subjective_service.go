package service

import (
	"fmt"
	"html"
	"log/slog"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

type SubjectiveService struct {
	db  *gorm.DB
	log *slog.Logger
}

func NewSubjectiveService(db *gorm.DB, log *slog.Logger) *SubjectiveService {
	return &SubjectiveService{db: db, log: log}
}

func sanitizeHTML(input string) string {
	if input == "" {
		return ""
	}
	safe := html.EscapeString(input)
	safe = strings.ReplaceAll(safe, "&amp;nbsp;", "&nbsp;")
	return safe
}

func roundScore(score float64, decimals int) float64 {
	shift := math.Pow(10, float64(decimals))
	return math.Round(score*shift) / shift
}

func (s *SubjectiveService) ListQuestions() ([]models.SubjectiveQuestion, error) {
	var questions []models.SubjectiveQuestion
	if err := s.db.Preload("Creator").Order("id desc").Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("list subjective questions: %w", err)
	}
	return questions, nil
}

func (s *SubjectiveService) GetQuestion(id uint) (*models.SubjectiveQuestion, error) {
	var question models.SubjectiveQuestion
	if err := s.db.Preload("Creator").First(&question, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSubjectiveQuestionNotFound
		}
		return nil, fmt.Errorf("get subjective question: %w", err)
	}
	return &question, nil
}

func (s *SubjectiveService) CreateQuestion(input dto.SubjectiveQuestionInput, createdBy uint) (*models.SubjectiveQuestion, error) {
	if input.FullScore <= 0 {
		return nil, ErrInvalidScore
	}

	status := input.Status
	if status == "" {
		status = models.SubjectiveStatusActive
	}
	if status != models.SubjectiveStatusActive && status != models.SubjectiveStatusInactive {
		status = models.SubjectiveStatusActive
	}

	question := models.SubjectiveQuestion{
		Title:           sanitizeHTML(strings.TrimSpace(input.Title)),
		ReferenceAnswer: sanitizeHTML(input.ReferenceAnswer),
		FullScore:       roundScore(input.FullScore, 2),
		CreatedBy:       createdBy,
		Status:          status,
	}

	if err := s.db.Create(&question).Error; err != nil {
		return nil, fmt.Errorf("create subjective question: %w", err)
	}

	if err := s.db.Preload("Creator").First(&question, question.ID).Error; err != nil {
		return nil, fmt.Errorf("reload subjective question: %w", err)
	}

	s.log.Info("subjective question created", "questionID", question.ID, "createdBy", createdBy)
	return &question, nil
}

func (s *SubjectiveService) UpdateQuestion(id uint, input dto.SubjectiveQuestionInput) (*models.SubjectiveQuestion, error) {
	if input.FullScore <= 0 {
		return nil, ErrInvalidScore
	}

	var question models.SubjectiveQuestion
	if err := s.db.First(&question, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSubjectiveQuestionNotFound
		}
		return nil, fmt.Errorf("find subjective question: %w", err)
	}

	question.Title = sanitizeHTML(strings.TrimSpace(input.Title))
	question.ReferenceAnswer = sanitizeHTML(input.ReferenceAnswer)
	question.FullScore = roundScore(input.FullScore, 2)

	if input.Status != "" {
		if input.Status == models.SubjectiveStatusActive || input.Status == models.SubjectiveStatusInactive {
			question.Status = input.Status
		}
	}

	if err := s.db.Save(&question).Error; err != nil {
		return nil, fmt.Errorf("update subjective question: %w", err)
	}

	if err := s.db.Preload("Creator").First(&question, id).Error; err != nil {
		return nil, fmt.Errorf("reload subjective question: %w", err)
	}

	s.log.Info("subjective question updated", "questionID", id)
	return &question, nil
}

func (s *SubjectiveService) DeleteQuestion(id uint) error {
	res := s.db.Delete(&models.SubjectiveQuestion{}, id)
	if res.Error != nil {
		return fmt.Errorf("delete subjective question: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrSubjectiveQuestionNotFound
	}
	s.log.Info("subjective question deleted", "questionID", id)
	return nil
}

func (s *SubjectiveService) StudentListQuestions() ([]models.SubjectiveQuestion, error) {
	var questions []models.SubjectiveQuestion
	if err := s.db.Where("status = ?", models.SubjectiveStatusActive).
		Order("id desc").
		Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("list student subjective questions: %w", err)
	}
	return questions, nil
}

func (s *SubjectiveService) GetStudentQuestion(questionID uint) (*models.SubjectiveQuestion, error) {
	var question models.SubjectiveQuestion
	if err := s.db.Where("id = ? AND status = ?", questionID, models.SubjectiveStatusActive).
		First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSubjectiveQuestionNotFound
		}
		return nil, fmt.Errorf("get student subjective question: %w", err)
	}
	return &question, nil
}

func (s *SubjectiveService) SubmitAnswer(studentID uint, input dto.SubjectiveSubmitRequest) (*models.SubjectiveSubmission, error) {
	var question models.SubjectiveQuestion
	if err := s.db.First(&question, input.QuestionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSubjectiveQuestionNotFound
		}
		return nil, fmt.Errorf("find question: %w", err)
	}

	if question.Status != models.SubjectiveStatusActive {
		return nil, ErrQuestionInactive
	}

	var existing models.SubjectiveSubmission
	err := s.db.Where("question_id = ? AND student_id = ?", input.QuestionID, studentID).
		First(&existing).Error
	if err == nil {
		return nil, ErrSubmissionExists
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("check existing submission: %w", err)
	}

	now := time.Now()
	submission := models.SubjectiveSubmission{
		QuestionID:  input.QuestionID,
		StudentID:   studentID,
		Content:     sanitizeHTML(input.Content),
		SubmittedAt: now,
		Status:      models.SubmissionStatusPending,
		Version:     1,
	}

	if err := s.db.Create(&submission).Error; err != nil {
		return nil, fmt.Errorf("create submission: %w", err)
	}

	if err := s.db.Preload("Question").Preload("Student").First(&submission, submission.ID).Error; err != nil {
		return nil, fmt.Errorf("reload submission: %w", err)
	}

	s.log.Info("subjective answer submitted", "submissionID", submission.ID, "studentID", studentID, "questionID", input.QuestionID)
	return &submission, nil
}

func (s *SubjectiveService) GetStudentSubmissions(studentID uint) ([]models.SubjectiveSubmission, error) {
	var submissions []models.SubjectiveSubmission
	if err := s.db.Preload("Question").
		Where("student_id = ?", studentID).
		Order("submitted_at desc").
		Find(&submissions).Error; err != nil {
		return nil, fmt.Errorf("get student submissions: %w", err)
	}
	return submissions, nil
}

func (s *SubjectiveService) GetStudentSubmission(studentID uint, submissionID uint) (*models.SubjectiveSubmission, error) {
	var submission models.SubjectiveSubmission
	if err := s.db.Preload("Question").Preload("Grader").
		Where("id = ? AND student_id = ?", submissionID, studentID).
		First(&submission).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSubjectiveSubmissionNotFound
		}
		return nil, fmt.Errorf("get student submission: %w", err)
	}
	return &submission, nil
}

func (s *SubjectiveService) ListSubmissions(filter dto.SubjectiveSubmissionFilter) ([]models.SubjectiveSubmission, int64, error) {
	var submissions []models.SubjectiveSubmission
	var total int64

	query := s.db.Model(&models.SubjectiveSubmission{}).
		Preload("Question").
		Preload("Student").
		Preload("Grader")

	if filter.ClassID != nil {
		query = query.Joins("JOIN users ON users.id = subjective_submissions.student_id").
			Where("users.class_id = ?", *filter.ClassID)
	}

	if filter.QuestionID != nil {
		query = query.Where("subjective_submissions.question_id = ?", *filter.QuestionID)
	}

	if filter.Status != "" {
		query = query.Where("subjective_submissions.status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count submissions: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	if err := query.Order("subjective_submissions.status = 'pending' desc, subjective_submissions.submitted_at asc").
		Limit(pageSize).Offset(offset).
		Find(&submissions).Error; err != nil {
		return nil, 0, fmt.Errorf("list submissions: %w", err)
	}

	return submissions, total, nil
}

func (s *SubjectiveService) GetSubmission(id uint) (*models.SubjectiveSubmission, error) {
	var submission models.SubjectiveSubmission
	if err := s.db.Preload("Question").Preload("Student").Preload("Grader").
		First(&submission, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSubjectiveSubmissionNotFound
		}
		return nil, fmt.Errorf("get submission: %w", err)
	}
	return &submission, nil
}

func (s *SubjectiveService) GradeSubmission(submissionID uint, graderID uint, input dto.SubjectiveGradeRequest) (*models.SubjectiveSubmission, error) {
	if input.Score < 0 {
		return nil, ErrInvalidScore
	}

	var submission models.SubjectiveSubmission
	if err := s.db.First(&submission, submissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSubjectiveSubmissionNotFound
		}
		return nil, fmt.Errorf("find submission: %w", err)
	}

	var question models.SubjectiveQuestion
	if err := s.db.First(&question, submission.QuestionID).Error; err != nil {
		return nil, fmt.Errorf("find question: %w", err)
	}

	roundedScore := roundScore(input.Score, 2)
	if roundedScore > question.FullScore {
		return nil, ErrScoreExceedsFull
	}

	now := time.Now()
	currentVersion := submission.Version

	result := s.db.Model(&models.SubjectiveSubmission{}).
		Where("id = ? AND version = ?", submissionID, currentVersion).
		Updates(map[string]interface{}{
			"score":     roundedScore,
			"comment":   sanitizeHTML(input.Comment),
			"status":    models.SubmissionStatusGraded,
			"graded_by": graderID,
			"graded_at": now,
			"version":   currentVersion + 1,
		})

	if result.Error != nil {
		return nil, fmt.Errorf("grade submission: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrConcurrentUpdate
	}

	if err := s.db.Preload("Question").Preload("Student").Preload("Grader").
		First(&submission, submissionID).Error; err != nil {
		return nil, fmt.Errorf("reload submission: %w", err)
	}

	s.log.Info("subjective submission graded", "submissionID", submissionID, "graderID", graderID, "score", roundedScore)
	return &submission, nil
}

func (s *SubjectiveService) GetPendingCount() (int64, error) {
	var count int64
	if err := s.db.Model(&models.SubjectiveSubmission{}).
		Where("status = ?", models.SubmissionStatusPending).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count pending submissions: %w", err)
	}
	return count, nil
}
