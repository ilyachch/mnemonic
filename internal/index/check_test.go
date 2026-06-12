package index

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestQuickCheckHealthyDB(t *testing.T) {
	testutil.CleanEnvForTest(t)

	db, err := OpenDB("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, ApplySchema(db))

	path, err := Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	require.NoError(t, QuickCheck(path))
}

func TestQuickCheckCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.sqlite")
	require.NoError(t, os.WriteFile(path, []byte("not sqlite"), 0o644))

	err := QuickCheck(path)
	require.Error(t, err)
	var appErr *app.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, app.CodeCorrupted, appErr.Code)
}
