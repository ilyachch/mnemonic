package cli

import (
	"errors"
	"fmt"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/spf13/cobra"
)

func newProjectDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "doctor [PROJECT]",
		Short:             "Run project health checks",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeProjectNames,
		RunE:              runProjectDoctor,
	}
	cmd.Flags().Bool("all", false, "run doctor across all active projects")
	return cmd
}

func runProjectDoctor(cmd *cobra.Command, args []string) error {
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}

	selector, err := resolveProjectSelector(cmd, args, all)
	if err != nil {
		return err
	}

	container, err := bootstrapFromContext(commandContext(cmd))
	if err != nil {
		return err
	}

	if all {
		if container.Services.Maint == nil {
			return errors.New("maintenance service is not configured")
		}
		maintResult, maintErr := container.Services.Maint.DoctorAll(commandContext(cmd))
		if maintErr != nil {
			return maintErr
		}
		human := fmt.Sprintf("%d projects checked\n", maintResult.Total)
		if err = PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, maintResult); err != nil {
			return err
		}
		if maintResult.Failed > 0 {
			return apperr.Internal(fmt.Sprintf("%d project(s) failed", maintResult.Failed), nil)
		}
		return nil
	}

	runtime, err := runtimeAppForSelector(commandContext(cmd), selector)
	if err != nil {
		return err
	}

	indexService := runtime.Services.Index
	if indexService == nil {
		return errors.New("runtime index service is not configured")
	}

	result, err := indexService.Doctor(commandContext(cmd))
	if err != nil {
		return err
	}
	human := result.Status + "\n"
	return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, result)
}
