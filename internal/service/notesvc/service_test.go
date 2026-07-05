package notesvc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/assert"
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
	}, nil)
	require.Equal(t, root, svc.Notes.RootDir)
	require.Equal(t, stateDir, svc.Notes.StateDir)
	require.Equal(t, filepath.Join(stateDir, "index.sqlite"), svc.Notes.IndexPath)
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
		Notes: markdownstore.Store{RootDir: root, StateDir: stateDir, IndexPath: indexPath},
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

	seedServiceIndexFromFile(t, indexPath, filepath.Join(root, created.Path))

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

func seedServiceIndexFromFile(t *testing.T, indexPath, path string) {
	t.Helper()

	store := sqliteindex.Store{IndexPath: indexPath}
	db, err := store.Open()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqliteindex.ApplySchema(db))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)

	_, err = db.Exec(
		`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		note.MnemonicNoteID, "kb-1", note.EffectiveSlug(), filepath.Base(path), note.Title, "hash-"+note.MnemonicNoteID,
		note.CreatedAt.UTC().Format(time.RFC3339), note.UpdatedAt.UTC().Format(time.RFC3339),
	)
	require.NoError(t, err)
}

func TestServiceHydrateDelegatesToStore(t *testing.T) {
	testutil.CleanEnvForTest(t)

	root := t.TempDir()
	stateDir := t.TempDir()
	svc := New(kb.KnowledgeBase{
		ID:        "kb-1",
		RootDir:   root,
		StateDir:  stateDir,
		IndexPath: filepath.Join(stateDir, "index.sqlite"),
	}, nil)

	raw := []byte("# Delegated Note\n\nBody.\n")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "delegated.md"), raw, 0o644))

	result, err := svc.Hydrate(markdownstore.HydrateInput{
		Now:  func() time.Time { return time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC) },
		UUID: func() string { return "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" },
	})
	require.NoError(t, err)
	require.Len(t, result.Hydrated, 1)
	assert.Equal(t, "delegated.md", result.Hydrated[0].Path)
	assert.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", result.Hydrated[0].NoteID)
	assert.Equal(t, "Delegated Note", result.Hydrated[0].Title)
	assert.Equal(t, "delegated-note", result.Hydrated[0].Slug)

	data, err := os.ReadFile(filepath.Join(root, "delegated.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	assert.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", note.MnemonicNoteID)
}

func TestServiceHydrateDryRunDoesNotWrite(t *testing.T) {
	testutil.CleanEnvForTest(t)

	root := t.TempDir()
	stateDir := t.TempDir()
	svc := New(kb.KnowledgeBase{
		ID:        "kb-1",
		RootDir:   root,
		StateDir:  stateDir,
		IndexPath: filepath.Join(stateDir, "index.sqlite"),
	}, nil)

	raw := []byte("# Dry Note\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "dry.md"), raw, 0o644))

	result, err := svc.Hydrate(markdownstore.HydrateInput{
		DryRun: true,
		Now:    func() time.Time { return time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC) },
		UUID:   func() string { return "ffffffff-0000-0000-0000-000000000000" },
	})
	require.NoError(t, err)
	require.Len(t, result.Hydrated, 1)
	assert.True(t, result.DryRun)

	data, err := os.ReadFile(filepath.Join(root, "dry.md"))
	require.NoError(t, err)
	assert.Equal(t, raw, data)
}

func TestShowManyReturnsMissingForUnknownSelector(t *testing.T) {
	testutil.CleanEnvForTest(t)

	root := t.TempDir()
	stateDir := t.TempDir()
	svc := New(kb.KnowledgeBase{
		ID:        "kb-1",
		RootDir:   root,
		StateDir:  stateDir,
		IndexPath: filepath.Join(stateDir, "index.sqlite"),
	}, nil)

	output, err := svc.ShowMany(ReadManyInput{Selectors: []string{"nonexistent"}})
	require.NoError(t, err)
	assert.Empty(t, output.Notes)
	assert.Equal(t, []string{"nonexistent"}, output.Missing)
	assert.Empty(t, output.Issues)
}

func TestShowManySplitsFoundAndMissing(t *testing.T) {
	testutil.CleanEnvForTest(t)

	root := t.TempDir()
	stateDir := t.TempDir()
	svc := New(kb.KnowledgeBase{
		ID:        "kb-1",
		RootDir:   root,
		StateDir:  stateDir,
		IndexPath: filepath.Join(stateDir, "index.sqlite"),
	}, nil)

	created, err := svc.Create(CreateInput{
		Title: "Test note",
		Body:  []byte("# Test\ncontent"),
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
		},
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440001"
		},
	})
	require.NoError(t, err)

	output, err := svc.ShowMany(ReadManyInput{Selectors: []string{created.Slug, "does-not-exist"}})
	require.NoError(t, err)
	require.Len(t, output.Notes, 1)
	assert.Equal(t, created.Slug, output.Notes[0].ID)
	require.Len(t, output.Missing, 1)
	assert.Equal(t, "does-not-exist", output.Missing[0])
	assert.Empty(t, output.Issues)
}

func TestShowManyCorruptedNoteGoesToIssues(t *testing.T) {
	testutil.CleanEnvForTest(t)

	root := t.TempDir()
	stateDir := t.TempDir()
	indexPath := filepath.Join(stateDir, "index.sqlite")

	require.NoError(t, os.WriteFile(filepath.Join(root, "broken.md"), []byte("---\ntitle: [bad yaml\n---\nBody"), 0o644))

	store := sqliteindex.Store{IndexPath: indexPath, RootDir: root, StateDir: stateDir, KBID: "kb-1"}
	db, err := store.Open()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqliteindex.ApplySchema(db))

	_, err = db.Exec(
		`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"broken-id", "kb-1", "broken", "broken.md", "Broken", "hash-broken",
		time.Date(2026, time.June, 2, 0, 0, 0, 0, time.UTC).Unix(),
		time.Date(2026, time.June, 2, 0, 0, 0, 0, time.UTC).Unix(),
	)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	svc := &Service{
		Notes: markdownstore.Store{RootDir: root, StateDir: stateDir, IndexPath: indexPath},
		Index: sqliteindex.Store{IndexPath: indexPath, RootDir: root, StateDir: stateDir, KBID: "kb-1"},
	}

	output, err := svc.ShowMany(ReadManyInput{Selectors: []string{"broken"}})
	require.NoError(t, err)
	assert.Empty(t, output.Notes)
	assert.Empty(t, output.Missing)
	require.Len(t, output.Issues, 1)
	assert.Equal(t, "broken", output.Issues[0].Selector)
	assert.Equal(t, "corrupted", output.Issues[0].Kind)
	assert.Contains(t, output.Issues[0].Message, "parse")
}

func TestShowManyReturnsSuccessNotesAlongsideMissingAndIssues(t *testing.T) {
	testutil.CleanEnvForTest(t)

	root := t.TempDir()
	stateDir := t.TempDir()
	svc := New(kb.KnowledgeBase{
		ID:        "kb-1",
		RootDir:   root,
		StateDir:  stateDir,
		IndexPath: filepath.Join(stateDir, "index.sqlite"),
	}, nil)

	created, err := svc.Create(CreateInput{
		Title: "Working note",
		Body:  []byte("# Work\ncontent"),
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
		},
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440099"
		},
	})
	require.NoError(t, err)

	output, err := svc.ShowMany(ReadManyInput{Selectors: []string{created.Slug, "does-not-exist"}})
	require.NoError(t, err)
	require.Len(t, output.Notes, 1)
	assert.Equal(t, created.Slug, output.Notes[0].ID)
	require.Len(t, output.Missing, 1)
	assert.Equal(t, "does-not-exist", output.Missing[0])
	assert.Empty(t, output.Issues)
}

func TestShowManyOrderPreserved(t *testing.T) {
	testutil.CleanEnvForTest(t)

	root := t.TempDir()
	stateDir := t.TempDir()
	svc := New(kb.KnowledgeBase{
		ID:        "kb-1",
		RootDir:   root,
		StateDir:  stateDir,
		IndexPath: filepath.Join(stateDir, "index.sqlite"),
	}, nil)

	slugs := make([]string, 0, 2)
	for i, title := range []string{"Alpha", "Beta"} {
		created, err := svc.Create(CreateInput{
			Title: title,
			Body:  []byte("# " + title),
			Now: func() time.Time {
				return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
			},
			UUID: func() string {
				return "550e8400-e29b-41d4-a716-44665544001" + string(rune('0'+i))
			},
		})
		require.NoError(t, err)
		slugs = append(slugs, created.Slug)
	}

	output, err := svc.ShowMany(ReadManyInput{Selectors: []string{slugs[1], slugs[0]}})
	require.NoError(t, err)
	require.Len(t, output.Notes, 2)
	assert.Equal(t, slugs[1], output.Notes[0].ID)
	assert.Equal(t, slugs[0], output.Notes[1].ID)
}

func TestClassifyShowErrorNotFoundGoesToMissing(t *testing.T) {
	missing, issues := classifyShowError("s", apperr.NotFound("note not found", nil), nil, nil)
	assert.Equal(t, []string{"s"}, missing)
	assert.Empty(t, issues)
}

func TestClassifyShowErrorAmbiguousGoesToIssues(t *testing.T) {
	missing, issues := classifyShowError("s", apperr.Ambiguous("matches multiple", nil), nil, nil)
	assert.Empty(t, missing)
	require.Len(t, issues, 1)
	assert.Equal(t, "s", issues[0].Selector)
	assert.Equal(t, "ambiguous", issues[0].Kind)
}

func TestClassifyShowErrorCorruptedGoesToIssues(t *testing.T) {
	missing, issues := classifyShowError("s", apperr.Corrupted("bad yaml", nil), nil, nil)
	assert.Empty(t, missing)
	require.Len(t, issues, 1)
	assert.Equal(t, "corrupted", issues[0].Kind)
}

func TestClassifyShowErrorParseErrorBecomesCorrupted(t *testing.T) {
	err := errors.New("parse note \"test.md\": yaml: line 1: could not find expected ':'")
	missing, issues := classifyShowError("a", err, nil, nil)
	assert.Empty(t, missing)
	require.Len(t, issues, 1)
	assert.Equal(t, "corrupted", issues[0].Kind)
}

func TestClassifyShowErrorReadErrorBecomesIOError(t *testing.T) {
	err := errors.New("read note \"test.md\": permission denied")
	missing, issues := classifyShowError("a", err, nil, nil)
	assert.Empty(t, missing)
	require.Len(t, issues, 1)
	assert.Equal(t, "io_error", issues[0].Kind)
}

func TestClassifyShowErrorOSPermissionBecomesIOError(t *testing.T) {
	err := os.ErrPermission
	missing, issues := classifyShowError("a", err, nil, nil)
	assert.Empty(t, missing)
	require.Len(t, issues, 1)
	assert.Equal(t, "io_error", issues[0].Kind)
}

func TestClassifyShowErrorUnknownBecomesInternal(t *testing.T) {
	err := errors.New("something unexpected happened")
	missing, issues := classifyShowError("x", err, nil, nil)
	assert.Empty(t, missing)
	require.Len(t, issues, 1)
	assert.Equal(t, "internal", issues[0].Kind)
}
