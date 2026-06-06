package fs

import (
	"os"
	"path/filepath"
)

var (
	createTempFile = os.CreateTemp
	renameFile     = os.Rename
	removeFile     = os.Remove
	syncFile       = func(f *os.File) error { return f.Sync() }
	syncDir        = func(path string) error {
		dir, err := os.Open(path)
		if err != nil {
			return err
		}
		defer dir.Close()

		return dir.Sync()
	}
	writeAll = func(f *os.File, data []byte) error {
		for len(data) > 0 {
			n, err := f.Write(data)
			if err != nil {
				return err
			}
			data = data[n:]
		}

		return nil
	}
)

// AtomicWriteFile writes data to path by staging it in a temp file in the same
// directory, syncing it, and then atomically replacing the target.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := createTempFile(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = removeFile(tmpPath)
		}
	}()

	if err = tmp.Chmod(perm); err != nil {
		return err
	}
	if err = writeAll(tmp, data); err != nil {
		return err
	}
	if err = syncFile(tmp); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = renameFile(tmpPath, path); err != nil {
		return err
	}
	if err = syncDir(dir); err != nil {
		return err
	}

	return nil
}
