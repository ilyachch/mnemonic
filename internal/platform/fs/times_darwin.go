//go:build darwin

package fs

import (
	"os"
	"syscall"
	"time"
)

// GetFileTimes returns (birthTime, modTime, error) for Darwin.
func GetFileTimes(path string) (time.Time, time.Time, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		// Fall back to modtime if sys conversion fails
		return fi.ModTime().UTC(), fi.ModTime().UTC(), nil
	}

	// Birthtimespec represents the creation time on macOS.
	birthTime := time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec).UTC()
	return birthTime, fi.ModTime().UTC(), nil
}
