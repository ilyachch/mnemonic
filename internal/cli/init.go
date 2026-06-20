package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/config"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init NAME",
	Short: "Initialize a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		local, err := cmd.Flags().GetBool("local")
		if err != nil {
			return err
		}

		if len(args) == 0 {
			return app.NewCLIUsageError("init requires NAME", nil)
		}
		if len(args) > 1 {
			return app.NewCLIUsageError("init accepts exactly one NAME", nil)
		}

		discoveredConfigPath, err := config.DiscoverConfigFile("")
		if err != nil {
			return err
		}

		cfg := config.DefaultConfig()
		if discoveredConfigPath != "" {
			cfg, err = config.LoadConfig(discoveredConfigPath)
			if err != nil {
				return err
			}
		}

		effective, err := paths.ResolveEffectivePaths(paths.EffectiveInput{
			ConfigFile:            discoveredConfigPath,
			RawConfigMemoriesHome: cfg.Paths.MemoriesHome,
		})
		if err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		description, err := cmd.Flags().GetString("description")
		if err != nil {
			return err
		}

		mode := project.InitModeCentral
		if local {
			mode = project.InitModeLocal
		}

		input := project.InitInput{
			CWD:          cwd,
			MemoriesHome: effective.MemoriesHome,
			Name:         args[0],
			Description:  description,
			Mode:         mode,
		}

		if err := project.InitProject(input); err != nil {
			if strings.Contains(err.Error(), "project slug") && strings.Contains(err.Error(), "already exists") {
				return app.NewAmbiguousError(err.Error(), nil)
			}
			return err
		}

		// Build the initial index.
		slug, err := project.Slugify(args[0])
		if err != nil {
			return err
		}

		memoriesRoot, err := resolveInitMemoriesRoot(effective.MemoriesHome, cwd, slug, mode)
		if err != nil {
			return err
		}

		// Read the project ID from the manifest.
		manifestPath := filepath.Join(effective.MemoriesHome, slug, "mnemonic.toml")
		if local {
			manifestPath = filepath.Join(cwd, ".mnemonic-memories", slug, "mnemonic.toml")
		}
		manifest, err := project.ParseMnemonicManifestFromFile(manifestPath)
		if err != nil {
			return err
		}

		if _, err := index.RebuildProjectIndex(manifest.ProjectID, memoriesRoot); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	initCmd.Flags().Bool("local", false, "create a local project")
	initCmd.Flags().String("description", "", "optional description of this memory's knowledge scope")
	RootCmd.AddCommand(initCmd)
}

func resolveInitMemoriesRoot(memoriesHome, cwd, slug string, mode project.InitMode) (string, error) {
	switch mode {
	case project.InitModeCentral:
		return filepath.Join(memoriesHome, slug), nil
	case project.InitModeLocal:
		return filepath.Join(cwd, ".mnemonic-memories", slug), nil
	default:
		return "", fmt.Errorf("unknown init mode %q", mode)
	}
}
