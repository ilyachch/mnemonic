package cli

import (
	"fmt"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/graph"
	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
	"github.com/spf13/cobra"
)

var notesBacklinksCmd = &cobra.Command{
	Use:   "backlinks SELECTOR",
	Short: "Show backlinks for a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := runtimeAppForSelectedProject(cmd.Context())
		if err != nil {
			return err
		}

		exists, err := runtime.Services.Search.Index.Exists()
		if err != nil {
			return err
		}
		if !exists {
			return apperr.NotFound("index missing; run `mnemonic project reindex`", nil)
		}

		links, err := runtime.Services.Search.Backlinks(cmd.Context(), searchsvc.BacklinksInput{
			Identifier: args[0],
		})
		if err != nil {
			return err
		}

		output := notesBacklinksOutput{Links: links}
		if output.Links == nil {
			output.Links = []graph.Backlink{}
		}
		human := fmt.Sprintf("%d links\n", len(output.Links))
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	notesCmd.AddCommand(notesBacklinksCmd)
}

type notesBacklinksOutput struct {
	Links []graph.Backlink `json:"links"`
}
