package notes

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/platform/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestResolveFollowsSelectorPrecedence(t *testing.T) {
	root := t.TempDir()

	writeResolvedNote(t, root, "uuid.md", markdown.Note{
		MnemonicNoteID: "11111111-1111-1111-1111-111111111111",
		Title:          "UUID Note",
		Slug:           "uuid-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "slug.md", markdown.Note{
		MnemonicNoteID: "22222222-2222-2222-2222-222222222222",
		Title:          "Slug Note",
		Slug:           "slug-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, filepath.Join("folder", "path.md"), markdown.Note{
		MnemonicNoteID: "33333333-3333-3333-3333-333333333333",
		Title:          "Path Note",
		Slug:           "path-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "title.md", markdown.Note{
		MnemonicNoteID: "44444444-4444-4444-4444-444444444444",
		Title:          "Exact Title",
		Slug:           "exact-title-slug",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "normalized.md", markdown.Note{
		MnemonicNoteID: "55555555-5555-5555-5555-555555555555",
		Title:          "Normalized Title",
		Slug:           "custom-slug",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	cases := []struct {
		name     string
		selector string
		wantPath string
	}{
		{name: "uuid", selector: "11111111-1111-1111-1111-111111111111", wantPath: "uuid.md"},
		{name: "slug", selector: "slug-note", wantPath: "slug.md"},
		{name: "path", selector: filepath.ToSlash(filepath.Join("folder", "path.md")), wantPath: filepath.ToSlash(filepath.Join("folder", "path.md"))},
		{name: "title", selector: "Exact Title", wantPath: "title.md"},
		{name: "normalized title", selector: "normalized-title", wantPath: "normalized.md"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(root, tc.selector)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPath, got.Path)
		})
	}
}

func TestResolveReturnsAmbiguousError(t *testing.T) {
	root := t.TempDir()
	writeResolvedNote(t, root, "a.md", markdown.Note{
		MnemonicNoteID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:          "Duplicate Title",
		Slug:           "dup-a",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "b.md", markdown.Note{
		MnemonicNoteID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Title:          "Duplicate Title",
		Slug:           "dup-b",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	_, err := Resolve(root, "Duplicate Title")
	require.Error(t, err)
	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeAmbiguous, appErr.Code)
}

func TestResolveReturnsNotFoundError(t *testing.T) {
	root := t.TempDir()
	writeResolvedNote(t, root, "note.md", markdown.Note{
		MnemonicNoteID: "cccccccc-cccc-cccc-cccc-cccccccccccc",
		Title:          "Existing",
		Slug:           "existing",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	_, err := Resolve(root, "missing")
	require.Error(t, err)
	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeNotFound, appErr.Code)
}

func TestNormalizeResolvedPath(t *testing.T) {
	cases := map[string]string{
		"folder/../note.md": "note.md",
		filepath.ToSlash(filepath.Join("folder", "note.md")): filepath.ToSlash(filepath.Join("folder", "note.md")),
		".":            ".",
		"../note.md":   "../note.md",
		"/tmp/note.md": "/tmp/note.md",
	}

	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, want, normalizeResolvedPath(input))
		})
	}
}

func TestMatchResolvedNotesFromIndexUsesNormalizedTitle(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`CREATE TABLE notes (
		note_id TEXT PRIMARY KEY,
		slug TEXT NOT NULL,
		rel_path TEXT NOT NULL,
		title TEXT NOT NULL
	)`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO notes(note_id, slug, rel_path, title) VALUES (?, ?, ?, ?)`,
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"custom-slug",
		"folder/note.md",
		"Normalized Title",
	)
	require.NoError(t, err)

	matches, err := matchResolvedNotesFromIndex(db, "normalized-title")
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, "folder/note.md", matches[0].Path)
}

func TestResolveReturnsReadErrorWhenIndexedFileMissing(t *testing.T) {
	testutil.CleanEnvForTest(t)
	projectRoot := t.TempDir()

	writeResolvedNote(t, projectRoot, "indexed.md", markdown.Note{
		MnemonicNoteID: "66666666-6666-6666-6666-666666666666",
		Title:          "Indexed Note",
		Slug:           "indexed-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	require.NoError(t, os.Remove(filepath.Join(projectRoot, "indexed.md")))

	_, _, err := readResolvedNote(projectRoot, resolvedNote{
		selectorData: selectorData{
			Path: "indexed.md",
		},
	})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "read note"))
}

func TestResolveUsesProjectIndexWhenAvailable(t *testing.T) {
	projectRoot := filepath.Join(testutil.CleanEnvForTest(t), "project")
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))

	writeResolvedNote(t, projectRoot, "indexed.md", markdown.Note{
		MnemonicNoteID: "66666666-6666-6666-6666-666666666666",
		Title:          "Indexed Note",
		Slug:           "indexed-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	// Set up file-based registry with a central project
	setupMnemonicHomeWithProject(t, "project", "550e8400-e29b-41d4-a716-446655440000", projectRoot)

	mnemonicPaths, err := paths.GetMnemonicPaths()
	require.NoError(t, err)
	indexPath := filepath.Join(mnemonicPaths.StateHome, "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "index.sqlite")
	require.NoError(t, os.MkdirAll(filepath.Dir(indexPath), 0o755))

	indexDB, err := sql.Open("sqlite", indexPath)
	require.NoError(t, err)
	defer func() { _ = indexDB.Close() }()
	_, err = indexDB.Exec(`CREATE TABLE notes (
		note_id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE,
		rel_path TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	_, err = indexDB.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"66666666-6666-6666-6666-666666666666",
		"550e8400-e29b-41d4-a716-446655440000",
		"indexed-note",
		"indexed.md",
		"Indexed Note",
		"content-hash",
		noteTime().UTC().Format(time.RFC3339),
		noteTime().UTC().Format(time.RFC3339),
	)
	require.NoError(t, err)

	got, err := Resolve(projectRoot, "indexed-note")
	require.NoError(t, err)
	assert.Equal(t, "indexed.md", got.Path)
}

func TestResolveFallsBackToDiskWhenIndexIsStale(t *testing.T) {
	projectRoot := filepath.Join(testutil.CleanEnvForTest(t), "project")
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))

	writeResolvedNote(t, projectRoot, "indexed.md", markdown.Note{
		MnemonicNoteID: "66666666-6666-6666-6666-666666666666",
		Title:          "Indexed Note",
		Slug:           "indexed-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, projectRoot, "fresh.md", markdown.Note{
		MnemonicNoteID: "77777777-7777-7777-7777-777777777777",
		Title:          "Fresh Note",
		Slug:           "fresh-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	// Set up file-based registry with a central project
	setupMnemonicHomeWithProject(t, "project", "550e8400-e29b-41d4-a716-446655440000", projectRoot)

	mnemonicPaths, err := paths.GetMnemonicPaths()
	require.NoError(t, err)
	indexPath := filepath.Join(mnemonicPaths.StateHome, "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "index.sqlite")
	require.NoError(t, os.MkdirAll(filepath.Dir(indexPath), 0o755))

	indexDB, err := sql.Open("sqlite", indexPath)
	require.NoError(t, err)
	defer func() { _ = indexDB.Close() }()
	_, err = indexDB.Exec(`CREATE TABLE notes (
		note_id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE,
		rel_path TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	_, err = indexDB.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"66666666-6666-6666-6666-666666666666",
		"550e8400-e29b-41d4-a716-446655440000",
		"indexed-note",
		"indexed.md",
		"Indexed Note",
		"content-hash",
		noteTime().UTC().Format(time.RFC3339),
		noteTime().UTC().Format(time.RFC3339),
	)
	require.NoError(t, err)

	got, err := Resolve(projectRoot, "77777777-7777-7777-7777-777777777777")
	require.NoError(t, err)
	assert.Equal(t, "fresh.md", got.Path)
}

func setupMnemonicHomeWithProject(t *testing.T, slug, projectID, memoriesRoot string) {
	t.Helper()
	mnemonicPaths, err := paths.GetMnemonicPaths()
	require.NoError(t, err)

	projectDir := filepath.Join(mnemonicPaths.MemoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = projectID
	manifest.Name = slug
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = noteTime()
	manifest.UpdatedAt = noteTime()
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest))
}

func writeResolvedNote(t *testing.T, root, relPath string, note markdown.Note) {
	t.Helper()

	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)
	path := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, rendered, 0o644))
}

func noteTime() time.Time {
	return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
}
