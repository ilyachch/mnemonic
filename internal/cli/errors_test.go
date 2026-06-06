package cli

import (
	"errors"
	"testing"

	"github.com/ilyachch/mnemonic/internal/app"
)

func TestExitCodeForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "Nil error",
			err:  nil,
			want: 0,
		},
		{
			name: "Standard error",
			err:  errors.New("generic error"),
			want: 1,
		},
		{
			name: "CLI usage error",
			err:  app.NewCLIUsageError("invalid usage", nil),
			want: 2,
		},
		{
			name: "Not found error",
			err:  app.NewNotFoundError("resource not found", nil),
			want: 3,
		},
		{
			name: "Ambiguous error",
			err:  app.NewAmbiguousError("ambiguous input", nil),
			want: 4,
		},
		{
			name: "Unsafe error",
			err:  app.NewUnsafeError("precondition failed", nil),
			want: 5,
		},
		{
			name: "Corrupted error",
			err:  app.NewCorruptedError("corrupted index", nil),
			want: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCodeForError(tt.err); got != tt.want {
				t.Errorf("ExitCodeForError() = %v, want %v", got, tt.want)
			}
		})
	}
}
