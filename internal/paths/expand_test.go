package paths

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandPath(t *testing.T) {
	t.Setenv("HOME", "/home/alice")

	tests := []struct {
		name     string
		input    string
		want     string
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:  "tilde root",
			input: "~/.mnemonic",
			want:  "/home/alice/.mnemonic",
		},
		{
			name:  "tilde nested",
			input: "~/x/y",
			want:  "/home/alice/x/y",
		},
		{
			name:  "bare tilde",
			input: "~",
			want:  "/home/alice",
		},
		{
			name:  "unsupported user",
			input: "~otheruser/x",
			errCheck: func(err error) bool {
				return err.Error() == `unsupported home expansion: "~otheruser/x"`
			},
		},
		{
			name:  "empty",
			input: "",
			errCheck: func(err error) bool {
				return errors.Is(err, errEmptyPath)
			},
		},
		{
			name:  "plain path unchanged",
			input: "relative/path",
			want:  "relative/path",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.input)
			if tt.errCheck != nil {
				require.Error(t, err, "expected error for %q", tt.input)
				assert.True(t, tt.errCheck(err), "unexpected error for %q: %v", tt.input, err)
				return
			}

			require.NoError(t, err, "unexpected error for %q: %v", tt.input, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExpandTilde(t *testing.T) {
	t.Setenv("HOME", "/home/alice")

	got, err := ExpandTilde("~/notes")
	require.NoError(t, err)
	assert.Equal(t, "/home/alice/notes", got)
}
