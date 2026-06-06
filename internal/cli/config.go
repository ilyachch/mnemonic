package cli

import (
	"fmt"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/config"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect configuration",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var configShowCmd = &cobra.Command{
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
				return app.NewCLIUsageError("invalid config file", err)
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

		return PrintOutput(cmd.OutOrStdout(), human, data)
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	RootCmd.AddCommand(configCmd)
}

type configShowOutput struct {
	ConfigHome   string `json:"config_home"`
	DataHome     string `json:"data_home"`
	StateHome    string `json:"state_home"`
	CacheHome    string `json:"cache_home"`
	MemoriesHome string `json:"memories_home"`
	ConfigFile   string `json:"config_file"`
}
