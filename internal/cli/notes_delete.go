package cli

import (
	"fmt"

	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/spf13/cobra"
)

var notesDeleteCmd = &cobra.Command{
	Use:   "delete SELECTOR",
	Short: "Delete a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectSelector, err := cmd.Flags().GetString("project")
		if err != nil {
			return err
		}
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

		root, err := resolveNotesProjectRoot(projectSelector)
		if err != nil {
			return err
		}

		deleted, err := notes.Delete(notes.DeleteInput{
			RootDir:  root,
			Selector: args[0],
			DryRun:   dryRun,
			Hard:     hard,
			Yes:      yes,
		})
		if err != nil {
			return err
		}

		return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s deleted\n", args[0]), deleted)
	},
}

func init() {
	notesDeleteCmd.Flags().String("project", "", "select a project")
	mustRegisterProjectFlagCompletion(notesDeleteCmd, "project")
	notesDeleteCmd.Flags().Bool("dry-run", false, "show the deletion result without changing files")
	notesDeleteCmd.Flags().Bool("hard", false, "delete the note permanently")
	notesDeleteCmd.Flags().Bool("yes", false, "confirm a hard delete")
	notesCmd.AddCommand(notesDeleteCmd)
}
