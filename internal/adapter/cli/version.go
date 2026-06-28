package cli

import (
	"github.com/ilyachch/mnemonic/internal/platform/buildinfo"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Prints the version of mnemonic",
		RunE: func(cmd *cobra.Command, args []string) error {
			version := buildinfo.Version()
			data := struct {
				Version string `json:"version"`
			}{
				Version: version,
			}
			return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), version+"\n", data)
		},
	}
}
