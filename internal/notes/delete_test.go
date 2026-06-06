package notes

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestDeleteDryRunReturnsTrashPath(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result, err := Delete(DeleteInput{
		RootDir:  root,
		Selector: created.Slug,
		DryRun:   true,
		Now: func() time.Time {
			return time.Date(2026, time.June, 2, 15, 4, 5, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if result.Mode != "trash" {
		t.Fatalf("Mode = %q, want trash", result.Mode)
	}
	if result.TrashPath == "" {
		t.Fatal("TrashPath is empty")
	}

	if _, err := os.Stat(filepath.Join(root, "auth-migration.md")); err != nil {
		t.Fatalf("original file stat error = %v", err)
	}
	if _, err := os.Stat(result.TrashPath); !os.IsNotExist(err) {
		t.Fatalf("trash file stat = %v, want not exist", err)
	}
}

func TestDeleteMovesNoteToTrash(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("body\n"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result, err := Delete(DeleteInput{
		RootDir:  root,
		Selector: created.Slug,
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if result.Mode != "trash" {
		t.Fatalf("Mode = %q, want trash", result.Mode)
	}
	if result.TrashPath == "" {
		t.Fatal("TrashPath is empty")
	}

	if _, err := os.Stat(filepath.Join(root, "auth-migration.md")); !os.IsNotExist(err) {
		t.Fatalf("source file stat = %v, want not exist", err)
	}
	if _, err := os.Stat(result.TrashPath); err != nil {
		t.Fatalf("trash file stat = %v", err)
	}

	list, err := List(root)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("List() len = %d, want 0", len(list))
	}
}

func TestDeleteHardRequiresConfirmation(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("body\n"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = Delete(DeleteInput{
		RootDir:  root,
		Selector: created.Slug,
		Hard:     true,
	})
	if err == nil {
		t.Fatal("Delete() error = nil, want unsafe error")
	}
}

func TestDeleteHardRemovesFileWhenConfirmed(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("body\n"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result, err := Delete(DeleteInput{
		RootDir:  root,
		Selector: created.Slug,
		Hard:     true,
		Yes:      true,
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if result.Mode != "hard" {
		t.Fatalf("Mode = %q, want hard", result.Mode)
	}

	if _, err := os.Stat(filepath.Join(root, "auth-migration.md")); !os.IsNotExist(err) {
		t.Fatalf("source file stat = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".trash")); !os.IsNotExist(err) {
		t.Fatalf("trash dir stat = %v, want not exist", err)
	}
}

func TestDeleteRejectsMissingSelector(t *testing.T) {
	_, err := Delete(DeleteInput{})
	if err == nil {
		t.Fatal("Delete() error = nil, want error")
	}
}

func TestDeleteReturnsBusyErrorWhenWriteLockHeld(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	guard, err := acquireWriteLock(root)
	if err != nil {
		t.Fatalf("acquireWriteLock() error = %v", err)
	}
	defer func() {
		if err := guard.Release(); err != nil {
			t.Fatalf("Release() error = %v", err)
		}
	}()

	_, err = Delete(DeleteInput{
		RootDir:  root,
		Selector: created.Slug,
	})
	if err == nil {
		t.Fatal("Delete() error = nil, want busy error")
	}

	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeUnsafe {
		t.Fatalf("Delete() error = %v, want unsafe error", err)
	}
}
