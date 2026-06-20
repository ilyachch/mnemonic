package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/registry"
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

		projects, err := loadProjectList(container.Paths)
		if err != nil {
			return err
		}

		output := projectListOutput{Projects: projects}
		if output.Projects == nil {
			output.Projects = []projectListItem{}
		}

		human := formatProjectListHuman(output.Projects)
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
	Type         string `json:"type"`
	MemoriesPath string `json:"memories_path"`
	StatePath    string `json:"state_path"`
	Status       string `json:"status"`
	Issue        string `json:"issue,omitempty"`
}

func loadProjectList(effectivePaths paths.EffectivePaths) ([]projectListItem, error) {
	entries, issues, err := registry.Scan(effectivePaths.MemoriesHome)
	if err != nil {
		return nil, fmt.Errorf("scan registry: %w", err)
	}

	// Build a map for quick issue lookup by slug.
	issueMap := make(map[string]registry.Issue, len(issues))
	for _, issue := range issues {
		issueMap[issue.Slug] = issue
	}

	projects := make([]projectListItem, 0, len(entries))
	for _, e := range entries {
		item := projectListItem{
			ProjectID:    e.ProjectID,
			Name:         e.Name,
			Slug:         e.Slug,
			Type:         e.Type,
			MemoriesPath: e.MemoriesAbs,
			StatePath:    filepath.Join(effectivePaths.StateHome, "mnemonic", "projects", e.ProjectID),
			Status:       "ok",
		}

		if issue, found := issueMap[e.Slug]; found {
			if issue.Corrupt {
				item.Status = "[CORRUPTED]"
			} else if issue.Orphan {
				item.Status = "[ORPHANED/MISSING]"
			}
			item.Issue = issue.Error
		}

		projects = append(projects, item)
	}

	return projects, nil
}

func formatProjectListHuman(projects []projectListItem) string {
	if len(projects) == 0 {
		return "0 projects\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d projects\n", len(projects))
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tSLUG\tTYPE\tPATH\tSTATUS")
	for _, p := range projects {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", p.Name, p.Slug, p.Type, p.MemoriesPath, p.Status)
	}
	_ = w.Flush()
	return b.String()
}
