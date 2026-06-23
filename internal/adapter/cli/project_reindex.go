package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilyachch/mnemonic/internal/apperr"
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

		container, err := bootstrapFromContext(commandContext(cmd))
		if err != nil {
			return err
		}

		switch {
		case all:
			if container.Services.Maint == nil {
				return errors.New("maintenance service is not configured")
			}
			result, err := container.Services.Maint.ReindexAll(commandContext(cmd))
			if err != nil {
				return err
			}
			if err := PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), fmt.Sprintf("%d projects reindexed\n", result.Indexed), result); err != nil {
				return err
			}
			if result.Failed > 0 {
				return apperr.Internal(fmt.Sprintf("%d project(s) failed", result.Failed), nil)
			}
			return nil
		default:
			result, err := reindexSingleProject(commandContext(cmd), selector)
			if err != nil {
				return err
			}
			return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), result.ProjectID+" reindexed\n", result)
		}
	},
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
		return projectReindexProjectResult{}, errors.New("runtime index service is not configured")
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
