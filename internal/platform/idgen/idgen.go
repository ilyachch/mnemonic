package idgen

import "github.com/ilyachch/mnemonic/internal/platform/clock"

// NewUUID returns a UUID from the active clock.
func NewUUID() string {
	return clock.Current().UUID()
}
