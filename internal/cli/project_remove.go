package cli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/spf13/cobra"
)

type projectRemoveMode string

const (
	projectRemoveModeSoft         projectRemoveMode = "soft"
	projectRemoveModeHard         projectRemoveMode = "hard"
	projectRemoveModeMarkdownOnly projectRemoveMode = "markdown-only"
	projectRemoveModeWipe         projectRemoveMode = "wipe"
)

var projectRemoveCmd = &cobra.Command{
	Use:   "remove NAME_OR_UUID",
	Short: "Remove a project with selectable cleanup modes",
	Long: `By default, remove only soft-deletes the project from the registry.

Modes:
  remove NAME_OR_UUID
      Soft-delete in registry only.

  remove --hard NAME_OR_UUID
      Soft-delete + remove index/state marker artifacts.

  remove --delete-markdown NAME_OR_UUID
      Remove markdown + index/state marker artifacts, but keep registry active.

  remove --wipe NAME_OR_UUID
      Remove everything: soft-delete in registry + index/state marker artifacts + markdown.

Note:
  --hard and --delete-markdown are mutually exclusive.
  Use --wipe when you want full cleanup in one command.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		hard, err := cmd.Flags().GetBool("hard")
		if err != nil {
			return err
		}
		deleteMarkdown, err := cmd.Flags().GetBool("delete-markdown")
		if err != nil {
			return err
		}
		wipe, err := cmd.Flags().GetBool("wipe")
		if err != nil {
			return err
		}

		if hard && deleteMarkdown {
			return app.NewCLIUsageError("--hard and --delete-markdown cannot be combined; use --wipe for full cleanup", nil)
		}
		if wipe && (hard || deleteMarkdown) {
			return app.NewCLIUsageError("--wipe cannot be combined with --hard or --delete-markdown", nil)
		}

		mode := projectRemoveModeSoft
		switch {
		case wipe:
			mode = projectRemoveModeWipe
		case hard:
			mode = projectRemoveModeHard
		case deleteMarkdown:
			mode = projectRemoveModeMarkdownOnly
		}

		container, err := mustAppContainer()
		if err != nil {
			return err
		}
		db := container.Services.Registry

		projectRecord, err := queryProjectBySelector(args[0])
		if err != nil {
			return err
		}

		registryRemoved := false
		if mode == projectRemoveModeSoft || mode == projectRemoveModeHard || mode == projectRemoveModeWipe {
			if err := softDeleteProject(db, projectRecord.ProjectID); err != nil {
				return err
			}
			registryRemoved = true
		}

		indexDeleted := false
		stateMarkersDeleted := false
		if mode == projectRemoveModeHard || mode == projectRemoveModeMarkdownOnly || mode == projectRemoveModeWipe {
			indexDeleted, stateMarkersDeleted, err = removeProjectStateArtifacts(container.Paths.StateHome, projectRecord.ProjectID)
			if err != nil {
				return err
			}
		}

		markdownDeleted := false
		if mode == projectRemoveModeMarkdownOnly || mode == projectRemoveModeWipe {
			markdownDeleted, err = removeMarkdownArtifacts(projectRecord.Location.memoriesAbs)
			if err != nil {
				return err
			}
		}

		output := projectRemoveOutput{
			ProjectID:           projectRecord.ProjectID,
			Slug:                projectRecord.Slug,
			Mode:                string(mode),
			RegistryRemoved:     registryRemoved,
			IndexDeleted:        indexDeleted,
			StateMarkersDeleted: stateMarkersDeleted,
			MarkdownDeleted:     markdownDeleted,
			FullWipe:            mode == projectRemoveModeWipe,
		}
		human := fmt.Sprintf("%s removed (%s)\n", projectRecord.Slug, mode)
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	projectRemoveCmd.Flags().Bool("hard", false, "soft-delete and remove index/state marker artifacts")
	projectRemoveCmd.Flags().Bool("delete-markdown", false, "remove markdown plus index/state marker artifacts, keep registry active")
	projectRemoveCmd.Flags().Bool("wipe", false, "remove everything (registry, index/state marker artifacts, markdown)")
	projectCmd.AddCommand(projectRemoveCmd)
}

type projectRemoveOutput struct {
	ProjectID           string `json:"project_id"`
	Slug                string `json:"slug"`
	Mode                string `json:"mode"`
	RegistryRemoved     bool   `json:"registry_removed"`
	IndexDeleted        bool   `json:"index_deleted"`
	StateMarkersDeleted bool   `json:"state_markers_deleted"`
	MarkdownDeleted     bool   `json:"markdown_deleted"`
	FullWipe            bool   `json:"full_wipe"`
}

func softDeleteProject(db *sql.DB, projectID string) error {
	removedAt := project.Timestamp()
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin registry removal: %w", err)
	}
	if _, err := tx.Exec(`UPDATE projects SET removed_at = ? WHERE project_id = ? AND removed_at IS NULL`, removedAt, projectID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("mark project removed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("commit registry removal: %w", err)
	}
	return nil
}

func removeProjectStateArtifacts(stateHome, projectID string) (bool, bool, error) {
	indexPath, err := index.Path(projectID)
	if err != nil {
		return false, false, err
	}

	indexDeleted := false
	for _, path := range []string{indexPath, indexPath + "-wal", indexPath + "-shm"} {
		deleted, err := removePathIfExists(path)
		if err != nil {
			return false, false, fmt.Errorf("remove index artifact %q: %w", path, err)
		}
		if deleted {
			indexDeleted = true
		}
	}

	statePath := filepath.Join(stateHome, "mnemonic", "projects", projectID, "state.toml")
	stateMarkersDeleted, err := removePathIfExists(statePath)
	if err != nil {
		return false, false, fmt.Errorf("remove state marker %q: %w", statePath, err)
	}

	return indexDeleted, stateMarkersDeleted, nil
}

func removeMarkdownArtifacts(memoriesPath string) (bool, error) {
	if memoriesPath == "" {
		return false, nil
	}

	if _, err := os.Stat(memoriesPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat markdown path %q: %w", memoriesPath, err)
	}

	if err := os.RemoveAll(memoriesPath); err != nil {
		return false, fmt.Errorf("remove project markdown directory: %w", err)
	}

	return true, nil
}

func removePathIfExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if err := os.Remove(path); err != nil {
		return false, err
	}

	return true, nil
}
