package cli

import (
	"database/sql"
	"fmt"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/spf13/cobra"
)

var projectReindexCmd = &cobra.Command{
	Use:   "reindex [NAME_OR_UUID]",
	Short: "Rebuild project indexes",
	RunE: func(cmd *cobra.Command, args []string) error {
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}

		container, err := mustAppContainer()
		if err != nil {
			return err
		}
		db := container.Services.Registry
		effectivePaths := container.Paths

		switch {
		case len(args) == 1:
			result, err := reindexSingleProject(db, effectivePaths, args[0])
			if err != nil {
				return err
			}
			return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s reindexed\n", result.ProjectID), result)
		case all:
			result, err := reindexAllProjects(db, effectivePaths)
			if err != nil {
				return err
			}
			return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%d projects reindexed\n", result.Indexed), result)
		default:
			result, err := reindexPendingProjects(db, effectivePaths)
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

func reindexSingleProject(db *sql.DB, effectivePaths paths.EffectivePaths, selector string) (projectReindexProjectResult, error) {
	record, err := queryProjectBySelector(selector)
	if err != nil {
		return projectReindexProjectResult{}, err
	}
	res, err := rebuildProject(record, effectivePaths)
	if err != nil {
		return projectReindexProjectResult{}, err
	}
	if err := updateIndexStatus(db, record.ProjectID, true, false); err != nil {
		return projectReindexProjectResult{}, err
	}
	return res, nil
}

func reindexPendingProjects(db *sql.DB, effectivePaths paths.EffectivePaths) (projectReindexSummary, error) {
	rows, err := db.Query(`SELECT p.project_id, p.name, p.slug, p.kind, l.memories_abs, l.source_kind, s.index_present, s.needs_reindex
		FROM projects p
		JOIN project_locations l ON l.project_id = p.project_id
		JOIN project_status s ON s.project_id = p.project_id
		WHERE p.removed_at IS NULL
		ORDER BY p.slug`)
	if err != nil {
		return projectReindexSummary{}, err
	}
	defer rows.Close()
	var summary projectReindexSummary
	for rows.Next() {
		var record projectLookupResult
		var indexPresent, needsReindex int
		if err := rows.Scan(&record.ProjectID, &record.Name, &record.Slug, &record.Kind, &record.Location.memoriesAbs, &record.Location.sourceKind, &indexPresent, &needsReindex); err != nil {
			return projectReindexSummary{}, err
		}
		if indexPresent == 1 && needsReindex == 0 {
			summary.Skipped++
			continue
		}
		res, err := rebuildProject(record, effectivePaths)
		if err != nil {
			summary.Projects = append(summary.Projects, projectReindexProjectResult{ProjectID: record.ProjectID, Slug: record.Slug, Status: "error", Error: err.Error()})
			continue
		}
		summary.Indexed++
		summary.Projects = append(summary.Projects, res)
		if err := updateIndexStatus(db, record.ProjectID, true, false); err != nil {
			return summary, err
		}
	}
	return summary, rows.Err()
}

func reindexAllProjects(db *sql.DB, effectivePaths paths.EffectivePaths) (projectReindexSummary, error) {
	rows, err := db.Query(`SELECT p.project_id, p.name, p.slug, p.kind, l.memories_abs, l.source_kind
		FROM projects p
		JOIN project_locations l ON l.project_id = p.project_id
		WHERE p.removed_at IS NULL
		ORDER BY p.slug`)
	if err != nil {
		return projectReindexSummary{}, err
	}
	defer rows.Close()
	var summary projectReindexSummary
	for rows.Next() {
		var record projectLookupResult
		if err := rows.Scan(&record.ProjectID, &record.Name, &record.Slug, &record.Kind, &record.Location.memoriesAbs, &record.Location.sourceKind); err != nil {
			return projectReindexSummary{}, err
		}
		res, err := rebuildProject(record, effectivePaths)
		if err != nil {
			summary.Projects = append(summary.Projects, projectReindexProjectResult{ProjectID: record.ProjectID, Slug: record.Slug, Status: "error", Error: err.Error()})
			continue
		}
		summary.Indexed++
		summary.Projects = append(summary.Projects, res)
		if err := updateIndexStatus(db, record.ProjectID, true, false); err != nil {
			return summary, err
		}
	}
	return summary, rows.Err()
}

func rebuildProject(record projectLookupResult, effectivePaths paths.EffectivePaths) (projectReindexProjectResult, error) {
	result, err := index.RebuildProjectIndex(record.ProjectID, record.Location.memoriesAbs)
	if err != nil {
		return projectReindexProjectResult{}, err
	}
	return projectReindexProjectResult{
		ProjectID:    result.ProjectID,
		Slug:         record.Slug,
		NotesSeen:    result.NotesSeen,
		NotesIndexed: result.NotesIndexed,
		Status:       result.Status,
	}, nil
}

func updateIndexStatus(db *sql.DB, projectID string, present bool, needsReindex bool) error {
	_, err := db.Exec(`UPDATE project_status SET index_present = ?, needs_reindex = ?, last_seen_at = CURRENT_TIMESTAMP WHERE project_id = ?`,
		boolToInt(present), boolToInt(needsReindex), projectID)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
