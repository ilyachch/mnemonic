package markdown

import (
	"bytes"
	"testing"
	"time"
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
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}

	if note.MnemonicNoteID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("MnemonicNoteID = %q", note.MnemonicNoteID)
	}
	if note.Title != "Example Note" {
		t.Fatalf("Title = %q", note.Title)
	}
	if note.Slug != "example-note" {
		t.Fatalf("Slug = %q", note.Slug)
	}
	if got, want := note.Tags, []string{"django", "auth"}; len(got) != len(want) {
		t.Fatalf("Tags = %#v, want %#v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Tags = %#v, want %#v", got, want)
			}
		}
	}
	wantCreatedAt := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	if !note.CreatedAt.Equal(wantCreatedAt) {
		t.Fatalf("CreatedAt = %s, want %s", note.CreatedAt, wantCreatedAt)
	}
	wantUpdatedAt := time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC)
	if !note.UpdatedAt.Equal(wantUpdatedAt) {
		t.Fatalf("UpdatedAt = %s, want %s", note.UpdatedAt, wantUpdatedAt)
	}
	if note.Type != "decision" {
		t.Fatalf("Type = %q", note.Type)
	}
	if note.Permalink != "example-note" {
		t.Fatalf("Permalink = %q", note.Permalink)
	}
	if note.Frontmatter["extra_field"] != "keep-me" {
		t.Fatalf("Frontmatter[extra_field] = %#v, want %q", note.Frontmatter["extra_field"], "keep-me")
	}
	if !bytes.Equal(note.Body, []byte("# Heading\nBody text\n")) {
		t.Fatalf("Body = %q", note.Body)
	}
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
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if note.Slug != "permalink-note" {
		t.Fatalf("Slug = %q, want permalink-note", note.Slug)
	}
}

func TestParseNoteWithoutFrontmatterReturnsFullBody(t *testing.T) {
	t.Parallel()

	input := []byte("# Heading\nBody\n")
	note, err := ParseNote(input)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if len(note.Frontmatter) != 0 {
		t.Fatalf("Frontmatter = %#v, want empty", note.Frontmatter)
	}
	if !bytes.Equal(note.Body, input) {
		t.Fatalf("Body = %q, want %q", note.Body, input)
	}
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
	if err == nil {
		t.Fatal("ParseNote() error = nil, want wrong type error")
	}
}
