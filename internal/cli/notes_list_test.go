package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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

	require.NoError(t, writeLocalProjectFixture(t, cwd, "backend"))

	result := executeCommand("notes", "list", "--project", "backend", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got notesListJSON
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.NotNil(t, got.Notes)
	require.Empty(t, got.Notes)
}

func TestNotesListCommandReturnsMarkdownNotes(t *testing.T) {
	cwd := testutil.CleanEnvForTest(t)

	restore := chdirForTest(t, cwd)
	defer restore()

	require.NoError(t, writeLocalProjectFixture(t, cwd, "backend"))

	projectRoot := filepath.Join(cwd, ".mnemonic-memories", "backend")
	noteBody := []byte(`---
mnemonic_note_id: note-123
title: Intro
slug: intro
updated_at: 2026-06-02T10:00:00Z
---
# Intro
`)
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "docs", "intro.md"), noteBody, 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, ".trash"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, ".trash", "deleted.md"), []byte("trash"), 0o644))

	result := executeCommand("notes", "list", "--project", "backend", "--json")
	require.NoError(t, result.Err, "stderr: %s", result.Stderr)

	var got notesListJSON
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &got), "stdout: %s", result.Stdout)
	require.NotNil(t, got.Notes)
	require.Len(t, got.Notes, 1)

	note := got.Notes[0]
	require.Equal(t, "note-123", note.NoteID)
	require.Equal(t, "intro", note.Slug)
	require.Equal(t, "Intro", note.Title)
	require.Equal(t, "docs/intro.md", note.Path)
	wantUpdatedAt, err := time.Parse(time.RFC3339, "2026-06-02T10:00:00Z")
	require.NoError(t, err)
	require.True(t, note.UpdatedAt.Equal(wantUpdatedAt))
	require.NotEmpty(t, note.ContentHash)
	require.Equal(t, notes.HashBytes(noteBody), note.ContentHash)
}

func chdirForTest(t *testing.T, dir string) func() {
	t.Helper()

	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
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