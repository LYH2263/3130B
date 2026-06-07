package dto

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	ClassID  uint   `json:"classId" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type QuestionOptionInput struct {
	Content   string `json:"content" binding:"required,min=1,max=200"`
	IsCorrect bool   `json:"isCorrect"`
}

type QuestionInput struct {
	Title       string                `json:"title" binding:"required,min=2,max=1000"`
	Description string                `json:"description" binding:"max=2000"`
	Options     []QuestionOptionInput `json:"options" binding:"required,min=2,max=6,dive"`
}

type QuestionUpdateInput struct {
	QuestionInput
	ChangeNote string `json:"changeNote" binding:"max=500"`
}

type UploadQuestionPayload struct {
	Questions []QuestionInput `json:"questions"`
}

type SubmitAnswerItem struct {
	QuestionID uint `json:"questionId" binding:"required"`
	OptionID   uint `json:"optionId" binding:"required"`
}

type SubmitRequest struct {
	Answers []SubmitAnswerItem `json:"answers" binding:"required,min=1,dive"`
}

type SubjectiveQuestionInput struct {
	Title           string  `json:"title" binding:"required,min=2"`
	ReferenceAnswer string  `json:"referenceAnswer"`
	FullScore       float64 `json:"fullScore" binding:"required,min=0.01"`
	Status          string  `json:"status"`
}

type SubjectiveSubmitRequest struct {
	QuestionID uint   `json:"questionId" binding:"required"`
	Content    string `json:"content" binding:"required,min=1"`
}

type SubjectiveGradeRequest struct {
	Score   float64 `json:"score" binding:"required,min=0"`
	Comment string  `json:"comment"`
}

type SubjectiveSubmissionFilter struct {
	ClassID    *uint  `form:"classId"`
	QuestionID *uint  `form:"questionId"`
	Status     string `form:"status"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"pageSize,default=20"`
}

type ExamCreateInput struct {
	Name          string `json:"name" binding:"required,min=1,max=128"`
	QuestionSetID *uint  `json:"questionSetId"`
	StartTime     string `json:"startTime" binding:"required"`
	EndTime       string `json:"endTime" binding:"required"`
	Duration      int    `json:"duration" binding:"required,min=1"`
	ClassIDs      []uint `json:"classIds" binding:"required,min=1,dive"`
}

type ExamUpdateInput struct {
	Name          string `json:"name" binding:"omitempty,min=1,max=128"`
	QuestionSetID *uint  `json:"questionSetId"`
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
	Duration      int    `json:"duration" binding:"omitempty,min=1"`
	ClassIDs      []uint `json:"classIds" binding:"omitempty,min=1,dive"`
	Status        string `json:"status"`
}

type ExamFilter struct {
	Status   string `form:"status"`
	ClassID  *uint  `form:"classId"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"pageSize,default=20"`
}

type ExamSubmitRequest struct {
	Answers []SubmitAnswerItem `json:"answers" binding:"required,min=1,dive"`
}

type CreateDiscussionRequest struct {
	QuestionID uint   `json:"questionId" binding:"required"`
	Content    string `json:"content" binding:"required,min=1,max=1000"`
	ParentID   *uint  `json:"parentId"`
}

type DiscussionFilter struct {
	QuestionID uint   `form:"questionId" binding:"required"`
	Sort       string `form:"sort,default=hot"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"pageSize,default=10"`
}

type ReplyFilter struct {
	ParentID uint `form:"parentId" binding:"required"`
	Page     int  `form:"page,default=1"`
	PageSize int  `form:"pageSize,default=5"`
}

type CheckinCalendarRequest struct {
	Year  int `form:"year" binding:"required"`
	Month int `form:"month" binding:"required,min=1,max=12"`
}

type CheckinCalendarDay struct {
	Date         string  `json:"date"`
	IsCheckedIn  bool    `json:"isCheckedIn"`
	QuestionCount int    `json:"questionCount"`
	AccuracyRate float64 `json:"accuracyRate"`
}

type CheckinCalendarResponse struct {
	Year     int                   `json:"year"`
	Month    int                   `json:"month"`
	Days     []CheckinCalendarDay  `json:"days"`
}

type CheckinStatusResponse struct {
	TodayCheckedIn bool   `json:"todayCheckedIn"`
	CurrentStreak  int    `json:"currentStreak"`
	LongestStreak  int    `json:"longestStreak"`
	TodayDate      string `json:"todayDate"`
	QuestionCount  int    `json:"questionCount"`
	CorrectCount   int    `json:"correctCount"`
}

type ManualCheckinRequest struct {
	QuestionCount int `json:"questionCount" binding:"required,min=1"`
	CorrectCount  int `json:"correctCount" binding:"required,min=0"`
}

type CheckinAwardBadge struct {
	BadgeID     uint   `json:"badgeId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type CheckinResult struct {
	CheckedIn       bool                  `json:"checkedIn"`
	IsNewCheckin    bool                  `json:"isNewCheckin"`
	CurrentStreak   int                   `json:"currentStreak"`
	NewlyAwarded    []CheckinAwardBadge   `json:"newlyAwarded"`
	QuestionCount   int                   `json:"questionCount"`
	CorrectCount    int                   `json:"correctCount"`
	AccuracyRate    float64               `json:"accuracyRate"`
}

type UserBadgeResponse struct {
	BadgeID     uint   `json:"badgeId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Type        string `json:"type"`
	Condition   int    `json:"condition"`
	AwardedAt   string `json:"awardedAt"`
	Owned       bool   `json:"owned"`
}

type CreatePkRoomRequest struct {
	QuestionCount int `json:"questionCount" binding:"omitempty,min=3,max=20"`
	TimePerQuestion int `json:"timePerQuestion" binding:"omitempty,min=5,max=60"`
}

type JoinPkRoomRequest struct {
	RoomCode string `json:"roomCode" binding:"required,len=6"`
}

type PkRoomResponse struct {
	ID              uint   `json:"id"`
	RoomCode        string `json:"roomCode"`
	Status          string `json:"status"`
	QuestionCount   int    `json:"questionCount"`
	TimePerQuestion int    `json:"timePerQuestion"`
	PlayerAID       *uint  `json:"playerAId"`
	PlayerAName     string `json:"playerAName"`
	PlayerBID       *uint  `json:"playerBId"`
	PlayerBName     string `json:"playerBName"`
	ScoreA          int    `json:"scoreA"`
	ScoreB          int    `json:"scoreB"`
	WinnerID        *uint  `json:"winnerId"`
	StartedAt       string `json:"startedAt"`
	FinishedAt      string `json:"finishedAt"`
}

type PkWsMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type PkAnswerData struct {
	QuestionID uint `json:"questionId"`
	OptionID   uint `json:"optionId"`
}

type PkRoundData struct {
	RoundIndex   int             `json:"roundIndex"`
	QuestionID   uint            `json:"questionId"`
	Title        string          `json:"title"`
	Options      []PkOptionBrief `json:"options"`
	TimeLimitSec int             `json:"timeLimitSec"`
	StartAt      int64           `json:"startAt"`
}

type PkOptionBrief struct {
	ID      uint   `json:"id"`
	Content string `json:"content"`
}

type PkRoundResultData struct {
	RoundIndex      int  `json:"roundIndex"`
	QuestionID      uint `json:"questionId"`
	CorrectOptionID uint `json:"correctOptionId"`
	PlayerACorrect  bool `json:"playerACorrect"`
	PlayerBCorrect  bool `json:"playerBCorrect"`
	PlayerATimeMs   int  `json:"playerATimeMs"`
	PlayerBTimeMs   int  `json:"playerBTimeMs"`
	ScoreA          int  `json:"scoreA"`
	ScoreB          int  `json:"scoreB"`
}

type PkGameOverData struct {
	WinnerID   uint   `json:"winnerId"`
	WinnerName string `json:"winnerName"`
	IsDraw     bool   `json:"isDraw"`
	ScoreA     int    `json:"scoreA"`
	ScoreB     int    `json:"scoreB"`
}

type PkPlayerLeaveData struct {
	PlayerID   uint   `json:"playerId"`
	PlayerName string `json:"playerName"`
	Reason     string `json:"reason"`
}

type ExportRequest struct {
	Format    string  `json:"format" binding:"required,oneof=xlsx csv"`
	Dimension string  `json:"dimension" binding:"required,oneof=class exam time"`
	ClassIDs  []uint  `json:"classIds"`
	ExamID    *uint   `json:"examId"`
	StartTime *string `json:"startTime"`
	EndTime   *string `json:"endTime"`
}

type ExportTaskResponse struct {
	ID           uint   `json:"id"`
	Format       string `json:"format"`
	Dimension    string `json:"dimension"`
	Status       string `json:"status"`
	FileName     string `json:"fileName"`
	FileURL      string `json:"fileUrl"`
	FileSize     int64  `json:"fileSize"`
	TotalRecords int    `json:"totalRecords"`
	Progress     int    `json:"progress"`
	ErrorMsg     string `json:"errorMsg"`
	ExpiresAt    string `json:"expiresAt"`
	CreatedAt    string `json:"createdAt"`
	IsAsync      bool   `json:"isAsync"`
}

type ExportListFilter struct {
	Status   string `form:"status"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"pageSize,default=10"`
}

type ClassOverviewStat struct {
	ClassID    uint    `json:"classId"`
	ClassName  string  `json:"className"`
	StudentCount int   `json:"studentCount"`
	AvgScore   float64 `json:"avgScore"`
	MaxScore   int     `json:"maxScore"`
	MinScore   int     `json:"minScore"`
	PassRate   float64 `json:"passRate"`
	TotalScore int     `json:"totalScore"`
}

type StudentScoreItem struct {
	StudentID   uint   `json:"studentId"`
	StudentName string `json:"studentName"`
	ClassID     uint   `json:"classId"`
	ClassName   string `json:"className"`
	AttemptID   uint   `json:"attemptId"`
	Score       int    `json:"score"`
	Total       int    `json:"total"`
	Rate        string `json:"rate"`
	CreatedAt   string `json:"createdAt"`
}
