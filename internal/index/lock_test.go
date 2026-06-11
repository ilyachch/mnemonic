package index

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/lock"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestRebuildProjectIndexReturnsBusyErrorWhenLockHeld(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesRoot := filepath.Join(t.TempDir(), "memories")
	require.NoError(t, createIndexTestNote(memoriesRoot, "alpha.md", "# Alpha\n"))

	guard, err := lock.Acquire(lock.AcquireInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      reindexLockName,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = guard.Release() })

	_, err = RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot)
	require.Error(t, err)

	var appErr *app.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, app.CodeUnsafe, appErr.Code)
}

func createIndexTestNote(root string, relPath string, content string) error {
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(relPath)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, relPath), []byte(content), 0o644)
}