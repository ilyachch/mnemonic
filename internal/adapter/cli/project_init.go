package cli

import (
	"fmt"
	"os"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/spf13/cobra"
)

var projectInitCmd = &cobra.Command{
	Use:   "init NAME",
	Short: "Initialize a project",
	Args: func(cmd *cobra.Command, args []string) error {
		switch len(args) {
		case 0:
			return apperr.CLIUsage("init requires NAME", nil)
		case 1:
			return nil
		default:
			return apperr.CLIUsage("init accepts exactly one NAME", nil)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		local, err := cmd.Flags().GetBool("local")
		if err != nil {
			return err
		}

		description, err := cmd.Flags().GetString("description")
		if err != nil {
			return err
		}

		container, err := bootstrapFromContext(commandContext(cmd))
		if err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		mode := catalogsvc.InitModeCentral
		if local {
			mode = catalogsvc.InitModeLocal
		}

		result, err := container.Services.Catalog.Init(commandContext(cmd), catalogsvc.InitInput{
			WorkingDir:  cwd,
			Name:        args[0],
			Description: description,
			Mode:        mode,
		})
		if err != nil {
			return err
		}

		human := fmt.Sprintf("%s initialized\n", result.Slug)
		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, result)
	},
}
