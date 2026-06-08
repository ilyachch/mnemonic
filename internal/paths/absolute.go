package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizeAbsolutePath expands path-like input and returns an absolute path.
// On macOS it prefers the logical /var-style path when the physical path lives
// under /private and the shorter alias exists.
func NormalizeAbsolutePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	expanded, err := ExpandPath(path)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		abs, err := filepath.Abs(expanded)
		if err != nil {
			return "", fmt.Errorf("resolve absolute path %q: %w", path, err)
		}
		expanded = abs
	}

	cleaned := filepath.Clean(expanded)
	return preferLogicalDarwinPath(cleaned), nil
}

func preferLogicalDarwinPath(path string) string {
	if runtime.GOOS != "darwin" {
		return path
	}
	if !strings.HasPrefix(path, "/private/") {
		return path
	}

	logicalPath := strings.TrimPrefix(path, "/private")
	if _, err := os.Stat(logicalPath); err == nil {
		return logicalPath
	}

	return path
}
