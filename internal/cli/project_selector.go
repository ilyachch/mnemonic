package cli

import (
	"os"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/spf13/cobra"
)

func resolveProjectSelector(cmd *cobra.Command, args []string, all bool) (string, error) {
	if all {
		if len(args) > 0 || projectFlag != "" {
			return "", apperr.CLIUsage("--all cannot be combined with a project selector", nil)
		}
		return "", nil
	}

	switch {
	case len(args) > 0:
		return args[0], nil
	case projectFlag != "":
		return projectFlag, nil
	case os.Getenv("MNEMONIC_PROJECT") != "":
		return os.Getenv("MNEMONIC_PROJECT"), nil
	default:
		return "", apperr.CLIUsage("project selector is required", nil)
	}
}
