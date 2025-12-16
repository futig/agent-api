package entity

import "errors"

// Domain errors
var (
	// Project errors
	ErrProjectNotFound = errors.New("project not found")
	ErrInvalidProject  = errors.New("invalid project data")

	// File errors
	ErrInvalidFile       = errors.New("invalid file")
	ErrFileTooLarge      = errors.New("file too large")
	ErrTooManyFiles      = errors.New("too many files")
	ErrInvalidExtension  = errors.New("invalid file extension")
	ErrTotalSizeTooLarge = errors.New("total file size too large")

	// Session errors
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionNotActive     = errors.New("session is not active")
	ErrSessionCancelled     = errors.New("session is cancelled")
	ErrSessionCompleted     = errors.New("session is already completed")
	ErrInvalidSessionStatus = errors.New("invalid session status")
	ErrIterationNotFound    = errors.New("iteration not found")
	ErrIterationExists      = errors.New("iteration already exists")
	ErrInvalidIteration     = errors.New("invalid iteration number")
	ErrQuestionNotFound     = errors.New("question not found")
	ErrNoResult             = errors.New("session result not available")

	// Validation errors
	ErrMissingField     = errors.New("required field is missing")
	ErrInvalidFormat    = errors.New("invalid format")
	ErrInvalidParameter = errors.New("invalid parameter")
)
