package cli

import (
	"fmt"
	"strings"

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
