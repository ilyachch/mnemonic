package lock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPathUsesStateDir(t *testing.T) {
	stateDir := t.TempDir()

	got, err := Path(stateDir, "write")
	require.NoError(t, err)

	want := filepath.Join(stateDir, "locks", "write.lock")
	require.Equal(t, want, got)
}

func TestAcquireBlocksUntilBusyTimeout(t *testing.T) {
	stateDir := t.TempDir()

	first, err := Acquire(AcquireInput{StateDir: stateDir, Name: "write"})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, first.Release())
	}()

	start := time.Now()
	_, err = Acquire(AcquireInput{
		StateDir:     stateDir,
		Name:         "write",
		Timeout:      50 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
	})
	require.True(t, errors.Is(err, ErrBusy))
	require.GreaterOrEqual(t, time.Since(start), 40*time.Millisecond)
}

func TestReleaseFreesLock(t *testing.T) {
	stateDir := t.TempDir()

	first, err := Acquire(AcquireInput{StateDir: stateDir, Name: "write"})
	require.NoError(t, err)

	lockPath, err := Path(stateDir, "write")
	require.NoError(t, err)
	_, err = os.Stat(lockPath)
	require.NoError(t, err)

	require.NoError(t, first.Release())
	info, err := os.Stat(lockPath)
	require.NoError(t, err)
	require.False(t, info.IsDir())

	second, err := Acquire(AcquireInput{StateDir: stateDir, Name: "write"})
	require.NoError(t, err)
	require.NoError(t, second.Release())
}

func TestAcquireIgnoresPreexistingLockFile(t *testing.T) {
	stateDir := t.TempDir()

	lockPath, err := Path(stateDir, "write")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(lockPath), 0o755))
	require.NoError(t, os.WriteFile(lockPath, []byte("stale"), 0o644))

	guard, err := Acquire(AcquireInput{StateDir: stateDir, Name: "write"})
	require.NoError(t, err)
	require.NoError(t, guard.Release())
}
