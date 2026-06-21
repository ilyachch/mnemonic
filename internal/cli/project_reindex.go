package cli

import (
	"fmt"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/spf13/cobra"
)

var projectReindexCmd = &cobra.Command{
	Use:               "reindex [SLUG]",
	Short:             "Rebuild project indexes",
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}

		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		switch {
		case all && len(args) > 0:
			return fmt.Errorf("--all cannot be combined with a project selector")
		case len(args) == 1:
			result, err := reindexSingleProject(container.Paths.MemoriesHome, args[0])
			if err != nil {
				return err
			}
			return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s reindexed\n", result.ProjectID), result)
		case all:
			if container.Services.Maint == nil {
				return fmt.Errorf("maintenance service is not configured")
			}
			result, err := container.Services.Maint.ReindexAll(cmd.Context())
			if err != nil {
				return err
			}
			return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%d projects reindexed\n", result.Indexed), result)
		default:
			if container.Services.Maint == nil {
				return fmt.Errorf("maintenance service is not configured")
			}
			result, err := container.Services.Maint.ReindexAll(cmd.Context())
			if err != nil {
				return err
			}
			return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%d projects reindexed\n", result.Indexed), result)
		}
	},
}

func init() {
	projectReindexCmd.Flags().Bool("all", false, "reindex all active projects")
	projectCmd.AddCommand(projectReindexCmd)
}

type projectReindexProjectResult struct {
	ProjectID    string `json:"project_id"`
	Slug         string `json:"slug"`
	NotesSeen    int    `json:"notes_seen"`
	NotesIndexed int    `json:"notes_indexed"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

func reindexSingleProject(memoriesHome, selector string) (projectReindexProjectResult, error) {
	entry, err := registry.Resolve(memoriesHome, selector)
	if err != nil {
		return projectReindexProjectResult{}, err
	}

	if entry.MemoriesAbs == "" {
		return projectReindexProjectResult{}, fmt.Errorf("no memories path for project %q", selector)
	}

	result, err := index.RebuildProjectIndex(entry.ProjectID, entry.MemoriesAbs)
	if err != nil {
		return projectReindexProjectResult{}, err
	}

	return projectReindexProjectResult{
		ProjectID:    result.ProjectID,
		Slug:         entry.Slug,
		NotesSeen:    result.NotesSeen,
		NotesIndexed: result.NotesIndexed,
		Status:       result.Status,
	}, nil
}
