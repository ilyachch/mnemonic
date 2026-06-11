package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFileCreatesParentsAndReplacesContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "project", "mnemonic.toml")

	first := []byte("version = 1\n")
	require.NoError(t, WriteFile(path, first, 0o644))

	_, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(first), string(got))

	second := []byte("version = 2\n")
	require.NoError(t, WriteFile(path, second, 0o600))

	got, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(second), string(got))
}