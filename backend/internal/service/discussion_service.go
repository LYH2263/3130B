package service

import (
	"fmt"
	"html"
	"log/slog"
	"strings"

	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

type DiscussionService struct {
	db  *gorm.DB
	log *slog.Logger
}

func NewDiscussionService(db *gorm.DB, log *slog.Logger) *DiscussionService {
	return &DiscussionService{db: db, log: log}
}

func (s *DiscussionService) CreateDiscussion(userID uint, userRole string, input dto.CreateDiscussionRequest) (*models.Discussion, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, ErrInvalidDiscussion
	}

	var parent *models.Discussion
	if input.ParentID != nil {
		var p models.Discussion
		if err := s.db.First(&p, *input.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, ErrDiscussionNotFound
			}
			return nil, fmt.Errorf("find parent discussion: %w", err)
		}
		parent = &p

		if parent.ParentID != nil {
			return nil, ErrReplyTooDeep
		}

		if parent.QuestionID != input.QuestionID {
			return nil, ErrInvalidDiscussion
		}
	}

	var floor int
	if input.ParentID == nil {
		var maxFloor int
		s.db.Model(&models.Discussion{}).
			Where("question_id = ? AND parent_id IS NULL", input.QuestionID).
			Select("COALESCE(MAX(floor), 0)").
			Scan(&maxFloor)
		floor = maxFloor + 1
	}

	discussion := models.Discussion{
		QuestionID: input.QuestionID,
		AuthorID:   userID,
		Content:    sanitizeHTML(content),
		ParentID:   input.ParentID,
		LikeCount:  0,
		Status:     models.DiscussionStatusNormal,
		Floor:      floor,
	}

	if err := s.db.Create(&discussion).Error; err != nil {
		return nil, fmt.Errorf("create discussion: %w", err)
	}

	if err := s.db.Preload("Author").First(&discussion, discussion.ID).Error; err != nil {
		return nil, fmt.Errorf("reload discussion: %w", err)
	}

	s.log.Info("discussion created", "discussionID", discussion.ID, "userID", userID, "questionID", input.QuestionID)
	return &discussion, nil
}

func (s *DiscussionService) ListDiscussions(filter dto.DiscussionFilter, userID uint) ([]models.Discussion, int64, error) {
	var discussions []models.Discussion
	var total int64

	query := s.db.Model(&models.Discussion{}).
		Preload("Author").
		Where("question_id = ? AND parent_id IS NULL AND status = ?", filter.QuestionID, models.DiscussionStatusNormal)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count discussions: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	orderExpr := "like_count desc, created_at desc"
	if filter.Sort == "time" {
		orderExpr = "created_at desc"
	}

	if err := query.Order(orderExpr).
		Limit(pageSize).Offset(offset).
		Find(&discussions).Error; err != nil {
		return nil, 0, fmt.Errorf("list discussions: %w", err)
	}

	return discussions, total, nil
}

func (s *DiscussionService) ListReplies(filter dto.ReplyFilter, userID uint) ([]models.Discussion, int64, error) {
	var replies []models.Discussion
	var total int64

	query := s.db.Model(&models.Discussion{}).
		Preload("Author").
		Where("parent_id = ? AND status = ?", filter.ParentID, models.DiscussionStatusNormal)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count replies: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 5
	}
	offset := (page - 1) * pageSize

	if err := query.Order("created_at asc").
		Limit(pageSize).Offset(offset).
		Find(&replies).Error; err != nil {
		return nil, 0, fmt.Errorf("list replies: %w", err)
	}

	return replies, total, nil
}

func (s *DiscussionService) GetDiscussion(id uint) (*models.Discussion, error) {
	var discussion models.Discussion
	if err := s.db.Preload("Author").First(&discussion, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrDiscussionNotFound
		}
		return nil, fmt.Errorf("get discussion: %w", err)
	}
	return &discussion, nil
}

func (s *DiscussionService) ToggleLike(discussionID uint, userID uint) (bool, int, error) {
	var discussion models.Discussion
	if err := s.db.First(&discussion, discussionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, 0, ErrDiscussionNotFound
		}
		return false, 0, fmt.Errorf("find discussion: %w", err)
	}

	if discussion.Status != models.DiscussionStatusNormal {
		return false, 0, ErrDiscussionNotFound
	}

	var existingLike models.DiscussionLike
	err := s.db.Where("discussion_id = ? AND user_id = ?", discussionID, userID).
		First(&existingLike).Error

	isLiked := false
	var likeCount := discussion.LikeCount

	if err == nil {
		if err := s.db.Delete(&existingLike).Error; err != nil {
			return false, likeCount, fmt.Errorf("remove like: %w", err)
		}
		likeCount--
		isLiked = false
	} else if err == gorm.ErrRecordNotFound {
		like := models.DiscussionLike{
			DiscussionID: discussionID,
			UserID:       userID,
		}
		if err := s.db.Create(&like).Error; err != nil {
			return false, likeCount, fmt.Errorf("add like: %w", err)
		}
		likeCount++
		isLiked = true
	} else {
		return false, likeCount, fmt.Errorf("check like: %w", err)
	}

	if err := s.db.Model(&discussion).Update("like_count", likeCount).Error; err != nil {
		return isLiked, likeCount, fmt.Errorf("update like count: %w", err)
	}

	return isLiked, likeCount, nil
}

func (s *DiscussionService) HasLiked(discussionID uint, userID uint) (bool, error) {
	var count int64
	err := s.db.Model(&models.DiscussionLike{}).
		Where("discussion_id = ? AND user_id = ?", discussionID, userID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check like status: %w", err)
	}
	return count > 0, nil
}

func (s *DiscussionService) GetUserLikedMap(discussionIDs []uint, userID uint) (map[uint]bool, error) {
	if len(discussionIDs) == 0 {
		return map[uint]bool{}, nil
	}

	var likes []models.DiscussionLike
	if err := s.db.Where("discussion_id IN ? AND user_id = ?", discussionIDs, userID).
		Find(&likes).Error; err != nil {
		return nil, fmt.Errorf("get user likes: %w", err)
	}

	result := make(map[uint]bool, len(likes))
	for _, like := range likes {
		result[like.DiscussionID] = true
	}
	return result, nil
}

func (s *DiscussionService) DeleteDiscussion(discussionID uint, userID uint, userRole string) error {
	var discussion models.Discussion
	if err := s.db.First(&discussion, discussionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrDiscussionNotFound
		}
		return fmt.Errorf("find discussion: %w", err)
	}

	if discussion.AuthorID != userID && userRole != models.RoleTeacher {
		return ErrCannotDeleteDiscussion
	}

	if discussion.Status == models.DiscussionStatusDeleted {
		return nil
	}

	result := s.db.Model(&discussion).Update("status", models.DiscussionStatusDeleted)
	if result.Error != nil {
		return fmt.Errorf("delete discussion: %w", result.Error)
	}

	s.log.Info("discussion deleted", "discussionID", discussionID, "userID", userID)
	return nil
}
