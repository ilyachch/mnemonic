package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/spf13/cobra"
)

var projectRemoveCmd = &cobra.Command{
	Use:   "remove NAME_OR_UUID",
	Short: "Remove a registered project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deleteMarkdown, err := cmd.Flags().GetBool("delete-markdown")
		if err != nil {
			return err
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

		removedAt := project.Timestamp()
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin registry removal: %w", err)
		}
		if _, err := tx.Exec(`UPDATE projects SET removed_at = ? WHERE project_id = ? AND removed_at IS NULL`, removedAt, projectRecord.ProjectID); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("mark project removed: %w", err)
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("commit registry removal: %w", err)
		}

		stateDir := filepath.Join(container.Paths.StateHome, "mnemonic", "projects", projectRecord.ProjectID)
		if err := os.RemoveAll(stateDir); err != nil {
			return fmt.Errorf("remove project state directory: %w", err)
		}

		markdownDeleted := false
		if deleteMarkdown && projectRecord.Location.memoriesAbs != "" {
			if err := os.RemoveAll(projectRecord.Location.memoriesAbs); err != nil {
				return fmt.Errorf("remove project markdown directory: %w", err)
			}
			markdownDeleted = true
		}

		output := projectRemoveOutput{
			ProjectID:       projectRecord.ProjectID,
			Slug:            projectRecord.Slug,
			Removed:         true,
			MarkdownDeleted: markdownDeleted,
			StateDeleted:    true,
		}
		return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s removed\n", projectRecord.Slug), output)
	},
}

func init() {
	projectRemoveCmd.Flags().Bool("delete-markdown", false, "delete the markdown directory as well")
	projectCmd.AddCommand(projectRemoveCmd)
}

type projectRemoveOutput struct {
	ProjectID       string `json:"project_id"`
	Slug            string `json:"slug"`
	Removed         bool   `json:"removed"`
	MarkdownDeleted bool   `json:"markdown_deleted"`
	StateDeleted    bool   `json:"state_deleted"`
}
