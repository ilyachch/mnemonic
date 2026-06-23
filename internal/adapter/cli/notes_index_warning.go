package cli

import (
	"fmt"
	"strings"

	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/spf13/cobra"
)

func printIndexWarning(cmd *cobra.Command, status, errText string) {
	if status != "stale" {
		return
	}

	errText = strings.TrimSpace(errText)
	if errText == "" {
		return
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: index is stale: %s\n", errText)
}

func printIndexWarnings(cmd *cobra.Command, warnings []catalogsvc.ImportIndexError) {
	for _, warning := range warnings {
		slug := strings.TrimSpace(warning.Slug)
		errText := strings.TrimSpace(warning.Error)
		if slug == "" || errText == "" {
			continue
		}

		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: index is stale for %s: %s\n", slug, errText)
	}
}
