package index

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestQuickCheckHealthyDB(t *testing.T) {
	testutil.CleanEnvForTest(t)

	db, err := OpenDB("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}

	path, err := Path("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if err := QuickCheck(path); err != nil {
		t.Fatalf("QuickCheck() error = %v", err)
	}
}

func TestQuickCheckCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.sqlite")
	if err := os.WriteFile(path, []byte("not sqlite"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := QuickCheck(path)
	if err == nil {
		t.Fatal("QuickCheck() error = nil, want corruption error")
	}
	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeCorrupted {
		t.Fatalf("QuickCheck() error = %v, want corruption error", err)
	}
}
