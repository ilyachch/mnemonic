package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var projectRemoveCmd = &cobra.Command{
	Use:   "remove SLUG",
	Short: "Remove a project",
	Long: `By default, removes the registry entry while keeping markdown notes intact.

  remove SLUG
      Delete the registry entry (pointer file or manifest).

  remove --wipe SLUG
      Delete the registry entry and all markdown notes.

Index and lock files are always cleaned up.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		wipe, err := cmd.Flags().GetBool("wipe")
		if err != nil {
			return err
		}

		container, err := bootstrapFromContext(commandContext(cmd))
		if err != nil {
			return err
		}

		result, err := container.Services.Catalog.Remove(args[0], wipe)
		if err != nil {
			return err
		}

		output := projectRemoveOutput{
			ProjectID:       result.ProjectID,
			Slug:            result.Slug,
			RegistryRemoved: result.RegistryRemoved,
			IndexDeleted:    result.IndexDeleted,
			MarkdownDeleted: result.MarkdownDeleted,
			FullWipe:        result.FullWipe,
		}
		human := fmt.Sprintf("%s removed\n", result.Slug)
		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
	},
}

type projectRemoveOutput struct {
	ProjectID       string `json:"project_id"`
	Slug            string `json:"slug"`
	RegistryRemoved bool   `json:"registry_removed"`
	IndexDeleted    bool   `json:"index_deleted"`
	MarkdownDeleted bool   `json:"markdown_deleted"`
	FullWipe        bool   `json:"full_wipe"`
}
