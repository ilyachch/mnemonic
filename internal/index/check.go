package index

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/apperr"
	_ "modernc.org/sqlite"
)

// QuickCheck runs SQLite integrity checks for an index database file.
func QuickCheck(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path is required")
	}

	db, err := sql.Open(sqliteDriverName, path)
	if err != nil {
		return apperr.Corrupted("index database is corrupted", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.Ping(); err != nil {
		return classifyCorruption(err)
	}

	var result string
	if err := db.QueryRow(`PRAGMA quick_check`).Scan(&result); err != nil {
		return classifyCorruption(err)
	}
	if result != "ok" {
		return apperr.Corrupted("index database is corrupted", fmt.Errorf("quick_check = %s", result))
	}
	return nil
}

func classifyCorruption(err error) error {
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return apperr.NotFound("index database not found", err)
	}
	return apperr.Corrupted("index database is corrupted", err)
}
