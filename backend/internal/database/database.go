package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"label3130/backend/internal/models"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve sql db: %w", err)
	}

	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := autoMigrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

func autoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.ClassRoom{},
		&models.User{},
		&models.Question{},
		&models.QuestionOption{},
		&models.Attempt{},
		&models.AttemptAnswer{},
		&models.SubjectiveQuestion{},
		&models.SubjectiveSubmission{},
		&models.Exam{},
		&models.ExamParticipant{},
		&models.Discussion{},
		&models.DiscussionLike{},
		&models.Checkin{},
		&models.Streak{},
		&models.Badge{},
		&models.UserBadge{},
		&models.PkRoom{},
		&models.PkRoundResult{},
		&models.ExportTask{},
		&models.QuestionVersion{},
		&models.ProctorEvent{},
		&models.ProctorConfig{},
		&models.QuestionStats{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
