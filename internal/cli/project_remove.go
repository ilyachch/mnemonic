package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
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

		_, err = mustAppContainer()
		if err != nil {
			return err
		}

		runtime, err := runtimeAppForSelector(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		// Remove registry entry.
		registryRemoved := false
		if runtime.KB.ManifestPath != "" {
			if err := os.Remove(runtime.KB.ManifestPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove registry entry %q: %w", runtime.KB.ManifestPath, err)
			}
			registryRemoved = true
		}

		// Remove index and lock artifacts.
		indexDeleted := false
		if runtime.KB.IndexPath != "" {
			for _, p := range []string{runtime.KB.IndexPath, runtime.KB.IndexPath + "-wal", runtime.KB.IndexPath + "-shm"} {
				if _, err := os.Stat(p); err == nil {
					_ = os.Remove(p)
					indexDeleted = true
				}
			}
			_ = os.RemoveAll(runtime.KB.StateDir)
		}

		// Wipe markdown if requested.
		markdownDeleted := false
		if wipe && runtime.KB.RootDir != "" {
			if err := os.RemoveAll(runtime.KB.RootDir); err != nil {
				return fmt.Errorf("wipe markdown: %w", err)
			}
			markdownDeleted = true
		}

		output := projectRemoveOutput{
			ProjectID:       runtime.KB.ID,
			Slug:            runtime.KB.Slug,
			RegistryRemoved: registryRemoved,
			IndexDeleted:    indexDeleted,
			MarkdownDeleted: markdownDeleted,
			FullWipe:        wipe,
		}
		human := fmt.Sprintf("%s removed\n", runtime.KB.Slug)
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	projectRemoveCmd.Flags().Bool("wipe", false, "also remove all markdown notes")
	projectCmd.AddCommand(projectRemoveCmd)
}

type projectRemoveOutput struct {
	ProjectID       string `json:"project_id"`
	Slug            string `json:"slug"`
	RegistryRemoved bool   `json:"registry_removed"`
	IndexDeleted    bool   `json:"index_deleted"`
	MarkdownDeleted bool   `json:"markdown_deleted"`
	FullWipe        bool   `json:"full_wipe"`
}
