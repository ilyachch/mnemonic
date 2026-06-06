package notes

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestWalkFindsMarkdownNotes(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "a.md", "root")
	writeFile(t, root, filepath.Join("dir", "b.md"), "b")
	writeFile(t, root, filepath.Join("dir", "sub", "c.md"), "c")
	writeFile(t, root, "mnemonic.toml", "ignore")
	writeFile(t, root, filepath.Join("dir", "mnemonic.toml"), "ignore")
	writeFile(t, root, filepath.Join(".trash", "deleted.md"), "trash")
	writeFile(t, root, filepath.Join("dir", ".trash", "nested.md"), "trash")
	writeFile(t, root, filepath.Join("dir", "note.txt"), "skip")

	got, err := Walk(root)
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	sort.Strings(got)

	want := []string{
		"a.md",
		"dir/b.md",
		"dir/sub/c.md",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Walk() = %#v, want %#v", got, want)
	}
}

func TestWalkReturnsSlashRelativePaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("dir", "note.md"), "note")

	got, err := Walk(root)
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Walk() len = %d, want 1", len(got))
	}
	if got[0] != "dir/note.md" {
		t.Fatalf("Walk() path = %q, want %q", got[0], "dir/note.md")
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
