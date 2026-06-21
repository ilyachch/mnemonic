package cli

import (
	"strconv"

	"github.com/ilyachch/mnemonic/internal/search"
	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
	"github.com/spf13/cobra"
)

var notesSearchCmd = &cobra.Command{
	Use:   "search QUERY",
	Short: "Search notes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, err := cmd.Flags().GetInt("limit")
		if err != nil {
			return err
		}
		tagFilter, err := cmd.Flags().GetString("tag")
		if err != nil {
			return err
		}

		runtime, err := runtimeAppForSelectedProject(cmd.Context())
		if err != nil {
			return err
		}

		if err := requireRuntimeSearchIndex(runtime); err != nil {
			return err
		}

		hits, err := runtime.Services.Search.Search(cmd.Context(), searchsvc.SearchInput{
			Query: args[0],
			Limit: limit,
			Tag:   tagFilter,
		})
		if err != nil {
			return err
		}

		output := notesSearchOutput{Hits: hits}
		human := formatNotesSearchHuman(hits)
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	notesSearchCmd.Flags().String("tag", "", "filter by tag")
	notesSearchCmd.Flags().Int("limit", 20, "maximum number of hits")
	notesCmd.AddCommand(notesSearchCmd)
}

type notesSearchOutput struct {
	Hits []search.Result `json:"hits"`
}

func formatNotesSearchHuman(hits []search.Result) string {
	if len(hits) == 0 {
		return "0 hits\n"
	}
	out := ""
	for _, hit := range hits {
		out += hit.Slug + " " + strconv.FormatFloat(hit.Score, 'f', 3, 64) + " " + hit.Path + "\n"
	}
	return out
}
