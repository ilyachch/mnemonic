package cli

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/search"
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

		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		entry, err := registry.Resolve(container.Paths.MemoriesHome, projectSelectorValue())
		if err != nil {
			return err
		}

		indexPath, err := index.Path(entry.ProjectID)
		if err != nil {
			return err
		}
		if _, err := os.Stat(indexPath); err != nil {
			if os.IsNotExist(err) {
				return app.NewNotFoundError("index missing; run `mnemonic project reindex`", nil)
			}
			return fmt.Errorf("stat index %q: %w", indexPath, err)
		}

		searchDB, err := sql.Open("sqlite", indexPath)
		if err != nil {
			return err
		}
		defer func() { _ = searchDB.Close() }()

		hits, err := search.Search(searchDB, args[0], limit, tagFilter)
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
