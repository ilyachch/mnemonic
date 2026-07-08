//go:build linux

package fs

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// GetFileTimes returns (birthTime, modTime, error) for Linux.
func GetFileTimes(path string) (time.Time, time.Time, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	var statx unix.Statx_t
	// AT_FDCWD means relative paths are evaluated relative to current directory
	if err = unix.Statx(unix.AT_FDCWD, path, unix.AT_STATX_SYNC_AS_STAT, unix.STATX_BTIME, &statx); err != nil {
		// Fall back to standard ModTime if statx fails (e.g. unsupported filesystem or older kernel).
		// Intentionally swallow statx errors: btime is best-effort here.
		return fi.ModTime().UTC(), fi.ModTime().UTC(), nil //nolint:nilerr // intentional fallback
	}

	// Verify that the kernel actually populated the Btime attribute
	if (statx.Mask & unix.STATX_BTIME) != 0 {
		birthTime := time.Unix(statx.Btime.Sec, int64(statx.Btime.Nsec)).UTC()
		return birthTime, fi.ModTime().UTC(), nil
	}

	return fi.ModTime().UTC(), fi.ModTime().UTC(), nil
}
