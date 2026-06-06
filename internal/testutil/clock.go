package testutil

import (
	"sync"
	"time"
)

// Clock is a deterministic project clock used by tests.
type Clock struct {
	mu    sync.Mutex
	now   time.Time
	uuids []string
	index int
}

// NewClock creates a deterministic clock with a fixed UTC time and UUID sequence.
func NewClock(now time.Time, uuids ...string) *Clock {
	return &Clock{
		now:   now.UTC(),
		uuids: append([]string(nil), uuids...),
	}
}

// Now returns the fixed time in UTC.
func (c *Clock) Now() time.Time {
	return c.now.UTC()
}

// UUID returns the next deterministic UUID from the sequence.
func (c *Clock) UUID() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.uuids) == 0 {
		return ""
	}
	if c.index >= len(c.uuids) {
		return c.uuids[len(c.uuids)-1]
	}

	uuid := c.uuids[c.index]
	c.index++
	return uuid
}
