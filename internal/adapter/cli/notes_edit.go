package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/spf13/cobra"
)

func newNotesEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit SELECTOR",
		Short: "Edit a note",
		Args:  cobra.ExactArgs(1),
		RunE:  runNotesEdit,
	}
	cmd.Flags().String("append", "", "append text to the note body")
	cmd.Flags().String("body-file", "", "replace the note body with the contents of a file")
	cmd.Flags().String("if-match", "", "only update if the current content hash matches")
	cmd.Flags().StringArray("set", nil, "set a frontmatter field")
	return cmd
}

func runNotesEdit(cmd *cobra.Command, args []string) error {
	parsed, err := parseNotesEditFlags(cmd)
	if err != nil {
		return err
	}

	runtime, err := runtimeAppForSelectedProject(cmd)
	if err != nil {
		return err
	}

	editInput := notesvc.EditInput{
		Selector: args[0],
		Set:      parsed.setFields,
		IfMatch:  parsed.ifMatch,
	}
	if parsed.bodyFile != "" {
		body, readErr := os.ReadFile(parsed.bodyFile)
		if readErr != nil {
			return fmt.Errorf("read body file: %w", readErr)
		}
		editInput.Body = body
		editInput.HasBody = true
	} else {
		editInput.Append = []byte(parsed.appendText)
	}

	edited, err := runtime.Services.Notes.Edit(editInput)
	if err != nil {
		return err
	}

	if !jsonOutputEnabled(cmd) {
		printIndexWarning(cmd, edited.IndexStatus, edited.IndexError)
	}
	return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), edited.Path+" updated\n", newNotesEditOutput(edited))
}

type notesEditFlags struct {
	appendText string
	bodyFile   string
	ifMatch    string
	setFields  map[string]string
}

func parseNotesEditFlags(cmd *cobra.Command) (notesEditFlags, error) {
	appendText, err := cmd.Flags().GetString("append")
	if err != nil {
		return notesEditFlags{}, err
	}
	bodyFile, err := cmd.Flags().GetString("body-file")
	if err != nil {
		return notesEditFlags{}, err
	}
	ifMatch, err := cmd.Flags().GetString("if-match")
	if err != nil {
		return notesEditFlags{}, err
	}
	var setFields map[string]string
	if cmd.Flags().Changed("set") {
		setValues, setErr := cmd.Flags().GetStringArray("set")
		if setErr != nil {
			return notesEditFlags{}, setErr
		}
		setFields, err = parseEditSetValues(setValues)
		if err != nil {
			return notesEditFlags{}, err
		}
	}
	if appendText == "" && bodyFile == "" && len(setFields) == 0 {
		return notesEditFlags{}, apperr.CLIUsage("edit requires --append, --body-file, or --set", nil)
	}
	if appendText != "" && bodyFile != "" {
		return notesEditFlags{}, apperr.CLIUsage("--append and --body-file cannot be combined", nil)
	}
	return notesEditFlags{appendText: appendText, bodyFile: bodyFile, ifMatch: ifMatch, setFields: setFields}, nil
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
