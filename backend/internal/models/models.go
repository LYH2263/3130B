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
	ID            uint             `gorm:"primaryKey" json:"id"`
	Title         string           `gorm:"type:text;not null" json:"title"`
	Description   string           `gorm:"type:text" json:"description"`
	CreatedBy     uint             `gorm:"index" json:"createdBy"`
	Options       []QuestionOption `json:"options"`
	Difficulty    string           `gorm:"size:16;default:medium;index" json:"difficulty"`
	QuestionType  string           `gorm:"size:32;default:single_choice;index" json:"questionType"`
	KnowledgeTags string           `gorm:"type:text" json:"knowledgeTags"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
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

const (
	ProctorEventTypeTabSwitch      = "tab_switch"
	ProctorEventTypeBlur           = "blur"
	ProctorEventTypeCopy           = "copy"
	ProctorEventTypePaste          = "paste"
	ProctorEventTypeFullscreenExit = "fullscreen_exit"
	ProctorEventTypeReconnect      = "reconnect"
)

const (
	ProctorSeverityLow    = "low"
	ProctorSeverityMedium = "medium"
	ProctorSeverityHigh   = "high"
)

const (
	ProctorStatusNormal  = "normal"
	ProctorStatusWarning = "warning"
	ProctorStatusSuspicious = "suspicious"
	ProctorStatusForceSubmitted = "force_submitted"
)

var ProctorEventSeverityMap = map[string]string{
	ProctorEventTypeTabSwitch:      ProctorSeverityMedium,
	ProctorEventTypeBlur:           ProctorSeverityLow,
	ProctorEventTypeCopy:           ProctorSeverityHigh,
	ProctorEventTypePaste:          ProctorSeverityHigh,
	ProctorEventTypeFullscreenExit: ProctorSeverityMedium,
	ProctorEventTypeReconnect:      ProctorSeverityLow,
}

var ProctorEventLabelMap = map[string]string{
	ProctorEventTypeTabSwitch:      "切屏",
	ProctorEventTypeBlur:           "失焦",
	ProctorEventTypeCopy:           "复制",
	ProctorEventTypePaste:          "粘贴",
	ProctorEventTypeFullscreenExit: "全屏退出",
	ProctorEventTypeReconnect:      "断线重连",
}

type ProctorEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ExamID      uint      `gorm:"index;not null" json:"examId"`
	Exam        *Exam     `gorm:"foreignKey:ExamID" json:"exam,omitempty"`
	StudentID   uint      `gorm:"index;not null" json:"studentId"`
	Student     *User     `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	EventType   string    `gorm:"size:32;not null;index" json:"eventType"`
	Severity    string    `gorm:"size:16;not null;index" json:"severity"`
	EventTime   time.Time `gorm:"not null;index" json:"eventTime"`
	ExtraInfo   string    `gorm:"type:text" json:"extraInfo"`
	ClientIP    string    `gorm:"size:64" json:"clientIp"`
	UserAgent   string    `gorm:"size:512" json:"userAgent"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (ProctorEvent) TableName() string {
	return "proctor_events"
}

type ProctorConfig struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	ExamID          *uint  `gorm:"uniqueIndex;index" json:"examId"`
	IsGlobal        bool   `gorm:"not null;default:false;index" json:"isGlobal"`
	WarningThreshold int   `gorm:"not null;default:3" json:"warningThreshold"`
	ForceSubmitThreshold int `gorm:"not null;default:5" json:"forceSubmitThreshold"`
	TabSwitchWeight   int  `gorm:"not null;default:1" json:"tabSwitchWeight"`
	BlurWeight        int  `gorm:"not null;default:1" json:"blurWeight"`
	CopyWeight        int  `gorm:"not null;default:2" json:"copyWeight"`
	PasteWeight       int  `gorm:"not null;default:2" json:"pasteWeight"`
	FullscreenExitWeight int `gorm:"not null;default:1" json:"fullscreenExitWeight"`
	ReconnectWeight   int  `gorm:"not null;default:1" json:"reconnectWeight"`
	AutoForceSubmit   bool  `gorm:"not null;default:false" json:"autoForceSubmit"`
	AutoMarkSuspicious bool `gorm:"not null;default:true" json:"autoMarkSuspicious"`
	Enabled           bool  `gorm:"not null;default:true" json:"enabled"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (ProctorConfig) TableName() string {
	return "proctor_configs"
}

func DefaultProctorConfig() ProctorConfig {
	return ProctorConfig{
		IsGlobal:            true,
		WarningThreshold:    3,
		ForceSubmitThreshold: 5,
		TabSwitchWeight:     1,
		BlurWeight:          1,
		CopyWeight:          2,
		PasteWeight:         2,
		FullscreenExitWeight: 1,
		ReconnectWeight:     1,
		AutoForceSubmit:     false,
		AutoMarkSuspicious:  true,
		Enabled:             true,
	}
}

const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

const (
	QuestionTypeSingleChoice = "single_choice"
	QuestionTypeMultipleChoice = "multiple_choice"
	QuestionTypeTrueFalse = "true_false"
)

var DifficultyLabelMap = map[string]string{
	DifficultyEasy:   "易",
	DifficultyMedium: "中",
	DifficultyHard:   "难",
}

var QuestionTypeLabelMap = map[string]string{
	QuestionTypeSingleChoice:   "单选题",
	QuestionTypeMultipleChoice: "多选题",
	QuestionTypeTrueFalse:      "判断题",
}

type PaperBlueprint struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"size:128;not null" json:"name"`
	Description     string    `gorm:"type:text" json:"description"`
	TotalScore      int       `gorm:"not null;default:100" json:"totalScore"`
	RuleJSON        string    `gorm:"type:longtext;not null" json:"-"`
	RuleData        *PaperRule `gorm:"-" json:"rule"`
	AvoidRepeatDays int       `gorm:"not null;default:0" json:"avoidRepeatDays"`
	CreatedBy       uint      `gorm:"index;not null" json:"createdBy"`
	Creator         *User     `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (PaperBlueprint) TableName() string {
	return "paper_blueprints"
}

type PaperRule struct {
	TotalQuestions int                 `json:"totalQuestions"`
	Difficulty     []DifficultyRule    `json:"difficulty"`
	QuestionTypes  []QuestionTypeRule  `json:"questionTypes"`
	KnowledgeTags  []KnowledgeTagRule  `json:"knowledgeTags"`
	PerQuestionScore int                `json:"perQuestionScore"`
}

type DifficultyRule struct {
	Level string `json:"level"`
	Count int    `json:"count"`
	Ratio float64 `json:"ratio"`
}

type QuestionTypeRule struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
	Ratio float64 `json:"ratio"`
}

type KnowledgeTagRule struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
	Ratio float64 `json:"ratio"`
}

type PaperSnapshot struct {
	ID           uint              `gorm:"primaryKey" json:"id"`
	BlueprintID  *uint             `gorm:"index" json:"blueprintId"`
	Blueprint    *PaperBlueprint   `gorm:"foreignKey:BlueprintID" json:"blueprint,omitempty"`
	Name         string            `gorm:"size:128;not null" json:"name"`
	Description  string            `gorm:"type:text" json:"description"`
	TotalScore   int               `gorm:"not null;default:100" json:"totalScore"`
	TotalQuestions int             `gorm:"not null;default:0" json:"totalQuestions"`
	QuestionsJSON string           `gorm:"type:longtext;not null" json:"-"`
	QuestionItems []PaperQuestionItem `gorm:"-" json:"questions"`
	Status       string            `gorm:"size:16;not null;default:draft;index" json:"status"`
	CreatedBy    uint              `gorm:"index;not null" json:"createdBy"`
	Creator      *User             `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

func (PaperSnapshot) TableName() string {
	return "paper_snapshots"
}

const (
	PaperStatusDraft    = "draft"
	PaperStatusPublished = "published"
)

type PaperQuestionItem struct {
	QuestionID   uint              `json:"questionId"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Options      []StudentOption   `json:"options"`
	Difficulty   string            `json:"difficulty"`
	QuestionType string            `json:"questionType"`
	KnowledgeTags string           `json:"knowledgeTags"`
	Score        int               `json:"score"`
}

type StudentOption struct {
	ID      uint   `json:"id"`
	Content string `json:"content"`
}

type PaperGapReport struct {
	TotalNeeded   int            `json:"totalNeeded"`
	TotalAvailable int           `json:"totalAvailable"`
	DifficultyGaps []GapItem     `json:"difficultyGaps"`
	QuestionTypeGaps []GapItem   `json:"questionTypeGaps"`
	KnowledgeTagGaps []GapItem   `json:"knowledgeTagGaps"`
	CanGenerate   bool           `json:"canGenerate"`
	Messages      []string       `json:"messages"`
}

type GapItem struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Needed   int    `json:"needed"`
	Available int   `json:"available"`
	Gap      int    `json:"gap"`
}
