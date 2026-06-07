package service

import "errors"

var (
	ErrUserExists        = errors.New("username already exists")
	ErrInvalidCredential = errors.New("invalid credentials")
	ErrClassNotFound     = errors.New("class not found")
	ErrInvalidQuestion   = errors.New("question must contain exactly one correct option")
	ErrQuestionNotFound  = errors.New("question not found")
	ErrNoQuestions       = errors.New("question bank is empty")
	ErrInvalidSubmission = errors.New("invalid submission")

	ErrSubjectiveQuestionNotFound   = errors.New("subjective question not found")
	ErrSubjectiveSubmissionNotFound = errors.New("subjective submission not found")
	ErrInvalidScore                 = errors.New("invalid score")
	ErrScoreExceedsFull             = errors.New("score exceeds full score")
	ErrAlreadyGraded                = errors.New("submission already graded")
	ErrConcurrentUpdate             = errors.New("concurrent update conflict")
	ErrSubmissionExists             = errors.New("submission already exists for this question")
	ErrQuestionInactive             = errors.New("question is inactive")

	ErrExamNotFound           = errors.New("exam not found")
	ErrExamTimeConflict       = errors.New("exam time conflict for same class")
	ErrExamInvalidTimeRange   = errors.New("invalid exam time range")
	ErrExamNotStarted         = errors.New("exam has not started")
	ErrExamAlreadyEnded       = errors.New("exam has already ended")
	ErrExamCancelled          = errors.New("exam has been cancelled")
	ErrNotInExamClass         = errors.New("student is not in the exam class")
	ErrParticipantNotFound    = errors.New("exam participant not found")
	ErrAlreadySubmitted       = errors.New("exam already submitted")
	ErrExamInProgress         = errors.New("exam is in progress, cannot modify")
	ErrInvalidExamStatus      = errors.New("invalid exam status")

	ErrDiscussionNotFound     = errors.New("discussion not found")
	ErrInvalidDiscussion      = errors.New("invalid discussion content")
	ErrReplyTooDeep           = errors.New("replies only support two levels")
	ErrCannotDeleteDiscussion = errors.New("no permission to delete this discussion")

	ErrExportTaskNotFound   = errors.New("export task not found")
	ErrExportFormatInvalid  = errors.New("invalid export format")
	ErrExportDimensionInvalid = errors.New("invalid export dimension")
	ErrExportGenerateFailed = errors.New("export generation failed")
	ErrExportExpired        = errors.New("export file expired")
	ErrExportNoData         = errors.New("no data to export")

	ErrVersionNotFound  = errors.New("version not found")
	ErrVersionMismatch  = errors.New("version does not belong to this question")
)
