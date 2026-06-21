package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/spf13/cobra"
)

var notesEditCmd = &cobra.Command{
	Use:   "edit SELECTOR",
	Short: "Edit a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appendText, err := cmd.Flags().GetString("append")
		if err != nil {
			return err
		}
		bodyFile, err := cmd.Flags().GetString("body-file")
		if err != nil {
			return err
		}
		ifMatch, err := cmd.Flags().GetString("if-match")
		if err != nil {
			return err
		}
		var setFields map[string]string
		if cmd.Flags().Changed("set") {
			setValues, err := cmd.Flags().GetStringArray("set")
			if err != nil {
				return err
			}
			setFields, err = parseEditSetValues(setValues)
			if err != nil {
				return err
			}
		}
		if appendText == "" && bodyFile == "" && len(setFields) == 0 {
			return apperr.CLIUsage("edit requires --append, --body-file, or --set", nil)
		}
		if appendText != "" && bodyFile != "" {
			return apperr.CLIUsage("--append and --body-file cannot be combined", nil)
		}

		root, err := resolveNotesProjectRoot()
		if err != nil {
			return err
		}

		editInput := notes.EditInput{
			RootDir:  root,
			Selector: args[0],
			Set:      setFields,
			IfMatch:  ifMatch,
		}
		if bodyFile != "" {
			body, err := os.ReadFile(bodyFile)
			if err != nil {
				return fmt.Errorf("read body file: %w", err)
			}
			editInput.Body = body
			editInput.HasBody = true
		} else {
			editInput.Append = []byte(appendText)
		}

		edited, err := notes.Edit(editInput)
		if err != nil {
			return err
		}

		output := notesEditOutput{
			NoteID:      edited.NoteID,
			Slug:        edited.Slug,
			Path:        edited.Path,
			ContentHash: edited.ContentHash,
			CreatedAt:   edited.CreatedAt,
			UpdatedAt:   edited.UpdatedAt,
		}
		return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s updated\n", edited.Path), output)
	},
}

func init() {
	notesEditCmd.Flags().String("append", "", "append text to the note body")
	notesEditCmd.Flags().String("body-file", "", "replace the note body from a file")
	notesEditCmd.Flags().String("if-match", "", "require the current content hash to match")
	notesEditCmd.Flags().StringArray("set", nil, "set a frontmatter field")
	notesCmd.AddCommand(notesEditCmd)
}

type notesEditOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func parseEditSetValues(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	out := make(map[string]string, len(values))
	for _, raw := range values {
		if raw == "" {
			continue
		}
		key, value, ok := strings.Cut(raw, "=")
		if !ok || key == "" {
			return nil, apperr.CLIUsage("--set values must use key=value", nil)
		}
		out[key] = value
	}

	return out, nil
}
