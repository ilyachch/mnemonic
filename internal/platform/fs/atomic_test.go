package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAtomicWriteFileCreatesParentsAndWritesContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "note.md")
	data := []byte("new content\n")

	require.NoError(t, AtomicWriteFile(path, data, 0o600))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(data), string(got))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestAtomicWriteFileReplacesContentAndRemovesTemp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	initial := []byte("old content\n")
	require.NoError(t, os.WriteFile(path, initial, 0o644))

	data := []byte("new content\n")
	require.NoError(t, AtomicWriteFile(path, data, 0o600))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(data), string(got))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	require.Equal(t, []string{"note.md"}, names)
}

func TestAtomicWriteFileKeepsTargetIntactOnRenameFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Make the target path a non-empty directory so the final rename fails
	// with ENOTEMPTY while the temp file is still created in the same dir.
	path := filepath.Join(dir, "note.md")
	require.NoError(t, os.MkdirAll(path, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(path, "blocker"), []byte("x"), 0o644))

	err := AtomicWriteFile(path, []byte("new content\n"), 0o600)
	require.Error(t, err)

	// The original directory (target) must remain untouched.
	got, err := os.ReadFile(filepath.Join(path, "blocker"))
	require.NoError(t, err)
	require.Equal(t, "x", string(got))

	// No leftover temp files in the parent directory.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	require.Equal(t, []string{"note.md"}, names)
}

func TestAtomicWriteFileReportsMkdirAllFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Place a file where a parent directory is expected so MkdirAll fails.
	blocker := filepath.Join(dir, "block")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	path := filepath.Join(blocker, "nested", "note.md")

	err := AtomicWriteFile(path, []byte("data\n"), 0o600)
	require.Error(t, err)
}
