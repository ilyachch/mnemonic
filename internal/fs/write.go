package fs

import (
	"os"
	"path/filepath"
)

// WriteFile creates parent directories as needed and writes data to path.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, perm)
}
