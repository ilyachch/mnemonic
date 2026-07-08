package markdown

import (
	"bytes"
	"errors"
	"testing"

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

func TestParseNoteAcceptsMinimalFrontmatter(t *testing.T) {
	t.Parallel()

	input := []byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440009\n" +
		"---\n" +
		"Body without heavy frontmatter.\n")

	note, err := ParseNote(input)
	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440009", note.MnemonicNoteID)
	require.Empty(t, note.Title)
	require.Empty(t, note.Slug)
	require.Empty(t, note.Tags)
	require.Contains(t, string(note.Body), "without heavy frontmatter")
}

func TestFrontmatterFieldError_TagsInvalidType(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440002\n" +
		"title: Note\n" +
		"tags: 123\n" +
		"---\n" +
		"Body\n"))
	require.Error(t, err)

	var fieldErr *FrontmatterFieldError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "tags", fieldErr.Field)
	assert.Equal(t, FieldErrKindInvalidStringList, fieldErr.Kind)
}

func TestFrontmatterFieldError_TagsScalarRejected(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440002\n" +
		"title: Note\n" +
		"tags: payment\n" +
		"---\n" +
		"Body\n"))
	require.Error(t, err)

	var fieldErr *FrontmatterFieldError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "tags", fieldErr.Field)
	assert.Equal(t, FieldErrKindInvalidStringList, fieldErr.Kind)
}

func TestFrontmatterFieldError_AliasesScalarRejected(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440002\n" +
		"title: Note\n" +
		"aliases: \"old title\"\n" +
		"---\n" +
		"Body\n"))
	require.Error(t, err)

	var fieldErr *FrontmatterFieldError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "aliases", fieldErr.Field)
	assert.Equal(t, FieldErrKindInvalidStringList, fieldErr.Kind)
}

func TestFrontmatterFieldError_TagsListWithNonString(t *testing.T) {
	t.Parallel()

	_, err := ParseNote([]byte("---\n" +
		"mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440002\n" +
		"title: Note\n" +
		"tags:\n" +
		"  - payment\n" +
		"  - 123\n" +
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

func TestGetOrDeriveTitleReturnsExplicitTitle(t *testing.T) {
	t.Parallel()

	note := Note{Title: "Explicit Title"}
	require.Equal(t, "Explicit Title", note.GetOrDeriveTitle("any.md"))
}

func TestGetOrDeriveTitleFallsBackToH1(t *testing.T) {
	t.Parallel()

	note := Note{Body: []byte("# My H1 Title\n\nBody text.\n")}
	require.Equal(t, "My H1 Title", note.GetOrDeriveTitle("untitled.md"))
}

func TestGetOrDeriveTitleFallsBackToFilename(t *testing.T) {
	t.Parallel()

	note := Note{Body: []byte("Just body text without a heading.\n")}
	require.Equal(t, "untitled-note", note.GetOrDeriveTitle("untitled-note.md"))
}

func TestGetOrDeriveSlugReturnsExplicitSlug(t *testing.T) {
	t.Parallel()

	note := Note{Slug: "explicit-slug"}
	require.Equal(t, "explicit-slug", note.GetOrDeriveSlug("any.md"))
}

func TestGetOrDeriveSlugDerivesFromTitle(t *testing.T) {
	t.Parallel()

	note := Note{Title: "Auth Migration Plan"}
	require.Equal(t, "auth-migration-plan", note.GetOrDeriveSlug("auth.md"))
}

func TestGetOrDeriveSlugFallsBackToFilename(t *testing.T) {
	t.Parallel()

	// Non-ASCII title forces Slugify to error; expect filename-based slug
	// derived from the file's base name.
	note := Note{Title: "Заметка", Body: []byte("body\n")}
	require.Equal(t, "my-note", note.GetOrDeriveSlug("My Note!.md"))
}
