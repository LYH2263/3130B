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

const (
	ExamStatusPending   = "pending"
	ExamStatusOngoing   = "ongoing"
	ExamStatusFinished  = "finished"
	ExamStatusCancelled = "cancelled"
)

const (
	ParticipantStatusNotJoined = "not_joined"
	ParticipantStatusOngoing   = "ongoing"
	ParticipantStatusSubmitted = "submitted"
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

type Exam struct {
	ID            uint        `gorm:"primaryKey" json:"id"`
	Name          string      `gorm:"size:128;not null" json:"name"`
	QuestionSetID *uint       `gorm:"index" json:"questionSetId"`
	StartTime     time.Time   `gorm:"index;not null" json:"startTime"`
	EndTime       time.Time   `gorm:"index;not null" json:"endTime"`
	Duration      int         `gorm:"not null;default:60" json:"duration"`
	ClassIDs      string      `gorm:"type:text;not null" json:"classIds"`
	Status        string      `gorm:"size:16;not null;default:pending;index" json:"status"`
	CreatedBy     uint        `gorm:"index;not null" json:"createdBy"`
	Creator       *User       `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Participants  []ExamParticipant `gorm:"foreignKey:ExamID" json:"participants,omitempty"`
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}

type ExamParticipant struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ExamID     uint      `gorm:"index;not null" json:"examId"`
	Exam       *Exam     `gorm:"foreignKey:ExamID" json:"exam,omitempty"`
	StudentID  uint      `gorm:"index;not null" json:"studentId"`
	Student    *User     `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Status     string    `gorm:"size:16;not null;default:not_joined;index" json:"status"`
	Score      *float64  `gorm:"type:decimal(10,2)" json:"score"`
	StartedAt  *time.Time `json:"startedAt"`
	SubmittedAt *time.Time `json:"submittedAt"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

const (
	DiscussionStatusNormal   = "normal"
	DiscussionStatusDeleted  = "deleted"
	DiscussionStatusFolded   = "folded"
)

type Discussion struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	QuestionID  uint      `gorm:"index;not null" json:"questionId"`
	AuthorID    uint      `gorm:"index;not null" json:"authorId"`
	Author      *User     `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	ParentID    *uint     `gorm:"index" json:"parentId"`
	Parent      *Discussion `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	LikeCount   int       `gorm:"not null;default:0;index" json:"likeCount"`
	Status      string    `gorm:"size:16;not null;default:normal;index" json:"status"`
	Floor       int       `gorm:"not null;default:0" json:"floor"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type DiscussionLike struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	DiscussionID uint      `gorm:"uniqueIndex:idx_discussion_user;not null" json:"discussionId"`
	UserID       uint      `gorm:"uniqueIndex:idx_discussion_user;not null" json:"userId"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (DiscussionLike) TableName() string {
	return "discussion_likes"
}
