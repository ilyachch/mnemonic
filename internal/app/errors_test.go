package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		want     string
		wantCode ErrCode
	}{
		{
			name:     "message only",
			err:      &AppError{Code: CodeNotFound, Message: "not found"},
			want:     "not found",
			wantCode: CodeNotFound,
		},
		{
			name:     "message with inner error",
			err:      &AppError{Code: CodeInternal, Message: "something broke", Err: errors.New("boom")},
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

func TestAppError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := &AppError{Code: CodeInternal, Message: "msg", Err: inner}
	unwrapped := errors.Unwrap(err)
	assert.Equal(t, inner, unwrapped)

	errNoInner := &AppError{Code: CodeSuccess, Message: "ok"}
	assert.Nil(t, errors.Unwrap(errNoInner))
}

func TestErrors_As(t *testing.T) {
	appErr := &AppError{Code: CodeUnsafe, Message: "precondition failed"}

	var target *AppError
	require.True(t, errors.As(appErr, &target))
	assert.Equal(t, CodeUnsafe, target.Code)
	assert.Equal(t, "precondition failed", target.Message)
}

func TestErrors_Is(t *testing.T) {
	inner := errors.New("inner")
	appErr := &AppError{Code: CodeInternal, Message: "wrapped", Err: inner}

	require.True(t, errors.Is(appErr, inner))
	require.False(t, errors.Is(appErr, errors.New("other")))
}

func TestNewAppError(t *testing.T) {
	inner := fmt.Errorf("disk full")
	err := NewAppError(CodeInternal, "write failed", inner)

	var appErr *AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, CodeInternal, appErr.Code)
	assert.Equal(t, "write failed", appErr.Message)
	assert.Equal(t, inner, appErr.Err)
}

func TestNewInternalError(t *testing.T) {
	err := NewInternalError("boom", nil)
	assert.Equal(t, CodeInternal, err.Code)
	assert.Equal(t, "boom", err.Message)

	errWithCause := NewInternalError("boom", errors.New("cause"))
	assert.Equal(t, CodeInternal, errWithCause.Code)
	assert.Equal(t, "boom", errWithCause.Message)
	assert.NotNil(t, errWithCause.Err)
}

func TestNewCLIUsageError(t *testing.T) {
	err := NewCLIUsageError("invalid flag", nil)
	assert.Equal(t, CodeCLIUsage, err.Code)
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("missing", nil)
	assert.Equal(t, CodeNotFound, err.Code)
}

func TestNewAmbiguousError(t *testing.T) {
	err := NewAmbiguousError("duplicate", nil)
	assert.Equal(t, CodeAmbiguous, err.Code)
}

func TestNewUnsafeError(t *testing.T) {
	err := NewUnsafeError("precondition", nil)
	assert.Equal(t, CodeUnsafe, err.Code)
}

func TestNewCorruptedError(t *testing.T) {
	err := NewCorruptedError("corrupt", nil)
	assert.Equal(t, CodeCorrupted, err.Code)
}

func TestApp_Close(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var a *App
		assert.NoError(t, a.Close())
	})

	t.Run("nil registry", func(t *testing.T) {
		a := &App{Services: Services{}}
		assert.NoError(t, a.Close())
	})
}
