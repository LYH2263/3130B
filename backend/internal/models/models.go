package models

import "time"

const (
	RoleTeacher = "teacher"
	RoleStudent = "student"
)

type ClassRoom struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;uniqueIndex;not null" json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Role         string     `gorm:"size:16;not null;index" json:"role"`
	ClassID      *uint      `gorm:"index" json:"classId"`
	ClassRoom    *ClassRoom `gorm:"foreignKey:ClassID" json:"classRoom,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type Question struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	Title       string           `gorm:"type:text;not null" json:"title"`
	Description string           `gorm:"type:text" json:"description"`
	CreatedBy   uint             `gorm:"index" json:"createdBy"`
	Options     []QuestionOption `json:"options"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

type QuestionOption struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	QuestionID uint      `gorm:"index;not null" json:"questionId"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	IsCorrect  bool      `gorm:"not null" json:"isCorrect"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Attempt struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	UserID    uint            `gorm:"index;not null" json:"userId"`
	User      User            `gorm:"foreignKey:UserID" json:"user"`
	ClassID   uint            `gorm:"index;not null" json:"classId"`
	ClassRoom ClassRoom       `gorm:"foreignKey:ClassID" json:"classRoom"`
	Score     int             `gorm:"not null" json:"score"`
	Total     int             `gorm:"not null" json:"total"`
	Answers   []AttemptAnswer `gorm:"foreignKey:AttemptID" json:"answers"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type AttemptAnswer struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	AttemptID        uint      `gorm:"index;not null" json:"attemptId"`
	QuestionID       uint      `gorm:"index;not null" json:"questionId"`
	SelectedOptionID uint      `gorm:"index;not null" json:"selectedOptionId"`
	IsCorrect        bool      `gorm:"index;not null" json:"isCorrect"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

const (
	SubjectiveStatusActive   = "active"
	SubjectiveStatusInactive = "inactive"

	SubmissionStatusPending  = "pending"
	SubmissionStatusGraded   = "graded"
)

type SubjectiveQuestion struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Title         string    `gorm:"type:text;not null" json:"title"`
	ReferenceAnswer string   `gorm:"type:text" json:"referenceAnswer"`
	FullScore     float64   `gorm:"type:decimal(10,2);not null;default:10" json:"fullScore"`
	CreatedBy     uint      `gorm:"index;not null" json:"createdBy"`
	Creator       *User     `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Status        string    `gorm:"size:16;not null;default:active;index" json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type SubjectiveSubmission struct {
	ID          uint                `gorm:"primaryKey" json:"id"`
	QuestionID  uint                `gorm:"index;not null" json:"questionId"`
	Question    *SubjectiveQuestion `gorm:"foreignKey:QuestionID" json:"question,omitempty"`
	StudentID   uint                `gorm:"index;not null" json:"studentId"`
	Student     *User               `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Content     string              `gorm:"type:text;not null" json:"content"`
	SubmittedAt time.Time           `json:"submittedAt"`
	Status      string              `gorm:"size:16;not null;default:pending;index" json:"status"`
	Score       *float64            `gorm:"type:decimal(10,2)" json:"score"`
	Comment     string              `gorm:"type:text" json:"comment"`
	GradedBy    *uint               `gorm:"index" json:"gradedBy"`
	Grader      *User               `gorm:"foreignKey:GradedBy" json:"grader,omitempty"`
	GradedAt    *time.Time          `json:"gradedAt"`
	Version     int                 `gorm:"not null;default:1" json:"version"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}
