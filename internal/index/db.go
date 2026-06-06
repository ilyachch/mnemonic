package index

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

// OpenDB opens the per-project index database, creating parents and applying the
// required connection pragmas.
func OpenDB(projectID string) (*sql.DB, error) {
	dbPath, err := Path(projectID)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepathDir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create index directory: %w", err)
	}

	db, err := sql.Open(sqliteDriverName, dbPath)
	if err != nil {
		return nil, fmt.Errorf("open index database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping index database: %w", err)
	}

	pragmas := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA user_version = 1`,
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply %s: %w", pragma, err)
		}
	}

	return db, nil
}

func filepathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
