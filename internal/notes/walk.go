package notes

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// Walk returns all Markdown note paths under root as slash-separated relative paths.
func Walk(root string) ([]string, error) {
	var notes []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("rel %q: %w", path, err)
		}
		rel = filepath.ToSlash(rel)

		if entry.IsDir() {
			if entry.Name() == ".trash" {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.Name() == "mnemonic.toml" {
			return nil
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			return nil
		}

		if strings.Contains(rel, "/.trash/") || strings.HasPrefix(rel, ".trash/") || rel == ".trash" {
			return nil
		}

		notes = append(notes, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return notes, nil
}
