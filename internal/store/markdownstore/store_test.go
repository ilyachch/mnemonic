package markdownstore

import (
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
	store := Store{RootDir: root}

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

func TestStoreResolveAndShowSelectorPrecedence(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root}

	writeNote(t, root, "uuid.md", markdown.Note{
		MnemonicNoteID: "11111111-1111-1111-1111-111111111111",
		Title:          "UUID Note",
		Slug:           "uuid-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeNote(t, root, "slug.md", markdown.Note{
		MnemonicNoteID: "22222222-2222-2222-2222-222222222222",
		Title:          "Slug Note",
		Slug:           "slug-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeNote(t, root, filepath.Join("folder", "path.md"), markdown.Note{
		MnemonicNoteID: "33333333-3333-3333-3333-333333333333",
		Title:          "Path Note",
		Slug:           "path-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeNote(t, root, "title.md", markdown.Note{
		MnemonicNoteID: "44444444-4444-4444-4444-444444444444",
		Title:          "Exact Title",
		Slug:           "exact-title-slug",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeNote(t, root, "normalized.md", markdown.Note{
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
			got, err := store.Resolve(tc.selector)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPath, got.Path)

			shown, err := store.Show(tc.selector)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPath, shown.Path)
		})
	}
}

func TestStoreListAndWalkIgnoreTrash(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root}

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

func writeNote(t *testing.T, root, relPath string, note markdown.Note) {
	t.Helper()

	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)
	writeTestFile(t, filepath.Join(root, relPath), rendered)
}

func writeTestFile(t *testing.T, path string, content []byte) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, content, 0o644))
}
