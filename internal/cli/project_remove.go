package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/registry"
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

		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		entry, err := registry.Resolve(container.Paths.MemoriesHome, args[0])
		if err != nil {
			if _, ok := err.(registry.ErrNotFound); ok {
				return apperr.NotFound(err.Error(), nil)
			}
			return err
		}

		// Remove registry entry.
		registryRemoved := false
		registryPath := registryPathForEntry(container.Paths.MemoriesHome, entry)
		if registryPath != "" {
			if entry.Type == "central" {
				if err := os.Remove(registryPath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("remove registry entry %q: %w", registryPath, err)
				}
			} else if err := os.Remove(registryPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove registry entry %q: %w", registryPath, err)
			}
			registryRemoved = true
		}

		// Remove index and lock artifacts.
		indexDeleted := false
		if entry.ProjectID != "" {
			idxPath, err := index.Path(entry.ProjectID)
			if err == nil {
				for _, p := range []string{idxPath, idxPath + "-wal", idxPath + "-shm"} {
					if _, err := os.Stat(p); err == nil {
						_ = os.Remove(p)
						indexDeleted = true
					}
				}
			}

			// Remove the entire state directory.
			stateDir := filepath.Join(container.Paths.StateHome, "mnemonic", "projects", entry.ProjectID)
			_ = os.RemoveAll(stateDir)
		}

		// Wipe markdown if requested.
		markdownDeleted := false
		if wipe && entry.MemoriesAbs != "" {
			if err := os.RemoveAll(entry.MemoriesAbs); err != nil {
				return fmt.Errorf("wipe markdown: %w", err)
			}
			markdownDeleted = true
		}

		output := projectRemoveOutput{
			ProjectID:       entry.ProjectID,
			Slug:            entry.Slug,
			RegistryRemoved: registryRemoved,
			IndexDeleted:    indexDeleted,
			MarkdownDeleted: markdownDeleted,
			FullWipe:        wipe,
		}
		human := fmt.Sprintf("%s removed\n", entry.Slug)
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

func registryPathForEntry(memoriesHome string, entry registry.Entry) string {
	switch entry.Type {
	case "central":
		return filepath.Join(memoriesHome, entry.Slug, "mnemonic.toml")
	case "local":
		return filepath.Join(memoriesHome, entry.Slug+".toml")
	default:
		return ""
	}
}
