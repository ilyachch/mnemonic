package cli

import (
	"fmt"

	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/spf13/cobra"
)

var projectImportCmd = &cobra.Command{
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

		result, err := project.ImportProject(project.ImportInput{Path: pathArg, DryRun: dryRun})
		if err != nil {
			return err
		}
		if !dryRun && len(result.Candidates) > 0 {
			container, err := mustAppContainer()
			if err != nil {
				return err
			}
			for _, candidate := range result.Candidates {
				if err := buildProjectIndex(container.Services.Registry, candidate.ProjectID, candidate.MemoriesPath); err != nil {
					return err
				}
				result.Indexed++
			}
		}

		output := projectImportOutput{
			Path:        result.Path,
			Imported:    result.Imported,
			CopiedFiles: result.CopiedFiles,
			Indexed:     result.Indexed,
			Candidates:  result.Candidates,
		}
		human := fmt.Sprintf("%d project(s) imported\n", result.Imported)
		if dryRun {
			human = fmt.Sprintf("%d project(s) would be imported\n", result.Imported)
		}
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	projectCmd.AddCommand(projectImportCmd)
	projectImportCmd.Flags().Bool("dry-run", false, "Show what would be imported without making changes")
}

type projectImportOutput struct {
	Path        string                    `json:"path"`
	Imported    int                       `json:"imported"`
	CopiedFiles int                       `json:"copied_files"`
	Indexed     int                       `json:"indexed"`
	Candidates  []project.ImportCandidate `json:"candidates,omitempty"`
}
