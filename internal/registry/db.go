package registry

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/ilyachch/mnemonic/internal/paths"
)

const sqliteDriverName = "sqlite"

// RegistryPath returns the absolute registry.sqlite path under the XDG data home.
func RegistryPath() (string, error) {
	mnemonicPaths, err := paths.GetMnemonicPaths()
	if err != nil {
		return "", err
	}

	return filepath.Join(mnemonicPaths.DataHome, "mnemonic", "registry.sqlite"), nil
}

// OpenDB opens the global registry database, creating parent directories and applying
// the required connection pragmas.
func OpenDB() (*sql.DB, error) {
	dbPath, err := RegistryPath()
	if err != nil {
		return nil, err
	}

	if err := ensureParentDir(dbPath); err != nil {
		return nil, err
	}

	db, err := sql.Open(sqliteDriverName, dbPath)
	if err != nil {
		return nil, fmt.Errorf("open registry database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping registry database: %w", err)
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return db, nil
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if filepath.Clean(dir) == "." {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create registry directory: %w", err)
	}
	return nil
}
