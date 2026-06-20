package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/paths"
)

// ImportInput configures import path parsing and validation.
type ImportInput struct {
	Path   string
	DryRun bool
}

// ImportResult reports the outcome of an import operation.
type ImportResult struct {
	Path        string
	Imported    int
	CopiedFiles int
	Indexed     int
	Candidates  []ImportCandidate
	DryRun      bool
}

// ImportCandidate describes a project that would be imported.
type ImportCandidate struct {
	ProjectID       string `json:"project_id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Kind            string `json:"kind"`
	MemoriesPath    string `json:"memories_path"`
	MnemonicFileAbs string `json:"mnemonic_file_abs,omitempty"`
	RepoRootAbs     string `json:"repo_root_abs"`
	ManifestAbs     string `json:"manifest_abs"`
}

// ResolveImportPath normalizes an import path to an absolute path and validates that it exists.
func ResolveImportPath(input ImportInput) (string, error) {
	path := input.Path
	if path == "" {
		path = "."
	}

	absPath, err := paths.NormalizeAbsolutePath(path)
	if err != nil {
		return "", fmt.Errorf("resolve import path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("import path %q not found", path)
		}
		return "", fmt.Errorf("stat import path %s: %w", absPath, err)
	}

	return absPath, nil
}

// ImportProject resolves an import path, parses the mnemonic.toml manifest at the
// path, and creates a pointer file for local projects.
func ImportProject(input ImportInput, memoriesHome string) (ImportResult, error) {
	resolvedPath, err := ResolveImportPath(input)
	if err != nil {
		return ImportResult{}, err
	}

	manifestPath := filepath.Join(resolvedPath, "mnemonic.toml")
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return ImportResult{}, fmt.Errorf("mnemonic.toml not found at %s", resolvedPath)
		}
		return ImportResult{}, fmt.Errorf("stat mnemonic.toml: %w", err)
	}

	manifest, err := loadMnemonicManifest(manifestPath)
	if err != nil {
		return ImportResult{}, err
	}

	candidate := ImportCandidate{
		ProjectID:    manifest.ProjectID,
		Name:         manifest.Name,
		Slug:         manifest.Slug,
		Kind:         kindFromManifest(manifest),
		MemoriesPath: resolvedPath,
		RepoRootAbs:  resolvedPath,
		ManifestAbs:  manifestPath,
	}

	result := ImportResult{
		Path:        resolvedPath,
		Imported:    1,
		CopiedFiles: 0,
		Indexed:     0,
		Candidates:  []ImportCandidate{candidate},
		DryRun:      input.DryRun,
	}

	if input.DryRun {
		return result, nil
	}

	// Create the pointer file for local projects.
	pointerPath := filepath.Join(memoriesHome, manifest.Slug+".toml")
	if _, err := os.Stat(pointerPath); err == nil {
		return ImportResult{}, fmt.Errorf("project slug %q already exists", manifest.Slug)
	} else if !os.IsNotExist(err) {
		return ImportResult{}, fmt.Errorf("stat pointer file: %w", err)
	}

	pointer := &PointerFile{ManifestPath: manifestPath}
	if err := WritePointerFile(pointerPath, pointer); err != nil {
		return ImportResult{}, err
	}

	return result, nil
}

func kindFromManifest(manifest *MnemonicManifest) string {
	if manifest.IsLocal() {
		return "local"
	}
	return "central"
}

func loadMnemonicManifest(path string) (*MnemonicManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mnemonic.toml: %w", err)
	}

	manifest, err := ParseMnemonicManifest(data)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}
