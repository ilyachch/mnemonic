package clock

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Clock provides time and UUID generation for deterministic tests.
type Clock interface {
	Now() time.Time
	UUID() string
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

func (systemClock) UUID() string { return uuid.NewString() }

var (
	mu     sync.RWMutex
	active Clock = systemClock{}
)

// SetClock replaces the active clock and returns a restore function.
func SetClock(c Clock) func() {
	if c == nil {
		c = systemClock{}
	}

	mu.Lock()
	prev := active
	active = c
	mu.Unlock()

	return func() {
		mu.Lock()
		active = prev
		mu.Unlock()
	}
}

// Current returns the active clock.
func Current() Clock {
	mu.RLock()
	defer mu.RUnlock()
	return active
}

// NowUTC returns the active clock time normalized to UTC.
func NowUTC() time.Time {
	return Current().Now().UTC()
}

// Timestamp returns the active clock time formatted as RFC3339 UTC.
func Timestamp() string {
	return NowUTC().Format(time.RFC3339)
}
