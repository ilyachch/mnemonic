package index

import "database/sql"

// SchemaStatus describes whether an existing index can be reused.
type SchemaStatus string

const (
	// SchemaStatusOK indicates that the database schema is compatible.
	SchemaStatusOK SchemaStatus = "ok"
	// SchemaStatusNeedsRebuild indicates that the index must be rebuilt.
	SchemaStatusNeedsRebuild SchemaStatus = "needs_rebuild"
)

// CheckSchemaStatus reports whether the current DB schema is compatible.
func CheckSchemaStatus(db *sql.DB) (SchemaStatus, error) {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return "", err
	}
	if version != 1 {
		return SchemaStatusNeedsRebuild, nil
	}
	return SchemaStatusOK, nil
}
