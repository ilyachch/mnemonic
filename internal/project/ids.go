package project

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"
)

// Clock abstracts time and UUID generation for production and tests.
type Clock interface {
	Now() time.Time
	UUID() string
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

func (systemClock) UUID() string {
	return randomUUID()
}

var (
	clockMu     sync.RWMutex
	activeClock Clock = systemClock{}
)

// SetClock replaces the active generator clock and returns a restore function.
func SetClock(clock Clock) func() {
	if clock == nil {
		clock = systemClock{}
	}

	clockMu.Lock()
	prev := activeClock
	activeClock = clock
	clockMu.Unlock()

	return func() {
		clockMu.Lock()
		activeClock = prev
		clockMu.Unlock()
	}
}

// NowUTC returns the active clock time normalized to UTC.
func NowUTC() time.Time {
	clockMu.RLock()
	clock := activeClock
	clockMu.RUnlock()
	return clock.Now().UTC()
}

// Timestamp returns the active clock time formatted as RFC3339 UTC.
func Timestamp() string {
	return NowUTC().Format(time.RFC3339)
}

// NewUUID returns a UUID from the active clock.
func NewUUID() string {
	clockMu.RLock()
	clock := activeClock
	clockMu.RUnlock()
	return clock.UUID()
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
