package project

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/registry"
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
// path, and registers the project without copying Markdown or building an index.
func ImportProject(input ImportInput) (ImportResult, error) {
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
		Kind:         string(manifest.Kind),
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

	db, err := registry.OpenDB()
	if err != nil {
		return ImportResult{}, err
	}
	defer func() {
		_ = db.Close()
	}()

	if err := registry.ApplySchema(db); err != nil {
		return ImportResult{}, err
	}

	now := NowUTC()
	if err := registerImportedProject(db, manifest, manifestPath, resolvedPath, now); err != nil {
		return ImportResult{}, err
	}

	return result, nil
}

func registerImportedProject(db *sql.DB, manifest *MnemonicManifest, manifestPath, repoRoot string, seenAt time.Time) error {
	return registry.RegisterProject(db, registry.RegisterProjectInput{
		ProjectID: manifest.ProjectID,
		Name:      manifest.Name,
		Slug:      manifest.Slug,
		Kind:      registry.ProjectKind(manifest.Kind),
		CreatedAt: manifest.CreatedAt,
		UpdatedAt: manifest.UpdatedAt,
		SeenAt:    seenAt,
		Location: registry.ProjectLocationInput{
			RepoRootAbs: repoRoot,
			MemoriesAbs: repoRoot,
			ManifestAbs: manifestPath,
			SourceKind:  registry.ProjectSourceKindImport,
		},
	})
}
