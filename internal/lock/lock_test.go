package lock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestPathUsesStateDir(t *testing.T) {
	testutil.CleanEnvForTest(t)

	got, err := Path("550e8400-e29b-41d4-a716-446655440000", "write")
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}

	want := filepath.Join(os.Getenv("XDG_STATE_HOME"), "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "locks", "write.lock")
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}

func TestAcquireBlocksUntilBusyTimeout(t *testing.T) {
	testutil.CleanEnvForTest(t)

	first, err := Acquire(AcquireInput{ProjectID: "project-1", Name: "write"})
	if err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}
	defer func() {
		if err := first.Release(); err != nil {
			t.Fatalf("Release() error = %v", err)
		}
	}()

	start := time.Now()
	_, err = Acquire(AcquireInput{
		ProjectID:    "project-1",
		Name:         "write",
		Timeout:      50 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
	})
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("second Acquire() error = %v, want ErrBusy", err)
	}
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Fatalf("second Acquire() elapsed = %v, want it to wait for timeout", elapsed)
	}
}

func TestReleaseFreesLock(t *testing.T) {
	testutil.CleanEnvForTest(t)

	first, err := Acquire(AcquireInput{ProjectID: "project-1", Name: "write"})
	if err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}

	lockPath, err := Path("project-1", "write")
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file stat error = %v", err)
	}

	if err := first.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock file stat after release = %v, want not exist", err)
	}

	second, err := Acquire(AcquireInput{ProjectID: "project-1", Name: "write"})
	if err != nil {
		t.Fatalf("second Acquire() error = %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatalf("second Release() error = %v", err)
	}
}

func TestAcquireIgnoresPreexistingLockFile(t *testing.T) {
	testutil.CleanEnvForTest(t)

	lockPath, err := Path("project-1", "write")
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(lockPath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	guard, err := Acquire(AcquireInput{ProjectID: "project-1", Name: "write"})
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if err := guard.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
}
