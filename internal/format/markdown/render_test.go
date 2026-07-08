package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderNoteRoundTripPreservesMetadata(t *testing.T) {
	t.Parallel()

	body := []byte("## Observations\n- [decision] Roll out by feature flag\n\n## Relations\n- depends_on [[Session storage redesign]]\n")
	note := Note{
		Frontmatter: map[string]any{
			"extra_field": "keep-me",
		},
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440010",
		Title:          "Auth migration plan",
		Slug:           "auth-migration-plan",
		Tags:           []string{"django", "auth"},
		Summary:        "Plan for migrating auth system",
		Aliases:        []string{"auth-plan", "migration-plan"},
		Type:           "note",
		Body:           body,
	}

	rendered, err := RenderNote(note)
	require.NoError(t, err)

	renderedText := string(rendered)
	require.NotContains(t, renderedText, "permalink:")
	require.NotContains(t, renderedText, "created_at:")
	require.NotContains(t, renderedText, "updated_at:")
	require.Contains(t, renderedText, "tags:\n  - django\n  - auth\n")
	require.Contains(t, renderedText, "summary: Plan for migrating auth system\n")
	require.Contains(t, renderedText, "aliases:\n  - auth-plan\n  - migration-plan\n")
	assertOrderedSubstrings(t, renderedText,
		"mnemonic_note_id:",
		"title:",
		"slug:",
		"tags:",
		"summary:",
		"aliases:",
		"type:",
		"extra_field:",
	)
	require.True(t, strings.HasSuffix(renderedText, string(body)))

	roundTripped, err := ParseNote(rendered)
	require.NoError(t, err)

	require.Equal(t, note.MnemonicNoteID, roundTripped.MnemonicNoteID)
	require.Equal(t, note.Title, roundTripped.Title)
	require.Equal(t, note.Slug, roundTripped.Slug)
	require.Equal(t, note.Tags, roundTripped.Tags)
	require.Equal(t, note.Summary, roundTripped.Summary)
	require.Equal(t, note.Aliases, roundTripped.Aliases)
	require.Equal(t, note.Type, roundTripped.Type)
	require.Equal(t, "keep-me", roundTripped.Frontmatter["extra_field"])
	require.True(t, bytes.Equal(roundTripped.Body, body))
}

func TestRenderNoteMinimalFrontmatterWritesNoOptionalKeys(t *testing.T) {
	t.Parallel()

	note := Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440040",
		Body:           []byte("body\n"),
	}

	rendered, err := RenderNote(note)
	require.NoError(t, err)

	renderedText := string(rendered)
	require.Contains(t, renderedText, "mnemonic_note_id: 550e8400-e29b-41d4-a716-446655440040\n")
	require.NotContains(t, renderedText, "created_at:")
	require.NotContains(t, renderedText, "updated_at:")
	require.NotContains(t, renderedText, "title:")
	require.NotContains(t, renderedText, "slug:")
}

func TestRenderNoteSingleTagUsesYAMLList(t *testing.T) {
	t.Parallel()

	note := Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440011",
		Slug:           "single-tag-note",
		Tags:           []string{"test"},
		Body:           []byte("body\n"),
	}

	rendered, err := RenderNote(note)
	require.NoError(t, err)

	renderedText := string(rendered)
	require.NotContains(t, renderedText, "tags: - test\n")
	require.Contains(t, renderedText, "tags:\n  - test\n")

	roundTripped, err := ParseNote(rendered)
	require.NoError(t, err)
	require.Equal(t, note.Tags, roundTripped.Tags)
}

func TestRenderNoteStripsPermalink(t *testing.T) {
	t.Parallel()

	note := Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440030",
		Slug:           "permalink-strip",
		Frontmatter: map[string]any{
			"permalink": "legacy-permalink",
		},
		Body: []byte("body\n"),
	}

	rendered, err := RenderNote(note)
	require.NoError(t, err)

	renderedText := string(rendered)
	require.NotContains(t, renderedText, "permalink:")
}

func TestRenderNoteStripsLegacyTimestamps(t *testing.T) {
	t.Parallel()

	note := Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440041",
		Slug:           "legacy-timestamps",
		Frontmatter: map[string]any{
			"created_at": 1741737600,
			"updated_at": 1741824000,
		},
		Body: []byte("body\n"),
	}

	rendered, err := RenderNote(note)
	require.NoError(t, err)

	renderedText := string(rendered)
	require.NotContains(t, renderedText, "created_at:")
	require.NotContains(t, renderedText, "updated_at:")
}

func assertOrderedSubstrings(t *testing.T, text string, substrings ...string) {
	t.Helper()

	last := -1
	for _, substring := range substrings {
		idx := strings.Index(text, substring)
		require.GreaterOrEqual(t, idx, 0, "rendered text missing %q: %q", substring, text)
		require.Greater(t, idx, last, "substring %q appears out of order in %q", substring, text)
		last = idx
	}
}
