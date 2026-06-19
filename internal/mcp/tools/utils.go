package tools

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
)

// indexedNote represents a minimal note record from the index database.
type IndexedNote struct {
	NoteID string
}

// QueryIndexedNoteByIdentifier looks up a note in the index database by identifier.
func QueryIndexedNoteByIdentifier(db *sql.DB, identifier string) (IndexedNote, error) {
	row := db.QueryRow(
		`SELECT note_id
		 FROM notes
		 WHERE note_id = ? OR slug = ? OR rel_path = ? OR title = ?`,
		identifier, identifier, identifier, identifier,
	)
	var note IndexedNote
	if err := row.Scan(&note.NoteID); err != nil {
		if err == sql.ErrNoRows {
			return IndexedNote{}, app.NewNotFoundError(fmt.Sprintf("note %q not found", identifier), nil)
		}
		return IndexedNote{}, fmt.Errorf("query note %q: %w", identifier, err)
	}
	return note, nil
}

func BoolPtr(value bool) *bool {
	return &value
}

// EnsurePathInsideRoot validates that the target path is inside the root directory.
func EnsurePathInsideRoot(root, target string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root path: %w", err)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}

	if absTarget != absRoot && !strings.HasPrefix(absTarget, absRoot+string(os.PathSeparator)) {
		return app.NewCLIUsageError("path must stay inside the root directory", nil)
	}

	return nil
}

// buildToolDescription prepends a custom knowledge-base description to the
// base strict instructions when a description is configured. This helps LLM
// agents route queries to the correct memory instance.
func buildToolDescription(description, baseInstructions string) string {
	if description == "" {
		return baseInstructions
	}
	return "Target Knowledge Base: " + description + "\n\n" + baseInstructions
}
