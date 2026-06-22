package cli

import (
	"fmt"
	"strings"

	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/spf13/cobra"
)

func printProjectImportIndexWarnings(cmd *cobra.Command, status string, indexErrors []catalogsvc.ImportIndexError) {
	if status != "stale" {
		return
	}

	for _, indexErr := range indexErrors {
		printProjectImportIndexWarning(cmd, indexErr.Slug, indexErr.Error)
	}
}

func printProjectImportIndexWarning(cmd *cobra.Command, slug, errText string) {
	slug = strings.TrimSpace(slug)
	errText = strings.TrimSpace(errText)
	if slug == "" || errText == "" {
		return
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: index is stale for %s: %s\n", slug, errText)
}
