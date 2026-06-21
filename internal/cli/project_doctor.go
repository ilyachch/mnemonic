package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var projectDoctorCmd = &cobra.Command{
	Use:               "doctor [PROJECT]",
	Short:             "Run project health checks",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}

		selector, err := resolveProjectSelector(cmd, args, all)
		if err != nil {
			return err
		}

		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		if all {
			if container.Services.Maint == nil {
				return fmt.Errorf("maintenance service is not configured")
			}
			result, err := container.Services.Maint.DoctorAll(cmd.Context())
			if err != nil {
				return err
			}
			human := fmt.Sprintf("%d projects checked\n", len(result.Projects))
			return PrintOutput(cmd.OutOrStdout(), human, result)
		}

		runtime, err := runtimeAppForSelector(cmd.Context(), selector)
		if err != nil {
			return err
		}

		indexService := runtime.Services.Index
		if indexService == nil {
			return fmt.Errorf("runtime index service is not configured")
		}

		result, err := indexService.Doctor(cmd.Context())
		if err != nil {
			return err
		}
		human := fmt.Sprintf("%s\n", result.Status)
		return PrintOutput(cmd.OutOrStdout(), human, result)
	},
}

func init() {
	projectDoctorCmd.Flags().Bool("all", false, "run doctor across all active projects")
	projectCmd.AddCommand(projectDoctorCmd)
}
