package cli

import (
	"fmt"
	"strings"

	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
	"github.com/spf13/cobra"
)

func newProjectSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync PROJECT_SELECTOR [FILES...]",
		Short: "Reconcile metadata for new or updated notes",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runProjectSync,
	}
	cmd.Flags().Bool("dry-run", false, "Show what would be synced without making changes")
	return cmd
}

func runProjectSync(cmd *cobra.Command, args []string) error {
	selector := args[0]
	files := args[1:]

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	runtime, err := runtimeAppForSelector(commandContext(cmd), selector)
	if err != nil {
		return err
	}

	hydrateResult, err := runtime.Services.Notes.Hydrate(markdownstore.HydrateInput{
		Files:  files,
		DryRun: dryRun,
	})
	if err != nil {
		return err
	}

	output := projectSyncOutput{
		ProjectID:   runtime.KB.ID,
		Slug:        runtime.KB.Slug,
		Hydrated:    hydrateResult.Hydrated,
		Skipped:     hydrateResult.Skipped,
		DryRun:      hydrateResult.DryRun,
		IndexStatus: "skipped",
	}

	if !dryRun {
		rebuild, rebuildErr := runtime.Services.Index.Rebuild(commandContext(cmd))
		if rebuildErr != nil {
			output.IndexStatus = "stale"
			output.IndexError = rebuildErr.Error()
		} else {
			output.Indexed = rebuild.NotesIndexed
			output.IndexStatus = "ok"
		}
	}

	if !jsonOutputEnabled(cmd) {
		printIndexWarning(cmd, output.IndexStatus, output.IndexError)
	}
	human := formatSyncHuman(output, dryRun)
	return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
}

func formatSyncHuman(output projectSyncOutput, dryRun bool) string {
	verb := "synced"
	if dryRun {
		verb = "would be synced"
	}
	human := fmt.Sprintf("%d note(s) %s for %s\n", len(output.Hydrated), verb, output.Slug)
	var humanSb77 strings.Builder
	for _, note := range output.Hydrated {
		prefix := "hydrated"
		if dryRun {
			prefix = "would hydrate"
		}
		fmt.Fprintf(&humanSb77, "%s %s (title=%q slug=%q id=%s)\n", prefix, note.Path, note.Title, note.Slug, note.NoteID)
	}
	human += humanSb77.String()
	var humanSb84 strings.Builder
	for _, path := range output.Skipped {
		fmt.Fprintf(&humanSb84, "skipped %s (already has mnemonic_note_id)\n", path)
	}
	human += humanSb84.String()
	return human
}

type projectSyncOutput struct {
	ProjectID   string                       `json:"project_id"`
	Slug        string                       `json:"slug"`
	Hydrated    []markdownstore.HydratedNote `json:"hydrated,omitempty"`
	Skipped     []string                     `json:"skipped,omitempty"`
	DryRun      bool                         `json:"dry_run"`
	Indexed     int                          `json:"indexed"`
	IndexStatus string                       `json:"index_status"`
	IndexError  string                       `json:"index_error,omitempty"`
}
