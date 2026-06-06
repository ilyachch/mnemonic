package notes

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/markdown"
)

func TestResolveFollowsSelectorPrecedence(t *testing.T) {
	root := t.TempDir()

	writeResolvedNote(t, root, "uuid.md", markdown.Note{
		MnemonicNoteID: "11111111-1111-1111-1111-111111111111",
		Title:          "UUID Note",
		Slug:           "uuid-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "slug.md", markdown.Note{
		MnemonicNoteID: "22222222-2222-2222-2222-222222222222",
		Title:          "Slug Note",
		Slug:           "slug-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, filepath.Join("folder", "path.md"), markdown.Note{
		MnemonicNoteID: "33333333-3333-3333-3333-333333333333",
		Title:          "Path Note",
		Slug:           "path-note",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "title.md", markdown.Note{
		MnemonicNoteID: "44444444-4444-4444-4444-444444444444",
		Title:          "Exact Title",
		Slug:           "exact-title-slug",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "normalized.md", markdown.Note{
		MnemonicNoteID: "55555555-5555-5555-5555-555555555555",
		Title:          "Normalized Title",
		Slug:           "custom-slug",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	cases := []struct {
		name     string
		selector string
		wantPath string
	}{
		{name: "uuid", selector: "11111111-1111-1111-1111-111111111111", wantPath: "uuid.md"},
		{name: "slug", selector: "slug-note", wantPath: "slug.md"},
		{name: "path", selector: filepath.ToSlash(filepath.Join("folder", "path.md")), wantPath: filepath.ToSlash(filepath.Join("folder", "path.md"))},
		{name: "title", selector: "Exact Title", wantPath: "title.md"},
		{name: "normalized title", selector: "normalized-title", wantPath: "normalized.md"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(root, tc.selector)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if got.Path != tc.wantPath {
				t.Fatalf("Resolve().Path = %q, want %q", got.Path, tc.wantPath)
			}
		})
	}
}

func TestResolveReturnsAmbiguousError(t *testing.T) {
	root := t.TempDir()
	writeResolvedNote(t, root, "a.md", markdown.Note{
		MnemonicNoteID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:          "Duplicate Title",
		Slug:           "dup-a",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})
	writeResolvedNote(t, root, "b.md", markdown.Note{
		MnemonicNoteID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Title:          "Duplicate Title",
		Slug:           "dup-b",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	_, err := Resolve(root, "Duplicate Title")
	if err == nil {
		t.Fatal("Resolve() error = nil, want ambiguous error")
	}
	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeAmbiguous {
		t.Fatalf("Resolve() error = %v, want ambiguous error", err)
	}
}

func TestResolveReturnsNotFoundError(t *testing.T) {
	root := t.TempDir()
	writeResolvedNote(t, root, "note.md", markdown.Note{
		MnemonicNoteID: "cccccccc-cccc-cccc-cccc-cccccccccccc",
		Title:          "Existing",
		Slug:           "existing",
		CreatedAt:      noteTime(),
		UpdatedAt:      noteTime(),
	})

	_, err := Resolve(root, "missing")
	if err == nil {
		t.Fatal("Resolve() error = nil, want not found error")
	}
	var appErr *app.AppError
	if !errors.As(err, &appErr) || appErr.Code != app.CodeNotFound {
		t.Fatalf("Resolve() error = %v, want not found error", err)
	}
}

func writeResolvedNote(t *testing.T, root, relPath string, note markdown.Note) {
	t.Helper()

	rendered, err := markdown.RenderNote(note)
	if err != nil {
		t.Fatalf("RenderNote() error = %v", err)
	}
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, rendered, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func noteTime() time.Time {
	return time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC)
}
