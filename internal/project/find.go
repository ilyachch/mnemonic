package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/app"
)

// FindNearestMnemonicFile walks upward from startDir and returns the first .mnemonic file it finds.
func FindNearestMnemonicFile(startDir string) (string, error) {
	if startDir == "" {
		return "", fmt.Errorf("start directory is required")
	}

	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve start directory: %w", err)
	}

	dir := absStart
	for {
		candidate := filepath.Join(dir, ".mnemonic")
		info, err := os.Stat(candidate)
		if err == nil {
			if info.IsDir() {
				return "", fmt.Errorf("%s is a directory", candidate)
			}
			return candidate, nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %s: %w", candidate, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", app.NewNotFoundError(fmt.Sprintf(".mnemonic not found from %s", absStart), nil)
}
