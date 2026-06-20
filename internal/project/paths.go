package project

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/paths"
)

// MemoriesRootInput configures absolute memories root resolution for a project.
type MemoriesRootInput struct {
	Kind         string
	Slug         string
	MemoriesPath string
	MemoriesHome string
	RepoRoot     string
}

// ResolveMemoriesRoot returns the absolute memories directory for a project.
//
// Central projects use memories_home/slug, local projects use
// repo_root/memories_path.
func ResolveMemoriesRoot(input MemoriesRootInput) (string, error) {
	switch input.Kind {
	case string(ProjectKindCentral):
		return resolveRelativePathWithinBase(input.MemoriesHome, input.Slug, "slug")
	case string(ProjectKindLocal):
		return resolveRelativePathWithinBase(input.RepoRoot, input.MemoriesPath, "memories_path")
	default:
		return "", fmt.Errorf("unsupported project kind %q", input.Kind)
	}
}

func resolveRelativePathWithinBase(baseDir string, relativePath string, fieldName string) (string, error) {
	if baseDir == "" {
		return "", fmt.Errorf("%s base directory is required", fieldName)
	}
	if relativePath == "" {
		return "", fmt.Errorf("%s is required", fieldName)
	}
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("%s must be relative", fieldName)
	}

	baseAbs, err := paths.NormalizeAbsolutePath(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve base directory: %w", err)
	}

	resolved := filepath.Clean(filepath.Join(baseAbs, relativePath))
	rel, err := filepath.Rel(baseAbs, resolved)
	if err != nil {
		return "", fmt.Errorf("validate %s: %w", fieldName, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s escapes base directory", fieldName)
	}

	return resolved, nil
}
