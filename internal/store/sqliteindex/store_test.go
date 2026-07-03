package sqliteindex

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/stretchr/testify/require"
)

func TestOpenCreatesFileAndPragmas(t *testing.T) {
	dir := t.TempDir()
	store := Store{IndexPath: filepath.Join(dir, "index.sqlite")}

	db, err := store.Open()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = os.Stat(store.IndexPath)
	require.NoError(t, err)

	var foreignKeys int
	require.NoError(t, db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys))
	require.Equal(t, 1, foreignKeys)

	var journalMode string
	require.NoError(t, db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode))
	require.Equal(t, "wal", journalMode)

	var busyTimeout int
	require.NoError(t, db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout))
	require.Equal(t, 5000, busyTimeout)

	var userVersion int
	require.NoError(t, db.QueryRow(`PRAGMA user_version`).Scan(&userVersion))
	require.Equal(t, 2, userVersion)
}

func TestOpenReadonlyUsesReadOnlyMode(t *testing.T) {
	dir := t.TempDir()
	store := Store{IndexPath: filepath.Join(dir, "index.sqlite")}

	db, err := store.Open()
	require.NoError(t, err)
	require.NoError(t, ApplySchema(db))
	require.NoError(t, db.Close())

	ro, err := store.OpenReadonly()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ro.Close() })

	_, err = ro.Exec(`CREATE TABLE should_fail (id INTEGER)`)
	require.Error(t, err)
}

func TestExistsAndRemove(t *testing.T) {
	dir := t.TempDir()
	store := Store{IndexPath: filepath.Join(dir, "index.sqlite")}

	exists, err := store.Exists()
	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, os.WriteFile(store.IndexPath, []byte("db"), 0o644))
	require.NoError(t, os.WriteFile(store.IndexPath+"-wal", []byte("wal"), 0o644))
	require.NoError(t, os.WriteFile(store.IndexPath+"-shm", []byte("shm"), 0o644))

	exists, err = store.Exists()
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, store.Remove())

	for _, path := range []string{store.IndexPath, store.IndexPath + "-wal", store.IndexPath + "-shm"} {
		_, err := os.Stat(path)
		require.True(t, os.IsNotExist(err), path)
	}
}

func TestRebuild(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(t.TempDir(), "state", "kb", "index.sqlite")
	store := Store{IndexPath: indexPath, RootDir: root, KBID: "kb-1"}

	note := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440001",
		Title:          "Test Note",
		Slug:           "test-note",
		Tags:           []string{"django"},
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("## Summary\n\nTest body.\n"),
	}
	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "test-note.md"), rendered, 0o644))

	result, err := store.Rebuild()
	require.NoError(t, err)
	require.Equal(t, "kb-1", result.KBID)
	require.Equal(t, 1, result.NotesSeen)
	require.Equal(t, 1, result.NotesIndexed)
	require.Equal(t, "ok", result.Status)

	db, err := store.OpenReadonly()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var noteCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&noteCount))
	require.Equal(t, 1, noteCount)

	var tagCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM note_tags`).Scan(&tagCount))
	require.Equal(t, 1, tagCount)

	require.NoError(t, store.QuickCheck())
}

func TestQuickCheckCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	store := Store{IndexPath: filepath.Join(dir, "broken.sqlite")}
	require.NoError(t, os.WriteFile(store.IndexPath, []byte("not sqlite"), 0o644))

	err := store.QuickCheck()
	require.Error(t, err)
	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeCorrupted, appErr.Code)
}

func TestCheckSchemaStatus(t *testing.T) {
	db, err := sql.Open(sqliteDriverName, "file::memory:?cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`PRAGMA user_version = 0`)
	require.NoError(t, err)

	status, err := (Store{}).CheckSchemaStatus(db)
	require.NoError(t, err)
	require.Equal(t, SchemaStatusNeedsRebuild, status)
}

func TestSchemaStatus_OpenError(t *testing.T) {
	store := Store{IndexPath: "/nonexistent/path/index.sqlite"}
	_, err := store.SchemaStatus()
	require.Error(t, err)
}

func TestUnresolvedLinkCount_OpenError(t *testing.T) {
	store := Store{IndexPath: "/nonexistent/path/index.sqlite"}
	_, err := store.UnresolvedLinkCount()
	require.Error(t, err)
}
