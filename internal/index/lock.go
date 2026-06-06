package index

import (
	"errors"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/lock"
)

const reindexLockName = "reindex"

func acquireReindexLock(projectID string) (*lock.Guard, error) {
	guard, err := lock.Acquire(lock.AcquireInput{
		ProjectID:    projectID,
		Name:         reindexLockName,
		Timeout:      150 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
	})
	if err == nil {
		return guard, nil
	}
	if errors.Is(err, lock.ErrBusy) {
		return nil, app.NewUnsafeError("project reindex lock is busy", err)
	}
	return nil, err
}
