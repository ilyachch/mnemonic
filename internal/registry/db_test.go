package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestOpenDBCreatesRegistryFileAndPragmas(t *testing.T) {
	dataHome := filepath.Join(testutil.CleanEnvForTest(t), "data")

	path, err := RegistryPath()
	if err != nil {
		t.Fatalf("RegistryPath() error = %v", err)
	}
	wantPath := filepath.Join(dataHome, "mnemonic", "registry.sqlite")
	if path != wantPath {
		t.Fatalf("RegistryPath() = %q, want %q", path, wantPath)
	}

	db, err := OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("parent directory missing: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("registry file missing: %v", err)
	}

	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("PRAGMA foreign_keys query failed: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("PRAGMA foreign_keys = %d, want 1", foreignKeys)
	}

	var userVersion int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&userVersion); err != nil {
		t.Fatalf("PRAGMA user_version query failed: %v", err)
	}
	if userVersion != 0 {
		t.Fatalf("PRAGMA user_version = %d, want 0 before migrations", userVersion)
	}
}
