package markdown

import (
	"bytes"
	"testing"
	"time"

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
		"created_at: 2026-06-02T10:00:00Z\n" +
		"updated_at: 2026-06-02T11:00:00Z\n" +
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
	require.Equal(t, "example-note", note.Permalink)
	require.Equal(t, "keep-me", note.Frontmatter["extra_field"])
	require.True(t, bytes.Equal(note.Body, []byte("# Heading\nBody text\n")))
}

func TestParseNoteUsesPermalinkFallbackWhenSlugMissing(t *testing.T) {
	t.Parallel()

	note, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440001\n" +
		"title: Permalink Note\n" +
		"permalink: permalink-note\n" +
		"created_at: 2026-06-02T10:00:00Z\n" +
		"updated_at: 2026-06-02T10:00:00Z\n" +
		"---\n" +
		"Body\n"))
	require.NoError(t, err)
	require.Equal(t, "permalink-note", note.Slug)
}

func TestParseNoteWithoutFrontmatterReturnsFullBody(t *testing.T) {
	t.Parallel()

	input := []byte("# Heading\nBody\n")
	note, err := ParseNote(input)
	require.NoError(t, err)
	require.Empty(t, note.Frontmatter)
	require.True(t, bytes.Equal(note.Body, input))
}

func TestParseNoteRejectsWrongFieldTypes(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440002\n" +
		"title: Bad Note\n" +
		"created_at: 123\n" +
		"updated_at: 2026-06-02T10:00:00Z\n" +
		"---\n" +
		"Body\n"))
	require.Error(t, err)
}
