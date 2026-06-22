package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var errEmptyPath = errors.New("path cannot be empty")

// ExpandPath expands the current user's home directory for paths that start
// with "~" and leaves other paths unchanged.
func ExpandPath(path string) (string, error) {
	return expandTilde(path)
}

// ExpandTilde is kept as a narrow alias for callers that want to express
// tilde-specific expansion explicitly.
func ExpandTilde(path string) (string, error) {
	return expandTilde(path)
}

func expandTilde(path string) (string, error) {
	if path == "" {
		return "", errEmptyPath
	}

	if path[0] != '~' {
		return path, nil
	}

	if len(path) == 1 {
		home, err := userHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}

	if path[1] != '/' {
		return "", fmt.Errorf("unsupported home expansion: %q", path)
	}

	home, err := userHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
}

func userHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home, nil
	}

	home = os.Getenv("HOME")
	if home != "" {
		return home, nil
	}

	if err != nil {
		return "", err
	}

	return "", errors.New("home directory is not set")
}
