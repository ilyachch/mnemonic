package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileCreatesParentsAndReplacesContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "project", "mnemonic.toml")

	first := []byte("version = 1\n")
	if err := WriteFile(path, first, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("parent directory missing after WriteFile(): %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(first) {
		t.Fatalf("first write = %q, want %q", got, first)
	}

	second := []byte("version = 2\n")
	if err := WriteFile(path, second, 0o600); err != nil {
		t.Fatalf("second WriteFile() error = %v", err)
	}

	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() after overwrite error = %v", err)
	}
	if string(got) != string(second) {
		t.Fatalf("overwrite = %q, want %q", got, second)
	}
}
