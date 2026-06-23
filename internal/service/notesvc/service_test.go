package notesvc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestServiceRebuildsIndexForCreateEditDelete(t *testing.T) {
	testutil.CleanEnvForTest(t)

	root := t.TempDir()
	stateDir := t.TempDir()
	svc := New(kb.KnowledgeBase{
		ID:        "kb-1",
		RootDir:   root,
		StateDir:  stateDir,
		IndexPath: filepath.Join(stateDir, "index.sqlite"),
	})
	require.Equal(t, root, svc.Notes.RootDir)
	require.Equal(t, stateDir, svc.Notes.StateDir)
	require.Equal(t, stateDir, svc.Index.StateDir)

	created, err := svc.Create(CreateInput{
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
	require.Equal(t, "ok", created.IndexStatus)
	require.Empty(t, created.IndexError)
	require.Equal(t, "auth-migration", created.Slug)

	assertIndexNoteCount(t, svc.Index, 1)

	list, err := svc.List()
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, created.NoteID, list[0].NoteID)

	shown, err := svc.Show(created.Slug)
	require.NoError(t, err)
	require.Equal(t, created.Path, shown.Path)
	require.Equal(t, created.ContentHash, shown.ContentHash)
	require.NotEmpty(t, shown.RawMarkdown)

	edited, err := svc.Edit(EditInput{
		Selector: created.Slug,
		Append:   []byte("Next step"),
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)
		},
	})
	require.NoError(t, err)
	require.Equal(t, "ok", edited.IndexStatus)
	require.Empty(t, edited.IndexError)
	require.NotEqual(t, created.ContentHash, edited.ContentHash)

	shown, err = svc.Show(created.Slug)
	require.NoError(t, err)
	require.Equal(t, "## Summary\nNext step", string(shown.Note.Body))
	require.Equal(t, shown.ContentHash, markdownstore.HashBytes(shown.RawMarkdown))

	assertIndexNoteCount(t, svc.Index, 1)

	deleted, err := svc.Delete(DeleteInput{
		Selector: created.Slug,
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 15, 4, 5, 0, time.UTC)
		},
	})
	require.NoError(t, err)
	require.Equal(t, "trash", deleted.Mode)
	require.Equal(t, "ok", deleted.IndexStatus)
	require.Empty(t, deleted.IndexError)

	_, err = os.Stat(filepath.Join(root, "auth-migration.md"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	assertIndexNoteCount(t, svc.Index, 0)
}

func TestServiceReturnsPartialSuccessWhenIndexRebuildFails(t *testing.T) {
	testutil.CleanEnvForTest(t)

	root := t.TempDir()
	stateDir := t.TempDir()
	indexPath := filepath.Join(stateDir, "index.sqlite")
	svc := &Service{
		Notes: markdownstore.Store{RootDir: root, StateDir: stateDir},
		Index: sqliteindex.Store{IndexPath: indexPath, StateDir: stateDir, KBID: "kb-1"},
	}
	svc.Index.RootDir = ""

	created, err := svc.Create(CreateInput{
		Title: "Auth migration",
		Body:  []byte("## Summary\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440000"
		},
	})
	require.NoError(t, err)
	require.Equal(t, "stale", created.IndexStatus)
	require.NotEmpty(t, created.IndexError)

	edited, err := svc.Edit(EditInput{
		Selector: created.Slug,
		Append:   []byte("Next step"),
	})
	require.NoError(t, err)
	require.Equal(t, "stale", edited.IndexStatus)
	require.NotEmpty(t, edited.IndexError)

	_, statErr := os.Stat(filepath.Join(root, "auth-migration.md"))
	require.NoError(t, statErr)

	deleted, err := svc.Delete(DeleteInput{
		Selector: created.Slug,
	})
	require.NoError(t, err)
	require.Equal(t, "stale", deleted.IndexStatus)
	require.NotEmpty(t, deleted.IndexError)

	_, statErr = os.Stat(filepath.Join(root, "auth-migration.md"))
	require.Error(t, statErr)
	require.True(t, os.IsNotExist(statErr))
}

func assertIndexNoteCount(t *testing.T, store sqliteindex.Store, want int) {
	t.Helper()

	db, err := store.OpenReadonly()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var got int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&got))
	require.Equal(t, want, got)
}
