package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var projectReindexCmd = &cobra.Command{
	Use:               "reindex [PROJECT]",
	Short:             "Rebuild project indexes",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}

		selector, err := resolveProjectSelector(cmd, args, all)
		if err != nil {
			return err
		}

		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		switch {
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
			result, err := reindexSingleProject(cmd.Context(), selector)
			if err != nil {
				return err
			}
			return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s reindexed\n", result.ProjectID), result)
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

func reindexSingleProject(ctx context.Context, selector string) (projectReindexProjectResult, error) {
	runtime, err := runtimeAppForSelector(ctx, selector)
	if err != nil {
		return projectReindexProjectResult{}, err
	}

	indexService := runtime.Services.Index
	if indexService == nil {
		return projectReindexProjectResult{}, fmt.Errorf("runtime index service is not configured")
	}

	result, err := indexService.Rebuild(ctx)
	if err != nil {
		return projectReindexProjectResult{}, err
	}

	return projectReindexProjectResult{
		ProjectID:    result.KBID,
		Slug:         runtime.KB.Slug,
		NotesSeen:    result.NotesSeen,
		NotesIndexed: result.NotesIndexed,
		Status:       result.Status,
	}, nil
}
