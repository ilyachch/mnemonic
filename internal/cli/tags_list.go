package cli

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Manage tags",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var tagsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		resolvedProject, err := container.Services.ProjectResolver.Resolve(app.ProjectResolveInput{
			ProjectSelector:  projectSelectorValue(),
			EnvironmentValue: os.Getenv(project.EnvironmentProjectSelector),
		})
		if err != nil {
			return err
		}

		indexPath, err := index.Path(resolvedProject.Project.ID)
		if err != nil {
			return err
		}
		if _, err := os.Stat(indexPath); err != nil {
			if os.IsNotExist(err) {
				return app.NewNotFoundError("index missing; run `mnemonic project reindex`", nil)
			}
			return fmt.Errorf("stat index %q: %w", indexPath, err)
		}

		db, err := sql.Open("sqlite", indexPath)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		tags, err := listTags(db)
		if err != nil {
			return err
		}

		output := tagsListOutput{Tags: tags}
		if output.Tags == nil {
			output.Tags = []tagsListItem{}
		}

		human := formatTagsListHuman(output.Tags)
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	tagsCmd.AddCommand(tagsListCmd)
	RootCmd.AddCommand(tagsCmd)
}

type tagsListOutput struct {
	Tags []tagsListItem `json:"tags"`
}

type tagsListItem struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func listTags(db *sql.DB) ([]tagsListItem, error) {
	rows, err := db.Query(
		`SELECT
			CASE
				WHEN instr(tag, ':') > 0 THEN substr(tag, instr(tag, ':') + 1)
				ELSE tag
			END AS tag_name,
			COUNT(DISTINCT note_id) AS count
		 FROM note_tags
		 GROUP BY tag_name
		 ORDER BY count DESC, tag_name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query tag list: %w", err)
	}
	defer rows.Close()

	tags := make([]tagsListItem, 0)
	for rows.Next() {
		var item tagsListItem
		if err := rows.Scan(&item.Tag, &item.Count); err != nil {
			return nil, fmt.Errorf("scan tag row: %w", err)
		}
		tags = append(tags, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tag rows: %w", err)
	}

	return tags, nil
}

func formatTagsListHuman(tags []tagsListItem) string {
	if len(tags) == 0 {
		return "0 tags\n"
	}
	var b strings.Builder
	for _, tag := range tags {
		fmt.Fprintf(&b, "%s %d\n", tag.Tag, tag.Count)
	}
	return b.String()
}
