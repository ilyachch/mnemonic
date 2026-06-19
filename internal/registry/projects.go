package registry

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrSlugAlreadyExists indicates that a project with the requested slug is
// already registered and active in the registry.
var ErrSlugAlreadyExists = errors.New("project slug already exists")

// ErrProjectSlugConflict wraps ErrSlugAlreadyExists with the conflicting slug.
type ErrProjectSlugConflict struct {
	Slug string
}

func (e ErrProjectSlugConflict) Error() string {
	return fmt.Sprintf("project slug %q already exists", e.Slug)
}

func (e ErrProjectSlugConflict) Unwrap() error {
	return ErrSlugAlreadyExists
}

// ProjectKind describes the project type stored in the registry.
type ProjectKind string

const (
	ProjectKindRegular  ProjectKind = "regular"
	ProjectKindLocal    ProjectKind = "local"
	ProjectKindDetached ProjectKind = "detached"
)

// ProjectSourceKind describes how the registry learned about a project.
type ProjectSourceKind string

const (
	ProjectSourceKindInit     ProjectSourceKind = "init"
	ProjectSourceKindImport   ProjectSourceKind = "import"
	ProjectSourceKindDiscover ProjectSourceKind = "discover"
)

// RegisterProjectInput captures the minimum data needed to register or update a project.
type RegisterProjectInput struct {
	ProjectID string
	Name      string
	Slug      string
	Kind      ProjectKind

	CreatedAt time.Time
	UpdatedAt time.Time
	SeenAt    time.Time

	Location ProjectLocationInput
}

// ProjectLocationInput stores the filesystem footprint of a registered project.
type ProjectLocationInput struct {
	MnemonicFileAbs string
	RepoRootAbs     string
	MemoriesAbs     string
	ManifestAbs     string
	SourceKind      ProjectSourceKind
}

// RegisterProject creates or updates the registry rows for a project.
func RegisterProject(db *sql.DB, input RegisterProjectInput) error {
	if db == nil {
		return fmt.Errorf("registry database is nil")
	}
	if err := ApplySchema(db); err != nil {
		return err
	}

	if input.SeenAt.IsZero() {
		input.SeenAt = input.UpdatedAt
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin registry transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var conflictProjectID string
	err = tx.QueryRow(
		`SELECT project_id FROM projects WHERE slug = ? AND removed_at IS NULL AND project_id <> ? LIMIT 1`,
		input.Slug,
		input.ProjectID,
	).Scan(&conflictProjectID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("check project slug conflict: %w", err)
	}
	if conflictProjectID != "" {
		return ErrProjectSlugConflict{Slug: input.Slug}
	}

	if _, err := tx.Exec(
		`INSERT INTO projects (project_id, name, slug, kind, created_at, updated_at, removed_at)
		 VALUES (?, ?, ?, ?, ?, ?, NULL)
		 ON CONFLICT(project_id) DO UPDATE SET
			name = excluded.name,
			slug = excluded.slug,
			kind = excluded.kind,
			updated_at = excluded.updated_at,
			removed_at = NULL`,
		input.ProjectID,
		input.Name,
		input.Slug,
		input.Kind,
		input.CreatedAt.UTC().Format(time.RFC3339),
		input.UpdatedAt.UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("upsert project: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO project_locations (
			project_id, mnemonic_file_abs, repo_root_abs, memories_abs, manifest_abs, source_kind, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id) DO UPDATE SET
			mnemonic_file_abs = excluded.mnemonic_file_abs,
			repo_root_abs = excluded.repo_root_abs,
			memories_abs = excluded.memories_abs,
			manifest_abs = excluded.manifest_abs,
			source_kind = excluded.source_kind,
			last_seen_at = excluded.last_seen_at`,
		input.ProjectID,
		nullString(input.Location.MnemonicFileAbs),
		nullString(input.Location.RepoRootAbs),
		input.Location.MemoriesAbs,
		nullString(input.Location.ManifestAbs),
		input.Location.SourceKind,
		input.SeenAt.UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("upsert project location: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO project_status (
			project_id, index_schema_version, last_indexed_at, last_seen_at, index_present, needs_reindex
		) VALUES (?, 0, NULL, ?, 0, 1)
		ON CONFLICT(project_id) DO UPDATE SET
			last_seen_at = excluded.last_seen_at`,
		input.ProjectID,
		input.SeenAt.UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("upsert project status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit registry transaction: %w", err)
	}

	return nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
