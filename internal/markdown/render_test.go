package markdown

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRenderNoteRoundTripPreservesMetadata(t *testing.T) {
	t.Parallel()

	body := []byte("## Observations\n- [decision] Roll out by feature flag\n\n## Relations\n- depends_on [[Session storage redesign]]\n")
	note := Note{
		Frontmatter: map[string]any{
			"extra_field": "keep-me",
			"permalink":   "legacy-slug",
		},
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440010",
		Title:          "Auth migration plan",
		Slug:           "auth-migration-plan",
		Tags:           []string{"django", "auth"},
		CreatedAt:      time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC),
		Type:           "note",
		Body:           body,
	}

	rendered, err := RenderNote(note)
	if err != nil {
		t.Fatalf("RenderNote() error = %v", err)
	}

	renderedText := string(rendered)
	if strings.Contains(renderedText, "permalink:") {
		t.Fatalf("rendered note contains permalink field: %q", renderedText)
	}
	if !strings.Contains(renderedText, "tags:\n  - django\n  - auth\n") {
		t.Fatalf("rendered note does not use a YAML list for tags: %q", renderedText)
	}
	assertOrderedSubstrings(t, renderedText,
		"mnemonic_note_id:",
		"title:",
		"slug:",
		"tags:",
		"created_at:",
		"updated_at:",
		"type:",
		"extra_field:",
	)
	if !strings.HasSuffix(renderedText, string(body)) {
		t.Fatalf("rendered body = %q, want suffix %q", renderedText, body)
	}

	roundTripped, err := ParseNote(rendered)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}

	if roundTripped.MnemonicNoteID != note.MnemonicNoteID {
		t.Fatalf("MnemonicNoteID = %q, want %q", roundTripped.MnemonicNoteID, note.MnemonicNoteID)
	}
	if roundTripped.Title != note.Title {
		t.Fatalf("Title = %q, want %q", roundTripped.Title, note.Title)
	}
	if roundTripped.Slug != note.Slug {
		t.Fatalf("Slug = %q, want %q", roundTripped.Slug, note.Slug)
	}
	if got, want := roundTripped.Tags, note.Tags; len(got) != len(want) {
		t.Fatalf("Tags = %#v, want %#v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Tags = %#v, want %#v", got, want)
			}
		}
	}
	if !roundTripped.CreatedAt.Equal(note.CreatedAt) {
		t.Fatalf("CreatedAt = %s, want %s", roundTripped.CreatedAt, note.CreatedAt)
	}
	if !roundTripped.UpdatedAt.Equal(note.UpdatedAt) {
		t.Fatalf("UpdatedAt = %s, want %s", roundTripped.UpdatedAt, note.UpdatedAt)
	}
	if roundTripped.Type != note.Type {
		t.Fatalf("Type = %q, want %q", roundTripped.Type, note.Type)
	}
	if got := roundTripped.Frontmatter["extra_field"]; got != "keep-me" {
		t.Fatalf("Frontmatter[extra_field] = %#v, want %q", got, "keep-me")
	}
	if _, ok := roundTripped.Frontmatter["permalink"]; ok {
		t.Fatalf("Frontmatter unexpectedly preserved permalink: %#v", roundTripped.Frontmatter)
	}
	if !bytes.Equal(roundTripped.Body, body) {
		t.Fatalf("Body = %q, want %q", roundTripped.Body, body)
	}
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
	if err != nil {
		t.Fatalf("RenderNote() error = %v", err)
	}

	renderedText := string(rendered)
	if strings.Contains(renderedText, "tags: - test\n") {
		t.Fatalf("rendered invalid inline sequence: %q", renderedText)
	}
	if !strings.Contains(renderedText, "tags:\n  - test\n") {
		t.Fatalf("rendered note does not use a YAML list for a single tag: %q", renderedText)
	}

	roundTripped, err := ParseNote(rendered)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if got, want := roundTripped.Tags, note.Tags; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("Tags = %#v, want %#v", got, want)
	}
}

func assertOrderedSubstrings(t *testing.T, text string, substrings ...string) {
	t.Helper()

	last := -1
	for _, substring := range substrings {
		idx := strings.Index(text, substring)
		if idx < 0 {
			t.Fatalf("rendered text missing %q: %q", substring, text)
		}
		if idx <= last {
			t.Fatalf("substring %q appears out of order in %q", substring, text)
		}
		last = idx
	}
}
