package cli

import (
	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/spf13/cobra"
)

func resolveProjectSelector(cmd *cobra.Command, args []string, all bool) (string, error) {
	shared := sharedOptionsFromContext(commandContext(cmd))
	if all {
		if len(args) > 0 {
			return "", apperr.CLIUsage("--all cannot be combined with a project selector", nil)
		}
		if shared != nil && shared.Project != "" {
			return "", apperr.CLIUsage("--all cannot be combined with a project selector", nil)
		}
		return "", nil
	}

	switch {
	case len(args) > 0:
		return args[0], nil
	default:
		if shared != nil {
			if selector := shared.ProjectSelector(); selector != "" {
				return selector, nil
			}
		}
		return "", apperr.CLIUsage("project selector is required", nil)
	}
}
