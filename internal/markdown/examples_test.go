package markdown

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExampleNoteParses(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "markdown", "example-note.md"))
	if err != nil {
		t.Fatalf("ReadFile(example-note.md) error = %v", err)
	}

	note, err := ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}

	if note.Title == "" {
		t.Fatal("Title is empty")
	}
	if len(note.Body) == 0 {
		t.Fatal("Body is empty")
	}
}
