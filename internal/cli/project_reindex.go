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
		case len(args) == 1:
			result, err := reindexSingleProject(container.Paths.MemoriesHome, args[0])
			if err != nil {
				return err
			}
			return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s reindexed\n", result.ProjectID), result)
		case all:
			result, err := reindexAllProjects(container.Paths.MemoriesHome)
			if err != nil {
				return err
			}
			return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%d projects reindexed\n", result.Indexed), result)
		default:
			result, err := reindexAllProjects(container.Paths.MemoriesHome)
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

type projectReindexSummary struct {
	Indexed  int                           `json:"indexed"`
	Skipped  int                           `json:"skipped,omitempty"`
	Projects []projectReindexProjectResult `json:"projects,omitempty"`
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

func reindexAllProjects(memoriesHome string) (projectReindexSummary, error) {
	entries, _, err := registry.Scan(memoriesHome)
	if err != nil {
		return projectReindexSummary{}, err
	}

	var summary projectReindexSummary
	for _, entry := range entries {
		if entry.ProjectID == "" || entry.MemoriesAbs == "" {
			summary.Skipped++
			continue
		}

		result, err := index.RebuildProjectIndex(entry.ProjectID, entry.MemoriesAbs)
		if err != nil {
			summary.Projects = append(summary.Projects, projectReindexProjectResult{
				ProjectID: entry.ProjectID,
				Slug:      entry.Slug,
				Status:    "error",
				Error:     err.Error(),
			})
			continue
		}

		summary.Indexed++
		summary.Projects = append(summary.Projects, projectReindexProjectResult{
			ProjectID:    result.ProjectID,
			Slug:         entry.Slug,
			NotesSeen:    result.NotesSeen,
			NotesIndexed: result.NotesIndexed,
			Status:       result.Status,
		})
	}
	return summary, nil
}
