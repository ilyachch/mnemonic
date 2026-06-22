package notes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/platform/lock"
)

const writeLockName = "write"

func acquireWriteLock(root string) (*lock.Guard, error) {
	projectKey, err := projectLockKey(root)
	if err != nil {
		return nil, err
	}

	guard, err := lock.Acquire(lock.AcquireInput{
		ProjectID:    projectKey,
		Name:         writeLockName,
		Timeout:      150 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
	})
	if err == nil {
		return guard, nil
	}
	if errors.Is(err, lock.ErrBusy) {
		return nil, apperr.Unsafe("project write lock is busy", err)
	}
	return nil, err
}

func projectLockKey(root string) (string, error) {
	cleaned := strings.TrimSpace(root)
	if cleaned == "" {
		return "", apperr.CLIUsage("root directory is required", nil)
	}
	absRoot, err := filepath.Abs(cleaned)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(filepath.Clean(absRoot)))
	return hex.EncodeToString(sum[:]), nil
}
