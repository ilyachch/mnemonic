package fs

import (
	"os"
	"path/filepath"
)

// AtomicWriteFile writes data to path by staging it in a temp file in the same
// directory, syncing it, and then atomically replacing the target.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if err = tmp.Chmod(perm); err != nil {
		return err
	}
	if err = writeAll(tmp, data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpPath, path); err != nil {
		return err
	}
	if err = syncDir(dir); err != nil {
		return err
	}

	return nil
}

// writeAll writes the entirety of data to f, retrying on short writes.
func writeAll(f *os.File, data []byte) error {
	for len(data) > 0 {
		n, err := f.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}

	return nil
}

// syncDir fsyncs the directory at path so that directory entries (renames)
// are persisted to disk.
func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = dir.Close() }()

	return dir.Sync()
}
