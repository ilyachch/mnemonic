package cli

import (
	"fmt"

	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/spf13/cobra"
)

func newNotesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "notes",
		Short: "Manage notes",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
}

func newNotesListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List notes",
		RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := runtimeAppForSelectedProject(cmd)
		if err != nil {
			return err
		}

		list, err := runtime.Services.Notes.List()
		if err != nil {
			return err
		}

		output := notesListOutput{Notes: list}
		human := fmt.Sprintf("%d notes\n", len(list))
		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
	},
	}
}

type notesListOutput struct {
	Notes []notesvc.NoteSummary `json:"notes"`
}
