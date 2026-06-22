package cli

import (
	"errors"
	"testing"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/stretchr/testify/assert"
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
			err:  apperr.CLIUsage("invalid usage", nil),
			want: 2,
		},
		{
			name: "Not found error",
			err:  apperr.NotFound("resource not found", nil),
			want: 3,
		},
		{
			name: "Ambiguous error",
			err:  apperr.Ambiguous("ambiguous input", nil),
			want: 4,
		},
		{
			name: "Unsafe error",
			err:  apperr.Unsafe("precondition failed", nil),
			want: 5,
		},
		{
			name: "Corrupted error",
			err:  apperr.Corrupted("corrupted index", nil),
			want: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExitCodeForError(tt.err))
		})
	}
}
