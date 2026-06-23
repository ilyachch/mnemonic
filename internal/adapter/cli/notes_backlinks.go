package cli

import (
	"fmt"

	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
	"github.com/spf13/cobra"
)

var notesBacklinksCmd = &cobra.Command{
	Use:   "backlinks SELECTOR",
	Short: "Show backlinks for a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := runtimeAppForSelectedProject(cmd)
		if err != nil {
			return err
		}

		if err = requireRuntimeSearchIndex(runtime); err != nil {
			return err
		}

		links, err := runtime.Services.Search.Backlinks(commandContext(cmd), searchsvc.BacklinksInput{
			Identifier: args[0],
		})
		if err != nil {
			return err
		}

		output := notesBacklinksOutput{Links: links}
		if output.Links == nil {
			output.Links = []searchsvc.Backlink{}
		}
		human := fmt.Sprintf("%d links\n", len(output.Links))
		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
	},
}

type notesBacklinksOutput struct {
	Links []searchsvc.Backlink `json:"links"`
}
