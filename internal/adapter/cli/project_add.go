package cli

import (
	"strings"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/spf13/cobra"
)

func newProjectAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [PATH]",
		Short: "Register an existing structured project",
		Args:  cobra.RangeArgs(0, 1),
		RunE:  runProjectAdd,
	}
	return cmd
}

func runProjectAdd(cmd *cobra.Command, args []string) error {
	pathArg := "."
	if len(args) == 1 {
		pathArg = args[0]
	}

	container, err := bootstrapFromContext(commandContext(cmd))
	if err != nil {
		return err
	}

	result, err := container.Services.Catalog.Add(commandContext(cmd), catalogsvc.AddInput{Path: pathArg})
	if err != nil {
		return wrapAddError(err)
	}

	if !jsonOutputEnabled(cmd) {
		printIndexWarning(cmd, result.IndexStatus, result.IndexError)
	}
	output := projectAddOutput{
		Path:        result.Path,
		Slug:        result.Slug,
		ProjectID:   result.ProjectID,
		Indexed:     result.Indexed,
		IndexStatus: result.IndexStatus,
		IndexError:  result.IndexError,
	}
	human := result.Slug + " added\n"
	return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
}

func wrapAddError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "add path") && strings.Contains(msg, "not found"):
		return apperr.NotFound(msg, nil)
	case strings.HasPrefix(msg, "mnemonic.toml not found at"):
		return apperr.NotFound(msg, nil)
	case strings.HasPrefix(msg, "project slug") && strings.Contains(msg, "already exists"):
		return apperr.Ambiguous(msg, nil)
	}
	return err
}

type projectAddOutput struct {
	Path        string `json:"path"`
	Slug        string `json:"slug"`
	ProjectID   string `json:"project_id"`
	Indexed     int    `json:"indexed"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}
