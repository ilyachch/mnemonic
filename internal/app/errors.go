package app

import (
	"fmt"
)

// ErrCode represents application exit codes and error classifications.
type ErrCode int

const (
	CodeSuccess   ErrCode = 0
	CodeInternal  ErrCode = 1
	CodeCLIUsage  ErrCode = 2
	CodeNotFound  ErrCode = 3
	CodeAmbiguous ErrCode = 4
	CodeUnsafe    ErrCode = 5
	CodeCorrupted ErrCode = 6
)

// AppError is the standard application error wrapping a code and details.
type AppError struct {
	Code    ErrCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new AppError with the specified code, message, and inner error.
func NewAppError(code ErrCode, msg string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: msg,
		Err:     err,
	}
}

// Helper functions to create specific types of errors

func NewInternalError(msg string, err error) *AppError {
	return NewAppError(CodeInternal, msg, err)
}

func NewCLIUsageError(msg string, err error) *AppError {
	return NewAppError(CodeCLIUsage, msg, err)
}

func NewNotFoundError(msg string, err error) *AppError {
	return NewAppError(CodeNotFound, msg, err)
}

func NewAmbiguousError(msg string, err error) *AppError {
	return NewAppError(CodeAmbiguous, msg, err)
}

func NewUnsafeError(msg string, err error) *AppError {
	return NewAppError(CodeUnsafe, msg, err)
}

func NewCorruptedError(msg string, err error) *AppError {
	return NewAppError(CodeCorrupted, msg, err)
}
