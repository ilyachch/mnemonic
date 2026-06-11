package project

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSlugifyUnicodeUnsupported(t *testing.T) {
	t.Parallel()

	got, err := Slugify("Привет мир")
	require.Empty(t, got)
	require.True(t, errors.Is(err, ErrUnsupportedSlugInput))
}

func TestSlugifyEmptyInput(t *testing.T) {
	t.Parallel()

	got, err := Slugify("")
	require.Empty(t, got)
	require.True(t, errors.Is(err, ErrEmptySlug))
}