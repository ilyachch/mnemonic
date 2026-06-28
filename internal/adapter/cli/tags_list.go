package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newTagsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tags",
		Short: "Manage tags",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
}

func newTagsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tags",
		RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := runtimeAppForSelectedProject(cmd)
		if err != nil {
			return err
		}

		if err = requireRuntimeSearchIndex(runtime); err != nil {
			return err
		}

		tags, err := runtime.Services.Search.ListTags(commandContext(cmd))
		if err != nil {
			return err
		}

		output := tagsListOutput{Tags: make([]tagsListItem, 0, len(tags.Tags))}
		for _, tag := range tags.Tags {
			output.Tags = append(output.Tags, tagsListItem{Tag: tag.Tag, Count: tag.Count})
		}
		if output.Tags == nil {
			output.Tags = []tagsListItem{}
		}

		human := formatTagsListHuman(output.Tags)
		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
	},
	}
}

type tagsListOutput struct {
	Tags []tagsListItem `json:"tags"`
}

type tagsListItem struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
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
