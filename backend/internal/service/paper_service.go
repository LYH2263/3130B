package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

type PaperService struct {
	db  *gorm.DB
	log *slog.Logger
	rng *rand.Rand
}

func NewPaperService(db *gorm.DB, log *slog.Logger) *PaperService {
	return &PaperService{
		db:  db,
		log: log,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *PaperService) ListBlueprints(filter dto.PaperBlueprintFilter) ([]models.PaperBlueprint, int64, error) {
	var blueprints []models.PaperBlueprint
	var total int64

	query := s.db.Model(&models.PaperBlueprint{})
	if filter.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+filter.Keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count blueprints: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Preload("Creator").Order("id desc").Offset(offset).Limit(filter.PageSize).
		Find(&blueprints).Error; err != nil {
		return nil, 0, fmt.Errorf("list blueprints: %w", err)
	}

	for i := range blueprints {
		if err := s.loadBlueprintRule(&blueprints[i]); err != nil {
			s.log.Warn("load blueprint rule failed", "id", blueprints[i].ID, "error", err.Error())
		}
	}

	return blueprints, total, nil
}

func (s *PaperService) GetBlueprint(id uint) (*models.PaperBlueprint, error) {
	var blueprint models.PaperBlueprint
	if err := s.db.Preload("Creator").First(&blueprint, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrPaperBlueprintNotFound
		}
		return nil, fmt.Errorf("get blueprint: %w", err)
	}
	if err := s.loadBlueprintRule(&blueprint); err != nil {
		return nil, fmt.Errorf("load rule: %w", err)
	}
	return &blueprint, nil
}

func (s *PaperService) CreateBlueprint(input dto.PaperBlueprintCreateInput, createdBy uint) (*models.PaperBlueprint, error) {
	if input.Rule.TotalQuestions <= 0 {
		return nil, ErrPaperInvalidRule
	}

	ruleJSON, err := json.Marshal(input.Rule)
	if err != nil {
		return nil, fmt.Errorf("marshal rule: %w", err)
	}

	blueprint := models.PaperBlueprint{
		Name:            strings.TrimSpace(input.Name),
		Description:     strings.TrimSpace(input.Description),
		TotalScore:      input.TotalScore,
		RuleJSON:        string(ruleJSON),
		AvoidRepeatDays: input.AvoidRepeatDays,
		CreatedBy:       createdBy,
	}

	if err := s.db.Create(&blueprint).Error; err != nil {
		return nil, fmt.Errorf("create blueprint: %w", err)
	}

	if err := s.db.Preload("Creator").First(&blueprint, blueprint.ID).Error; err != nil {
		return nil, fmt.Errorf("reload blueprint: %w", err)
	}

	if err := s.loadBlueprintRule(&blueprint); err != nil {
		return nil, fmt.Errorf("load rule: %w", err)
	}

	s.log.Info("paper blueprint created", "id", blueprint.ID, "createdBy", createdBy)
	return &blueprint, nil
}

func (s *PaperService) UpdateBlueprint(id uint, input dto.PaperBlueprintUpdateInput) (*models.PaperBlueprint, error) {
	var blueprint models.PaperBlueprint
	if err := s.db.First(&blueprint, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrPaperBlueprintNotFound
		}
		return nil, fmt.Errorf("find blueprint: %w", err)
	}

	updates := make(map[string]interface{})
	if input.Name != "" {
		updates["name"] = strings.TrimSpace(input.Name)
	}
	if input.Description != "" {
		updates["description"] = strings.TrimSpace(input.Description)
	}
	if input.TotalScore > 0 {
		updates["total_score"] = input.TotalScore
	}
	if input.AvoidRepeatDays >= 0 {
		updates["avoid_repeat_days"] = input.AvoidRepeatDays
	}
	if input.Rule != nil {
		ruleJSON, err := json.Marshal(input.Rule)
		if err != nil {
			return nil, fmt.Errorf("marshal rule: %w", err)
		}
		updates["rule_json"] = string(ruleJSON)
	}

	if err := s.db.Model(&blueprint).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update blueprint: %w", err)
	}

	if err := s.db.Preload("Creator").First(&blueprint, id).Error; err != nil {
		return nil, fmt.Errorf("reload blueprint: %w", err)
	}

	if err := s.loadBlueprintRule(&blueprint); err != nil {
		return nil, fmt.Errorf("load rule: %w", err)
	}

	return &blueprint, nil
}

func (s *PaperService) DeleteBlueprint(id uint) error {
	result := s.db.Delete(&models.PaperBlueprint{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete blueprint: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPaperBlueprintNotFound
	}
	s.log.Info("paper blueprint deleted", "id", id)
	return nil
}

func (s *PaperService) GeneratePaper(blueprintID uint) (*dto.PaperGenerateResult, error) {
	blueprint, err := s.GetBlueprint(blueprintID)
	if err != nil {
		return nil, err
	}

	rule := blueprint.RuleData
	if rule == nil {
		return nil, ErrPaperInvalidRule
	}

	candidates, err := s.getCandidateQuestions(blueprint)
	if err != nil {
		return nil, fmt.Errorf("get candidates: %w", err)
	}

	gapReport := s.evaluateGap(rule, candidates)

	if !gapReport.CanGenerate {
		return &dto.PaperGenerateResult{
			BlueprintID: blueprintID,
			Questions:   nil,
			TotalScore:  0,
			GapReport:   gapReport,
		}, nil
	}

	questions, err := s.selectQuestions(rule, candidates, blueprint)
	if err != nil {
		return nil, fmt.Errorf("select questions: %w", err)
	}

	perQuestionScore := rule.PerQuestionScore
	if perQuestionScore <= 0 {
		perQuestionScore = blueprint.TotalScore / len(questions)
	}

	resultQuestions := make([]dto.PaperQuestionItem, 0, len(questions))
	for _, q := range questions {
		opts := make([]dto.StudentOption, 0, len(q.Options))
		for _, opt := range q.Options {
			opts = append(opts, dto.StudentOption{
				ID:      opt.ID,
				Content: opt.Content,
			})
		}
		resultQuestions = append(resultQuestions, dto.PaperQuestionItem{
			QuestionID:    q.ID,
			Title:         q.Title,
			Description:   q.Description,
			Options:       opts,
			Difficulty:    q.Difficulty,
			QuestionType:  q.QuestionType,
			KnowledgeTags: q.KnowledgeTags,
			Score:         perQuestionScore,
		})
	}

	totalScore := perQuestionScore * len(resultQuestions)

	return &dto.PaperGenerateResult{
		BlueprintID: blueprintID,
		Questions:   resultQuestions,
		TotalScore:  totalScore,
		GapReport:   gapReport,
	}, nil
}

func (s *PaperService) GetReplacementQuestion(currentQuestionID uint, blueprintID uint) (*dto.PaperQuestionItem, error) {
	blueprint, err := s.GetBlueprint(blueprintID)
	if err != nil {
		return nil, err
	}

	var currentQ models.Question
	if err := s.db.First(&currentQ, currentQuestionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrPaperQuestionNotFound
		}
		return nil, fmt.Errorf("find current question: %w", err)
	}

	candidates, err := s.getCandidateQuestions(blueprint)
	if err != nil {
		return nil, fmt.Errorf("get candidates: %w", err)
	}

	var eligible []models.Question
	for _, q := range candidates {
		if q.ID == currentQuestionID {
			continue
		}
		if q.Difficulty == currentQ.Difficulty && q.QuestionType == currentQ.QuestionType {
			eligible = append(eligible, q)
		}
	}

	if len(eligible) == 0 {
		return nil, ErrPaperQuestionExcluded
	}

	pick := eligible[s.rng.Intn(len(eligible))]

	opts := make([]dto.StudentOption, 0, len(pick.Options))
	for _, opt := range pick.Options {
		opts = append(opts, dto.StudentOption{
			ID:      opt.ID,
			Content: opt.Content,
		})
	}

	perQuestionScore := 0
	if blueprint.RuleData != nil {
		perQuestionScore = blueprint.RuleData.PerQuestionScore
	}
	if perQuestionScore <= 0 {
		perQuestionScore = blueprint.TotalScore / blueprint.RuleData.TotalQuestions
	}

	return &dto.PaperQuestionItem{
		QuestionID:    pick.ID,
		Title:         pick.Title,
		Description:   pick.Description,
		Options:       opts,
		Difficulty:    pick.Difficulty,
		QuestionType:  pick.QuestionType,
		KnowledgeTags: pick.KnowledgeTags,
		Score:         perQuestionScore,
	}, nil
}

func (s *PaperService) SavePaperSnapshot(blueprintID *uint, questions []dto.PaperQuestionItem, input dto.PaperSaveRequest, createdBy uint) (*models.PaperSnapshot, error) {
	if len(questions) == 0 {
		return nil, ErrPaperInvalidRule
	}

	questionsJSON, err := json.Marshal(questions)
	if err != nil {
		return nil, fmt.Errorf("marshal questions: %w", err)
	}

	totalScore := 0
	for _, q := range questions {
		totalScore += q.Score
	}

	status := input.Status
	if status == "" {
		status = models.PaperStatusDraft
	}

	snapshot := models.PaperSnapshot{
		BlueprintID:    blueprintID,
		Name:           strings.TrimSpace(input.Name),
		Description:    strings.TrimSpace(input.Description),
		TotalScore:     totalScore,
		TotalQuestions: len(questions),
		QuestionsJSON:  string(questionsJSON),
		Status:         status,
		CreatedBy:      createdBy,
	}

	if err := s.db.Create(&snapshot).Error; err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}

	if err := s.db.Preload("Creator").Preload("Blueprint").First(&snapshot, snapshot.ID).Error; err != nil {
		return nil, fmt.Errorf("reload snapshot: %w", err)
	}

	if err := s.loadSnapshotQuestions(&snapshot); err != nil {
		return nil, fmt.Errorf("load questions: %w", err)
	}

	s.log.Info("paper snapshot saved", "id", snapshot.ID, "createdBy", createdBy, "status", status)
	return &snapshot, nil
}

func (s *PaperService) ListSnapshots(filter dto.PaperSnapshotFilter) ([]models.PaperSnapshot, int64, error) {
	var snapshots []models.PaperSnapshot
	var total int64

	query := s.db.Model(&models.PaperSnapshot{})
	if filter.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+filter.Keyword+"%")
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count snapshots: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Preload("Creator").Preload("Blueprint").Order("id desc").
		Offset(offset).Limit(filter.PageSize).Find(&snapshots).Error; err != nil {
		return nil, 0, fmt.Errorf("list snapshots: %w", err)
	}

	for i := range snapshots {
		if err := s.loadSnapshotQuestions(&snapshots[i]); err != nil {
			s.log.Warn("load snapshot questions failed", "id", snapshots[i].ID, "error", err.Error())
		}
	}

	return snapshots, total, nil
}

func (s *PaperService) GetSnapshot(id uint) (*models.PaperSnapshot, error) {
	var snapshot models.PaperSnapshot
	if err := s.db.Preload("Creator").Preload("Blueprint").First(&snapshot, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrPaperSnapshotNotFound
		}
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	if err := s.loadSnapshotQuestions(&snapshot); err != nil {
		return nil, fmt.Errorf("load questions: %w", err)
	}
	return &snapshot, nil
}

func (s *PaperService) DeleteSnapshot(id uint) error {
	result := s.db.Delete(&models.PaperSnapshot{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete snapshot: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPaperSnapshotNotFound
	}
	s.log.Info("paper snapshot deleted", "id", id)
	return nil
}

func (s *PaperService) GetKnowledgeTags() ([]dto.KnowledgeTagOption, error) {
	type result struct {
		KnowledgeTags string
		Count         int
	}

	var results []result
	if err := s.db.Model(&models.Question{}).
		Select("knowledge_tags, COUNT(*) as count").
		Where("knowledge_tags IS NOT NULL AND knowledge_tags != ''").
		Group("knowledge_tags").
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("get tags: %w", err)
	}

	tagCountMap := make(map[string]int)
	for _, r := range results {
		tags := strings.Split(r.KnowledgeTags, ",")
		for _, t := range tags {
			t = strings.TrimSpace(t)
			if t != "" {
				tagCountMap[t] += r.Count
			}
		}
	}

	var options []dto.KnowledgeTagOption
	for tag, count := range tagCountMap {
		options = append(options, dto.KnowledgeTagOption{
			Tag:   tag,
			Count: count,
		})
	}

	return options, nil
}

func (s *PaperService) loadBlueprintRule(blueprint *models.PaperBlueprint) error {
	if blueprint.RuleJSON == "" {
		return nil
	}
	var rule models.PaperRule
	if err := json.Unmarshal([]byte(blueprint.RuleJSON), &rule); err != nil {
		return fmt.Errorf("unmarshal rule: %w", err)
	}
	blueprint.RuleData = &rule
	return nil
}

func (s *PaperService) loadSnapshotQuestions(snapshot *models.PaperSnapshot) error {
	if snapshot.QuestionsJSON == "" {
		return nil
	}
	var questions []models.PaperQuestionItem
	if err := json.Unmarshal([]byte(snapshot.QuestionsJSON), &questions); err != nil {
		return fmt.Errorf("unmarshal questions: %w", err)
	}
	snapshot.QuestionItems = questions
	return nil
}

func (s *PaperService) getCandidateQuestions(blueprint *models.PaperBlueprint) ([]models.Question, error) {
	var questions []models.Question

	query := s.db.Preload("Options")

	if blueprint.AvoidRepeatDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -blueprint.AvoidRepeatDays)
		query = query.Where("id NOT IN (?)",
			s.db.Table("attempt_answers").
				Select("DISTINCT question_id").
				Where("created_at >= ?", cutoff),
		)
	}

	if err := query.Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("load questions: %w", err)
	}

	return questions, nil
}

func (s *PaperService) evaluateGap(rule *models.PaperRule, candidates []models.Question) *dto.PaperGapReport {
	report := &dto.PaperGapReport{
		TotalNeeded:    rule.TotalQuestions,
		TotalAvailable: len(candidates),
		CanGenerate:    true,
		Messages:       []string{},
	}

	difficultyCount := make(map[string]int)
	questionTypeCount := make(map[string]int)
	knowledgeTagCount := make(map[string]int)

	for _, q := range candidates {
		difficultyCount[q.Difficulty]++
		questionTypeCount[q.QuestionType]++
		if q.KnowledgeTags != "" {
			tags := strings.Split(q.KnowledgeTags, ",")
			for _, t := range tags {
				t = strings.TrimSpace(t)
				if t != "" {
					knowledgeTagCount[t]++
				}
			}
		}
	}

	for _, diffRule := range rule.Difficulty {
		if diffRule.Count > 0 {
			available := difficultyCount[diffRule.Level]
			gap := diffRule.Count - available
			if gap < 0 {
				gap = 0
			}
			label := models.DifficultyLabelMap[diffRule.Level]
			if label == "" {
				label = diffRule.Level
			}
			report.DifficultyGaps = append(report.DifficultyGaps, dto.GapItem{
				Name:      diffRule.Level,
				Label:     label,
				Needed:    diffRule.Count,
				Available: available,
				Gap:       gap,
			})
			if gap > 0 {
				report.CanGenerate = false
				report.Messages = append(report.Messages, fmt.Sprintf("%s题不足：需要%d道，现有%d道，缺口%d道", label, diffRule.Count, available, gap))
			}
		}
	}

	for _, typeRule := range rule.QuestionTypes {
		if typeRule.Count > 0 {
			available := questionTypeCount[typeRule.Type]
			gap := typeRule.Count - available
			if gap < 0 {
				gap = 0
			}
			label := models.QuestionTypeLabelMap[typeRule.Type]
			if label == "" {
				label = typeRule.Type
			}
			report.QuestionTypeGaps = append(report.QuestionTypeGaps, dto.GapItem{
				Name:      typeRule.Type,
				Label:     label,
				Needed:    typeRule.Count,
				Available: available,
				Gap:       gap,
			})
			if gap > 0 {
				report.CanGenerate = false
				report.Messages = append(report.Messages, fmt.Sprintf("%s不足：需要%d道，现有%d道，缺口%d道", label, typeRule.Count, available, gap))
			}
		}
	}

	for _, tagRule := range rule.KnowledgeTags {
		if tagRule.Count > 0 {
			available := knowledgeTagCount[tagRule.Tag]
			gap := tagRule.Count - available
			if gap < 0 {
				gap = 0
			}
			report.KnowledgeTagGaps = append(report.KnowledgeTagGaps, dto.GapItem{
				Name:      tagRule.Tag,
				Label:     tagRule.Tag,
				Needed:    tagRule.Count,
				Available: available,
				Gap:       gap,
			})
			if gap > 0 {
				report.CanGenerate = false
				report.Messages = append(report.Messages, fmt.Sprintf("知识点「%s」不足：需要%d道，现有%d道，缺口%d道", tagRule.Tag, tagRule.Count, available, gap))
			}
		}
	}

	if len(candidates) < rule.TotalQuestions {
		report.CanGenerate = false
		report.Messages = append(report.Messages, fmt.Sprintf("总题量不足：需要%d道，现有%d道，缺口%d道", rule.TotalQuestions, len(candidates), rule.TotalQuestions-len(candidates)))
	}

	return report
}

func (s *PaperService) selectQuestions(rule *models.PaperRule, candidates []models.Question, blueprint *models.PaperBlueprint) ([]models.Question, error) {
	selected := make([]models.Question, 0, rule.TotalQuestions)
	selectedIDs := make(map[uint]bool)

	type bucketKey struct {
		difficulty   string
		questionType string
	}
	buckets := make(map[bucketKey][]models.Question)

	for _, q := range candidates {
		key := bucketKey{q.Difficulty, q.QuestionType}
		buckets[key] = append(buckets[key], q)
	}

	difficultyTarget := make(map[string]int)
	for _, dr := range rule.Difficulty {
		if dr.Count > 0 {
			difficultyTarget[dr.Level] = dr.Count
		}
	}

	typeTarget := make(map[string]int)
	for _, tr := range rule.QuestionTypes {
		if tr.Count > 0 {
			typeTarget[tr.Type] = tr.Count
		}
	}

	if len(difficultyTarget) == 0 && len(typeTarget) == 0 {
		shuffled := make([]models.Question, len(candidates))
		copy(shuffled, candidates)
		s.rng.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})
		if len(shuffled) > rule.TotalQuestions {
			shuffled = shuffled[:rule.TotalQuestions]
		}
		return s.filterByKnowledgeTags(shuffled, rule.KnowledgeTags, rule.TotalQuestions), nil
	}

	remaining := rule.TotalQuestions
	difficultySelected := make(map[string]int)
	typeSelected := make(map[string]int)

	hasDifficultyConstraint := len(difficultyTarget) > 0
	hasTypeConstraint := len(typeTarget) > 0

	if hasDifficultyConstraint && hasTypeConstraint {
		for diff, diffCount := range difficultyTarget {
			typeRemainForDiff := diffCount
			for qType, typeCount := range typeTarget {
				allocCount := typeCount
				if allocCount > typeRemainForDiff {
					allocCount = typeRemainForDiff
				}
				if allocCount <= 0 {
					continue
				}

				key := bucketKey{diff, qType}
				bucket := buckets[key]
				if len(bucket) == 0 {
					continue
				}

				s.rng.Shuffle(len(bucket), func(i, j int) {
					bucket[i], bucket[j] = bucket[j], bucket[i]
				})

				pickCount := allocCount
				if pickCount > len(bucket) {
					pickCount = len(bucket)
				}

				picked := 0
				for _, q := range bucket {
					if picked >= pickCount {
						break
					}
					if !selectedIDs[q.ID] {
						selected = append(selected, q)
						selectedIDs[q.ID] = true
						difficultySelected[diff]++
						typeSelected[qType]++
						picked++
						remaining--
						typeRemainForDiff--
					}
				}
			}
		}
	} else if hasDifficultyConstraint {
		for diff, diffCount := range difficultyTarget {
			var bucket []models.Question
			for key, qs := range buckets {
				if key.difficulty == diff {
					bucket = append(bucket, qs...)
				}
			}

			s.rng.Shuffle(len(bucket), func(i, j int) {
				bucket[i], bucket[j] = bucket[j], bucket[i]
			})

			pickCount := diffCount
			if pickCount > len(bucket) {
				pickCount = len(bucket)
			}

			picked := 0
			for _, q := range bucket {
				if picked >= pickCount {
					break
				}
				if !selectedIDs[q.ID] {
					selected = append(selected, q)
					selectedIDs[q.ID] = true
					difficultySelected[diff]++
					typeSelected[q.QuestionType]++
					picked++
					remaining--
				}
			}
		}
	} else if hasTypeConstraint {
		for qType, typeCount := range typeTarget {
			var bucket []models.Question
			for key, qs := range buckets {
				if key.questionType == qType {
					bucket = append(bucket, qs...)
				}
			}

			s.rng.Shuffle(len(bucket), func(i, j int) {
				bucket[i], bucket[j] = bucket[j], bucket[i]
			})

			pickCount := typeCount
			if pickCount > len(bucket) {
				pickCount = len(bucket)
			}

			picked := 0
			for _, q := range bucket {
				if picked >= pickCount {
					break
				}
				if !selectedIDs[q.ID] {
					selected = append(selected, q)
					selectedIDs[q.ID] = true
					difficultySelected[q.Difficulty]++
					typeSelected[qType]++
					picked++
					remaining--
				}
			}
		}
	}

	if remaining > 0 {
		var remainingPool []models.Question
		for _, q := range candidates {
			if !selectedIDs[q.ID] {
				remainingPool = append(remainingPool, q)
			}
		}

		s.rng.Shuffle(len(remainingPool), func(i, j int) {
			remainingPool[i], remainingPool[j] = remainingPool[j], remainingPool[i]
		})

		for _, q := range remainingPool {
			if remaining <= 0 {
				break
			}
			selected = append(selected, q)
			selectedIDs[q.ID] = true
			remaining--
		}
	}

	if len(rule.KnowledgeTags) > 0 {
		selected = s.filterByKnowledgeTags(selected, rule.KnowledgeTags, len(selected))
	}

	s.rng.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})

	if len(selected) > rule.TotalQuestions {
		selected = selected[:rule.TotalQuestions]
	}

	return selected, nil
}

func (s *PaperService) filterByKnowledgeTags(questions []models.Question, tagRules []models.KnowledgeTagRule, totalNeeded int) []models.Question {
	if len(tagRules) == 0 {
		return questions
	}

	result := make([]models.Question, 0, totalNeeded)
	usedIDs := make(map[uint]bool)

	tagPools := make(map[string][]models.Question)
	var untagged []models.Question

	for _, q := range questions {
		if q.KnowledgeTags == "" {
			untagged = append(untagged, q)
			continue
		}
		tags := strings.Split(q.KnowledgeTags, ",")
		for _, t := range tags {
			t = strings.TrimSpace(t)
			if t != "" {
				tagPools[t] = append(tagPools[t], q)
			}
		}
	}

	for tag, pool := range tagPools {
		s.rng.Shuffle(len(pool), func(i, j int) {
			pool[i], pool[j] = pool[j], pool[i]
		})
		tagPools[tag] = pool
	}

	for _, tr := range tagRules {
		if tr.Count <= 0 {
			continue
		}
		pool := tagPools[tr.Tag]
		picked := 0
		for _, q := range pool {
			if picked >= tr.Count {
				break
			}
			if !usedIDs[q.ID] {
				result = append(result, q)
				usedIDs[q.ID] = true
				picked++
			}
		}
	}

	remaining := totalNeeded - len(result)
	if remaining > 0 {
		var others []models.Question
		for _, q := range questions {
			if !usedIDs[q.ID] {
				others = append(others, q)
			}
		}

		s.rng.Shuffle(len(others), func(i, j int) {
			others[i], others[j] = others[j], others[i]
		})

		for _, q := range others {
			if remaining <= 0 {
				break
			}
			result = append(result, q)
			usedIDs[q.ID] = true
			remaining--
		}
	}

	return result
}
