package index

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/lock"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestRebuildProjectIndexReturnsBusyErrorWhenLockHeld(t *testing.T) {
	testutil.CleanEnvForTest(t)

	memoriesRoot := filepath.Join(t.TempDir(), "memories")
	if err := createIndexTestNote(memoriesRoot, "alpha.md", "# Alpha\n"); err != nil {
		t.Fatalf("createIndexTestNote() error = %v", err)
	}

	guard, err := lock.Acquire(lock.AcquireInput{
		ProjectID: "550e8400-e29b-41d4-a716-446655440000",
		Name:      reindexLockName,
	})
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	t.Cleanup(func() { _ = guard.Release() })

	_, err = RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot)
	if err == nil {
		t.Fatal("RebuildProjectIndex() error = nil, want unsafe error")
	}

	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeUnsafe {
		t.Fatalf("RebuildProjectIndex() error = %v, want unsafe error", err)
	}
}

func createIndexTestNote(root string, relPath string, content string) error {
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(relPath)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, relPath), []byte(content), 0o644)
}
