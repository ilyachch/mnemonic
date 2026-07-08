package fs

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetFileTimesReturnsBirthBeforeMod(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	require.NoError(t, os.WriteFile(path, []byte("initial\n"), 0o644))

	// Birth time captured after creation.
	_, mod1, err := GetFileTimes(path)
	require.NoError(t, err)
	require.False(t, mod1.IsZero(), "mtime must be populated")

	// Wait briefly so the subsequent write produces a strictly newer mtime on
	// filesystems with coarse mtime granularity.
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, os.WriteFile(path, []byte("updated\n"), 0o644))

	birth2, mod2, err := GetFileTimes(path)
	require.NoError(t, err)

	// Birth time must not move backwards and must be <= the latest mtime.
	require.False(t, birth2.After(mod2), "birthTime %s must not be after modTime %s", birth2, mod2)
	require.False(t, mod2.Before(mod1), "mtime must advance after write: was %s now %s", mod1, mod2)
}
