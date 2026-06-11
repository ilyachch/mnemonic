package project

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestTimestampUsesUTCAndInjectedClock(t *testing.T) {
	clock := testutil.NewClock(time.Date(2026, time.June, 2, 10, 15, 0, 0, time.FixedZone("EEST", 2*60*60)))
	t.Cleanup(SetClock(clock))

	got := Timestamp()
	want := "2026-06-02T08:15:00Z"
	require.Equal(t, want, got)
}

func TestNewUUIDUsesInjectedClock(t *testing.T) {
	clock := testutil.NewClock(time.Date(2026, time.June, 2, 10, 15, 0, 0, time.UTC), "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222")
	t.Cleanup(SetClock(clock))

	require.Equal(t, "11111111-1111-4111-8111-111111111111", NewUUID())
	require.Equal(t, "22222222-2222-4222-8222-222222222222", NewUUID())
}