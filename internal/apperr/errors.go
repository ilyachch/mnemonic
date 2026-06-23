package apperr

import "fmt"

// Code represents application exit codes and error classifications.
type Code int

const (
	CodeSuccess   Code = 0
	CodeInternal  Code = 1
	CodeCLIUsage  Code = 2
	CodeNotFound  Code = 3
	CodeAmbiguous Code = 4
	CodeUnsafe    Code = 5
	CodeCorrupted Code = 6
)

// Error is the standard application error wrapping a code and details.
type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// New creates a new Error with the specified code, message, and inner error.
func New(code Code, msg string, err error) *Error {
	return &Error{
		Code:    code,
		Message: msg,
		Err:     err,
	}
}

func Internal(msg string, err error) *Error {
	return New(CodeInternal, msg, err)
}

func CLIUsage(msg string, err error) *Error {
	return New(CodeCLIUsage, msg, err)
}

func NotFound(msg string, err error) *Error {
	return New(CodeNotFound, msg, err)
}

func Ambiguous(msg string, err error) *Error {
	return New(CodeAmbiguous, msg, err)
}

func Unsafe(msg string, err error) *Error {
	return New(CodeUnsafe, msg, err)
}

func Corrupted(msg string, err error) *Error {
	return New(CodeCorrupted, msg, err)
}
