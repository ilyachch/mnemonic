package markdown

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNoteReadsCanonicalFieldsAndPreservesBody(t *testing.T) {
	t.Parallel()

	input := []byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440000\n" +
		"title: Example Note\n" +
		"slug: example-note\n" +
		"tags:\n" +
		"  - django\n" +
		"  - auth\n" +
		"created_at: 1780394400\n" +
		"updated_at: 1780398000\n" +
		"type: decision\n" +
		"permalink: example-note\n" +
		"extra_field: keep-me\n" +
		"---\n" +
		"# Heading\n" +
		"Body text\n")

	note, err := ParseNote(input)
	require.NoError(t, err)

	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", note.MnemonicNoteID)
	require.Equal(t, "Example Note", note.Title)
	require.Equal(t, "example-note", note.Slug)
	require.Equal(t, []string{"django", "auth"}, note.Tags)
	wantCreatedAt := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	require.True(t, note.CreatedAt.Equal(wantCreatedAt))
	wantUpdatedAt := time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC)
	require.True(t, note.UpdatedAt.Equal(wantUpdatedAt))
	require.Equal(t, "decision", note.Type)
	require.Equal(t, 20, len(note.Body))
	require.True(t, bytes.HasSuffix(note.Body, []byte("Body text\n")))
}

func TestParseNoteParsesSummaryAndAliases(t *testing.T) {
	t.Parallel()

	input := []byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440000\n" +
		"title: Note With Extras\n" +
		"slug: note-with-extras\n" +
		"summary: A brief summary of the note\n" +
		"aliases:\n" +
		"  - alias-one\n" +
		"  - alias-two\n" +
		"created_at: 1780394400\n" +
		"updated_at: 1780394400\n" +
		"---\n" +
		"Body\n")

	note, err := ParseNote(input)
	require.NoError(t, err)

	require.Equal(t, "Note With Extras", note.Title)
	require.Equal(t, "A brief summary of the note", note.Summary)
	require.Equal(t, []string{"alias-one", "alias-two"}, note.Aliases)
}

func TestParseNoteWithoutSlugReturnsEmpty(t *testing.T) {
	t.Parallel()

	note, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440001\n" +
		"title: Permalink Note\n" +
		"permalink: permalink-note\n" +
		"created_at: 1780394400\n" +
		"updated_at: 1780394400\n" +
		"---\n" +
		"Body\n"))
	require.NoError(t, err)
	require.Empty(t, note.Slug)
}

func TestParseNoteWithoutFrontmatterReturnsFullBody(t *testing.T) {
	t.Parallel()

	input := []byte("# Heading\nBody\n")
	note, err := ParseNote(input)
	require.NoError(t, err)
	require.Empty(t, note.Frontmatter)
	require.True(t, bytes.Equal(note.Body, input))
}

func TestParseNoteRejectsRFC3339StringTimestamp(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440002\n" +
		"title: Bad Note\n" +
		"created_at: 2026-06-02T10:00:00Z\n" +
		"updated_at: 1780394400\n" +
		"---\n" +
		"Body\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "created_at")
}

func TestParseNoteRejectsStringTimestamp(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440003\n" +
		"title: Bad Note\n" +
		"created_at: 1780394400\n" +
		"updated_at: \"2026-06-02T10:00:00Z\"\n" +
		"---\n" +
		"Body\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "updated_at")
}

func TestNoteTimeField_IntType(t *testing.T) {
	t.Parallel()

	raw := map[string]any{"created_at": 1780394400}
	result, err := noteTimeField(raw, "created_at")
	require.NoError(t, err)
	want := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	require.Equal(t, want, result)
}

func TestNoteTimeField_MissingField(t *testing.T) {
	t.Parallel()

	raw := map[string]any{"other": "value"}
	result, err := noteTimeField(raw, "created_at")
	require.NoError(t, err)
	require.True(t, result.IsZero())
}

func TestNoteTimeField_NilValue(t *testing.T) {
	t.Parallel()

	raw := map[string]any{"created_at": nil}
	result, err := noteTimeField(raw, "created_at")
	require.NoError(t, err)
	require.True(t, result.IsZero())
}

func TestFrontmatterFieldError_Timestamp(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440002\n" +
		"title: Bad Note\n" +
		"created_at: yesterday\n" +
		"updated_at: 1780394400\n" +
		"---\n" +
		"Body\n"))
	require.Error(t, err)

	var fieldErr *FrontmatterFieldError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "created_at", fieldErr.Field)
	assert.Equal(t, FieldErrKindInvalidTimestamp, fieldErr.Kind)
}

func TestFrontmatterFieldError_TagsInvalidType(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440002\n" +
		"title: Note\n" +
		"created_at: 1780394400\n" +
		"updated_at: 1780394400\n" +
		"tags: 123\n" +
		"---\n" +
		"Body\n"))
	require.Error(t, err)

	var fieldErr *FrontmatterFieldError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "tags", fieldErr.Field)
	assert.Equal(t, FieldErrKindInvalidStringList, fieldErr.Kind)
}

func TestFrontmatterFieldError_TitleInvalidType(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440002\n" +
		"title: 42\n" +
		"created_at: 1780394400\n" +
		"updated_at: 1780394400\n" +
		"---\n" +
		"Body\n"))
	require.Error(t, err)

	var fieldErr *FrontmatterFieldError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "title", fieldErr.Field)
	assert.Equal(t, FieldErrKindInvalidString, fieldErr.Kind)
}

func TestFrontmatterFieldError_YAMLSyntaxIsNotFieldError(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\ntitle: [bad yaml\n---\nBody"))
	require.Error(t, err)

	var fieldErr *FrontmatterFieldError
	assert.False(t, errors.As(err, &fieldErr))
	assert.Contains(t, err.Error(), "parse frontmatter")
}
