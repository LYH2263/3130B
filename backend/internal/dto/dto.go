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
