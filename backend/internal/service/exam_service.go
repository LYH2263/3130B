package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

type ExamService struct {
	db  *gorm.DB
	log *slog.Logger
}

func NewExamService(db *gorm.DB, log *slog.Logger) *ExamService {
	return &ExamService{db: db, log: log}
}

func parseClassIDs(classIDs []uint) string {
	data, _ := json.Marshal(classIDs)
	return string(data)
}

func getClassIDs(exam *models.Exam) []uint {
	var ids []uint
	if exam.ClassIDs == "" {
		return ids
	}
	_ = json.Unmarshal([]byte(exam.ClassIDs), &ids)
	return ids
}

func (s *ExamService) refreshExamStatus(exam *models.Exam) {
	if exam.Status == models.ExamStatusCancelled {
		return
	}
	now := time.Now()
	if now.Before(exam.StartTime) {
		exam.Status = models.ExamStatusPending
	} else if now.After(exam.EndTime) {
		exam.Status = models.ExamStatusFinished
	} else {
		exam.Status = models.ExamStatusOngoing
	}
}

func (s *ExamService) checkTimeConflict(classIDs []uint, startTime, endTime time.Time, excludeExamID *uint) error {
	classIDsJSON := parseClassIDs(classIDs)

	var exams []models.Exam
	query := s.db.Model(&models.Exam{}).
		Where("status != ?", models.ExamStatusCancelled).
		Where("start_time < ? AND end_time > ?", endTime, startTime)

	if excludeExamID != nil {
		query = query.Where("id != ?", *excludeExamID)
	}

	if err := query.Find(&exams).Error; err != nil {
		return fmt.Errorf("check time conflict: %w", err)
	}

	for _, exam := range exams {
		examClassIDs := getClassIDs(&exam)
		for _, cid := range classIDs {
			for _, ecid := range examClassIDs {
				if cid == ecid {
					return ErrExamTimeConflict
				}
			}
		}
	}
	_ = classIDsJSON
	return nil
}

func (s *ExamService) ListExams(filter dto.ExamFilter) ([]models.Exam, int64, error) {
	var exams []models.Exam
	var total int64

	query := s.db.Model(&models.Exam{}).Preload("Creator")

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count exams: %w", err)
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

	if err := query.Order("start_time desc").
		Limit(pageSize).Offset(offset).
		Find(&exams).Error; err != nil {
		return nil, 0, fmt.Errorf("list exams: %w", err)
	}

	for i := range exams {
		s.refreshExamStatus(&exams[i])
	}

	return exams, total, nil
}

func (s *ExamService) GetExam(id uint) (*models.Exam, error) {
	var exam models.Exam
	if err := s.db.Preload("Creator").First(&exam, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrExamNotFound
		}
		return nil, fmt.Errorf("get exam: %w", err)
	}
	s.refreshExamStatus(&exam)
	return &exam, nil
}

func (s *ExamService) CreateExam(input dto.ExamCreateInput, createdBy uint) (*models.Exam, error) {
	startTime, err := time.Parse(time.RFC3339, input.StartTime)
	if err != nil {
		return nil, ErrExamInvalidTimeRange
	}
	endTime, err := time.Parse(time.RFC3339, input.EndTime)
	if err != nil {
		return nil, ErrExamInvalidTimeRange
	}

	if !endTime.After(startTime) {
		return nil, ErrExamInvalidTimeRange
	}

	if input.Duration <= 0 {
		return nil, ErrInvalidScore
	}

	if err := s.checkTimeConflict(input.ClassIDs, startTime, endTime, nil); err != nil {
		return nil, err
	}

	status := models.ExamStatusPending
	now := time.Now()
	if now.Before(startTime) {
		status = models.ExamStatusPending
	} else if now.After(endTime) {
		status = models.ExamStatusFinished
	} else {
		status = models.ExamStatusOngoing
	}

	exam := models.Exam{
		Name:          input.Name,
		QuestionSetID: input.QuestionSetID,
		StartTime:     startTime,
		EndTime:       endTime,
		Duration:      input.Duration,
		ClassIDs:      parseClassIDs(input.ClassIDs),
		Status:        status,
		CreatedBy:     createdBy,
	}

	if err := s.db.Create(&exam).Error; err != nil {
		return nil, fmt.Errorf("create exam: %w", err)
	}

	if err := s.db.Preload("Creator").First(&exam, exam.ID).Error; err != nil {
		return nil, fmt.Errorf("reload exam: %w", err)
	}

	s.log.Info("exam created", "examID", exam.ID, "createdBy", createdBy)
	return &exam, nil
}

func (s *ExamService) UpdateExam(id uint, input dto.ExamUpdateInput) (*models.Exam, error) {
	var exam models.Exam
	if err := s.db.First(&exam, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrExamNotFound
		}
		return nil, fmt.Errorf("find exam: %w", err)
	}

	if exam.Status == models.ExamStatusOngoing {
		return nil, ErrExamInProgress
	}

	startTime := exam.StartTime
	endTime := exam.EndTime
	classIDs := getClassIDs(&exam)

	needCheckConflict := false

	if input.StartTime != "" {
		t, err := time.Parse(time.RFC3339, input.StartTime)
		if err != nil {
			return nil, ErrExamInvalidTimeRange
		}
		startTime = t
		needCheckConflict = true
	}
	if input.EndTime != "" {
		t, err := time.Parse(time.RFC3339, input.EndTime)
		if err != nil {
			return nil, ErrExamInvalidTimeRange
		}
		endTime = t
		needCheckConflict = true
	}
	if input.ClassIDs != nil && len(input.ClassIDs) > 0 {
		classIDs = input.ClassIDs
		needCheckConflict = true
	}

	if !endTime.After(startTime) {
		return nil, ErrExamInvalidTimeRange
	}

	if needCheckConflict {
		if err := s.checkTimeConflict(classIDs, startTime, endTime, &id); err != nil {
			return nil, err
		}
	}

	if input.Name != "" {
		exam.Name = input.Name
	}
	if input.QuestionSetID != nil {
		exam.QuestionSetID = input.QuestionSetID
	}
	if input.StartTime != "" {
		exam.StartTime = startTime
	}
	if input.EndTime != "" {
		exam.EndTime = endTime
	}
	if input.Duration > 0 {
		exam.Duration = input.Duration
	}
	if input.ClassIDs != nil && len(input.ClassIDs) > 0 {
		exam.ClassIDs = parseClassIDs(classIDs)
	}
	if input.Status != "" {
		validStatuses := map[string]bool{
			models.ExamStatusPending:   true,
			models.ExamStatusOngoing:   true,
			models.ExamStatusFinished:  true,
			models.ExamStatusCancelled: true,
		}
		if !validStatuses[input.Status] {
			return nil, ErrInvalidExamStatus
		}
		exam.Status = input.Status
	} else {
		s.refreshExamStatus(&exam)
	}

	if err := s.db.Save(&exam).Error; err != nil {
		return nil, fmt.Errorf("update exam: %w", err)
	}

	if err := s.db.Preload("Creator").First(&exam, id).Error; err != nil {
		return nil, fmt.Errorf("reload exam: %w", err)
	}

	s.log.Info("exam updated", "examID", id)
	return &exam, nil
}

func (s *ExamService) DeleteExam(id uint) error {
	var exam models.Exam
	if err := s.db.First(&exam, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrExamNotFound
		}
		return fmt.Errorf("find exam: %w", err)
	}

	if exam.Status == models.ExamStatusOngoing {
		return ErrExamInProgress
	}

	res := s.db.Delete(&models.Exam{}, id)
	if res.Error != nil {
		return fmt.Errorf("delete exam: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrExamNotFound
	}

	s.db.Where("exam_id = ?", id).Delete(&models.ExamParticipant{})

	s.log.Info("exam deleted", "examID", id)
	return nil
}

func (s *ExamService) GetStudentExams(studentID uint, classID *uint) (map[string][]models.Exam, error) {
	var exams []models.Exam

	query := s.db.Model(&models.Exam{}).
		Where("status != ?", models.ExamStatusCancelled).
		Order("start_time desc")

	if err := query.Find(&exams).Error; err != nil {
		return nil, fmt.Errorf("get student exams: %w", err)
	}

	result := map[string][]models.Exam{
		"pending":  {},
		"ongoing":  {},
		"finished": {},
	}

	for i := range exams {
		exam := exams[i]
		classIDs := getClassIDs(&exam)
		if classID != nil {
			found := false
			for _, cid := range classIDs {
				if cid == *classID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		s.refreshExamStatus(&exam)

		switch exam.Status {
		case models.ExamStatusPending:
			result["pending"] = append(result["pending"], exam)
		case models.ExamStatusOngoing:
			result["ongoing"] = append(result["ongoing"], exam)
		case models.ExamStatusFinished:
			result["finished"] = append(result["finished"], exam)
		}
	}

	return result, nil
}

func (s *ExamService) getOrCreateParticipant(examID, studentID uint) (*models.ExamParticipant, error) {
	var participant models.ExamParticipant
	err := s.db.Where("exam_id = ? AND student_id = ?", examID, studentID).
		First(&participant).Error
	if err == nil {
		return &participant, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("find participant: %w", err)
	}

	participant = models.ExamParticipant{
		ExamID:    examID,
		StudentID: studentID,
		Status:    models.ParticipantStatusNotJoined,
	}
	if err := s.db.Create(&participant).Error; err != nil {
		return nil, fmt.Errorf("create participant: %w", err)
	}
	return &participant, nil
}

func (s *ExamService) EnterExam(studentID uint, classID uint, examID uint) (*models.Exam, *models.ExamParticipant, error) {
	exam, err := s.GetExam(examID)
	if err != nil {
		return nil, nil, err
	}

	if exam.Status == models.ExamStatusCancelled {
		return nil, nil, ErrExamCancelled
	}

	examClassIDs := getClassIDs(exam)
	found := false
	for _, cid := range examClassIDs {
		if cid == classID {
			found = true
			break
		}
	}
	if !found {
		return nil, nil, ErrNotInExamClass
	}

	now := time.Now()
	if now.Before(exam.StartTime) {
		return nil, nil, ErrExamNotStarted
	}
	if now.After(exam.EndTime) {
		return nil, nil, ErrExamAlreadyEnded
	}

	participant, err := s.getOrCreateParticipant(examID, studentID)
	if err != nil {
		return nil, nil, err
	}

	if participant.Status == models.ParticipantStatusSubmitted {
		return nil, nil, ErrAlreadySubmitted
	}

	if participant.Status == models.ParticipantStatusNotJoined {
		now := time.Now()
		participant.Status = models.ParticipantStatusOngoing
		participant.StartedAt = &now
		if err := s.db.Save(participant).Error; err != nil {
			return nil, nil, fmt.Errorf("update participant: %w", err)
		}
	}

	s.log.Info("student entered exam", "examID", examID, "studentID", studentID)
	return exam, participant, nil
}

func (s *ExamService) SubmitExam(studentID uint, examID uint, req dto.ExamSubmitRequest) (*models.ExamParticipant, error) {
	exam, err := s.GetExam(examID)
	if err != nil {
		return nil, err
	}

	if exam.Status == models.ExamStatusCancelled {
		return nil, ErrExamCancelled
	}

	var participant models.ExamParticipant
	err = s.db.Where("exam_id = ? AND student_id = ?", examID, studentID).
		First(&participant).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrParticipantNotFound
		}
		return nil, fmt.Errorf("find participant: %w", err)
	}

	if participant.Status == models.ParticipantStatusSubmitted {
		return nil, ErrAlreadySubmitted
	}

	var questions []models.Question
	if err := s.db.Preload("Options").Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("load questions: %w", err)
	}

	if len(questions) == 0 {
		return nil, ErrNoQuestions
	}

	questionMap := make(map[uint]models.Question)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	correctCount := 0
	total := len(questions)

	for _, answer := range req.Answers {
		q, ok := questionMap[answer.QuestionID]
		if !ok {
			continue
		}
		for _, opt := range q.Options {
			if opt.ID == answer.OptionID && opt.IsCorrect {
				correctCount++
				break
			}
		}
	}

	score := float64(correctCount) / float64(total) * 100
	roundedScore := float64(int(score*100)) / 100

	now := time.Now()
	participant.Status = models.ParticipantStatusSubmitted
	participant.Score = &roundedScore
	participant.SubmittedAt = &now

	if err := s.db.Save(&participant).Error; err != nil {
		return nil, fmt.Errorf("submit exam: %w", err)
	}

	if err := s.db.Preload("Exam").Preload("Student").First(&participant, participant.ID).Error; err != nil {
		return nil, fmt.Errorf("reload participant: %w", err)
	}

	s.log.Info("exam submitted", "examID", examID, "studentID", studentID, "score", roundedScore)
	return &participant, nil
}

func (s *ExamService) GetStudentParticipant(studentID uint, examID uint) (*models.ExamParticipant, error) {
	var participant models.ExamParticipant
	err := s.db.Preload("Exam").Preload("Student").
		Where("exam_id = ? AND student_id = ?", examID, studentID).
		First(&participant).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrParticipantNotFound
		}
		return nil, fmt.Errorf("get participant: %w", err)
	}
	return &participant, nil
}

func (s *ExamService) GetExamParticipants(examID uint) ([]models.ExamParticipant, error) {
	var participants []models.ExamParticipant
	if err := s.db.Preload("Student").
		Where("exam_id = ?", examID).
		Order("status desc, id asc").
		Find(&participants).Error; err != nil {
		return nil, fmt.Errorf("get exam participants: %w", err)
	}
	return participants, nil
}
