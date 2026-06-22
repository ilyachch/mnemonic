package clock

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"
)

// Clock provides time and UUID generation for deterministic tests.
type Clock interface {
	Now() time.Time
	UUID() string
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

func (systemClock) UUID() string { return randomUUID() }

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

func randomUUID() string {
	var buf [16]byte
	if _, err := io.ReadFull(rand.Reader, buf[:]); err != nil {
		panic(fmt.Errorf("generate random uuid: %w", err))
	}

	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	encoded := hex.EncodeToString(buf[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		encoded[0:8],
		encoded[8:12],
		encoded[12:16],
		encoded[16:20],
		encoded[20:32],
	)
}
