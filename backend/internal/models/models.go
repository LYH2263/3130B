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

type Checkin struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"uniqueIndex:idx_user_date;not null;index" json:"userId"`
	User           *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CheckinDate    string    `gorm:"size:10;uniqueIndex:idx_user_date;not null;index" json:"checkinDate"`
	QuestionCount  int       `gorm:"not null;default:0" json:"questionCount"`
	CorrectCount   int       `gorm:"not null;default:0" json:"correctCount"`
	AccuracyRate   float64   `gorm:"type:decimal(5,2);not null;default:0" json:"accuracyRate"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Streak struct {
	UserID          uint      `gorm:"primaryKey" json:"userId"`
	User            *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CurrentStreak   int       `gorm:"not null;default:0;index" json:"currentStreak"`
	LongestStreak   int       `gorm:"not null;default:0" json:"longestStreak"`
	LastCheckinDate string    `gorm:"size:10;index" json:"lastCheckinDate"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type Badge struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;not null" json:"name"`
	Description string    `gorm:"size:255;not null" json:"description"`
	Icon        string    `gorm:"size:32;not null;default:🏆" json:"icon"`
	Type        string    `gorm:"size:16;not null;default:streak;index" json:"type"`
	Condition   int       `gorm:"not null;default:0" json:"condition"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type UserBadge struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:idx_user_badge;not null;index" json:"userId"`
	BadgeID   uint      `gorm:"uniqueIndex:idx_user_badge;not null;index" json:"badgeId"`
	Badge     *Badge    `gorm:"foreignKey:BadgeID" json:"badge,omitempty"`
	AwardedAt time.Time `json:"awardedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

func (UserBadge) TableName() string {
	return "user_badges"
}

const (
	BadgeTypeStreak = "streak"
)

var MilestoneBadges = []Badge{
	{Name: "连续打卡7天", Description: "连续打卡满7天", Icon: "🔥", Type: BadgeTypeStreak, Condition: 7},
	{Name: "连续打卡14天", Description: "连续打卡满14天", Icon: "⚡", Type: BadgeTypeStreak, Condition: 14},
	{Name: "连续打卡21天", Description: "连续打卡满21天", Icon: "🌟", Type: BadgeTypeStreak, Condition: 21},
	{Name: "连续打卡30天", Description: "连续打卡满30天", Icon: "👑", Type: BadgeTypeStreak, Condition: 30},
	{Name: "连续打卡60天", Description: "连续打卡满60天", Icon: "💎", Type: BadgeTypeStreak, Condition: 60},
	{Name: "连续打卡100天", Description: "连续打卡满100天", Icon: "🏆", Type: BadgeTypeStreak, Condition: 100},
}

const (
	PkRoomStatusWaiting  = "waiting"
	PkRoomStatusOngoing  = "ongoing"
	PkRoomStatusFinished = "finished"
)

type PkRoom struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	RoomCode      string     `gorm:"size:8;uniqueIndex;not null" json:"roomCode"`
	Status        string     `gorm:"size:16;not null;default:waiting;index" json:"status"`
	QuestionCount int        `gorm:"not null;default:10" json:"questionCount"`
	TimePerQuestion int     `gorm:"not null;default:15" json:"timePerQuestion"`
	PlayerAID     *uint      `gorm:"index" json:"playerAId"`
	PlayerA       *User      `gorm:"foreignKey:PlayerAID" json:"playerA,omitempty"`
	PlayerBID     *uint      `gorm:"index" json:"playerBId"`
	PlayerB       *User      `gorm:"foreignKey:PlayerBID" json:"playerB,omitempty"`
	ScoreA        int        `gorm:"not null;default:0" json:"scoreA"`
	ScoreB        int        `gorm:"not null;default:0" json:"scoreB"`
	WinnerID      *uint      `gorm:"index" json:"winnerId"`
	Winner        *User      `gorm:"foreignKey:WinnerID" json:"winner,omitempty"`
	Questions     string     `gorm:"type:text" json:"-"`
	StartedAt     *time.Time `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type PkRoundResult struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RoomID     uint      `gorm:"index;not null" json:"roomId"`
	RoundIndex int       `gorm:"not null" json:"roundIndex"`
	QuestionID uint      `gorm:"not null" json:"questionId"`
	PlayerAOptionID *uint `gorm:"index" json:"playerAOptionId"`
	PlayerAIsCorrect *bool `gorm:"index" json:"playerAIsCorrect"`
	PlayerATimeMs *int   `json:"playerATimeMs"`
	PlayerBOptionID *uint `gorm:"index" json:"playerBOptionId"`
	PlayerBIsCorrect *bool `gorm:"index" json:"playerBIsCorrect"`
	PlayerBTimeMs *int   `json:"playerBTimeMs"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (PkRoundResult) TableName() string {
	return "pk_round_results"
}

const (
	ExportStatusProcessing = "processing"
	ExportStatusCompleted  = "completed"
	ExportStatusFailed     = "failed"
)

const (
	ExportFormatExcel = "xlsx"
	ExportFormatCSV   = "csv"
)

const (
	ExportDimClass   = "class"
	ExportDimExam    = "exam"
	ExportDimTime    = "time"
)

type ExportTask struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"userId"`
	User        *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Format      string    `gorm:"size:8;not null;index" json:"format"`
	Dimension   string    `gorm:"size:16;not null" json:"dimension"`
	ClassIDs    string    `gorm:"type:text" json:"classIds"`
	ExamID      *uint     `gorm:"index" json:"examId"`
	StartTime   *time.Time `json:"startTime"`
	EndTime     *time.Time `json:"endTime"`
	Status      string    `gorm:"size:16;not null;default:processing;index" json:"status"`
	FileName    string    `gorm:"size:255" json:"fileName"`
	FileURL     string    `gorm:"size:512" json:"fileUrl"`
	FileSize    int64     `gorm:"default:0" json:"fileSize"`
	TotalRecords int      `gorm:"default:0" json:"totalRecords"`
	Progress    int       `gorm:"default:0" json:"progress"`
	ErrorMsg    string    `gorm:"type:text" json:"errorMsg"`
	ExpiresAt   *time.Time `gorm:"index" json:"expiresAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (ExportTask) TableName() string {
	return "export_tasks"
}

type QuestionVersion struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	QuestionID    uint      `gorm:"index;not null" json:"questionId"`
	VersionNumber int       `gorm:"not null;index" json:"versionNumber"`
	Snapshot      string    `gorm:"type:longtext;not null" json:"-"`
	SnapshotData  *QuestionSnapshot `gorm:"-" json:"snapshot"`
	ModifiedBy    uint      `gorm:"index;not null" json:"modifiedBy"`
	Modifier      *User     `gorm:"foreignKey:ModifiedBy" json:"modifier,omitempty"`
	ChangeNote    string    `gorm:"type:text" json:"changeNote"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (QuestionVersion) TableName() string {
	return "question_versions"
}

type QuestionSnapshot struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Options     []QuestionOptionSnap `json:"options"`
}

type QuestionOptionSnap struct {
	ID        uint   `json:"id"`
	Content   string `json:"content"`
	IsCorrect bool   `json:"isCorrect"`
}
