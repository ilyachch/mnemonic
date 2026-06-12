package tools

import "database/sql"

// Dependencies abstracts server dependencies so tools don't depend on mcp.Server directly.
type Dependencies interface {
	GetMemoriesRoot() (string, error)
	GetIndexDB() (*sql.DB, error)
	RebuildIndex(root string) error
}
