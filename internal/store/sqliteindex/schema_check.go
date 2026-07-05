package sqliteindex

import (
	"database/sql"
	"fmt"
)

const reindexHint = "; run `mnemonic project reindex` to resolve"

var requiredTables = map[string][]string{
	"notes":        {"note_id", "project_id", "slug", "rel_path", "title", "content_hash", "summary", "created_at", "updated_at"},
	"note_tags":    {"note_id", "tag"},
	"links":        {"link_id", "note_id", "to_note_id", "target", "label", "link_style", "source_kind", "relation_type", "is_resolved", "is_ambiguous", "source_line"},
	"observations": {"observation_id", "note_id", "kind", "value"},
	"notes_fts":    {"note_id", "title", "summary", "tags", "aliases", "body"},
}

// ValidateSchema checks that the required tables and columns exist in the
// current index schema. It does not compare versions or attempt migrations.
func ValidateSchema(db *sql.DB) error {
	for table, columns := range requiredTables {
		existing, err := tableColumns(db, table)
		if err != nil {
			return fmt.Errorf("index is invalid: table %q is missing%s", table, reindexHint)
		}
		for _, col := range columns {
			if !existing[col] {
				return fmt.Errorf("index is invalid: column %q.%q is missing%s", table, col, reindexHint)
			}
		}
	}
	return nil
}

func tableColumns(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("scan table_info %q: %w", table, err)
		}
		columns[name] = true
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("table %q does not exist", table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate table_info %q: %w", table, err)
	}
	return columns, nil
}
