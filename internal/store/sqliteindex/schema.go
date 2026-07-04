package sqliteindex

import "database/sql"

// ApplySchema creates the index schema for version 2.
func ApplySchema(db *sql.DB) error {
	stmts := []string{
		`PRAGMA application_id = 1095521358`,
		`PRAGMA user_version = 2`,
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS notes (
			note_id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			rel_path TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			summary TEXT DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_updated_at ON notes(updated_at)`,
		`CREATE TABLE IF NOT EXISTS note_tags (
			note_id TEXT NOT NULL REFERENCES notes(note_id) ON DELETE CASCADE,
			tag TEXT NOT NULL,
			PRIMARY KEY (note_id, tag)
		)`,
		`CREATE TABLE IF NOT EXISTS observations (
			observation_id TEXT PRIMARY KEY,
			note_id TEXT NOT NULL REFERENCES notes(note_id) ON DELETE CASCADE,
			kind TEXT NOT NULL,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS note_aliases (
			note_id TEXT NOT NULL REFERENCES notes(note_id) ON DELETE CASCADE,
			alias TEXT NOT NULL,
			PRIMARY KEY (note_id, alias)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_note_aliases_alias ON note_aliases(alias)`,
		`CREATE TABLE IF NOT EXISTS links (
			link_id TEXT PRIMARY KEY,
			note_id TEXT NOT NULL REFERENCES notes(note_id) ON DELETE CASCADE,
			to_note_id TEXT REFERENCES notes(note_id) ON DELETE SET NULL,
			target TEXT NOT NULL,
			label TEXT DEFAULT '',
			link_style TEXT DEFAULT '',
			source_kind TEXT DEFAULT '',
			relation_type TEXT DEFAULT '',
			is_resolved INTEGER NOT NULL DEFAULT 0,
			is_ambiguous INTEGER NOT NULL DEFAULT 0,
			source_line INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS index_runs (
			run_id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			status TEXT NOT NULL
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
			note_id UNINDEXED,
			title,
			summary,
			tags,
			aliases,
			body,
			tokenize = 'unicode61'
		)`,
		`INSERT OR IGNORE INTO meta(key, value) VALUES ('schema_version', '2')`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

// CheckSchemaStatus reports whether the current DB schema is compatible.
func CheckSchemaStatus(db *sql.DB) (SchemaStatus, error) {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return "", err
	}
	if version != 2 {
		return SchemaStatusNeedsRebuild, nil
	}
	return SchemaStatusOK, nil
}
