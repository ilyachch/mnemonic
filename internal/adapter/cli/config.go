package cli

import (
	"fmt"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/platform/config"
	"github.com/ilyachch/mnemonic/internal/platform/paths"
	"github.com/spf13/cobra"
)

func newConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Inspect configuration",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show effective config and paths",
		RunE: func(cmd *cobra.Command, args []string) error {
		discoveredConfigPath, err := config.DiscoverConfigFile("")
		if err != nil {
			return err
		}

		cfg := config.DefaultConfig()
		if discoveredConfigPath != "" {
			cfg, err = config.LoadConfig(discoveredConfigPath)
			if err != nil {
				return apperr.CLIUsage("invalid config file", err)
			}
		}

		effective, err := paths.ResolveEffectivePaths(paths.EffectiveInput{
			ConfigFile:            discoveredConfigPath,
			RawConfigMemoriesHome: cfg.Paths.MemoriesHome,
		})
		if err != nil {
			return err
		}

		displayConfigPath := discoveredConfigPath
		if displayConfigPath == "" {
			displayConfigPath = filepath.Join(effective.ConfigHome, "mnemonic", "config.toml")
		}

		data := configShowOutput{
			ConfigHome:   effective.ConfigHome,
			DataHome:     effective.DataHome,
			StateHome:    effective.StateHome,
			CacheHome:    effective.CacheHome,
			MemoriesHome: effective.MemoriesHome,
			ConfigFile:   displayConfigPath,
		}

		human := fmt.Sprintf(
			"config_home=%s\ndata_home=%s\nstate_home=%s\ncache_home=%s\nmemories_home=%s\nconfig_file=%s\n",
			data.ConfigHome,
			data.DataHome,
			data.StateHome,
			data.CacheHome,
			data.MemoriesHome,
			data.ConfigFile,
		)

		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, data)
	},
	}
}

type configShowOutput struct {
	ConfigHome   string `json:"config_home"`
	DataHome     string `json:"data_home"`
	StateHome    string `json:"state_home"`
	CacheHome    string `json:"cache_home"`
	MemoriesHome string `json:"memories_home"`
	ConfigFile   string `json:"config_file"`
}
