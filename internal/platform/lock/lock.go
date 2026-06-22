package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

var ErrBusy = errors.New("lock busy")

const defaultPollInterval = 10 * time.Millisecond

// AcquireInput configures state-scoped file lock acquisition.
type AcquireInput struct {
	StateDir     string
	Name         string
	Timeout      time.Duration
	PollInterval time.Duration
}

// Guard holds an acquired lock until Release is called.
type Guard struct {
	path string
	lock *flock.Flock
}

// Path returns the absolute path for a state-scoped lock file.
func Path(stateDir string, name string) (string, error) {
	stateDir = strings.TrimSpace(stateDir)
	if stateDir == "" {
		return "", fmt.Errorf("state directory is required")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("lock name is required")
	}
	if strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("lock name must not contain path separators")
	}

	return filepath.Join(
		stateDir,
		"locks",
		name+".lock",
	), nil
}

// Acquire creates a state-scoped lock file and waits until timeout on conflict.
func Acquire(input AcquireInput) (*Guard, error) {
	lockPath, err := Path(input.StateDir, input.Name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	fileLock := flock.New(lockPath, flock.SetPermissions(0o644))
	pollInterval := input.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}

	deadline := time.Now().Add(input.Timeout)
	for {
		ok, err := fileLock.TryLock()
		if err != nil {
			return nil, fmt.Errorf("acquire lock: %w", err)
		}
		if ok {
			return &Guard{path: lockPath, lock: fileLock}, nil
		}
		if input.Timeout <= 0 || !time.Now().Before(deadline) {
			return nil, fmt.Errorf("%w: %s", ErrBusy, lockPath)
		}
		time.Sleep(pollInterval)
	}
}

// Release frees the lock and leaves the lock file on disk.
func (g *Guard) Release() error {
	if g == nil {
		return nil
	}
	if g.lock != nil {
		if err := g.lock.Close(); err != nil {
			return fmt.Errorf("close lock file: %w", err)
		}
		g.lock = nil
	}
	return nil
}
