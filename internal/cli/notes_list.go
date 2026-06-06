package cli

import (
	"fmt"

	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/spf13/cobra"
)

var notesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Manage notes",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var notesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectSelector, err := cmd.Flags().GetString("project")
		if err != nil {
			return err
		}

		root, err := resolveNotesProjectRoot(projectSelector)
		if err != nil {
			return err
		}

		list, err := notes.List(root)
		if err != nil {
			return err
		}

		output := notesListOutput{Notes: list}
		human := fmt.Sprintf("%d notes\n", len(list))
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	notesListCmd.Flags().String("project", "", "select a project")
	notesCmd.AddCommand(notesListCmd)
	RootCmd.AddCommand(notesCmd)
}

type notesListOutput struct {
	Notes []notes.NoteSummary `json:"notes"`
}
