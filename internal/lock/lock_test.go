package lock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestPathUsesStateDir(t *testing.T) {
	testutil.CleanEnvForTest(t)

	got, err := Path("550e8400-e29b-41d4-a716-446655440000", "write")
	require.NoError(t, err)

	want := filepath.Join(os.Getenv("XDG_STATE_HOME"), "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "locks", "write.lock")
	require.Equal(t, want, got)
}

func TestAcquireBlocksUntilBusyTimeout(t *testing.T) {
	testutil.CleanEnvForTest(t)

	first, err := Acquire(AcquireInput{ProjectID: "project-1", Name: "write"})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, first.Release())
	}()

	start := time.Now()
	_, err = Acquire(AcquireInput{
		ProjectID:    "project-1",
		Name:         "write",
		Timeout:      50 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
	})
	require.True(t, errors.Is(err, ErrBusy))
	require.GreaterOrEqual(t, time.Since(start), 40*time.Millisecond)
}

func TestReleaseFreesLock(t *testing.T) {
	testutil.CleanEnvForTest(t)

	first, err := Acquire(AcquireInput{ProjectID: "project-1", Name: "write"})
	require.NoError(t, err)

	lockPath, err := Path("project-1", "write")
	require.NoError(t, err)
	_, err = os.Stat(lockPath)
	require.NoError(t, err)

	require.NoError(t, first.Release())
	info, err := os.Stat(lockPath)
	require.NoError(t, err)
	require.False(t, info.IsDir())

	second, err := Acquire(AcquireInput{ProjectID: "project-1", Name: "write"})
	require.NoError(t, err)
	require.NoError(t, second.Release())
}

func TestAcquireIgnoresPreexistingLockFile(t *testing.T) {
	testutil.CleanEnvForTest(t)

	lockPath, err := Path("project-1", "write")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(lockPath), 0o755))
	require.NoError(t, os.WriteFile(lockPath, []byte("stale"), 0o644))

	guard, err := Acquire(AcquireInput{ProjectID: "project-1", Name: "write"})
	require.NoError(t, err)
	require.NoError(t, guard.Release())
}
