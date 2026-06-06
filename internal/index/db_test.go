package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestOpenDBCreatesFileAndPragmas(t *testing.T) {
	stateHome := filepath.Join(testutil.CleanEnvForTest(t), "state")

	db, err := OpenDB("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	path := filepath.Join(stateHome, "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "index.sqlite")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("index file missing: %v", err)
	}

	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("PRAGMA foreign_keys query failed: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("PRAGMA foreign_keys = %d, want 1", foreignKeys)
	}

	var journalMode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode query failed: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("PRAGMA journal_mode = %q, want wal", journalMode)
	}

	var busyTimeout int
	if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout query failed: %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("PRAGMA busy_timeout = %d, want 5000", busyTimeout)
	}

	var userVersion int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&userVersion); err != nil {
		t.Fatalf("PRAGMA user_version query failed: %v", err)
	}
	if userVersion != 1 {
		t.Fatalf("PRAGMA user_version = %d, want 1", userVersion)
	}
}
