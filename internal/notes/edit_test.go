package notes

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestEditAppendsBodyAndUpdatesTimestamps(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("## Summary\n"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	edited, err := Edit(EditInput{
		RootDir:  root,
		Selector: created.Slug,
		Append:   []byte("Next step"),
	})
	if err != nil {
		t.Fatalf("Edit() error = %v", err)
	}
	if edited.ContentHash == created.ContentHash {
		t.Fatalf("ContentHash = %q, want change", edited.ContentHash)
	}

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	note, err := markdown.ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if string(note.Body) != "## Summary\nNext step" {
		t.Fatalf("Body = %q", note.Body)
	}
	if note.CreatedAt.UTC().Format(time.RFC3339) != "2026-06-02T12:34:56Z" {
		t.Fatalf("CreatedAt = %s", note.CreatedAt)
	}
	if note.UpdatedAt.UTC().Format(time.RFC3339) != "2026-06-02T12:35:56Z" {
		t.Fatalf("UpdatedAt = %s", note.UpdatedAt)
	}
}

func TestEditSetsFrontmatterField(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	_, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("## Summary\n"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	restoreEditClock := project.SetClock(testutil.NewClock(time.Date(2026, time.June, 2, 12, 35, 56, 0, time.UTC)))
	defer restoreEditClock()

	if _, err := Edit(EditInput{
		RootDir:  root,
		Selector: "auth-migration",
		Set: map[string]string{
			"type": "decision",
		},
	}); err != nil {
		t.Fatalf("Edit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	note, err := markdown.ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}
	if note.Type != "decision" {
		t.Fatalf("Type = %q, want %q", note.Type, "decision")
	}
}

func TestEditRejectsProtectedFrontmatterField(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	_, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = Edit(EditInput{
		RootDir:  root,
		Selector: "auth-migration",
		Set: map[string]string{
			"created_at": "2026-06-02T12:00:00Z",
		},
	})
	if err == nil {
		t.Fatal("Edit() error = nil, want unsafe error")
	}
	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeUnsafe {
		t.Fatalf("Edit() error = %v, want unsafe error", err)
	}
}

func TestEditRejectsContentHashMismatch(t *testing.T) {
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

	_, err = Edit(EditInput{
		RootDir:  root,
		Selector: created.Slug,
		IfMatch:  "deadbeef",
		Append:   []byte("Next step"),
	})
	if err == nil {
		t.Fatal("Edit() error = nil, want unsafe error")
	}
	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeUnsafe {
		t.Fatalf("Edit() error = %v, want unsafe error", err)
	}
	data, readErr := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	note, parseErr := markdown.ParseNote(data)
	if parseErr != nil {
		t.Fatalf("ParseNote() error = %v", parseErr)
	}
	if string(note.Body) != "" {
		t.Fatalf("Body = %q, want unchanged", note.Body)
	}
}

func TestEditReturnsBusyErrorWhenWriteLockHeld(t *testing.T) {
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

	_, err = Edit(EditInput{
		RootDir:  root,
		Selector: created.Slug,
		Append:   []byte("blocked"),
	})
	if err == nil {
		t.Fatal("Edit() error = nil, want busy error")
	}

	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeUnsafe {
		t.Fatalf("Edit() error = %v, want unsafe error", err)
	}
}

func TestConcurrentEditWithSameHashAllowsOnlyOneSuccess(t *testing.T) {
	root := t.TempDir()
	restoreCreateClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreCreateClock()

	created, err := Create(CreateInput{
		RootDir: root,
		Title:   "Auth migration",
		Body:    []byte("## Summary\n"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	type outcome struct {
		result EditResult
		err    error
	}

	appends := [][]byte{
		[]byte("First change\n"),
		[]byte("Second change\n"),
	}
	results := make([]outcome, len(appends))

	start := make(chan struct{})
	var wg sync.WaitGroup
	for idx, appendBody := range appends {
		wg.Add(1)
		go func(i int, body []byte) {
			defer wg.Done()
			<-start
			results[i].result, results[i].err = Edit(EditInput{
				RootDir:  root,
				Selector: created.Slug,
				IfMatch:  created.ContentHash,
				Append:   body,
			})
		}(idx, appendBody)
	}

	close(start)
	wg.Wait()

	successes := 0
	preconditionFailures := 0
	for _, result := range results {
		if result.err == nil {
			successes++
			continue
		}
		var appErr *app.AppError
		if !errors.As(result.err, &appErr) || appErr.Code != app.CodeUnsafe || !strings.Contains(result.err.Error(), "content hash mismatch") {
			t.Fatalf("Edit() error = %v, want content hash mismatch unsafe error", result.err)
		}
		preconditionFailures++
	}

	if successes != 1 {
		t.Fatalf("successes = %d, want 1", successes)
	}
	if preconditionFailures != 1 {
		t.Fatalf("preconditionFailures = %d, want 1", preconditionFailures)
	}

	data, err := os.ReadFile(filepath.Join(root, "auth-migration.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	note, err := markdown.ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}

	body := string(note.Body)
	if strings.Contains(body, "First change\nSecond change\n") || strings.Contains(body, "Second change\nFirst change\n") {
		t.Fatalf("Body = %q, want exactly one applied change", body)
	}
	if body != "## Summary\nFirst change\n" && body != "## Summary\nSecond change\n" {
		t.Fatalf("Body = %q, want one winning change", body)
	}
}
