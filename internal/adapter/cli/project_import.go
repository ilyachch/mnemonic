package cli

import (
	"fmt"
	"strings"

	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
	"github.com/spf13/cobra"
)

func newProjectImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [PATH]",
		Short: "Onboard a raw directory of markdown notes",
		Args:  cobra.RangeArgs(0, 1),
		RunE:  runProjectImport,
	}
	cmd.Flags().Bool("dry-run", false, "Show what would be imported without making changes")
	return cmd
}

func runProjectImport(cmd *cobra.Command, args []string) error {
	pathArg := "."
	if len(args) == 1 {
		pathArg = args[0]
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	container, err := bootstrapFromContext(commandContext(cmd))
	if err != nil {
		return err
	}

	result, err := container.Services.Catalog.Import(commandContext(cmd), catalogsvc.ImportInput{Path: pathArg, DryRun: dryRun}, loggerFromContext(commandContext(cmd)))
	if err != nil {
		return err
	}

	if !jsonOutputEnabled(cmd) {
		printIndexWarnings(cmd, result.IndexErrors)
	}
	output := projectImportOutput{
		Path:            result.Path,
		Imported:        result.Imported,
		CopiedFiles:     result.CopiedFiles,
		Indexed:         result.Indexed,
		IndexStatus:     result.IndexStatus,
		IndexErrors:     result.IndexErrors,
		Candidates:      result.Candidates,
		ManifestCreated: result.ManifestCreated,
		Hydrated:        result.Hydrated,
		Skipped:         result.Skipped,
	}
	human := formatImportHuman(result, dryRun)
	return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
}

func formatImportHuman(result catalogsvc.ImportResult, dryRun bool) string {
	verb := "imported"
	if dryRun {
		verb = "would be imported"
	}
	human := fmt.Sprintf("%d project(s) %s\n", result.Imported, verb)
	if result.ManifestCreated {
		if dryRun {
			human += "manifest: would be created\n"
		} else {
			human += "manifest: created\n"
		}
	}
	var humanSb76 strings.Builder
	for _, note := range result.Hydrated {
		prefix := "hydrated"
		if dryRun {
			prefix = "would hydrate"
		}
		fmt.Fprintf(&humanSb76, "%s %s (title=%q slug=%q id=%s)\n", prefix, note.Path, note.Title, note.Slug, note.NoteID)
	}
	human += humanSb76.String()
	var humanSb83 strings.Builder
	for _, path := range result.Skipped {
		fmt.Fprintf(&humanSb83, "skipped %s (already has mnemonic_note_id)\n", path)
	}
	human += humanSb83.String()
	return human
}

type projectImportOutput struct {
	Path            string                        `json:"path"`
	Imported        int                           `json:"imported"`
	CopiedFiles     int                           `json:"copied_files"`
	Indexed         int                           `json:"indexed"`
	IndexStatus     string                        `json:"index_status"`
	IndexErrors     []catalogsvc.ImportIndexError `json:"index_errors,omitempty"`
	Candidates      []catalogsvc.ImportCandidate  `json:"candidates,omitempty"`
	ManifestCreated bool                          `json:"manifest_created"`
	Hydrated        []markdownstore.HydratedNote  `json:"hydrated,omitempty"`
	Skipped         []string                      `json:"skipped,omitempty"`
}
