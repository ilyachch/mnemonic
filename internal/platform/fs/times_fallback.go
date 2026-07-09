//go:build !darwin && !linux

package fs

import (
	"os"
	"time"
)

// GetFileTimes is a portable fallback that returns modtime for both attributes.
func GetFileTimes(path string) (time.Time, time.Time, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return fi.ModTime().UTC(), fi.ModTime().UTC(), nil
}
