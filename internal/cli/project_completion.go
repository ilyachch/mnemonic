package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func completeProjectNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	projects, err := loadActiveProjectNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if toComplete == "" {
		return projects, cobra.ShellCompDirectiveNoFileComp
	}

	matches := make([]string, 0, len(projects))
	for _, name := range projects {
		if strings.HasPrefix(name, toComplete) {
			matches = append(matches, name)
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

func loadActiveProjectNames() ([]string, error) {
	container, err := mustAppContainer()
	if err != nil {
		return nil, err
	}

	rows, err := container.Services.Registry.Query(
		`SELECT slug
		 FROM projects
		 WHERE removed_at IS NULL
		 ORDER BY slug`,
	)
	if err != nil {
		return nil, fmt.Errorf("query active project names: %w", err)
	}
	defer rows.Close()

	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan active project name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active project names: %w", err)
	}

	return names, nil
}
