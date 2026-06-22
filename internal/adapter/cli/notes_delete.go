package cli

import (
	"fmt"

	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/spf13/cobra"
)

var notesDeleteCmd = &cobra.Command{
	Use:   "delete SELECTOR",
	Short: "Delete a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		hard, err := cmd.Flags().GetBool("hard")
		if err != nil {
			return err
		}
		yes, err := cmd.Flags().GetBool("yes")
		if err != nil {
			return err
		}

		runtime, err := runtimeAppForSelectedProject(cmd)
		if err != nil {
			return err
		}

		deleted, err := runtime.Services.Notes.Delete(notesvc.DeleteInput{
			Selector: args[0],
			DryRun:   dryRun,
			Hard:     hard,
			Yes:      yes,
		})
		if err != nil {
			return err
		}

		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), fmt.Sprintf("%s deleted\n", args[0]), deleted)
	},
}
