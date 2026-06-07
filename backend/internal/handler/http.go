package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"label3130/backend/internal/auth"
	"label3130/backend/internal/dto"
	"label3130/backend/internal/middleware"
	"label3130/backend/internal/models"
	"label3130/backend/internal/service"
)

type HTTPHandler struct {
	authSvc       *service.AuthService
	questionSvc   *service.QuestionService
	attemptSvc    *service.AttemptService
	subjectiveSvc *service.SubjectiveService
	examSvc       *service.ExamService
	discussionSvc *service.DiscussionService
	checkinSvc    *service.CheckinService
	pkSvc         *service.PkService
	tokens        *auth.TokenManager
	log           *slog.Logger
	wsUpgrader    *websocket.Upgrader
}

func New(
	authSvc *service.AuthService,
	questionSvc *service.QuestionService,
	attemptSvc *service.AttemptService,
	subjectiveSvc *service.SubjectiveService,
	examSvc *service.ExamService,
	discussionSvc *service.DiscussionService,
	checkinSvc *service.CheckinService,
	pkSvc *service.PkService,
	tokens *auth.TokenManager,
	log *slog.Logger,
) *HTTPHandler {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return &HTTPHandler{
		authSvc:       authSvc,
		questionSvc:   questionSvc,
		attemptSvc:    attemptSvc,
		subjectiveSvc: subjectiveSvc,
		examSvc:       examSvc,
		discussionSvc: discussionSvc,
		checkinSvc:    checkinSvc,
		pkSvc:         pkSvc,
		tokens:        tokens,
		log:           log,
		wsUpgrader:    upgrader,
	}
}

func (h *HTTPHandler) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(h.requestLogger())
	r.Use(cors())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.GET("/classes", h.listClasses)
		api.POST("/auth/register", h.register)
		api.POST("/auth/login", h.login)

		authed := api.Group("", middleware.AuthRequired(h.tokens))
		{
			authed.GET("/me", h.me)

			teacher := authed.Group("/teacher", middleware.RequireRole(models.RoleTeacher))
			{
				teacher.GET("/overview", h.teacherOverview)
				teacher.GET("/class-stats", h.teacherClassStats)
				teacher.GET("/attempts", h.teacherAttempts)

				teacher.GET("/questions", h.listQuestions)
				teacher.POST("/questions", h.createQuestion)
				teacher.PUT("/questions/:id", h.updateQuestion)
				teacher.DELETE("/questions/:id", h.deleteQuestion)
				teacher.POST("/questions/upload", h.uploadQuestions)

				teacher.GET("/subjective-questions", h.listSubjectiveQuestions)
				teacher.GET("/subjective-questions/:id", h.getSubjectiveQuestion)
				teacher.POST("/subjective-questions", h.createSubjectiveQuestion)
				teacher.PUT("/subjective-questions/:id", h.updateSubjectiveQuestion)
				teacher.DELETE("/subjective-questions/:id", h.deleteSubjectiveQuestion)

				teacher.GET("/subjective-submissions", h.listSubjectiveSubmissions)
				teacher.GET("/subjective-submissions/:id", h.getSubjectiveSubmission)
				teacher.POST("/subjective-submissions/:id/grade", h.gradeSubjectiveSubmission)
				teacher.GET("/subjective-pending-count", h.subjectivePendingCount)

				teacher.GET("/exams", h.listExams)
				teacher.GET("/exams/:id", h.getExam)
				teacher.POST("/exams", h.createExam)
				teacher.PUT("/exams/:id", h.updateExam)
				teacher.DELETE("/exams/:id", h.deleteExam)
				teacher.GET("/exams/:id/participants", h.getExamParticipants)
			}

			student := authed.Group("/student", middleware.RequireRole(models.RoleStudent))
			{
				student.GET("/questions", h.studentQuestions)
				student.POST("/submit", h.submit)
				student.GET("/mistakes", h.studentMistakes)
				student.GET("/attempts", h.studentAttempts)

				student.GET("/subjective-questions", h.studentSubjectiveQuestions)
				student.GET("/subjective-questions/:id", h.studentSubjectiveQuestion)
				student.POST("/subjective-submit", h.studentSubjectiveSubmit)
				student.GET("/subjective-submissions", h.studentSubjectiveSubmissions)
				student.GET("/subjective-submissions/:id", h.studentSubjectiveSubmission)

				student.GET("/exams", h.studentExams)
				student.GET("/exams/:id", h.studentExamDetail)
				student.POST("/exams/:id/enter", h.enterExam)
				student.POST("/exams/:id/submit", h.submitExam)
				student.GET("/exams/:id/result", h.examResult)

				student.GET("/discussions", h.listDiscussions)
				student.GET("/discussions/replies", h.listReplies)
				student.POST("/discussions", h.createDiscussion)
				student.POST("/discussions/:id/like", h.toggleLike)
				student.DELETE("/discussions/:id", h.deleteDiscussion)

				student.GET("/checkin/status", h.getCheckinStatus)
				student.POST("/checkin", h.manualCheckin)
				student.GET("/checkin/calendar", h.getCheckinCalendar)
				student.GET("/checkin/badges", h.getUserBadges)

				student.POST("/pk/rooms", h.createPkRoom)
				student.POST("/pk/rooms/join", h.joinPkRoom)
				student.GET("/pk/rooms/:code", h.getPkRoom)
				student.GET("/pk/rooms/:id/results", h.getPkRoundResults)
			}

			teacher := authed.Group("/teacher", middleware.RequireRole(models.RoleTeacher))
			{
				teacher.GET("/discussions", h.listDiscussions)
				teacher.GET("/discussions/replies", h.listReplies)
				teacher.POST("/discussions", h.createDiscussion)
				teacher.POST("/discussions/:id/like", h.toggleLike)
				teacher.DELETE("/discussions/:id", h.deleteDiscussion)
			}
		}
	}

	r.GET("/api/pk/ws/:roomCode", h.pkWebSocket)

	return r
}

func (h *HTTPHandler) register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid register payload"})
		return
	}
	result, err := h.authSvc.RegisterStudent(req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *HTTPHandler) login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid login payload"})
		return
	}
	result, err := h.authSvc.Login(req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *HTTPHandler) me(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	user, err := h.authSvc.GetUser(claims.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *HTTPHandler) listClasses(c *gin.Context) {
	classes, err := h.authSvc.ListClasses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load classes"})
		return
	}
	c.JSON(http.StatusOK, classes)
}

func (h *HTTPHandler) listQuestions(c *gin.Context) {
	questions, err := h.questionSvc.ListQuestions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load questions"})
		return
	}
	c.JSON(http.StatusOK, questions)
}

func (h *HTTPHandler) createQuestion(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var req dto.QuestionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid question payload"})
		return
	}
	question, err := h.questionSvc.CreateQuestion(req, claims.UserID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, question)
}

func (h *HTTPHandler) updateQuestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid question id"})
		return
	}
	var req dto.QuestionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid question payload"})
		return
	}
	question, err := h.questionSvc.UpdateQuestion(uint(id), req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, question)
}

func (h *HTTPHandler) deleteQuestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid question id"})
		return
	}
	if err := h.questionSvc.DeleteQuestion(uint(id)); err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "question deleted"})
}

func (h *HTTPHandler) uploadQuestions(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing file"})
		return
	}
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "open file failed"})
		return
	}
	defer opened.Close()

	data, err := io.ReadAll(opened)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "read file failed"})
		return
	}

	count, err := h.questionSvc.UploadFromJSON(data, claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "upload success", "count": count})
}

func (h *HTTPHandler) teacherOverview(c *gin.Context) {
	overview, err := h.attemptSvc.Overview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "load overview failed"})
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *HTTPHandler) teacherClassStats(c *gin.Context) {
	stats, err := h.attemptSvc.ClassWrongStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "load class stats failed"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *HTTPHandler) teacherAttempts(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	items, err := h.attemptSvc.TeacherRecentAttempts(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "load attempts failed"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *HTTPHandler) studentQuestions(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	questions, err := h.questionSvc.GetQuizQuestions(limit)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, questions)
}

func (h *HTTPHandler) submit(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims.ClassID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid student context"})
		return
	}

	var req dto.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid submit payload"})
		return
	}
	result, err := h.attemptSvc.Submit(claims.UserID, *claims.ClassID, req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	checkinResult, checkinErr := h.checkinSvc.AutoCheckin(claims.UserID, result.Total, result.Score)
	if checkinErr != nil {
		h.log.Warn("auto checkin failed", "error", checkinErr.Error(), "userID", claims.UserID)
	}

	response := gin.H{
		"attemptId": result.AttemptID,
		"score":     result.Score,
		"total":     result.Total,
		"rate":      result.Rate,
	}

	if checkinResult != nil && checkinResult.CheckedIn {
		response["checkin"] = checkinResult
		if len(checkinResult.NewlyAwarded) > 0 {
			response["newlyAwarded"] = checkinResult.NewlyAwarded
		}
	}

	c.JSON(http.StatusCreated, response)
}

func (h *HTTPHandler) studentMistakes(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	items, err := h.attemptSvc.StudentMistakes(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "load mistakes failed"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *HTTPHandler) studentAttempts(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	items, err := h.attemptSvc.StudentAttempts(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "load attempts failed"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *HTTPHandler) listSubjectiveQuestions(c *gin.Context) {
	questions, err := h.subjectiveSvc.ListQuestions()
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, questions)
}

func (h *HTTPHandler) getSubjectiveQuestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid question id"})
		return
	}
	question, err := h.subjectiveSvc.GetQuestion(uint(id))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, question)
}

func (h *HTTPHandler) createSubjectiveQuestion(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var req dto.SubjectiveQuestionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payload"})
		return
	}
	question, err := h.subjectiveSvc.CreateQuestion(req, claims.UserID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, question)
}

func (h *HTTPHandler) updateSubjectiveQuestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid question id"})
		return
	}
	var req dto.SubjectiveQuestionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payload"})
		return
	}
	question, err := h.subjectiveSvc.UpdateQuestion(uint(id), req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, question)
}

func (h *HTTPHandler) deleteSubjectiveQuestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid question id"})
		return
	}
	if err := h.subjectiveSvc.DeleteQuestion(uint(id)); err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "question deleted"})
}

func (h *HTTPHandler) listSubjectiveSubmissions(c *gin.Context) {
	var filter dto.SubjectiveSubmissionFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid filter"})
		return
	}
	submissions, total, err := h.subjectiveSvc.ListSubmissions(filter)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": submissions, "total": total, "page": filter.Page, "pageSize": filter.PageSize})
}

func (h *HTTPHandler) getSubjectiveSubmission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid submission id"})
		return
	}
	submission, err := h.subjectiveSvc.GetSubmission(uint(id))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, submission)
}

func (h *HTTPHandler) gradeSubjectiveSubmission(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid submission id"})
		return
	}
	var req dto.SubjectiveGradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid grade payload"})
		return
	}
	submission, err := h.subjectiveSvc.GradeSubmission(uint(id), claims.UserID, req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, submission)
}

func (h *HTTPHandler) subjectivePendingCount(c *gin.Context) {
	count, err := h.subjectiveSvc.GetPendingCount()
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

func (h *HTTPHandler) studentSubjectiveQuestions(c *gin.Context) {
	questions, err := h.subjectiveSvc.StudentListQuestions()
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, questions)
}

func (h *HTTPHandler) studentSubjectiveQuestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid question id"})
		return
	}
	question, err := h.subjectiveSvc.GetStudentQuestion(uint(id))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, question)
}

func (h *HTTPHandler) studentSubjectiveSubmit(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var req dto.SubjectiveSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid submit payload"})
		return
	}
	submission, err := h.subjectiveSvc.SubmitAnswer(claims.UserID, req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, submission)
}

func (h *HTTPHandler) studentSubjectiveSubmissions(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	submissions, err := h.subjectiveSvc.GetStudentSubmissions(claims.UserID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, submissions)
}

func (h *HTTPHandler) studentSubjectiveSubmission(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid submission id"})
		return
	}
	submission, err := h.subjectiveSvc.GetStudentSubmission(claims.UserID, uint(id))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, submission)
}

func (h *HTTPHandler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserExists):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrInvalidCredential):
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrClassNotFound), errors.Is(err, service.ErrQuestionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrInvalidQuestion), errors.Is(err, service.ErrInvalidSubmission):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrNoQuestions):
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrSubjectiveQuestionNotFound), errors.Is(err, service.ErrSubjectiveSubmissionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrInvalidScore), errors.Is(err, service.ErrScoreExceedsFull), errors.Is(err, service.ErrQuestionInactive):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrAlreadyGraded), errors.Is(err, service.ErrSubmissionExists):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrConcurrentUpdate):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrExamNotFound), errors.Is(err, service.ErrParticipantNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrExamTimeConflict), errors.Is(err, service.ErrAlreadySubmitted):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrExamInvalidTimeRange), errors.Is(err, service.ErrInvalidExamStatus):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrExamNotStarted), errors.Is(err, service.ErrExamAlreadyEnded),
		errors.Is(err, service.ErrExamCancelled), errors.Is(err, service.ErrNotInExamClass),
		errors.Is(err, service.ErrExamInProgress):
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrDiscussionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrInvalidDiscussion), errors.Is(err, service.ErrReplyTooDeep):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrCannotDeleteDiscussion):
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrPkRoomNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrPkRoomFull):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrPkRoomNotWaiting), errors.Is(err, service.ErrPkGameEnded),
		errors.Is(err, service.ErrPkAlreadyInRoom), errors.Is(err, service.ErrPkNotInRoom),
		errors.Is(err, service.ErrPkInvalidAnswer):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	default:
		h.log.Error("service error", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
	}
}

func (h *HTTPHandler) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		h.log.Info("http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
		)
	}
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (h *HTTPHandler) listExams(c *gin.Context) {
	var filter dto.ExamFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid filter"})
		return
	}
	exams, total, err := h.examSvc.ListExams(filter)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": exams, "total": total, "page": filter.Page, "pageSize": filter.PageSize})
}

func (h *HTTPHandler) getExam(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid exam id"})
		return
	}
	exam, err := h.examSvc.GetExam(uint(id))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, exam)
}

func (h *HTTPHandler) createExam(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var req dto.ExamCreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid exam payload"})
		return
	}
	exam, err := h.examSvc.CreateExam(req, claims.UserID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, exam)
}

func (h *HTTPHandler) updateExam(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid exam id"})
		return
	}
	var req dto.ExamUpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid exam payload"})
		return
	}
	exam, err := h.examSvc.UpdateExam(uint(id), req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, exam)
}

func (h *HTTPHandler) deleteExam(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid exam id"})
		return
	}
	if err := h.examSvc.DeleteExam(uint(id)); err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "exam deleted"})
}

func (h *HTTPHandler) getExamParticipants(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid exam id"})
		return
	}
	participants, err := h.examSvc.GetExamParticipants(uint(id))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, participants)
}

func (h *HTTPHandler) studentExams(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	exams, err := h.examSvc.GetStudentExams(claims.UserID, claims.ClassID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, exams)
}

func (h *HTTPHandler) studentExamDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid exam id"})
		return
	}
	exam, err := h.examSvc.GetExam(uint(id))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, exam)
}

func (h *HTTPHandler) enterExam(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims.ClassID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid student context"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid exam id"})
		return
	}
	exam, participant, err := h.examSvc.EnterExam(claims.UserID, *claims.ClassID, uint(id))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"exam": exam, "participant": participant})
}

func (h *HTTPHandler) submitExam(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid exam id"})
		return
	}
	var req dto.ExamSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid submit payload"})
		return
	}
	participant, err := h.examSvc.SubmitExam(claims.UserID, uint(id), req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, participant)
}

func (h *HTTPHandler) examResult(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid exam id"})
		return
	}
	participant, err := h.examSvc.GetStudentParticipant(claims.UserID, uint(id))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, participant)
}

func (h *HTTPHandler) createDiscussion(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var req dto.CreateDiscussionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid discussion payload"})
		return
	}
	discussion, err := h.discussionSvc.CreateDiscussion(claims.UserID, claims.Role, req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, discussion)
}

func (h *HTTPHandler) listDiscussions(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var filter dto.DiscussionFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid filter"})
		return
	}
	discussions, total, err := h.discussionSvc.ListDiscussions(filter, claims.UserID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	ids := make([]uint, 0, len(discussions))
	for _, d := range discussions {
		ids = append(ids, d.ID)
	}
	likedMap, _ := h.discussionSvc.GetUserLikedMap(ids, claims.UserID)

	type discussionWithLike struct {
		*models.Discussion
		IsLiked bool `json:"isLiked"`
	}

	result := make([]discussionWithLike, len(discussions))
	for i, d := range discussions {
		result[i] = discussionWithLike{Discussion: &discussions[i], IsLiked: likedMap[d.ID]}
	}

	c.JSON(http.StatusOK, gin.H{"items": result, "total": total, "page": filter.Page, "pageSize": filter.PageSize})
}

func (h *HTTPHandler) listReplies(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var filter dto.ReplyFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid filter"})
		return
	}
	replies, total, err := h.discussionSvc.ListReplies(filter, claims.UserID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	ids := make([]uint, 0, len(replies))
	for _, r := range replies {
		ids = append(ids, r.ID)
	}
	likedMap, _ := h.discussionSvc.GetUserLikedMap(ids, claims.UserID)

	type replyWithLike struct {
		*models.Discussion
		IsLiked bool `json:"isLiked"`
	}

	result := make([]replyWithLike, len(replies))
	for i, r := range replies {
		result[i] = replyWithLike{Discussion: &replies[i], IsLiked: likedMap[r.ID]}
	}

	c.JSON(http.StatusOK, gin.H{"items": result, "total": total, "page": filter.Page, "pageSize": filter.PageSize})
}

func (h *HTTPHandler) toggleLike(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid discussion id"})
		return
	}
	isLiked, likeCount, err := h.discussionSvc.ToggleLike(uint(id), claims.UserID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"isLiked": isLiked, "likeCount": likeCount})
}

func (h *HTTPHandler) deleteDiscussion(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid discussion id"})
		return
	}
	if err := h.discussionSvc.DeleteDiscussion(uint(id), claims.UserID, claims.Role); err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "discussion deleted"})
}

func (h *HTTPHandler) getCheckinStatus(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	status, err := h.checkinSvc.GetCheckinStatus(claims.UserID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *HTTPHandler) manualCheckin(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var req dto.ManualCheckinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid checkin payload"})
		return
	}
	result, err := h.checkinSvc.ManualCheckin(claims.UserID, req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *HTTPHandler) getCheckinCalendar(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var req dto.CheckinCalendarRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid calendar query"})
		return
	}
	calendar, err := h.checkinSvc.GetCalendar(claims.UserID, req.Year, req.Month)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, calendar)
}

func (h *HTTPHandler) getUserBadges(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	badges, err := h.checkinSvc.GetUserBadges(claims.UserID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, badges)
}

func (h *HTTPHandler) createPkRoom(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var req dto.CreatePkRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payload"})
		return
	}
	result, err := h.pkSvc.CreateRoom(claims.UserID, claims.Username, req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *HTTPHandler) joinPkRoom(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}
	var req dto.JoinPkRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payload"})
		return
	}
	result, err := h.pkSvc.JoinRoom(claims.UserID, claims.Username, req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *HTTPHandler) getPkRoom(c *gin.Context) {
	roomCode := c.Param("code")
	result, err := h.pkSvc.GetRoomInfo(roomCode)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *HTTPHandler) getPkRoundResults(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid room id"})
		return
	}
	results, err := h.pkSvc.GetRoundResults(uint(id))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, results)
}

func (h *HTTPHandler) pkWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
			token = authHeader[7:]
		}
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "missing token"})
		return
	}

	claims, err := h.tokens.Parse(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}

	roomCode := c.Param("roomCode")
	if roomCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing room code"})
		return
	}

	conn, err := h.wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Warn("ws upgrade failed", "error", err.Error())
		return
	}

	h.pkSvc.HandleWebSocket(conn, claims.UserID, claims.Username, roomCode)
}
