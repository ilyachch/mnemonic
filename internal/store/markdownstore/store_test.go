package markdownstore

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreCreateEditAndDelete(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	stateDir := t.TempDir()
	indexPath := filepath.Join(stateDir, "index.sqlite")
	store := Store{RootDir: root, StateDir: stateDir, IndexPath: indexPath}

	created, err := store.Create(CreateInput{
		Title: "Auth migration",
		Body:  []byte("## Summary\n"),
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
		},
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440000"
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "auth-migration", created.Slug)
	assert.Equal(t, "auth-migration.md", created.Path)

	db := openSeedIndex(t, indexPath)
	defer func() { _ = db.Close() }()
	seedIndexedNoteFromPath(t, db, filepath.Join(root, created.Path), created.Path)

	edited, err := store.Edit(EditInput{
		Selector: created.Slug,
		Append:   []byte("Next step"),
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)
		},
	})
	require.NoError(t, err)
	assert.NotEqual(t, created.ContentHash, edited.ContentHash)

	show, err := store.Show(created.Slug)
	require.NoError(t, err)
	assert.Equal(t, "Auth migration", show.Note.Title)
	assert.Equal(t, "## Summary\nNext step", string(show.Note.Body))
	assert.Equal(t, edited.ContentHash, show.ContentHash)
	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	require.NoError(t, err)
	assert.Equal(t, data, show.RawMarkdown)
	assert.Equal(t, HashBytes(data), show.ContentHash)

	result, err := store.Delete(DeleteInput{
		Selector: created.Slug,
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 15, 4, 5, 0, time.UTC)
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "trash", result.Mode)
	assert.NotEmpty(t, result.TrashPath)

	_, err = os.Stat(filepath.Join(root, "auth-migration.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestStoreResolveSelectorPrecedenceFromIndex(t *testing.T) {
	testutil.CleanEnvForTest(t)
	indexPath := filepath.Join(t.TempDir(), "index.sqlite")
	store := Store{IndexPath: indexPath}
	db := openSeedIndex(t, indexPath)
	defer func() { _ = db.Close() }()

	seedIndexedNote(t, db, markdown.Note{
		MnemonicNoteID: "11111111-1111-1111-1111-111111111111",
		Title:          "UUID Note",
		Slug:           "uuid-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	}, "uuid.md")
	seedIndexedNote(t, db, markdown.Note{
		MnemonicNoteID: "22222222-2222-2222-2222-222222222222",
		Title:          "Slug Note",
		Slug:           "slug-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	}, "slug.md")
	seedIndexedNote(t, db, markdown.Note{
		MnemonicNoteID: "33333333-3333-3333-3333-333333333333",
		Title:          "Path Note",
		Slug:           "path-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	}, filepath.ToSlash(filepath.Join("folder", "path.md")))
	seedIndexedNote(t, db, markdown.Note{
		MnemonicNoteID: "44444444-4444-4444-4444-444444444444",
		Title:          "Exact Title",
		Slug:           "exact-title-slug",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	}, "title.md")
	seedIndexedNote(t, db, markdown.Note{
		MnemonicNoteID: "55555555-5555-5555-5555-555555555555",
		Title:          "Normalized Title",
		Slug:           "custom-slug",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	}, "normalized.md")

	cases := []struct {
		name     string
		selector string
		wantPath string
	}{
		{name: "uuid", selector: "11111111-1111-1111-1111-111111111111", wantPath: "uuid.md"},
		{name: "slug", selector: "slug-note", wantPath: "slug.md"},
		{name: "path", selector: filepath.Join("folder", "path.md"), wantPath: filepath.ToSlash(filepath.Join("folder", "path.md"))},
		{name: "title", selector: "Exact Title", wantPath: "title.md"},
		{name: "windows path", selector: `folder\path.md`, wantPath: "folder/path.md"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := store.Resolve(tc.selector)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPath, got.Path)
		})
	}
}

func TestStoreResolveMissingIndex(t *testing.T) {
	testutil.CleanEnvForTest(t)
	store := Store{IndexPath: filepath.Join(t.TempDir(), "missing.sqlite")}

	_, err := store.Resolve("slug-note")
	require.EqualError(t, err, "index missing; run `mnemonic project reindex` before resolving")
}

func TestStoreResolveNotFoundAndAmbiguous(t *testing.T) {
	testutil.CleanEnvForTest(t)
	indexPath := filepath.Join(t.TempDir(), "index.sqlite")
	store := Store{IndexPath: indexPath}
	db := openSeedIndex(t, indexPath)
	defer func() { _ = db.Close() }()

	seedIndexedNote(t, db, markdown.Note{
		MnemonicNoteID: "66666666-6666-6666-6666-666666666666",
		Title:          "Shared Title",
		Slug:           "shared-one",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	}, "shared-one.md")
	seedIndexedNote(t, db, markdown.Note{
		MnemonicNoteID: "77777777-7777-7777-7777-777777777777",
		Title:          "Shared Title",
		Slug:           "shared-two",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	}, "shared-two.md")

	_, err := store.Resolve("missing")
	require.EqualError(t, err, `note "missing" not found`)

	_, err = store.Resolve("Shared Title")
	require.EqualError(t, err, `note selector "Shared Title" matches multiple notes`)
}

func TestStoreListAndWalkIgnoreTrash(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root, StateDir: t.TempDir()}

	note := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Auth Migration",
		Slug:           "auth-migration",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
		Body:           []byte("# Heading\n"),
	}
	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)

	writeTestFile(t, filepath.Join(root, "projects", "auth.md"), rendered)
	writeTestFile(t, filepath.Join(root, ".trash", "projects", "deleted.md"), rendered)

	paths, err := store.Walk()
	require.NoError(t, err)
	sort.Strings(paths)
	assert.Equal(t, []string{"projects/auth.md"}, paths)

	got, err := store.List()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "projects/auth.md", got[0].Path)
	assert.Equal(t, HashBytes(rendered), got[0].ContentHash)
}

func noteTime() time.Time {
	return time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
}

func writeTestFile(t *testing.T, path string, content []byte) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, content, 0o644))
}

func openSeedIndex(t *testing.T, indexPath string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(indexPath)+"?mode=rwc")
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS notes (
		note_id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		slug TEXT NOT NULL,
		rel_path TEXT NOT NULL,
		title TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	return db
}

func seedIndexedNoteFromPath(t *testing.T, db *sql.DB, path, relPath string) {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	seedIndexedNote(t, db, note, filepath.ToSlash(relPath))
}

func seedIndexedNote(t *testing.T, db *sql.DB, note markdown.Note, relPath string) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		note.MnemonicNoteID, "kb-1", note.EffectiveSlug(), relPath, note.Title, "hash-"+note.MnemonicNoteID,
		note.CreatedAt.UTC().Format(time.RFC3339), note.UpdatedAt.UTC().Format(time.RFC3339),
	)
	require.NoError(t, err)
}
