package notes

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	sort.Strings(got)

	want := []string{
		"a.md",
		"dir/b.md",
		"dir/sub/c.md",
	}
	assert.Equal(t, want, got)
}

func TestWalkReturnsSlashRelativePaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("dir", "note.md"), "note")

	got, err := Walk(root)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "dir/note.md", got[0])
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
