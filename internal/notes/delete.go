package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	mnemonicfs "github.com/ilyachch/mnemonic/internal/fs"
)

// DeleteInput configures note deletion.
type DeleteInput struct {
	RootDir  string
	Selector string
	DryRun   bool
	Hard     bool
	TrashDir string
	Yes      bool
	Now      func() time.Time
}

// DeleteResult describes the outcome of a note deletion.
type DeleteResult struct {
	Mode      string `json:"mode"`
	Path      string `json:"path,omitempty"`
	TrashPath string `json:"trash_path,omitempty"`
}

// Delete removes or trashes a note.
func Delete(input DeleteInput) (DeleteResult, error) {
	if input.RootDir == "" {
		return DeleteResult{}, fmt.Errorf("root directory is required")
	}
	if input.Selector == "" {
		return DeleteResult{}, app.NewNotFoundError("note selector is required", nil)
	}

	guard, err := acquireWriteLock(input.RootDir)
	if err != nil {
		return DeleteResult{}, err
	}
	defer func() { _ = guard.Release() }()

	resolved, err := Resolve(input.RootDir, input.Selector)
	if err != nil {
		return DeleteResult{}, err
	}

	absPath := filepath.Join(input.RootDir, filepath.FromSlash(resolved.Path))
	if input.Hard {
		if !input.Yes {
			return DeleteResult{}, app.NewUnsafeError("hard delete requires --yes", nil)
		}
		if input.DryRun {
			return DeleteResult{Mode: "hard", Path: resolved.Path}, nil
		}
		if err := os.Remove(absPath); err != nil {
			return DeleteResult{}, fmt.Errorf("remove note: %w", err)
		}
		return DeleteResult{Mode: "hard", Path: resolved.Path}, nil
	}

	trashPath, err := ResolveTrashPath(TrashPathInput{
		RootDir:      input.RootDir,
		OriginalPath: resolved.Path,
		TrashDirName: input.TrashDir,
		Now:          input.Now,
	})
	if err != nil {
		return DeleteResult{}, err
	}

	if input.DryRun {
		return DeleteResult{Mode: "trash", Path: resolved.Path, TrashPath: trashPath}, nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return DeleteResult{}, fmt.Errorf("read note: %w", err)
	}
	if err := mnemonicfs.WriteFile(trashPath, data, 0o644); err != nil {
		return DeleteResult{}, fmt.Errorf("move note to trash: %w", err)
	}
	if err := os.Remove(absPath); err != nil {
		return DeleteResult{}, fmt.Errorf("remove source note: %w", err)
	}

	return DeleteResult{Mode: "trash", Path: resolved.Path, TrashPath: trashPath}, nil
}
