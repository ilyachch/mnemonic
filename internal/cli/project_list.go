package cli

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		projects, err := loadProjectList(container.Services.Registry, container.Paths)
		if err != nil {
			return err
		}

		output := projectListOutput{Projects: projects}
		if output.Projects == nil {
			output.Projects = []projectListItem{}
		}

		human := fmt.Sprintf("%d projects\n", len(output.Projects))
		return PrintOutput(cmd.OutOrStdout(), human, output)
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
	RootCmd.AddCommand(projectCmd)
}

type projectListOutput struct {
	Projects []projectListItem `json:"projects"`
}

type projectListItem struct {
	ProjectID    string `json:"project_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Kind         string `json:"kind"`
	MemoriesPath string `json:"memories_path"`
	StatePath    string `json:"state_path"`
	NeedsReindex bool   `json:"needs_reindex"`
	IndexPresent bool   `json:"index_present"`
}

func loadProjectList(db *sql.DB, effectivePaths paths.EffectivePaths) ([]projectListItem, error) {
	rows, err := db.Query(
		`SELECT p.project_id, p.name, p.slug, p.kind, l.memories_abs, s.needs_reindex, s.index_present
		 FROM projects p
		 JOIN project_locations l ON l.project_id = p.project_id
		 JOIN project_status s ON s.project_id = p.project_id
		 WHERE p.removed_at IS NULL
		 ORDER BY p.slug`,
	)
	if err != nil {
		return nil, fmt.Errorf("query project list: %w", err)
	}
	defer rows.Close()

	projects := make([]projectListItem, 0)
	for rows.Next() {
		var item projectListItem
		var memoriesPath string
		var needsReindex, indexPresent int
		if err := rows.Scan(&item.ProjectID, &item.Name, &item.Slug, &item.Kind, &memoriesPath, &needsReindex, &indexPresent); err != nil {
			return nil, fmt.Errorf("scan project list row: %w", err)
		}

		item.MemoriesPath = memoriesPath
		item.StatePath = filepath.Join(effectivePaths.StateHome, "mnemonic", "projects", item.ProjectID, "state.toml")
		item.NeedsReindex = needsReindex == 1
		item.IndexPresent = indexPresent == 1
		projects = append(projects, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project list: %w", err)
	}

	return projects, nil
}
