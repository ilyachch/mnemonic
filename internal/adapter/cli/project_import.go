package cli

import (
	"fmt"
	"strings"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/spf13/cobra"
)

func newProjectImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [PATH]",
		Short: "Import a project from a path",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

		result, err := container.Services.Catalog.Import(commandContext(cmd), catalogsvc.ImportInput{Path: pathArg, DryRun: dryRun})
		if err != nil {
			return wrapImportError(err)
		}

		if !jsonOutputEnabled(cmd) {
			printIndexWarnings(cmd, result.IndexErrors)
		}
		output := projectImportOutput{
			Path:        result.Path,
			Imported:    result.Imported,
			CopiedFiles: result.CopiedFiles,
			Indexed:     result.Indexed,
			IndexStatus: result.IndexStatus,
			IndexErrors: result.IndexErrors,
			Candidates:  result.Candidates,
		}
		human := fmt.Sprintf("%d project(s) imported\n", result.Imported)
		if dryRun {
			human = fmt.Sprintf("%d project(s) would be imported\n", result.Imported)
		}
		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
	},
	}
	cmd.Flags().Bool("dry-run", false, "Show what would be imported without making changes")
	return cmd
}

func wrapImportError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "import path") && strings.Contains(msg, "not found"):
		return apperr.NotFound(msg, nil)
	case strings.HasPrefix(msg, "mnemonic.toml not found at"):
		return apperr.NotFound(msg, nil)
	case strings.HasPrefix(msg, "project slug") && strings.Contains(msg, "already exists"):
		return apperr.Ambiguous(msg, nil)
	}
	return err
}

type projectImportOutput struct {
	Path        string                        `json:"path"`
	Imported    int                           `json:"imported"`
	CopiedFiles int                           `json:"copied_files"`
	Indexed     int                           `json:"indexed"`
	IndexStatus string                        `json:"index_status"`
	IndexErrors []catalogsvc.ImportIndexError `json:"index_errors,omitempty"`
	Candidates  []catalogsvc.ImportCandidate  `json:"candidates,omitempty"`
}
