package cli

import (
	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/spf13/cobra"
)

func newNotesDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
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

		if !jsonOutputEnabled(cmd) {
			printIndexWarning(cmd, deleted.IndexStatus, deleted.IndexError)
		}
		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), args[0]+" deleted\n", newNotesDeleteOutput(deleted))
	},
	}
	cmd.Flags().Bool("dry-run", false, "show what would be deleted without making changes")
	cmd.Flags().Bool("hard", false, "delete the note file instead of moving it to trash")
	cmd.Flags().Bool("yes", false, "confirm deletion without prompting")
	return cmd
}
