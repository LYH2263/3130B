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

type ProctorEventItem struct {
	EventType string `json:"eventType" binding:"required,oneof=tab_switch blur copy paste fullscreen_exit reconnect"`
	EventTime int64  `json:"eventTime" binding:"required"`
	ExtraInfo string `json:"extraInfo"`
}

type ProctorReportRequest struct {
	ExamID  uint              `json:"examId" binding:"required"`
	Events  []ProctorEventItem `json:"events" binding:"required,min=1,max=50,dive"`
}

type ProctorReportResponse struct {
	ReportedCount    int    `json:"reportedCount"`
	ViolationScore   int    `json:"violationScore"`
	WarningThreshold int    `json:"warningThreshold"`
	ForceThreshold   int    `json:"forceThreshold"`
	Status           string `json:"status"`
	RemainingWarns   int    `json:"remainingWarns"`
}

type ProctorStudentStat struct {
	StudentID      uint            `json:"studentId"`
	StudentName    string          `json:"studentName"`
	TotalEvents    int             `json:"totalEvents"`
	ViolationScore int             `json:"violationScore"`
	EventBreakdown map[string]int  `json:"eventBreakdown"`
	Status         string          `json:"status"`
	LastEventTime  string          `json:"lastEventTime"`
}

type ProctorExamStatsResponse struct {
	ExamID      uint                 `json:"examId"`
	ExamName    string               `json:"examName"`
	TotalStudents int                 `json:"totalStudents"`
	TotalEvents  int                 `json:"totalEvents"`
	SuspiciousCount int              `json:"suspiciousCount"`
	WarningCount  int                 `json:"warningCount"`
	StudentStats []ProctorStudentStat `json:"studentStats"`
}

type ProctorConfigInput struct {
	WarningThreshold     int  `json:"warningThreshold" binding:"min=1"`
	ForceSubmitThreshold int  `json:"forceSubmitThreshold" binding:"min=1"`
	TabSwitchWeight      int  `json:"tabSwitchWeight" binding:"min=0"`
	BlurWeight           int  `json:"blurWeight" binding:"min=0"`
	CopyWeight           int  `json:"copyWeight" binding:"min=0"`
	PasteWeight          int  `json:"pasteWeight" binding:"min=0"`
	FullscreenExitWeight int  `json:"fullscreenExitWeight" binding:"min=0"`
	ReconnectWeight      int  `json:"reconnectWeight" binding:"min=0"`
	AutoForceSubmit      bool `json:"autoForceSubmit"`
	AutoMarkSuspicious   bool `json:"autoMarkSuspicious"`
	Enabled              bool `json:"enabled"`
}

type ProctorStudentStatusResponse struct {
	ExamID           uint           `json:"examId"`
	StudentID        uint           `json:"studentId"`
	TotalEvents      int            `json:"totalEvents"`
	ViolationScore   int            `json:"violationScore"`
	WarningThreshold int            `json:"warningThreshold"`
	ForceThreshold   int            `json:"forceThreshold"`
	Status           string         `json:"status"`
	RemainingWarns   int            `json:"remainingWarns"`
	EventBreakdown   map[string]int `json:"eventBreakdown"`
	RecentEvents     []ProctorEventBrief `json:"recentEvents"`
}

type ProctorEventBrief struct {
	ID        uint   `json:"id"`
	EventType string `json:"eventType"`
	EventTime string `json:"eventTime"`
	Severity  string `json:"severity"`
}

type DifficultyRuleInput struct {
	Level string  `json:"level" binding:"required,oneof=easy medium hard"`
	Count int     `json:"count" binding:"min=0"`
	Ratio float64 `json:"ratio" binding:"min=0,max=1"`
}

type QuestionTypeRuleInput struct {
	Type  string  `json:"type" binding:"required,oneof=single_choice multiple_choice true_false"`
	Count int     `json:"count" binding:"min=0"`
	Ratio float64 `json:"ratio" binding:"min=0,max=1"`
}

type KnowledgeTagRuleInput struct {
	Tag   string  `json:"tag" binding:"required,min=1,max=64"`
	Count int     `json:"count" binding:"min=0"`
	Ratio float64 `json:"ratio" binding:"min=0,max=1"`
}

type PaperRuleInput struct {
	TotalQuestions    int                     `json:"totalQuestions" binding:"required,min=1"`
	Difficulty        []DifficultyRuleInput   `json:"difficulty"`
	QuestionTypes     []QuestionTypeRuleInput `json:"questionTypes"`
	KnowledgeTags     []KnowledgeTagRuleInput `json:"knowledgeTags"`
	PerQuestionScore  int                     `json:"perQuestionScore" binding:"min=1"`
}

type PaperBlueprintCreateInput struct {
	Name            string          `json:"name" binding:"required,min=1,max=128"`
	Description     string          `json:"description" binding:"max=1000"`
	TotalScore      int             `json:"totalScore" binding:"required,min=1"`
	Rule            PaperRuleInput  `json:"rule" binding:"required"`
	AvoidRepeatDays int             `json:"avoidRepeatDays" binding:"min=0"`
}

type PaperBlueprintUpdateInput struct {
	Name            string          `json:"name" binding:"omitempty,min=1,max=128"`
	Description     string          `json:"description" binding:"omitempty,max=1000"`
	TotalScore      int             `json:"totalScore" binding:"omitempty,min=1"`
	Rule            *PaperRuleInput `json:"rule"`
	AvoidRepeatDays int             `json:"avoidRepeatDays" binding:"omitempty,min=0"`
}

type PaperBlueprintFilter struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"pageSize,default=20"`
	Keyword  string `form:"keyword"`
}

type PaperGenerateRequest struct {
	BlueprintID uint `json:"blueprintId" binding:"required"`
}

type PaperReplaceQuestionRequest struct {
	BlueprintID uint `json:"blueprintId" binding:"required"`
	CurrentQuestionID uint `json:"currentQuestionId" binding:"required"`
}

type PaperSaveRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=128"`
	Description string `json:"description" binding:"max=1000"`
	Status      string `json:"status" binding:"omitempty,oneof=draft published"`
}

type PaperSnapshotFilter struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"pageSize,default=20"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
}

type KnowledgeTagOption struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type PaperGenerateResult struct {
	BlueprintID uint                `json:"blueprintId"`
	Questions   []PaperQuestionItem `json:"questions"`
	TotalScore  int                 `json:"totalScore"`
	GapReport   *PaperGapReport     `json:"gapReport,omitempty"`
}

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
	TotalNeeded      int       `json:"totalNeeded"`
	TotalAvailable   int       `json:"totalAvailable"`
	DifficultyGaps   []GapItem `json:"difficultyGaps"`
	QuestionTypeGaps []GapItem `json:"questionTypeGaps"`
	KnowledgeTagGaps []GapItem `json:"knowledgeTagGaps"`
	CanGenerate      bool      `json:"canGenerate"`
	Messages         []string  `json:"messages"`
}

type GapItem struct {
	Name      string `json:"name"`
	Label     string `json:"label"`
	Needed    int    `json:"needed"`
	Available int    `json:"available"`
	Gap       int    `json:"gap"`
}
