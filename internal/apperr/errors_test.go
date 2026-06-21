package apperr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		want     string
		wantCode Code
	}{
		{
			name:     "message only",
			err:      &Error{Code: CodeNotFound, Message: "not found"},
			want:     "not found",
			wantCode: CodeNotFound,
		},
		{
			name:     "message with inner error",
			err:      &Error{Code: CodeInternal, Message: "something broke", Err: errors.New("boom")},
			want:     "something broke: boom",
			wantCode: CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
			assert.Equal(t, tt.wantCode, tt.err.Code)
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := &Error{Code: CodeInternal, Message: "msg", Err: inner}
	unwrapped := errors.Unwrap(err)
	assert.Equal(t, inner, unwrapped)

	errNoInner := &Error{Code: CodeSuccess, Message: "ok"}
	assert.Nil(t, errors.Unwrap(errNoInner))
}

func TestErrors_As(t *testing.T) {
	appErr := &Error{Code: CodeUnsafe, Message: "precondition failed"}

	var target *Error
	require.True(t, errors.As(appErr, &target))
	assert.Equal(t, CodeUnsafe, target.Code)
	assert.Equal(t, "precondition failed", target.Message)
}

func TestErrors_Is(t *testing.T) {
	inner := errors.New("inner")
	appErr := &Error{Code: CodeInternal, Message: "wrapped", Err: inner}

	require.True(t, errors.Is(appErr, inner))
	require.False(t, errors.Is(appErr, errors.New("other")))
}

func TestNew(t *testing.T) {
	inner := fmt.Errorf("disk full")
	err := New(CodeInternal, "write failed", inner)

	var appErr *Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, CodeInternal, appErr.Code)
	assert.Equal(t, "write failed", appErr.Message)
	assert.Equal(t, inner, appErr.Err)
}

func TestInternal(t *testing.T) {
	err := Internal("boom", nil)
	assert.Equal(t, CodeInternal, err.Code)
	assert.Equal(t, "boom", err.Message)

	errWithCause := Internal("boom", errors.New("cause"))
	assert.Equal(t, CodeInternal, errWithCause.Code)
	assert.Equal(t, "boom", errWithCause.Message)
	assert.NotNil(t, errWithCause.Err)
}

func TestCLIUsage(t *testing.T) {
	err := CLIUsage("invalid flag", nil)
	assert.Equal(t, CodeCLIUsage, err.Code)
}

func TestNotFound(t *testing.T) {
	err := NotFound("missing", nil)
	assert.Equal(t, CodeNotFound, err.Code)
}

func TestAmbiguous(t *testing.T) {
	err := Ambiguous("duplicate", nil)
	assert.Equal(t, CodeAmbiguous, err.Code)
}

func TestUnsafe(t *testing.T) {
	err := Unsafe("precondition", nil)
	assert.Equal(t, CodeUnsafe, err.Code)
}

func TestCorrupted(t *testing.T) {
	err := Corrupted("corrupt", nil)
	assert.Equal(t, CodeCorrupted, err.Code)
}
