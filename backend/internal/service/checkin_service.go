package service

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

const (
	MinQuestionsForCheckin = 3
)

type CheckinService struct {
	db  *gorm.DB
	log *slog.Logger
}

func NewCheckinService(db *gorm.DB, log *slog.Logger) *CheckinService {
	return &CheckinService{db: db, log: log}
}

func (s *CheckinService) InitBadges() error {
	for _, badge := range models.MilestoneBadges {
		var existing models.Badge
		err := s.db.Where("name = ? AND type = ?", badge.Name, badge.Type).First(&existing).Error
		if err == nil {
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("check badge: %w", err)
		}
		if err := s.db.Create(&badge).Error; err != nil {
			return fmt.Errorf("create badge %s: %w", badge.Name, err)
		}
	}
	return nil
}

func (s *CheckinService) GetCheckinStatus(userID uint) (*dto.CheckinStatusResponse, error) {
	today := time.Now().Format("2006-01-02")

	var streak models.Streak
	err := s.db.Where("user_id = ?", userID).First(&streak).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("load streak: %w", err)
	}

	var checkin models.Checkin
	var questionCount, correctCount int
	err = s.db.Where("user_id = ? AND checkin_date = ?", userID, today).First(&checkin).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("load today checkin: %w", err)
	}
	todayCheckedIn := err == nil
	if todayCheckedIn {
		questionCount = checkin.QuestionCount
		correctCount = checkin.CorrectCount
	}

	return &dto.CheckinStatusResponse{
		TodayCheckedIn: todayCheckedIn,
		CurrentStreak:  streak.CurrentStreak,
		LongestStreak:  streak.LongestStreak,
		TodayDate:      today,
		QuestionCount:  questionCount,
		CorrectCount:   correctCount,
	}, nil
}

func (s *CheckinService) AutoCheckin(userID uint, questionCount int, correctCount int) (*dto.CheckinResult, error) {
	if questionCount < MinQuestionsForCheckin {
		return &dto.CheckinResult{
			CheckedIn:     false,
			IsNewCheckin:  false,
			CurrentStreak: 0,
			NewlyAwarded:  []dto.CheckinAwardBadge{},
			QuestionCount: questionCount,
			CorrectCount:  correctCount,
		}, nil
	}

	today := time.Now().Format("2006-01-02")
	var result *dto.CheckinResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existingCheckin models.Checkin
		err := tx.Where("user_id = ? AND checkin_date = ?", userID, today).First(&existingCheckin).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("load checkin: %w", err)
		}

		isNew := err == gorm.ErrRecordNotFound
		accuracyRate := 0.0
		if questionCount > 0 {
			accuracyRate = float64(correctCount) / float64(questionCount) * 100
		}

		if isNew {
			checkin := models.Checkin{
				UserID:        userID,
				CheckinDate:   today,
				QuestionCount: questionCount,
				CorrectCount:  correctCount,
				AccuracyRate:  accuracyRate,
			}
			if err := tx.Create(&checkin).Error; err != nil {
				return fmt.Errorf("create checkin: %w", err)
			}

			streak, newlyAwarded, err := s.updateStreak(tx, userID, today)
			if err != nil {
				return err
			}

			result = &dto.CheckinResult{
				CheckedIn:     true,
				IsNewCheckin:  true,
				CurrentStreak: streak.CurrentStreak,
				NewlyAwarded:  newlyAwarded,
				QuestionCount: questionCount,
				CorrectCount:  correctCount,
				AccuracyRate:  accuracyRate,
			}
		} else {
			existingCheckin.QuestionCount += questionCount
			existingCheckin.CorrectCount += correctCount
			if existingCheckin.QuestionCount > 0 {
				existingCheckin.AccuracyRate = float64(existingCheckin.CorrectCount) / float64(existingCheckin.QuestionCount) * 100
			}
			if err := tx.Save(&existingCheckin).Error; err != nil {
				return fmt.Errorf("update checkin: %w", err)
			}

			var streak models.Streak
			if err := tx.Where("user_id = ?", userID).First(&streak).Error; err != nil && err != gorm.ErrRecordNotFound {
				return fmt.Errorf("load streak: %w", err)
			}

			result = &dto.CheckinResult{
				CheckedIn:     true,
				IsNewCheckin:  false,
				CurrentStreak: streak.CurrentStreak,
				NewlyAwarded:  []dto.CheckinAwardBadge{},
				QuestionCount: existingCheckin.QuestionCount,
				CorrectCount:  existingCheckin.CorrectCount,
				AccuracyRate:  existingCheckin.AccuracyRate,
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *CheckinService) ManualCheckin(userID uint, req dto.ManualCheckinRequest) (*dto.CheckinResult, error) {
	return s.AutoCheckin(userID, req.QuestionCount, req.CorrectCount)
}

func (s *CheckinService) updateStreak(tx *gorm.DB, userID uint, today string) (*models.Streak, []dto.CheckinAwardBadge, error) {
	var streak models.Streak
	err := tx.Where("user_id = ?", userID).First(&streak).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, nil, fmt.Errorf("load streak: %w", err)
	}

	isNewStreak := err == gorm.ErrRecordNotFound

	if isNewStreak {
		streak = models.Streak{
			UserID:          userID,
			CurrentStreak:   1,
			LongestStreak:   1,
			LastCheckinDate: today,
		}
		if err := tx.Create(&streak).Error; err != nil {
			return nil, nil, fmt.Errorf("create streak: %w", err)
		}
	} else {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

		if streak.LastCheckinDate == today {
			newlyAwarded, err := s.checkMilestoneBadges(tx, userID, streak.CurrentStreak)
			if err != nil {
				return nil, nil, err
			}
			return &streak, newlyAwarded, nil
		}

		if streak.LastCheckinDate == yesterday {
			streak.CurrentStreak++
		} else {
			streak.CurrentStreak = 1
		}

		if streak.CurrentStreak > streak.LongestStreak {
			streak.LongestStreak = streak.CurrentStreak
		}

		streak.LastCheckinDate = today

		if err := tx.Save(&streak).Error; err != nil {
			return nil, nil, fmt.Errorf("update streak: %w", err)
		}
	}

	newlyAwarded, err := s.checkMilestoneBadges(tx, userID, streak.CurrentStreak)
	if err != nil {
		return nil, nil, err
	}

	return &streak, newlyAwarded, nil
}

func (s *CheckinService) checkMilestoneBadges(tx *gorm.DB, userID uint, currentStreak int) ([]dto.CheckinAwardBadge, error) {
	var newlyAwarded []dto.CheckinAwardBadge

	var badges []models.Badge
	if err := tx.Where("type = ? AND condition <= ?", models.BadgeTypeStreak, currentStreak).
		Order("condition asc").Find(&badges).Error; err != nil {
		return nil, fmt.Errorf("load badges: %w", err)
	}

	for _, badge := range badges {
		var userBadge models.UserBadge
		err := tx.Where("user_id = ? AND badge_id = ?", userID, badge.ID).First(&userBadge).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("check user badge: %w", err)
		}

		newUserBadge := models.UserBadge{
			UserID:    userID,
			BadgeID:   badge.ID,
			AwardedAt: time.Now(),
		}
		if err := tx.Create(&newUserBadge).Error; err != nil {
			return nil, fmt.Errorf("award badge: %w", err)
		}

		newlyAwarded = append(newlyAwarded, dto.CheckinAwardBadge{
			BadgeID:     badge.ID,
			Name:        badge.Name,
			Description: badge.Description,
			Icon:        badge.Icon,
		})
	}

	return newlyAwarded, nil
}

func (s *CheckinService) GetCalendar(userID uint, year int, month int) (*dto.CheckinCalendarResponse, error) {
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	lastDay := firstDay.AddDate(0, 1, -1)

	firstStr := firstDay.Format("2006-01-02")
	lastStr := lastDay.Format("2006-01-02")

	var checkins []models.Checkin
	err := s.db.Where("user_id = ? AND checkin_date >= ? AND checkin_date <= ?", userID, firstStr, lastStr).
		Find(&checkins).Error
	if err != nil {
		return nil, fmt.Errorf("load checkins: %w", err)
	}

	checkinMap := make(map[string]models.Checkin)
	for _, c := range checkins {
		checkinMap[c.CheckinDate] = c
	}

	days := make([]dto.CheckinCalendarDay, 0, lastDay.Day())
	for d := 1; d <= lastDay.Day(); d++ {
		date := time.Date(year, time.Month(month), d, 0, 0, 0, 0, time.Local)
		dateStr := date.Format("2006-01-02")

		day := dto.CheckinCalendarDay{
			Date:         dateStr,
			IsCheckedIn:  false,
			QuestionCount: 0,
			AccuracyRate: 0,
		}

		if checkin, ok := checkinMap[dateStr]; ok {
			day.IsCheckedIn = true
			day.QuestionCount = checkin.QuestionCount
			day.AccuracyRate = checkin.AccuracyRate
		}

		days = append(days, day)
	}

	return &dto.CheckinCalendarResponse{
		Year:  year,
		Month: month,
		Days:  days,
	}, nil
}

func (s *CheckinService) GetUserBadges(userID uint) ([]dto.UserBadgeResponse, error) {
	var allBadges []models.Badge
	if err := s.db.Where("type = ?", models.BadgeTypeStreak).
		Order("condition asc").Find(&allBadges).Error; err != nil {
		return nil, fmt.Errorf("load all badges: %w", err)
	}

	var userBadges []models.UserBadge
	if err := s.db.Preload("Badge").Where("user_id = ?", userID).Find(&userBadges).Error; err != nil {
		return nil, fmt.Errorf("load user badges: %w", err)
	}

	userBadgeMap := make(map[uint]models.UserBadge)
	for _, ub := range userBadges {
		userBadgeMap[ub.BadgeID] = ub
	}

	result := make([]dto.UserBadgeResponse, 0, len(allBadges))
	for _, badge := range allBadges {
		item := dto.UserBadgeResponse{
			BadgeID:     badge.ID,
			Name:        badge.Name,
			Description: badge.Description,
			Icon:        badge.Icon,
			Type:        badge.Type,
			Condition:   badge.Condition,
			AwardedAt:   "",
			Owned:       false,
		}

		if ub, ok := userBadgeMap[badge.ID]; ok {
			item.Owned = true
			item.AwardedAt = ub.AwardedAt.Format("2006-01-02 15:04:05")
		}

		result = append(result, item)
	}

	return result, nil
}
