package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func completeProjectNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	projects, err := loadActiveProjectNames(commandContext(cmd))
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

func loadActiveProjectNames(ctx context.Context) ([]string, error) {
	boot, err := bootstrapFromContext(ctx)
	if err != nil {
		return nil, err
	}

	slugs, err := boot.Services.Catalog.Slugs()
	if err != nil {
		return nil, fmt.Errorf("list active project names: %w", err)
	}

	return slugs, nil
}
