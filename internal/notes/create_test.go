package notes

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/project"
)

func TestCreateWritesCanonicalNote(t *testing.T) {
	root := t.TempDir()

	got, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got.Slug != "auth-migration" {
		t.Fatalf("Slug = %q, want %q", got.Slug, "auth-migration")
	}
	if got.Path != "auth-migration.md" {
		t.Fatalf("Path = %q, want %q", got.Path, "auth-migration.md")
	}
	if got.NoteID == "" {
		t.Fatal("NoteID is empty")
	}
	if got.ContentHash == "" {
		t.Fatal("ContentHash is empty")
	}

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	parsed, err := markdown.ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if parsed.Title != "Auth migration" {
		t.Fatalf("Title = %q", parsed.Title)
	}
	if parsed.Slug != "auth-migration" {
		t.Fatalf("Slug = %q", parsed.Slug)
	}
}

func TestCreateUsesInjectedClockAndUUID(t *testing.T) {
	root := t.TempDir()
	now := func() time.Time {
		return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
	}
	uuidFn := func() string {
		return "550e8400-e29b-41d4-a716-446655440000"
	}

	got, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("## Summary\n"),
		Now:     now,
		UUID:    uuidFn,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	want := CreateResult{
		NoteID:      "550e8400-e29b-41d4-a716-446655440000",
		Slug:        "auth-migration",
		Path:        "auth-migration.md",
		ContentHash: got.ContentHash,
	}
	if got.NoteID != want.NoteID || got.Slug != want.Slug || got.Path != want.Path {
		t.Fatalf("Create() = %#v, want %#v", got, want)
	}

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	parsed, err := markdown.ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if parsed.MnemonicNoteID != want.NoteID {
		t.Fatalf("MnemonicNoteID = %q, want %q", parsed.MnemonicNoteID, want.NoteID)
	}
	if !parsed.CreatedAt.Equal(now()) || !parsed.UpdatedAt.Equal(now()) {
		t.Fatalf("timestamps = %s/%s, want %s", parsed.CreatedAt, parsed.UpdatedAt, now())
	}
}

func TestCreateRejectsDuplicateSlug(t *testing.T) {
	root := t.TempDir()
	if _, err := Create(CreateInput{RootDir: root, Title: "Auth migration"}); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	if _, err := Create(CreateInput{RootDir: root, Title: "Auth migration"}); err == nil {
		t.Fatal("Create() error = nil, want duplicate slug error")
	}
}

func TestCreateRespectsProjectClock(t *testing.T) {
	root := t.TempDir()
	restore := project.SetClock(testClock{})
	defer restore()

	got, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got.NoteID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("NoteID = %q", got.NoteID)
	}
}

func TestCreateReturnsBusyErrorWhenWriteLockHeld(t *testing.T) {
	root := t.TempDir()

	guard, err := acquireWriteLock(root)
	if err != nil {
		t.Fatalf("acquireWriteLock() error = %v", err)
	}
	defer func() {
		if err := guard.Release(); err != nil {
			t.Fatalf("Release() error = %v", err)
		}
	}()

	_, err = Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	if err == nil {
		t.Fatal("Create() error = nil, want busy error")
	}

	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeUnsafe {
		t.Fatalf("Create() error = %v, want unsafe error", err)
	}
}

type testClock struct{}

func (testClock) Now() time.Time {
	return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
}

func (testClock) UUID() string {
	return "550e8400-e29b-41d4-a716-446655440000"
}
