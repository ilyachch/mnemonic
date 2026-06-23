package fs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAtomicWriteFileReplacesContentAndRemovesTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "note.md")
	initial := []byte("old content\n")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, initial, 0o644))

	var tempPath string
	originalCreateTempFile := createTempFile
	originalRenameFile := renameFile
	originalRemoveFile := removeFile
	originalSyncFile := syncFile
	originalSyncDir := syncDir
	originalWriteAll := writeAll
	t.Cleanup(func() {
		createTempFile = originalCreateTempFile
		renameFile = originalRenameFile
		removeFile = originalRemoveFile
		syncFile = originalSyncFile
		syncDir = originalSyncDir
		writeAll = originalWriteAll
	})

	createTempFile = func(dir string, pattern string) (*os.File, error) {
		f, err := os.CreateTemp(dir, "atomic-write-*")
		if err != nil {
			return nil, err
		}
		tempPath = f.Name()
		return f, nil
	}
	renameFile = func(oldpath, newpath string) error {
		require.Equal(t, filepath.Dir(newpath), filepath.Dir(oldpath))
		return os.Rename(oldpath, newpath)
	}
	syncFile = func(f *os.File) error {
		return nil
	}
	syncDir = func(path string) error {
		return nil
	}

	data := []byte("new content\n")
	require.NoError(t, AtomicWriteFile(path, data, 0o600))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(data), string(got))
	require.NotEmpty(t, tempPath)
	_, err = os.Stat(tempPath)
	require.True(t, errors.Is(err, os.ErrNotExist))
}

func TestAtomicWriteFileKeepsTargetIntactOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	initial := []byte("safe content\n")
	require.NoError(t, os.WriteFile(path, initial, 0o644))

	var tempPath string
	originalCreateTempFile := createTempFile
	originalRenameFile := renameFile
	originalRemoveFile := removeFile
	originalSyncFile := syncFile
	originalSyncDir := syncDir
	originalWriteAll := writeAll
	t.Cleanup(func() {
		createTempFile = originalCreateTempFile
		renameFile = originalRenameFile
		removeFile = originalRemoveFile
		syncFile = originalSyncFile
		syncDir = originalSyncDir
		writeAll = originalWriteAll
	})

	createTempFile = func(dir string, pattern string) (*os.File, error) {
		f, err := os.CreateTemp(dir, "atomic-write-*")
		if err != nil {
			return nil, err
		}
		tempPath = f.Name()
		return f, nil
	}
	writeAll = func(f *os.File, data []byte) error {
		if _, err := f.Write(data[:4]); err != nil {
			return err
		}
		return errors.New("injected write failure")
	}
	syncFile = func(f *os.File) error {
		require.FailNow(t, "sync should not be called after write failure")
		return nil
	}
	renameFile = func(oldpath, newpath string) error {
		require.FailNow(t, "rename should not be called after write failure")
		return nil
	}
	syncDir = func(path string) error {
		require.FailNow(t, "dir sync should not be called after write failure")
		return nil
	}

	err := AtomicWriteFile(path, []byte("corrupted content\n"), 0o600)
	require.Error(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(initial), string(got))
	require.NotEmpty(t, tempPath)
	_, err = os.Stat(tempPath)
	require.True(t, errors.Is(err, os.ErrNotExist))
}

func TestAtomicWriteFileKeepsTargetIntactOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	initial := []byte("safe content\n")
	require.NoError(t, os.WriteFile(path, initial, 0o644))

	var tempPath string
	originalCreateTempFile := createTempFile
	originalRenameFile := renameFile
	originalRemoveFile := removeFile
	originalSyncFile := syncFile
	originalSyncDir := syncDir
	originalWriteAll := writeAll
	t.Cleanup(func() {
		createTempFile = originalCreateTempFile
		renameFile = originalRenameFile
		removeFile = originalRemoveFile
		syncFile = originalSyncFile
		syncDir = originalSyncDir
		writeAll = originalWriteAll
	})

	createTempFile = func(dir string, pattern string) (*os.File, error) {
		f, err := os.CreateTemp(dir, "atomic-write-*")
		if err != nil {
			return nil, err
		}
		tempPath = f.Name()
		return f, nil
	}
	syncFile = func(f *os.File) error {
		return nil
	}
	renameFile = func(oldpath, newpath string) error {
		return errors.New("injected rename failure")
	}
	syncDir = func(path string) error {
		require.FailNow(t, "dir sync should not be called after rename failure")
		return nil
	}

	err := AtomicWriteFile(path, []byte("new content\n"), 0o600)
	require.Error(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(initial), string(got))
	require.NotEmpty(t, tempPath)
	_, err = os.Stat(tempPath)
	require.True(t, errors.Is(err, os.ErrNotExist))
}
