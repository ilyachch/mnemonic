package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
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
	cmd.Flags().StringSlice("kind", nil, "filter diagnostics by kind (invalid_frontmatter, missing_required_field, missing_summary, duplicate_slug, duplicate_alias, unresolved_link, ambiguous_link, empty_body)")
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
		return handleDoctorAll(cmd, container)
	}

	return handleDoctorSingle(cmd, selector)
}

func handleDoctorAll(cmd *cobra.Command, container *app.Bootstrap) error {
	if container.Services.Maint == nil {
		return errors.New("maintenance service is not configured")
	}
	maintResult, maintErr := container.Services.Maint.DoctorAll(commandContext(cmd), loggerFromContext(commandContext(cmd)))
	if maintErr != nil {
		return maintErr
	}
	human := fmt.Sprintf("%d projects checked\n", maintResult.Total)
	if err := PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, maintResult); err != nil {
		return err
	}
	if maintResult.Failed > 0 {
		return apperr.Internal(fmt.Sprintf("%d project(s) failed", maintResult.Failed), nil)
	}
	return nil
}

func handleDoctorSingle(cmd *cobra.Command, selector string) error {
	runtime, err := runtimeAppForSelector(commandContext(cmd), selector)
	if err != nil {
		return err
	}

	svc := runtime.Services.Index
	if svc == nil {
		return errors.New("runtime index service is not configured")
	}

	kindsFlag, _ := cmd.Flags().GetStringSlice("kind")
	if len(kindsFlag) > 0 {
		return runDoctorKinds(cmd, svc, kindsFlag)
	}

	result, err := svc.Doctor(commandContext(cmd))
	if err != nil {
		return err
	}
	return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), result.Status+"\n", result)
}

func runDoctorKinds(cmd *cobra.Command, svc *indexsvc.Service, kindsFlag []string) error {
	kinds := make([]indexsvc.DiagnosticKind, 0, len(kindsFlag))
	for _, k := range kindsFlag {
		kinds = append(kinds, indexsvc.DiagnosticKind(strings.TrimSpace(k)))
	}

	result, err := svc.Diagnose(commandContext(cmd), indexsvc.DiagnoseInput{Kinds: kinds})
	if err != nil {
		return err
	}

	jsonOutput := jsonOutputEnabled(cmd)
	if jsonOutput {
		return PrintOutput(cmd.OutOrStdout(), true, "", result)
	}
	return PrintOutput(cmd.OutOrStdout(), false, formatDiagnoseTable(result), nil)
}

func formatDiagnoseTable(result indexsvc.DiagnoseOutput) string {
	if len(result.Issues) == 0 {
		return "No issues found.\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d issue(s)", result.TotalCount)
	if result.NextCursor != 0 {
		fmt.Fprintf(&b, " (showing first %d)", len(result.Issues))
	}
	b.WriteString(":\n\n")

	for _, issue := range result.Issues {
		writeIssueRow(&b, issue)
	}

	return b.String()
}

func writeIssueRow(w io.Writer, issue indexsvc.DiagnosticIssue) {
	_, _ = fmt.Fprintf(w, "[%s] ", issue.Kind)
	if issue.Path != "" {
		_, _ = fmt.Fprint(w, issue.Path)
	}
	if issue.NoteID != "" {
		_, _ = fmt.Fprintf(w, " (%s)", issue.NoteID)
	}
	_, _ = fmt.Fprint(w, "\n")
	if issue.Detail != "" {
		_, _ = fmt.Fprintf(w, "  %s\n", issue.Detail)
	}
	if len(issue.Candidates) > 0 {
		_, _ = fmt.Fprint(w, "  Suggestions:\n")
		for _, c := range issue.Candidates {
			_, _ = fmt.Fprintf(w, "    - %s (%s) [%s]\n", c.Title, c.Slug, c.Path)
		}
	}
}
