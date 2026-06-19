package project

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/ilyachch/mnemonic/internal/registry"
)

// EnvironmentProjectSelector is the environment variable used to provide an
// implicit project selector when the CLI flag is not set.
const EnvironmentProjectSelector = "MNEMONIC_PROJECT"

// ResolveProjectInput configures project resolution precedence.
type ResolveProjectInput struct {
	ProjectSelector  string
	EnvironmentValue string
	Registry         *sql.DB
}

// ResolvedProject is the selected project together with the resolved
// filesystem pointers from the registry.
type ResolvedProject struct {
	MnemonicFilePath string
	RepoRootAbs      string
	MemoriesAbs      string
	ManifestAbs      string
	Project          ResolvedProjectEntry
}

// ResolvedProjectEntry captures the registry columns needed by callers.
type ResolvedProjectEntry struct {
	ID        string
	Name      string
	Slug      string
	Kind      ProjectKind
	CreatedAt string
	UpdatedAt string
}

// ErrNoProjectSelected indicates that neither the --project flag nor the
// MNEMONIC_PROJECT environment variable was set.
type ErrNoProjectSelected struct{}

// Error implements the error interface.
func (ErrNoProjectSelected) Error() string {
	return "no project selected; specify --project or set " + EnvironmentProjectSelector
}

// ErrProjectNotFound is returned when the requested selector does not match
// any active project in the registry.
type ErrProjectNotFound struct {
	Selector string
}

// Error implements the error interface.
func (e ErrProjectNotFound) Error() string {
	return fmt.Sprintf("project %q not found", e.Selector)
}

// ResolveProject selects a project using the explicit selector or the
// environment variable. There is no filesystem-walking fallback.
func ResolveProject(input ResolveProjectInput) (ResolvedProject, error) {
	if input.Registry == nil {
		return ResolvedProject{}, fmt.Errorf("registry database is required")
	}

	selector := input.ProjectSelector
	if selector == "" {
		selector = input.EnvironmentValue
	}

	if selector == "" {
		return ResolvedProject{}, ErrNoProjectSelected{}
	}

	row, err := queryProjectRegistry(input.Registry, selector)
	if err != nil {
		return ResolvedProject{}, err
	}

	return ResolvedProject{
		Project: ResolvedProjectEntry{
			ID:        row.projectID,
			Name:      row.name,
			Slug:      row.slug,
			Kind:      ProjectKind(row.kind),
			CreatedAt: row.createdAt,
			UpdatedAt: row.updatedAt,
		},
		MemoriesAbs:      row.memoriesAbs,
		ManifestAbs:      row.manifestAbs,
		RepoRootAbs:      row.repoRootAbs,
		MnemonicFilePath: row.mnemonicFileAbs,
	}, nil
}

// ResolveProjectFromEnv is a convenience that resolves a project using only the
// CLI selector (or the MNEMONIC_PROJECT environment variable as fallback) and
// opens the registry database internally.
func ResolveProjectFromEnv(projectSelector string) (ResolvedProject, error) {
	registryDB, err := registry.OpenDB()
	if err != nil {
		return ResolvedProject{}, fmt.Errorf("open registry: %w", err)
	}
	defer func() {
		_ = registryDB.Close()
	}()

	return ResolveProject(ResolveProjectInput{
		ProjectSelector:  projectSelector,
		EnvironmentValue: os.Getenv(EnvironmentProjectSelector),
		Registry:         registryDB,
	})
}

type registryProjectRow struct {
	projectID       string
	name            string
	slug            string
	kind            string
	mnemonicFileAbs string
	repoRootAbs     string
	memoriesAbs     string
	manifestAbs     string
	createdAt       string
	updatedAt       string
}

func queryProjectRegistry(db *sql.DB, selector string) (registryProjectRow, error) {
	row := registryProjectRow{}
	err := db.QueryRow(
		`SELECT p.project_id, p.name, p.slug, p.kind,
		        COALESCE(l.mnemonic_file_abs, ''),
		        COALESCE(l.repo_root_abs, ''),
		        l.memories_abs,
		        COALESCE(l.manifest_abs, ''),
		        COALESCE(CAST(p.created_at AS TEXT), ''),
		        COALESCE(CAST(p.updated_at AS TEXT), '')
		 FROM projects p
		 JOIN project_locations l ON l.project_id = p.project_id
		 WHERE p.removed_at IS NULL AND p.project_id = ?`,
		selector,
	).Scan(&row.projectID, &row.name, &row.slug, &row.kind, &row.mnemonicFileAbs, &row.repoRootAbs, &row.memoriesAbs, &row.manifestAbs, &row.createdAt, &row.updatedAt)
	if err != nil && err != sql.ErrNoRows {
		return registryProjectRow{}, fmt.Errorf("query project %q: %w", selector, err)
	}
	if err == nil {
		return row, nil
	}

	err = db.QueryRow(
		`SELECT p.project_id, p.name, p.slug, p.kind,
		        COALESCE(l.mnemonic_file_abs, ''),
		        COALESCE(l.repo_root_abs, ''),
		        l.memories_abs,
		        COALESCE(l.manifest_abs, ''),
		        COALESCE(CAST(p.created_at AS TEXT), ''),
		        COALESCE(CAST(p.updated_at AS TEXT), '')
		 FROM projects p
		 JOIN project_locations l ON l.project_id = p.project_id
		 WHERE p.removed_at IS NULL AND p.slug = ?`,
		selector,
	).Scan(&row.projectID, &row.name, &row.slug, &row.kind, &row.mnemonicFileAbs, &row.repoRootAbs, &row.memoriesAbs, &row.manifestAbs, &row.createdAt, &row.updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return registryProjectRow{}, ErrProjectNotFound{Selector: selector}
		}
		return registryProjectRow{}, fmt.Errorf("query project %q: %w", selector, err)
	}

	return row, nil
}
