package idgen

import (
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/platform/clock"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUUID_WithTestClock(t *testing.T) {
	fixedTime := time.Date(2026, time.June, 23, 12, 0, 0, 0, time.UTC)
	testUUID := "550e8400-e29b-41d4-a716-446655440000"
	tc := testutil.NewClock(fixedTime, testUUID)

	restore := clock.SetClock(tc)
	defer restore()

	got := NewUUID()
	assert.Equal(t, testUUID, got)
}

func TestNewUUID_MultipleCalls(t *testing.T) {
	fixedTime := time.Date(2026, time.June, 23, 12, 0, 0, 0, time.UTC)
	uuid1 := "550e8400-e29b-41d4-a716-446655440001"
	uuid2 := "550e8400-e29b-41d4-a716-446655440002"
	tc := testutil.NewClock(fixedTime, uuid1, uuid2)

	restore := clock.SetClock(tc)
	defer restore()

	assert.Equal(t, uuid1, NewUUID())
	assert.Equal(t, uuid2, NewUUID())
}

func TestNewUUID_DefaultClock(t *testing.T) {
	// Returns a UUID v4 format string with the default clock
	got := NewUUID()
	require.NotEmpty(t, got)

	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	// where y is 8, 9, a, or b
	parts := strings.Split(got, "-")
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
