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

	ErrSubjectiveQuestionNotFound = errors.New("subjective question not found")
	ErrSubjectiveSubmissionNotFound = errors.New("subjective submission not found")
	ErrInvalidScore              = errors.New("invalid score")
	ErrScoreExceedsFull          = errors.New("score exceeds full score")
	ErrAlreadyGraded             = errors.New("submission already graded")
	ErrConcurrentUpdate          = errors.New("concurrent update conflict")
	ErrSubmissionExists          = errors.New("submission already exists for this question")
	ErrQuestionInactive          = errors.New("question is inactive")
)
