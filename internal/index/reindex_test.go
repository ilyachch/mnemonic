package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanNotes(t *testing.T) {
	root := t.TempDir()

	note := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Test Note",
		Slug:           "test-note",
		Tags:           []string{"django", "auth"},
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("## Summary\n\nTest body with #inline-tag.\n"),
	}
	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)

	notePath := filepath.Join(root, "test-note.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, rendered, 0o644))

	docs, errs, err := ScanNotes(root)
	require.NoError(t, err)
	require.Empty(t, errs)
	require.Len(t, docs, 1)

	doc := docs[0]
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", doc.NoteID)
	assert.Equal(t, "test-note", doc.Slug)
	assert.Equal(t, "Test Note", doc.Title)
	assert.Equal(t, "test-note.md", doc.RelPath)
	assert.NotEmpty(t, doc.ContentHash)
	assert.NotEmpty(t, doc.SearchText)

	foundDjango := false
	foundAuth := false
	foundInline := false
	for _, tag := range doc.Tags {
		if tag.Value == "django" {
			foundDjango = true
		}
		if tag.Value == "auth" {
			foundAuth = true
		}
		if tag.Value == "inline-tag" && tag.Source == "inline" {
			foundInline = true
		}
	}
	assert.True(t, foundDjango, "should have django tag")
	assert.True(t, foundAuth, "should have auth tag")
	assert.True(t, foundInline, "should have inline tag")
}

func TestScanNotesSkipsMnemonicToml(t *testing.T) {
	root := t.TempDir()

	note := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Test Note",
		Slug:           "test-note",
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
	}
	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(root, "test-note.md"), rendered, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "mnemonic.toml"), []byte("version = 1\n"), 0o644))

	docs, errs, err := ScanNotes(root)
	require.NoError(t, err)
	require.Empty(t, errs)
	require.Len(t, docs, 1)
	assert.Equal(t, "test-note.md", docs[0].RelPath)
}

func TestScanNotesWithRelations(t *testing.T) {
	root := t.TempDir()

	note := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Source Note",
		Slug:           "source-note",
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("## Relations\n- depends_on [[Target Note]]\n- relates_to [[Other Note]]\n"),
	}
	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(root, "source-note.md"), rendered, 0o644))

	docs, errs, err := ScanNotes(root)
	require.NoError(t, err)
	require.Empty(t, errs)
	require.Len(t, docs, 1)

	assert.Len(t, docs[0].Links, 2)
}

func TestScanNotesWithObservations(t *testing.T) {
	root := t.TempDir()

	note := markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440000",
		Title:          "Observed Note",
		Slug:           "observed-note",
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("## Observations\n- [query] SELECT * FROM users\n- [metric] 42 requests/sec\n"),
	}
	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(root, "observed-note.md"), rendered, 0o644))

	docs, errs, err := ScanNotes(root)
	require.NoError(t, err)
	require.Empty(t, errs)
	require.Len(t, docs, 1)

	assert.Len(t, docs[0].Observations, 2)
}

func TestRebuildProjectIndex(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	root := filepath.Join(projectRoot, "memories")

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

	result, err := RebuildProjectIndex(projectID, root)
	require.NoError(t, err)
	assert.Equal(t, projectID, result.ProjectID)
	assert.Equal(t, 1, result.NotesSeen)
	assert.Equal(t, 1, result.NotesIndexed)
	assert.Equal(t, "ok", result.Status)

	indexPath, err := Path(projectID)
	require.NoError(t, err)
	_, err = os.Stat(indexPath)
	require.NoError(t, err)

	err = QuickCheck(indexPath)
	require.NoError(t, err)
}

func TestRebuildProjectIndexEmptyProject(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	root := filepath.Join(projectRoot, "memories")
	require.NoError(t, os.MkdirAll(root, 0o755))

	result, err := RebuildProjectIndex(projectID, root)
	require.NoError(t, err)
	assert.Equal(t, 0, result.NotesSeen)
	assert.Equal(t, 0, result.NotesIndexed)
}

func TestNormalizeTitleSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Auth migration", "auth-migration"},
		{"  Leading spaces  ", "leading-spaces"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"under_score", "under-score"},
		{"already-slug", "already-slug"},
		{"Mixed_Case-Text", "mixed-case-text"},
		{"Special!@#Chars", "specialchars"},
		{"", ""},
		{"---", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeTitleSlug(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHashNoteBytes(t *testing.T) {
	got := hashNoteBytes([]byte("test"))
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "sha256:")
}

func TestHashString(t *testing.T) {
	got := hashString("test")
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "sha256:")

	assert.Equal(t, hashString("test"), hashString("test"))
	assert.NotEqual(t, hashString("test"), hashString("other"))
}

func TestBoolToInt(t *testing.T) {
	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))
}

func TestResolveLinkTarget(t *testing.T) {
	docs := []NoteDoc{
		{NoteID: "id-1", Slug: "slug-1", Title: "Title One", RelPath: "one.md"},
		{NoteID: "id-2", Slug: "slug-2", Title: "Title Two", RelPath: "two.md"},
	}
	norms := map[string]int{"title-one": 1, "title-two": 1}

	t.Run("by note_id", func(t *testing.T) {
		id, ok := resolveLinkTarget(docs, norms, "id-1")
		require.True(t, ok)
		assert.Equal(t, "id-1", id)
	})

	t.Run("by slug", func(t *testing.T) {
		id, ok := resolveLinkTarget(docs, norms, "slug-2")
		require.True(t, ok)
		assert.Equal(t, "id-2", id)
	})

	t.Run("by rel_path", func(t *testing.T) {
		id, ok := resolveLinkTarget(docs, norms, "one.md")
		require.True(t, ok)
		assert.Equal(t, "id-1", id)
	})

	t.Run("by title", func(t *testing.T) {
		id, ok := resolveLinkTarget(docs, norms, "Title Two")
		require.True(t, ok)
		assert.Equal(t, "id-2", id)
	})

	t.Run("by normalized title", func(t *testing.T) {
		id, ok := resolveLinkTarget(docs, norms, "title-one")
		require.True(t, ok)
		assert.Equal(t, "id-1", id)
	})

	t.Run("ambiguous normalized title", func(t *testing.T) {
		ambigNorms := map[string]int{"title-one": 2}
		_, ok := resolveLinkTarget(docs, ambigNorms, "title-one")
		require.False(t, ok)
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := resolveLinkTarget(docs, norms, "nonexistent")
		require.False(t, ok)
	})
}

func TestTempIndexPath(t *testing.T) {
	testutil.CleanEnvForTest(t)
	projectID := "550e8400-e29b-41d4-a716-446655440000"

	got, err := tempIndexPath(projectID)
	require.NoError(t, err)
	assert.Contains(t, got, projectID)
	assert.Contains(t, got, ".new.sqlite")
}

func TestRemoveExistingIndexFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.sqlite")
	walPath := path + "-wal"
	shmPath := path + "-shm"

	require.NoError(t, os.WriteFile(path, []byte("data"), 0o644))
	require.NoError(t, os.WriteFile(walPath, []byte("wal"), 0o644))
	require.NoError(t, os.WriteFile(shmPath, []byte("shm"), 0o644))

	err := removeExistingIndexFiles(path)
	require.NoError(t, err)

	_, err = os.Stat(path)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(walPath)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(shmPath)
	require.True(t, os.IsNotExist(err))

	err = removeExistingIndexFiles(path)
	require.NoError(t, err)
}
