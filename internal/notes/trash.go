package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const trashTimestampLayout = "20060102T150405Z"

// TrashPathInput configures trash path resolution for a note deletion.
type TrashPathInput struct {
	RootDir      string
	OriginalPath string
	TrashDirName string
	Now          func() time.Time
}

// ResolveTrashPath returns a unique trash path for a note and creates parent
// directories under the configured trash root.
func ResolveTrashPath(input TrashPathInput) (string, error) {
	rootDir := input.RootDir
	if rootDir == "" {
		return "", fmt.Errorf("root directory is required")
	}

	trashDirName := input.TrashDirName
	if trashDirName == "" {
		trashDirName = ".trash"
	}
	trashDirName, err := cleanRelativePath(trashDirName)
	if err != nil {
		return "", fmt.Errorf("trash directory: %w", err)
	}

	now := input.Now
	if now == nil {
		now = time.Now
	}

	originalPath, err := cleanRelativePath(input.OriginalPath)
	if err != nil {
		return "", err
	}

	trashParent := filepath.Join(rootDir, trashDirName, filepath.Dir(originalPath))
	if filepath.Dir(originalPath) == "." {
		trashParent = filepath.Join(rootDir, trashDirName)
	}
	if err := os.MkdirAll(trashParent, 0o755); err != nil {
		return "", fmt.Errorf("create trash parent dirs: %w", err)
	}

	base := filepath.Base(originalPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	stamp := now().UTC().Format(trashTimestampLayout)

	for suffix := 0; ; suffix++ {
		name := fmt.Sprintf("%s.deleted-%s%s", stem, stamp, ext)
		if suffix > 0 {
			name = fmt.Sprintf("%s.deleted-%s-%d%s", stem, stamp, suffix, ext)
		}

		candidate := filepath.Join(trashParent, name)
		_, statErr := os.Stat(candidate)
		switch {
		case statErr == nil:
			continue
		case !os.IsNotExist(statErr):
			return "", fmt.Errorf("check trash path %q: %w", candidate, statErr)
		default:
			return candidate, nil
		}
	}
}

func cleanRelativePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("original path is required")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("original path must be relative")
	}

	cleaned := filepath.Clean(path)
	if cleaned == "." {
		return "", fmt.Errorf("original path is required")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("original path escapes root directory")
	}

	return cleaned, nil
}
