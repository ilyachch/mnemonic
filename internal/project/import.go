package project

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
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
	MnemonicFileAbs string `json:"mnemonic_file_abs"`
	RepoRootAbs     string `json:"repo_root_abs"`
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
			return "", app.NewNotFoundError(fmt.Sprintf("import path %q not found", path), nil)
		}
		return "", fmt.Errorf("stat import path %s: %w", absPath, err)
	}

	return absPath, nil
}

// ImportProject resolves an import path, finds the nearest .mnemonic file, and registers
// the projects declared in that file without copying Markdown or building an index.
func ImportProject(input ImportInput) (ImportResult, error) {
	resolvedPath, err := ResolveImportPath(input)
	if err != nil {
		return ImportResult{}, err
	}

	mnemonicPath, err := FindNearestMnemonicFile(resolvedPath)
	if err != nil {
		return ImportResult{}, err
	}

	file, err := loadMnemonicFile(mnemonicPath)
	if err != nil {
		return ImportResult{}, err
	}

	repoRoot := filepath.Dir(mnemonicPath)
	candidates := make([]ImportCandidate, 0, len(file.Projects))
	for i := range file.Projects {
		projectEntry := file.Projects[i]
		if projectEntry.Kind != ProjectKindLocal {
			continue
		}
		candidates = append(candidates, buildImportCandidate(mnemonicPath, repoRoot, projectEntry))
	}

	result := ImportResult{
		Path:        resolvedPath,
		Imported:    len(candidates),
		CopiedFiles: 0,
		Indexed:     0,
		Candidates:  candidates,
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
	for i := range file.Projects {
		projectEntry := file.Projects[i]
		if projectEntry.Kind != ProjectKindLocal {
			continue
		}
		if err := registerImportedProject(db, mnemonicPath, repoRoot, projectEntry, now); err != nil {
			return ImportResult{}, err
		}
	}

	return result, nil
}

func buildImportCandidate(mnemonicPath, repoRoot string, projectEntry MnemonicProject) ImportCandidate {
	return ImportCandidate{
		ProjectID:       projectEntry.ID,
		Name:            projectEntry.Name,
		Slug:            projectEntry.Slug,
		Kind:            string(projectEntry.Kind),
		MemoriesPath:    filepath.Join(repoRoot, projectEntry.MemoriesPath),
		MnemonicFileAbs: mnemonicPath,
		RepoRootAbs:     repoRoot,
	}
}

func registerImportedProject(db *sql.DB, mnemonicPath, repoRoot string, projectEntry MnemonicProject, seenAt time.Time) error {
	memoriesAbs := filepath.Join(repoRoot, projectEntry.MemoriesPath)

	return registry.RegisterProject(db, registry.RegisterProjectInput{
		ProjectID: projectEntry.ID,
		Name:      projectEntry.Name,
		Slug:      projectEntry.Slug,
		Kind:      registry.ProjectKind(projectEntry.Kind),
		CreatedAt: projectEntry.CreatedAt,
		UpdatedAt: projectEntry.UpdatedAt,
		SeenAt:    seenAt,
		Location: registry.ProjectLocationInput{
			MnemonicFileAbs: mnemonicPath,
			RepoRootAbs:     repoRoot,
			MemoriesAbs:     memoriesAbs,
			SourceKind:      registry.ProjectSourceKindImport,
		},
	})
}
