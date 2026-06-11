package notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListReturnsEmptySliceForEmptyProject(t *testing.T) {
	root := t.TempDir()

	got, err := List(root)
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestListReturnsNoteSummariesAndIgnoresTrash(t *testing.T) {
	root := t.TempDir()
	note := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Auth Migration",
		Slug:           "auth-migration",
		CreatedAt:      time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC),
		Body:           []byte("# Heading\n"),
	}
	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)

	path := filepath.Join(root, "projects", "auth.md")
	writeTestFile(t, path, rendered)
	writeTestFile(t, filepath.Join(root, ".trash", "projects", "deleted.md"), rendered)

	got, err := List(root)
	require.NoError(t, err)
	require.Len(t, got, 1)

	want := NoteSummary{
		NoteID:      "550e8400-e29b-41d4-a716-446655440000",
		Slug:        "auth-migration",
		Title:       "Auth Migration",
		Path:        "projects/auth.md",
		UpdatedAt:   "2026-06-02T11:00:00Z",
		ContentHash: HashBytes(rendered),
	}
	assert.Equal(t, want, got[0])
}

func writeTestFile(t *testing.T, path string, content []byte) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, content, 0o644))
}