package paths

import (
	"errors"
	"testing"
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
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tt.input)
				}
				if !tt.errCheck(err) {
					t.Fatalf("unexpected error for %q: %v", tt.input, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandTilde(t *testing.T) {
	t.Setenv("HOME", "/home/alice")

	got, err := ExpandTilde("~/notes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/home/alice/notes" {
		t.Fatalf("ExpandTilde returned %q, want %q", got, "/home/alice/notes")
	}
}
