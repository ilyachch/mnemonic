package cli

import (
	"errors"
	"os"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/config"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/registry"
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

		detached, err := cmd.Flags().GetBool("detached")
		if err != nil {
			return err
		}

		if len(args) == 0 {
			return app.NewCLIUsageError("init requires NAME", nil)
		}
		if len(args) > 1 {
			return app.NewCLIUsageError("init accepts exactly one NAME", nil)
		}
		if local && detached {
			return app.NewCLIUsageError("--local and --detached cannot be combined", nil)
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

		input := project.InitInput{
			CWD:          cwd,
			MemoriesHome: effective.MemoriesHome,
			Name:         args[0],
			Mode:         project.InitModeRegular,
		}
		if local {
			input.Mode = project.InitModeLocal
		}
		if detached {
			input.Mode = project.InitModeDetached
		}
		if err := project.InitProject(input); err != nil {
			var conflict registry.ErrProjectSlugConflict
			if errors.As(err, &conflict) {
				return app.NewAmbiguousError(conflict.Error(), nil)
			}
			return err
		}

		slug, err := project.Slugify(args[0])
		if err != nil {
			return err
		}

		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		record, err := queryProjectBySelector(slug)
		if err != nil {
			return err
		}

		return buildProjectIndex(container.Services.Registry, record.ProjectID, record.Location.memoriesAbs)
	},
}

func init() {
	initCmd.Flags().Bool("local", false, "create a local project")
	initCmd.Flags().Bool("detached", false, "create a detached project")
	RootCmd.AddCommand(initCmd)
}
