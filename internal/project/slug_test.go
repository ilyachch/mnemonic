package project

import (
	"errors"
	"testing"
)

func TestSlugifyASCIIExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "backend",
			input: "Backend",
			want:  "backend",
		},
		{
			name:  "migration plan",
			input: "Auth migration plan",
			want:  "auth-migration-plan",
		},
		{
			name:  "collapse separators",
			input: "  A__B  C  ",
			want:  "a-b-c",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Slugify(tt.input)
			if err != nil {
				t.Fatalf("Slugify() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Slugify() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSlugifyUnicodeUnsupported(t *testing.T) {
	t.Parallel()

	got, err := Slugify("Привет мир")
	if got != "" {
		t.Fatalf("Slugify() = %q, want empty slug on unsupported input", got)
	}
	if !errors.Is(err, ErrUnsupportedSlugInput) {
		t.Fatalf("Slugify() error = %v, want ErrUnsupportedSlugInput", err)
	}
}

func TestSlugifyEmptyInput(t *testing.T) {
	t.Parallel()

	got, err := Slugify("")
	if got != "" {
		t.Fatalf("Slugify() = %q, want empty slug", got)
	}
	if !errors.Is(err, ErrEmptySlug) {
		t.Fatalf("Slugify() error = %v, want ErrEmptySlug", err)
	}
}
