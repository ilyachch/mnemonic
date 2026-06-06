package fs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteFileReplacesContentAndRemovesTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "note.md")
	initial := []byte("old content\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

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
		if filepath.Dir(oldpath) != filepath.Dir(newpath) {
			t.Fatalf("temp dir = %q, target dir = %q", filepath.Dir(oldpath), filepath.Dir(newpath))
		}
		return os.Rename(oldpath, newpath)
	}
	syncFile = func(f *os.File) error {
		return nil
	}
	syncDir = func(path string) error {
		return nil
	}

	data := []byte("new content\n")
	if err := AtomicWriteFile(path, data, 0o600); err != nil {
		t.Fatalf("AtomicWriteFile() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("target content = %q, want %q", got, data)
	}
	if tempPath == "" {
		t.Fatal("temp path was not recorded")
	}
	if _, err := os.Stat(tempPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp file still exists: %v", err)
	}
}

func TestAtomicWriteFileKeepsTargetIntactOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	initial := []byte("safe content\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

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
		t.Fatal("sync should not be called after write failure")
		return nil
	}
	renameFile = func(oldpath, newpath string) error {
		t.Fatal("rename should not be called after write failure")
		return nil
	}
	syncDir = func(path string) error {
		t.Fatal("dir sync should not be called after write failure")
		return nil
	}

	err := AtomicWriteFile(path, []byte("corrupted content\n"), 0o600)
	if err == nil {
		t.Fatal("AtomicWriteFile() error = nil, want failure")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(initial) {
		t.Fatalf("target content = %q, want %q", got, initial)
	}
	if tempPath == "" {
		t.Fatal("temp path was not recorded")
	}
	if _, err := os.Stat(tempPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp file still exists after failure: %v", err)
	}
}

func TestAtomicWriteFileKeepsTargetIntactOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	initial := []byte("safe content\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

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
		t.Fatal("dir sync should not be called after rename failure")
		return nil
	}

	err := AtomicWriteFile(path, []byte("new content\n"), 0o600)
	if err == nil {
		t.Fatal("AtomicWriteFile() error = nil, want failure")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(initial) {
		t.Fatalf("target content = %q, want %q", got, initial)
	}
	if tempPath == "" {
		t.Fatal("temp path was not recorded")
	}
	if _, err := os.Stat(tempPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp file still exists after rename failure: %v", err)
	}
}
