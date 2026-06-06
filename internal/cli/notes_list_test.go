package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

type notesListJSON struct {
	Notes []struct {
		NoteID      string    `json:"note_id"`
		Slug        string    `json:"slug"`
		Title       string    `json:"title"`
		Path        string    `json:"path"`
		UpdatedAt   time.Time `json:"updated_at"`
		ContentHash string    `json:"content_hash"`
	} `json:"notes"`
}

func TestNotesListCommandEmptyProject(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)

	restore := chdirForTest(t, cwd)
	defer restore()

	if err := writeLocalProjectFixture(t, cwd, "backend"); err != nil {
		t.Fatalf("writeLocalProjectFixture() error = %v", err)
	}

	result := executeCommand("notes", "list", "--project", "backend", "--json")
	if result.Err != nil {
		t.Fatalf("notes list returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got notesListJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.Notes == nil {
		t.Fatal("notes is nil, want empty array")
	}
	if len(got.Notes) != 0 {
		t.Fatalf("len(notes) = %d, want 0", len(got.Notes))
	}
}

func TestNotesListCommandReturnsMarkdownNotes(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)

	restore := chdirForTest(t, cwd)
	defer restore()

	if err := writeLocalProjectFixture(t, cwd, "backend"); err != nil {
		t.Fatalf("writeLocalProjectFixture() error = %v", err)
	}

	projectRoot := filepath.Join(cwd, ".mnemonic-memories", "backend")
	noteBody := []byte(`---
mnemonic_note_id: note-123
title: Intro
slug: intro
updated_at: 2026-06-02T10:00:00Z
---
# Intro
`)
	if err := os.MkdirAll(filepath.Join(projectRoot, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "docs", "intro.md"), noteBody, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".trash"), 0o755); err != nil {
		t.Fatalf("MkdirAll() trash error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ".trash", "deleted.md"), []byte("trash"), 0o644); err != nil {
		t.Fatalf("WriteFile() trash error = %v", err)
	}

	result := executeCommand("notes", "list", "--project", "backend", "--json")
	if result.Err != nil {
		t.Fatalf("notes list returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got notesListJSON
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("failed to decode JSON: %v\nstdout: %s", err, result.Stdout)
	}
	if got.Notes == nil {
		t.Fatal("notes is nil, want populated array")
	}
	if len(got.Notes) != 1 {
		t.Fatalf("len(notes) = %d, want 1", len(got.Notes))
	}

	note := got.Notes[0]
	if note.NoteID != "note-123" {
		t.Fatalf("note_id = %q, want %q", note.NoteID, "note-123")
	}
	if note.Slug != "intro" {
		t.Fatalf("slug = %q, want %q", note.Slug, "intro")
	}
	if note.Title != "Intro" {
		t.Fatalf("title = %q, want %q", note.Title, "Intro")
	}
	if note.Path != "docs/intro.md" {
		t.Fatalf("path = %q, want %q", note.Path, "docs/intro.md")
	}
	wantUpdatedAt, err := time.Parse(time.RFC3339, "2026-06-02T10:00:00Z")
	if err != nil {
		t.Fatalf("time.Parse() error = %v", err)
	}
	if !note.UpdatedAt.Equal(wantUpdatedAt) {
		t.Fatalf("updated_at = %v, want %v", note.UpdatedAt, wantUpdatedAt)
	}
	if note.ContentHash == "" {
		t.Fatal("content_hash is empty, want SHA-256 value")
	}
	if note.ContentHash != notes.HashBytes(noteBody) {
		t.Fatalf("content_hash = %q, want %q", note.ContentHash, notes.HashBytes(noteBody))
	}
}

func chdirForTest(t *testing.T, dir string) func() {
	t.Helper()

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}
	return func() {
		_ = os.Chdir(prev)
	}
}

func writeLocalProjectFixture(t *testing.T, cwd, slug string) error {
	t.Helper()

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	file := &project.MnemonicFile{
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		Projects: []project.MnemonicProject{
			{
				ID:                    "550e8400-e29b-41d4-a716-446655440000",
				Name:                  slug,
				Slug:                  slug,
				Kind:                  project.ProjectKindLocal,
				MemoriesPath:          filepath.Join(".mnemonic-memories", slug),
				MarkdownFormatVersion: 1,
				CreatedAt:             now,
				UpdatedAt:             now,
			},
		},
	}

	if err := os.MkdirAll(filepath.Join(cwd, ".mnemonic-memories", slug), 0o755); err != nil {
		return err
	}
	return project.WriteMnemonicFile(filepath.Join(cwd, ".mnemonic"), file)
}
