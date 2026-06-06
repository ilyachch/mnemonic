package registryschema

import (
	"database/sql"
	"fmt"
)

const (
	ApplicationID = 1095521357
	SchemaVersion = 1
)

type migration struct {
	from  int
	to    int
	apply func(*sql.Tx) error
}

var migrations = []migration{
	{from: 0, to: 1, apply: applyV1Schema},
}

// Apply migrates the registry database to the current schema version.
func Apply(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("registry database is nil")
	}

	version, err := currentVersion(db)
	if err != nil {
		return err
	}
	if version > SchemaVersion {
		return fmt.Errorf("registry schema version %d is newer than supported version %d", version, SchemaVersion)
	}

	for version < SchemaVersion {
		step, ok := migrationFor(version)
		if !ok {
			return fmt.Errorf("no registry migration from version %d", version)
		}
		if err := runMigration(db, step); err != nil {
			return err
		}
		version = step.to
	}

	return ensureSchema(db, version)
}

func currentVersion(db *sql.DB) (int, error) {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return 0, fmt.Errorf("query registry schema version: %w", err)
	}
	return version, nil
}

func migrationFor(version int) (migration, bool) {
	for _, step := range migrations {
		if step.from == version {
			return step, true
		}
	}
	return migration{}, false
}

func runMigration(db *sql.DB, step migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin registry migration %d->%d: %w", step.from, step.to, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := step.apply(tx); err != nil {
		return fmt.Errorf("apply registry migration %d->%d: %w", step.from, step.to, err)
	}
	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, step.to)); err != nil {
		return fmt.Errorf("set registry schema version %d: %w", step.to, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit registry migration %d->%d: %w", step.from, step.to, err)
	}
	return nil
}

func ensureSchema(db *sql.DB, version int) error {
	switch version {
	case 1:
		return ensureSchemaV1(db)
	default:
		return fmt.Errorf("unsupported registry schema version %d", version)
	}
}

func ensureSchemaV1(db *sql.DB) error {
	stmts := []string{
		fmt.Sprintf(`PRAGMA application_id = %d`, ApplicationID),
		`CREATE TABLE IF NOT EXISTS registry_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS projects (
			project_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			kind TEXT NOT NULL CHECK (kind IN ('regular', 'local', 'detached')),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			removed_at TEXT
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS projects_active_slug_idx
			ON projects(slug)
			WHERE removed_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS project_locations (
			project_id TEXT PRIMARY KEY REFERENCES projects(project_id) ON DELETE CASCADE,
			mnemonic_file_abs TEXT,
			repo_root_abs TEXT,
			memories_abs TEXT NOT NULL,
			manifest_abs TEXT,
			source_kind TEXT NOT NULL CHECK (source_kind IN ('init', 'import', 'discover')),
			last_seen_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS project_status (
			project_id TEXT PRIMARY KEY REFERENCES projects(project_id) ON DELETE CASCADE,
			index_schema_version INTEGER NOT NULL DEFAULT 0,
			last_indexed_at TEXT,
			last_seen_at TEXT NOT NULL,
			index_present INTEGER NOT NULL DEFAULT 0 CHECK (index_present IN (0, 1)),
			needs_reindex INTEGER NOT NULL DEFAULT 1 CHECK (needs_reindex IN (0, 1))
		)`,
		`INSERT OR IGNORE INTO registry_meta(key, value) VALUES ('schema_version', '1')`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func applyV1Schema(tx *sql.Tx) error {
	stmts := []string{
		fmt.Sprintf(`PRAGMA application_id = %d`, ApplicationID),
		`CREATE TABLE IF NOT EXISTS registry_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS projects (
			project_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			kind TEXT NOT NULL CHECK (kind IN ('regular', 'local', 'detached')),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			removed_at TEXT
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS projects_active_slug_idx
			ON projects(slug)
			WHERE removed_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS project_locations (
			project_id TEXT PRIMARY KEY REFERENCES projects(project_id) ON DELETE CASCADE,
			mnemonic_file_abs TEXT,
			repo_root_abs TEXT,
			memories_abs TEXT NOT NULL,
			manifest_abs TEXT,
			source_kind TEXT NOT NULL CHECK (source_kind IN ('init', 'import', 'discover')),
			last_seen_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS project_status (
			project_id TEXT PRIMARY KEY REFERENCES projects(project_id) ON DELETE CASCADE,
			index_schema_version INTEGER NOT NULL DEFAULT 0,
			last_indexed_at TEXT,
			last_seen_at TEXT NOT NULL,
			index_present INTEGER NOT NULL DEFAULT 0 CHECK (index_present IN (0, 1)),
			needs_reindex INTEGER NOT NULL DEFAULT 1 CHECK (needs_reindex IN (0, 1))
		)`,
		`INSERT OR IGNORE INTO registry_meta(key, value) VALUES ('schema_version', '1')`,
	}

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
