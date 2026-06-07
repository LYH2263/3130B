package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

type QuestionVersionService struct {
	db  *gorm.DB
	log *slog.Logger
}

func NewQuestionVersionService(db *gorm.DB, log *slog.Logger) *QuestionVersionService {
	return &QuestionVersionService{db: db, log: log}
}

func (s *QuestionVersionService) CreateSnapshot(questionID uint, modifiedBy uint, changeNote string) (*models.QuestionVersion, error) {
	var question models.Question
	if err := s.db.Preload("Options").First(&question, questionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("find question: %w", err)
	}

	var maxVersion int
	s.db.Model(&models.QuestionVersion{}).
		Where("question_id = ?", questionID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&maxVersion)

	snapshot := models.QuestionSnapshot{
		Title:       question.Title,
		Description: question.Description,
		Options:     make([]models.QuestionOptionSnap, 0, len(question.Options)),
	}
	for _, opt := range question.Options {
		snapshot.Options = append(snapshot.Options, models.QuestionOptionSnap{
			ID:        opt.ID,
			Content:   opt.Content,
			IsCorrect: opt.IsCorrect,
		})
	}

	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}

	version := models.QuestionVersion{
		QuestionID:    questionID,
		VersionNumber: maxVersion + 1,
		Snapshot:      string(snapshotJSON),
		ModifiedBy:    modifiedBy,
		ChangeNote:    changeNote,
	}

	if err := s.db.Create(&version).Error; err != nil {
		return nil, fmt.Errorf("create version: %w", err)
	}

	version.SnapshotData = &snapshot

	s.log.Info("question version snapshot created", "questionID", questionID, "version", version.VersionNumber)
	return &version, nil
}

func (s *QuestionVersionService) ListVersions(questionID uint) ([]models.QuestionVersion, error) {
	var versions []models.QuestionVersion
	if err := s.db.Preload("Modifier").
		Where("question_id = ?", questionID).
		Order("version_number desc").
		Find(&versions).Error; err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}

	for i := range versions {
		var snap models.QuestionSnapshot
		if err := json.Unmarshal([]byte(versions[i].Snapshot), &snap); err == nil {
			versions[i].SnapshotData = &snap
		}
	}

	return versions, nil
}

func (s *QuestionVersionService) GetVersion(versionID uint) (*models.QuestionVersion, error) {
	var version models.QuestionVersion
	if err := s.db.Preload("Modifier").First(&version, versionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("get version: %w", err)
	}

	var snap models.QuestionSnapshot
	if err := json.Unmarshal([]byte(version.Snapshot), &snap); err == nil {
		version.SnapshotData = &snap
	}

	return &version, nil
}

func (s *QuestionVersionService) RollbackToVersion(questionID uint, versionID uint, modifiedBy uint) (*models.Question, error) {
	var targetVersion models.QuestionVersion
	if err := s.db.First(&targetVersion, versionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("find target version: %w", err)
	}
	if targetVersion.QuestionID != questionID {
		return nil, ErrVersionMismatch
	}

	var snap models.QuestionSnapshot
	if err := json.Unmarshal([]byte(targetVersion.Snapshot), &snap); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}

	var question models.Question
	if err := s.db.Preload("Options").First(&question, questionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("find question: %w", err)
	}

	if _, err := s.CreateSnapshot(questionID, modifiedBy, fmt.Sprintf("回滚到版本 v%d", targetVersion.VersionNumber)); err != nil {
		return nil, fmt.Errorf("create pre-rollback snapshot: %w", err)
	}

	question.Title = snap.Title
	question.Description = snap.Description

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&question).Updates(map[string]any{
			"title":       question.Title,
			"description": question.Description,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("question_id = ?", question.ID).Delete(&models.QuestionOption{}).Error; err != nil {
			return err
		}
		options := make([]models.QuestionOption, 0, len(snap.Options))
		for _, opt := range snap.Options {
			options = append(options, models.QuestionOption{
				QuestionID: question.ID,
				Content:    opt.Content,
				IsCorrect:  opt.IsCorrect,
			})
		}
		if err := tx.Create(&options).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("rollback question: %w", err)
	}

	if err := s.db.Preload("Options").First(&question, question.ID).Error; err != nil {
		return nil, fmt.Errorf("reload question: %w", err)
	}

	s.log.Info("question rolled back", "questionID", questionID, "toVersion", targetVersion.VersionNumber)
	return &question, nil
}

type VersionDiff struct {
	Title       DiffField     `json:"title"`
	Description DiffField     `json:"description"`
	Options     []OptionDiff  `json:"options"`
}

type DiffField struct {
	OldValue string `json:"oldValue"`
	NewValue string `json:"newValue"`
	Changed  bool   `json:"changed"`
}

type OptionDiff struct {
	Index     int       `json:"index"`
	Status    string    `json:"status"`
	Content   DiffField `json:"content"`
	IsCorrect DiffField `json:"isCorrect"`
	OldIndex  *int      `json:"oldIndex,omitempty"`
	NewIndex  *int      `json:"newIndex,omitempty"`
}

func (s *QuestionVersionService) DiffVersions(questionID uint, oldVersionID uint, newVersionID uint) (*VersionDiff, error) {
	var oldVer, newVer models.QuestionVersion
	if err := s.db.First(&oldVer, oldVersionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("find old version: %w", err)
	}
	if err := s.db.First(&newVer, newVersionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("find new version: %w", err)
	}
	if oldVer.QuestionID != questionID || newVer.QuestionID != questionID {
		return nil, ErrVersionMismatch
	}

	var oldSnap, newSnap models.QuestionSnapshot
	if err := json.Unmarshal([]byte(oldVer.Snapshot), &oldSnap); err != nil {
		return nil, fmt.Errorf("unmarshal old snapshot: %w", err)
	}
	if err := json.Unmarshal([]byte(newVer.Snapshot), &newSnap); err != nil {
		return nil, fmt.Errorf("unmarshal new snapshot: %w", err)
	}

	diff := &VersionDiff{
		Title: DiffField{
			OldValue: oldSnap.Title,
			NewValue: newSnap.Title,
			Changed:  oldSnap.Title != newSnap.Title,
		},
		Description: DiffField{
			OldValue: oldSnap.Description,
			NewValue: newSnap.Description,
			Changed:  oldSnap.Description != newSnap.Description,
		},
	}

	diff.Options = computeOptionDiffs(oldSnap.Options, newSnap.Options)

	return diff, nil
}

func computeOptionDiffs(oldOpts, newOpts []models.QuestionOptionSnap) []OptionDiff {
	result := make([]OptionDiff, 0)

	oldMap := make(map[uint]int)
	for i, opt := range oldOpts {
		oldMap[opt.ID] = i
	}

	newMap := make(map[uint]int)
	for i, opt := range newOpts {
		newMap[opt.ID] = i
	}

	matched := make(map[uint]bool)

	for i, newOpt := range newOpts {
		if oldIdx, ok := oldMap[newOpt.ID]; ok {
			oldOpt := oldOpts[oldIdx]
			contentChanged := oldOpt.Content != newOpt.Content
			isCorrectChanged := oldOpt.IsCorrect != newOpt.IsCorrect
			if contentChanged || isCorrectChanged {
				result = append(result, OptionDiff{
					Index:    i,
					Status:   "modified",
					OldIndex:  &oldIdx,
					NewIndex:  &i,
					Content: DiffField{
						OldValue: oldOpt.Content,
						NewValue: newOpt.Content,
						Changed:  contentChanged,
					},
					IsCorrect: DiffField{
						OldValue: fmt.Sprintf("%v", oldOpt.IsCorrect),
						NewValue: fmt.Sprintf("%v", newOpt.IsCorrect),
						Changed:  isCorrectChanged,
					},
				})
			} else {
				result = append(result, OptionDiff{
					Index:    i,
					Status:   "unchanged",
					OldIndex:  &oldIdx,
					NewIndex:  &i,
					Content: DiffField{
						OldValue: oldOpt.Content,
						NewValue: newOpt.Content,
						Changed:  false,
					},
					IsCorrect: DiffField{
						OldValue: fmt.Sprintf("%v", oldOpt.IsCorrect),
						NewValue: fmt.Sprintf("%v", newOpt.IsCorrect),
						Changed:  false,
					},
				})
			}
			matched[newOpt.ID] = true
		} else {
			idx := i
			result = append(result, OptionDiff{
				Index:    i,
				Status:   "added",
				NewIndex: &idx,
				Content: DiffField{
					NewValue: newOpt.Content,
					Changed:  true,
				},
				IsCorrect: DiffField{
					NewValue: fmt.Sprintf("%v", newOpt.IsCorrect),
					Changed:  true,
				},
			})
		}
	}

	for i, oldOpt := range oldOpts {
		if !matched[oldOpt.ID] {
			idx := i
			result = append(result, OptionDiff{
				Index:    i,
				Status:   "deleted",
				OldIndex: &idx,
				Content: DiffField{
					OldValue: oldOpt.Content,
					Changed:  true,
				},
				IsCorrect: DiffField{
					OldValue: fmt.Sprintf("%v", oldOpt.IsCorrect),
					Changed:  true,
				},
			})
		}
	}

	return result
}

func (s *QuestionVersionService) UpdateQuestionWithVersion(questionID uint, input dto.QuestionInput, modifiedBy uint, changeNote string) (*models.Question, error) {
	if _, err := s.CreateSnapshot(questionID, modifiedBy, changeNote); err != nil {
		return nil, fmt.Errorf("create snapshot before update: %w", err)
	}

	questionSvc := NewQuestionService(s.db, s.log)
	question, err := questionSvc.UpdateQuestion(questionID, input)
	if err != nil {
		return nil, err
	}

	return question, nil
}

func (s *QuestionVersionService) CreateQuestionWithVersion(input dto.QuestionInput, createdBy uint) (*models.Question, error) {
	questionSvc := NewQuestionService(s.db, s.log)
	question, err := questionSvc.CreateQuestion(input, createdBy)
	if err != nil {
		return nil, err
	}

	if _, err := s.CreateSnapshot(question.ID, createdBy, "初始版本"); err != nil {
		s.log.Warn("failed to create initial version", "questionID", question.ID, "error", err.Error())
	}

	return question, nil
}
