package slug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{
			name:  "simple word",
			input: "Hello",
			want:  "hello",
		},
		{
			name:  "multiple words with spaces",
			input: "Hello World",
			want:  "hello-world",
		},
		{
			name:  "mixed case and digits",
			input: "My 1st Note",
			want:  "my-1st-note",
		},
		{
			name:  "special characters become dashes",
			input: "foo@bar#baz",
			want:  "foo-bar-baz",
		},
		{
			name:  "consecutive special chars become single dash",
			input: "foo!!!bar???baz",
			want:  "foo-bar-baz",
		},
		{
			name:  "leading special chars trimmed",
			input: "!!!Hello",
			want:  "hello",
		},
		{
			name:  "trailing special chars trimmed",
			input: "Hello!!!",
			want:  "hello",
		},
		{
			name:  "leading and trailing whitespace-like chars",
			input: "  Hello World  ",
			want:  "hello-world",
		},
		{
			name:  "underscores become dashes",
			input: "hello_world",
			want:  "hello-world",
		},
		{
			name:  "dots become dashes",
			input: "hello.world",
			want:  "hello-world",
		},
		{
			name:  "only letters and digits",
			input: "abc123",
			want:  "abc123",
		},
		{
			name:    "empty string returns ErrEmptySlug",
			input:   "",
			wantErr: ErrEmptySlug,
		},
		{
			name:    "only special chars returns ErrEmptySlug",
			input:   "!!!???",
			wantErr: ErrEmptySlug,
		},
		{
			name:    "unicode input returns ErrUnsupportedSlugInput",
			input:   "привет",
			wantErr: ErrUnsupportedSlugInput,
		},
		{
			name:    "mixed unicode in middle",
			input:   "hello мир world",
			wantErr: ErrUnsupportedSlugInput,
		},
		{
			name:    "single emoji",
			input:   "🎉",
			wantErr: ErrUnsupportedSlugInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Slugify(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
