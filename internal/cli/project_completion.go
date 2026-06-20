package cli

import (
	"fmt"
	"strings"

	"github.com/ilyachch/mnemonic/internal/registry"
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

	slugs, err := registry.Slugs(container.Paths.MemoriesHome)
	if err != nil {
		return nil, fmt.Errorf("list active project names: %w", err)
	}

	return slugs, nil
}
