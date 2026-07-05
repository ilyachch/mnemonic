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
	cmd.Flags().StringArray("set-tags", nil, "set the tags list")
	cmd.Flags().StringArray("set-aliases", nil, "set the aliases list")
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
	if len(parsed.setTags) > 0 {
		tags := parsed.setTags
		editInput.Tags = &tags
	}
	if len(parsed.setAliases) > 0 {
		aliases := parsed.setAliases
		editInput.Aliases = &aliases
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
	setTags    []string
	setAliases []string
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
	setFields, err := parseSetFieldsFlag(cmd)
	if err != nil {
		return notesEditFlags{}, err
	}
	setTags, err := cmd.Flags().GetStringArray("set-tags")
	if err != nil {
		return notesEditFlags{}, err
	}
	setAliases, err := cmd.Flags().GetStringArray("set-aliases")
	if err != nil {
		return notesEditFlags{}, err
	}
	return validateEditFlags(appendText, bodyFile, setFields, setTags, setAliases, ifMatch)
}

func parseSetFieldsFlag(cmd *cobra.Command) (map[string]string, error) {
	if !cmd.Flags().Changed("set") {
		return nil, nil
	}
	setValues, err := cmd.Flags().GetStringArray("set")
	if err != nil {
		return nil, err
	}
	return parseEditSetValues(setValues)
}

func validateEditFlags(appendText, bodyFile string, setFields map[string]string, setTags, setAliases []string, ifMatch string) (notesEditFlags, error) {
	hasContent := appendText != "" || bodyFile != "" || len(setFields) > 0 || len(setTags) > 0 || len(setAliases) > 0
	if !hasContent {
		return notesEditFlags{}, apperr.CLIUsage("edit requires --append, --body-file, --set, --set-tags, or --set-aliases", nil)
	}
	if appendText != "" && bodyFile != "" {
		return notesEditFlags{}, apperr.CLIUsage("--append and --body-file cannot be combined", nil)
	}
	return notesEditFlags{appendText: appendText, bodyFile: bodyFile, ifMatch: ifMatch, setFields: setFields, setTags: setTags, setAliases: setAliases}, nil
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
