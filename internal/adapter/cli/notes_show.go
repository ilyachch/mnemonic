package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var notesShowCmd = &cobra.Command{
	Use:   "show SELECTOR",
	Short: "Show a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := runtimeAppForSelectedProject(cmd)
		if err != nil {
			return err
		}

		resolved, err := runtime.Services.Notes.Show(args[0])
		if err != nil {
			return err
		}

		data, err := os.ReadFile(filepath.Join(runtime.KB.RootDir, filepath.FromSlash(resolved.Path)))
		if err != nil {
			return fmt.Errorf("read note %q: %w", resolved.Path, err)
		}

		output := notesShowOutput{
			Note: notesShowItem{
				NoteID:      resolved.Note.MnemonicNoteID,
				Slug:        resolved.Note.EffectiveSlug(),
				Title:       resolved.Note.Title,
				Path:        resolved.Path,
				Frontmatter: resolved.Note.Frontmatter,
				Body:        string(resolved.Note.Body),
				ContentHash: resolved.ContentHash,
				UpdatedAt:   resolved.Note.UpdatedAt.UTC().Format(time.RFC3339),
			},
		}
		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), string(data), output)
	},
}

type notesShowOutput struct {
	Note notesShowItem `json:"note"`
}

type notesShowItem struct {
	NoteID      string         `json:"note_id"`
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Path        string         `json:"path"`
	Frontmatter map[string]any `json:"frontmatter"`
	Body        string         `json:"body"`
	ContentHash string         `json:"content_hash"`
	UpdatedAt   string         `json:"updated_at"`
}
