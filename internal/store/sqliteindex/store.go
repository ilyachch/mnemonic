package sqliteindex

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/index"
	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

// Store provides path-explicit access to a SQLite index database.
type Store struct {
	IndexPath string
	RootDir   string
	KBID      string
}

// Open opens the index database, creating parent directories and applying the
// required connection pragmas.
func (s Store) Open() (*sql.DB, error) {
	if err := s.validateIndexPath(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(s.IndexPath), 0o755); err != nil {
		return nil, fmt.Errorf("create index directory: %w", err)
	}
	return openDB(s.IndexPath)
}

// OpenReadonly opens the index database in read-only mode.
func (s Store) OpenReadonly() (*sql.DB, error) {
	if err := s.validateIndexPath(); err != nil {
		return nil, err
	}
	db, err := sql.Open(sqliteDriverName, "file:"+filepath.ToSlash(s.IndexPath)+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open index database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping index database: %w", err)
	}
	return db, nil
}

// Exists reports whether the index database file exists.
func (s Store) Exists() (bool, error) {
	if err := s.validateIndexPath(); err != nil {
		return false, err
	}
	_, err := os.Stat(s.IndexPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Remove deletes the index database and SQLite sidecars.
func (s Store) Remove() error {
	if err := s.validateIndexPath(); err != nil {
		return err
	}
	return removeIndexFiles(s.IndexPath)
}

// QuickCheck runs SQLite integrity checks for the explicit index path.
func (s Store) QuickCheck() error {
	if err := s.validateIndexPath(); err != nil {
		return err
	}
	return index.QuickCheck(s.IndexPath)
}

// CheckSchemaStatus reports whether the current DB schema is compatible.
func (s Store) CheckSchemaStatus(db *sql.DB) (index.SchemaStatus, error) {
	return index.CheckSchemaStatus(db)
}

func (s Store) validateIndexPath() error {
	if strings.TrimSpace(s.IndexPath) == "" {
		return fmt.Errorf("index path is required")
	}
	return nil
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open(sqliteDriverName, path)
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

func acquireRebuildLock(indexPath string) (*flock.Flock, error) {
	lockPath := rebuildLockPath(indexPath)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	fileLock := flock.New(lockPath, flock.SetPermissions(0o644))
	deadline := time.Now().Add(150 * time.Millisecond)
	for {
		ok, err := fileLock.TryLock()
		if err != nil {
			return nil, fmt.Errorf("acquire lock: %w", err)
		}
		if ok {
			return fileLock, nil
		}
		if !time.Now().Before(deadline) {
			return nil, apperr.Unsafe("index rebuild lock is busy", fmt.Errorf("%s", lockPath))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func rebuildLockPath(indexPath string) string {
	return filepath.Join(filepath.Dir(indexPath), "reindex.lock")
}

func removeIndexFiles(path string) error {
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		if err := os.Remove(candidate); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
