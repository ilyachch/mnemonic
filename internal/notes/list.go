package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ilyachch/mnemonic/internal/format/markdown"
)

// NoteSummary is the note listing projection used by the CLI.
type NoteSummary struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Path        string `json:"path"`
	UpdatedAt   string `json:"updated_at"`
	ContentHash string `json:"content_hash"`
}

// List returns all note summaries under root, excluding .trash notes.
func List(root string) ([]NoteSummary, error) {
	if root == "" {
		return nil, fmt.Errorf("root directory is required")
	}

	paths, err := Walk(root)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	notes := make([]NoteSummary, 0, len(paths))
	for _, relPath := range paths {
		absPath := filepath.Join(root, filepath.FromSlash(relPath))
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read note %q: %w", relPath, err)
		}

		note, err := markdown.ParseNote(data)
		if err != nil {
			return nil, fmt.Errorf("parse note %q: %w", relPath, err)
		}
		if note.MnemonicNoteID == "" {
			return nil, fmt.Errorf("note %q is missing mnemonic_note_id", relPath)
		}
		if note.UpdatedAt.IsZero() {
			return nil, fmt.Errorf("note %q is missing updated_at", relPath)
		}

		notes = append(notes, NoteSummary{
			NoteID:      note.MnemonicNoteID,
			Slug:        note.EffectiveSlug(),
			Title:       note.Title,
			Path:        relPath,
			UpdatedAt:   note.UpdatedAt.UTC().Format(time.RFC3339),
			ContentHash: HashBytes(data),
		})
	}

	return notes, nil
}
