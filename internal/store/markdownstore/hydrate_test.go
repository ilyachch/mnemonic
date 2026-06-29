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

func TestStoreWalkSkipsHiddenDirectories(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root, StateDir: t.TempDir()}

	note := renderedListTestNote(t, "id-1", "Real", "real", noteTime())
	writeTestFile(t, filepath.Join(root, "real.md"), note)
	writeTestFile(t, filepath.Join(root, ".git", "config.md"), note)
	writeTestFile(t, filepath.Join(root, ".obsidian", "workspace.md"), note)
	writeTestFile(t, filepath.Join(root, "sub", ".cache", "cached.md"), note)
	writeTestFile(t, filepath.Join(root, "sub", "keep.md"), note)

	paths, err := store.Walk()
	require.NoError(t, err)
	sort.Strings(paths)
	assert.Equal(t, []string{"real.md", "sub/keep.md"}, paths)
}

func TestStoreHydrateAddsFrontmatterToRawNote(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root, StateDir: t.TempDir()}

	raw := []byte("# My Raw Note\n\nSome body text.\n")
	writeTestFile(t, filepath.Join(root, "raw.md"), raw)

	fixedTime := time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC)
	result, err := store.Hydrate(HydrateInput{
		Now:  func() time.Time { return fixedTime },
		UUID: func() string { return "11111111-2222-3333-4444-555555555555" },
	})
	require.NoError(t, err)
	require.Len(t, result.Hydrated, 1)
	assert.Equal(t, "raw.md", result.Hydrated[0].Path)
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", result.Hydrated[0].NoteID)
	assert.Equal(t, "My Raw Note", result.Hydrated[0].Title)
	assert.Equal(t, "my-raw-note", result.Hydrated[0].Slug)

	data, err := os.ReadFile(filepath.Join(root, "raw.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", note.MnemonicNoteID)
	assert.Equal(t, "My Raw Note", note.Title)
	assert.Equal(t, "my-raw-note", note.Slug)
	assert.Equal(t, "My Raw Note", note.Title)
	assert.Contains(t, string(note.Body), "Some body text.")
}

func TestStoreHydratePreservesCustomFrontmatter(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root, StateDir: t.TempDir()}

	raw := []byte("---\nauthor: jane\ntitle: Custom Title\nstatus: draft\n---\n# Custom Title\n\nBody.\n")
	writeTestFile(t, filepath.Join(root, "custom.md"), raw)

	result, err := store.Hydrate(HydrateInput{
		Now:  func() time.Time { return noteTime() },
		UUID: func() string { return "22222222-3333-4444-5555-666666666666" },
	})
	require.NoError(t, err)
	require.Len(t, result.Hydrated, 1)
	assert.Equal(t, "Custom Title", result.Hydrated[0].Title)

	data, err := os.ReadFile(filepath.Join(root, "custom.md"))
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	assert.Equal(t, "22222222-3333-4444-5555-666666666666", note.MnemonicNoteID)
	assert.Equal(t, "Custom Title", note.Title)
	assert.Equal(t, "custom-title", note.Slug)
	assert.Equal(t, "jane", note.Frontmatter["author"])
	assert.Equal(t, "draft", note.Frontmatter["status"])
	assert.Contains(t, string(note.Body), "Body.")
}

func TestStoreHydrateSkipsNotesWithExistingID(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root, StateDir: t.TempDir()}

	existing := renderedListTestNote(t, "existing-id", "Existing", "existing", noteTime())
	writeTestFile(t, filepath.Join(root, "existing.md"), existing)

	raw := []byte("# New Note\n\nBody.\n")
	writeTestFile(t, filepath.Join(root, "new.md"), raw)

	result, err := store.Hydrate(HydrateInput{
		Now:  func() time.Time { return noteTime() },
		UUID: func() string { return "33333333-4444-5555-6666-777777777777" },
	})
	require.NoError(t, err)
	require.Len(t, result.Hydrated, 1)
	assert.Equal(t, "new.md", result.Hydrated[0].Path)
	assert.Equal(t, []string{"existing.md"}, result.Skipped)

	existingData, err := os.ReadFile(filepath.Join(root, "existing.md"))
	require.NoError(t, err)
	assert.Equal(t, existing, existingData)
}

func TestStoreHydrateFallsBackToBaseNameWhenNoH1(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root, StateDir: t.TempDir()}

	raw := []byte("Just some text without a heading.\n")
	writeTestFile(t, filepath.Join(root, "untitled-note.md"), raw)

	result, err := store.Hydrate(HydrateInput{
		Now:  func() time.Time { return noteTime() },
		UUID: func() string { return "44444444-5555-6666-7777-888888888888" },
	})
	require.NoError(t, err)
	require.Len(t, result.Hydrated, 1)
	assert.Equal(t, "untitled-note", result.Hydrated[0].Title)
	assert.Equal(t, "untitled-note", result.Hydrated[0].Slug)
}

func TestStoreHydrateDryRunDoesNotWrite(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root, StateDir: t.TempDir()}

	raw := []byte("# Dry Run Note\n\nBody.\n")
	writeTestFile(t, filepath.Join(root, "dry.md"), raw)

	result, err := store.Hydrate(HydrateInput{
		DryRun: true,
		Now:    func() time.Time { return noteTime() },
		UUID:   func() string { return "55555555-6666-7777-8888-999999999999" },
	})
	require.NoError(t, err)
	require.Len(t, result.Hydrated, 1)
	assert.True(t, result.DryRun)

	data, err := os.ReadFile(filepath.Join(root, "dry.md"))
	require.NoError(t, err)
	assert.Equal(t, raw, data)
}

func TestStoreHydrateUsesFileMtimeForTimestamps(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root, StateDir: t.TempDir()}

	raw := []byte("# Mtime Note\n\nBody.\n")
	notePath := filepath.Join(root, "mtime.md")
	writeTestFile(t, notePath, raw)

	mtime := time.Date(2025, time.December, 25, 8, 30, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(notePath, mtime, mtime))

	fallback := time.Date(2026, time.June, 29, 10, 0, 0, 0, time.UTC)
	_, err := store.Hydrate(HydrateInput{
		Now:  func() time.Time { return fallback },
		UUID: func() string { return "66666666-7777-8888-9999-000000000000" },
	})
	require.NoError(t, err)

	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	note, err := markdown.ParseNote(data)
	require.NoError(t, err)
	assert.True(t, note.CreatedAt.Equal(mtime))
	assert.True(t, note.UpdatedAt.Equal(mtime))
}

func TestStoreHydrateRejectsFileOutsideRoot(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root, StateDir: t.TempDir()}

	external := t.TempDir()
	externalFile := filepath.Join(external, "outside.md")
	writeTestFile(t, externalFile, []byte("# Outside\n"))

	_, err := store.Hydrate(HydrateInput{
		Files: []string{externalFile},
		Now:   func() time.Time { return noteTime() },
		UUID:  func() string { return "77777777-8888-9999-0000-111111111111" },
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside project root")
}

func TestStoreHydrateProcessesExplicitFiles(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	store := Store{RootDir: root, StateDir: t.TempDir()}

	writeTestFile(t, filepath.Join(root, "a.md"), []byte("# A Note\n"))
	writeTestFile(t, filepath.Join(root, "b.md"), []byte("# B Note\n"))

	result, err := store.Hydrate(HydrateInput{
		Files: []string{"b.md"},
		Now:   func() time.Time { return noteTime() },
		UUID:  func() string { return "88888888-9999-0000-1111-222222222222" },
	})
	require.NoError(t, err)
	require.Len(t, result.Hydrated, 1)
	assert.Equal(t, "b.md", result.Hydrated[0].Path)

	aData, err := os.ReadFile(filepath.Join(root, "a.md"))
	require.NoError(t, err)
	assert.Equal(t, "# A Note\n", string(aData))
}
