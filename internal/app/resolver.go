package app

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/registry"
)

// FileResolver implements ProjectResolver by scanning the file-based registry.
type FileResolver struct {
	MemoriesHome string
}

// Resolve finds a project by slug using the file-based registry.
func (r *FileResolver) Resolve(input ProjectResolveInput) (ProjectResolution, error) {
	selector := input.ProjectSelector
	if selector == "" {
		selector = input.EnvironmentValue
	}

	if selector == "" {
		return ProjectResolution{}, NewNoProjectSelectedError()
	}

	entry, err := registry.Resolve(r.MemoriesHome, selector)
	if err != nil {
		return ProjectResolution{}, wrapRegistryError(err)
	}

	return ProjectResolution{
		RepoRootAbs: entry.RepoRootAbs,
		MemoriesAbs: entry.MemoriesAbs,
		ManifestAbs: entry.ManifestPath,
		Project: ProjectRecord{
			ID:           entry.ProjectID,
			Name:         entry.Name,
			Slug:         entry.Slug,
			Kind:         entry.Type,
			MemoriesPath: memoriesPathForEntry(entry),
		},
	}, nil
}

func wrapRegistryError(err error) error {
	if err == nil {
		return nil
	}
	var notFoundErr registry.ErrNotFound
	if errors.As(err, &notFoundErr) {
		return apperr.NotFound(notFoundErr.Error(), nil)
	}
	return err
}

func memoriesPathForEntry(entry registry.Entry) string {
	if entry.RepoRootAbs != "" && entry.MemoriesAbs != "" {
		if rel, err := filepath.Rel(entry.RepoRootAbs, entry.MemoriesAbs); err == nil && rel != "" && rel != "." {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.Base(entry.MemoriesAbs)
}

// NewNoProjectSelectedError returns a CLI usage error when no project is selected.
func NewNoProjectSelectedError() error {
	return apperr.CLIUsage("no project selected; specify --project or set MNEMONIC_PROJECT", nil)
}

// IsNoProjectSelected checks whether this error indicates no project was selected.
func IsNoProjectSelected(err error) bool {
	if err == nil {
		return false
	}
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return appErr.Code == apperr.CodeCLIUsage && strings.Contains(appErr.Message, "no project selected")
	}
	return false
}
