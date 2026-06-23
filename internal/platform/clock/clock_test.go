package clock

import (
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrent_Default(t *testing.T) {
	c := Current()
	require.NotNil(t, c)
	_, ok := c.(systemClock)
	assert.True(t, ok, "default clock should be systemClock")
}

func TestSetClock(t *testing.T) {
	fixedTime := time.Date(2026, time.June, 23, 12, 0, 0, 0, time.UTC)
	fixedUUID := "550e8400-e29b-41d4-a716-446655440000"
	tc := testutil.NewClock(fixedTime, fixedUUID)

	restore := SetClock(tc)
	defer restore()

	// After SetClock, Current returns a clock that uses our test clock's time and UUID
	current := Current()
	assert.Equal(t, fixedTime, current.Now())
	assert.Equal(t, fixedUUID, current.UUID())

	// Now returns the fixed time
	assert.Equal(t, fixedTime, NowUTC())

	// UUID returns the fixed UUID
	assert.Equal(t, fixedUUID, current.UUID())

	// Timestamp is RFC3339
	assert.Equal(t, "2026-06-23T12:00:00Z", Timestamp())

	// After restore, the system clock is back (time moves forward)
	restore()
	afterRestore := Current()
	// System clock should return current time that is not our fixed time
	assert.NotEqual(t, fixedTime, afterRestore.Now())
}

func TestSetClock_Nil(t *testing.T) {
	tc := testutil.NewClock(time.Now(), "test-uuid")
	restore := SetClock(tc)
	defer restore()

	// Nil restores systemClock
	nilRestore := SetClock(nil)
	c := Current()
	_, ok := c.(systemClock)
	assert.True(t, ok, "nil SetClock should restore systemClock")
	nilRestore()
}

func TestNowUTC(t *testing.T) {
	fixedTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	tc := testutil.NewClock(fixedTime)
	restore := SetClock(tc)
	defer restore()

	// NowUTC should return the fixed time in UTC (already UTC)
	got := NowUTC()
	assert.Equal(t, fixedTime, got)
	// The location should be UTC
	assert.Equal(t, time.UTC, got.Location())
}

func TestTimestamp(t *testing.T) {
	fixedTime := time.Date(2026, time.June, 23, 15, 30, 45, 0, time.UTC)
	tc := testutil.NewClock(fixedTime)
	restore := SetClock(tc)
	defer restore()

	got := Timestamp()
	assert.Equal(t, "2026-06-23T15:30:45Z", got)
}

func TestTimestamp_Default(t *testing.T) {
	// Just verify that Timestamp() doesn't panic with the default clock
	ts := Timestamp()
	require.NotEmpty(t, ts)

	// It should be parseable as RFC3339
	_, err := time.Parse(time.RFC3339, ts)
	require.NoError(t, err, "Timestamp should be valid RFC3339")
}

func TestUUID_DefaultClock(t *testing.T) {
	// Use the default (systemClock) to cover systemClock.UUID() and randomUUID()
	c := Current()
	uuid := c.UUID()
	require.NotEmpty(t, uuid)

	// UUID v4 format check
	parts := strings.Split(uuid, "-")
	require.Len(t, parts, 5)
	assert.Len(t, parts[0], 8)
	assert.Len(t, parts[1], 4)
	assert.Len(t, parts[2], 4)
	assert.Len(t, parts[3], 4)
	assert.Len(t, parts[4], 12)

	// Version nibble should be 4
	assert.Equal(t, '4', rune(parts[2][0]), "UUID version should be 4")

	// Variant nibble should be 8, 9, a, or b
	variant := parts[3][0]
	assert.True(t, variant == '8' || variant == '9' || variant == 'a' || variant == 'b',
		"UUID variant should be 8/9/a/b, got %c", variant)
}

// NewUUID_TestClock verifies Clock.UUID() through Current()
