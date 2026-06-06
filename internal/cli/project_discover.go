package cli

import (
	"fmt"

	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/spf13/cobra"
)

var projectDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover projects from mnemonic.toml files",
	RunE: func(cmd *cobra.Command, args []string) error {
		effectivePaths, err := paths.ResolveEffectivePaths(paths.EffectiveInput{})
		if err != nil {
			return err
		}

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}

		result, err := project.DiscoverProjects(project.DiscoverInput{
			MemoriesHome: effectivePaths.MemoriesHome,
			DryRun:       dryRun,
		})
		if err != nil {
			return err
		}

		output := projectDiscoverOutput{
			MemoriesHome: result.MemoriesHome,
			Discovered:   result.Discovered,
			Errors:       result.Errors,
		}

		human := fmt.Sprintf("%d project(s) discovered\n", result.Discovered)
		if dryRun {
			human = fmt.Sprintf("%d project(s) would be discovered\n", result.Discovered)
			if len(result.Errors) > 0 {
				for i := range result.Errors {
					issue := result.Errors[i]
					human += fmt.Sprintf("skipped %s: %s\n", issue.Path, issue.Error)
				}
			}
		}
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	projectCmd.AddCommand(projectDiscoverCmd)
	projectDiscoverCmd.Flags().Bool("dry-run", false, "Show what would be discovered without making changes")
}

type projectDiscoverOutput struct {
	MemoriesHome string                  `json:"memories_home"`
	Discovered   int                     `json:"discovered"`
	Errors       []project.DiscoverIssue `json:"errors,omitempty"`
}
