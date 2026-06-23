package cli

import (
	"bytes"
	"testing"

	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestPrintIndexWarning(t *testing.T) {
	cmd := &cobra.Command{}
	errBuf := &bytes.Buffer{}
	cmd.SetErr(errBuf)

	printIndexWarning(cmd, "stale", "  rebuild failed  ")

	require.Equal(t, "warning: index is stale: rebuild failed\n", errBuf.String())
}

func TestPrintIndexWarningSkipsNonStaleStatus(t *testing.T) {
	cmd := &cobra.Command{}
	errBuf := &bytes.Buffer{}
	cmd.SetErr(errBuf)

	printIndexWarning(cmd, "ok", "rebuild failed")

	require.Empty(t, errBuf.String())
}

func TestPrintIndexWarnings(t *testing.T) {
	cmd := &cobra.Command{}
	errBuf := &bytes.Buffer{}
	cmd.SetErr(errBuf)

	printIndexWarnings(cmd, []catalogsvc.ImportIndexError{
		{Slug: "backend", Error: "  rebuild failed  "},
		{Slug: " ", Error: "ignored"},
		{Slug: "frontend", Error: "missing index"},
	})

	require.Equal(t, "warning: index is stale for backend: rebuild failed\nwarning: index is stale for frontend: missing index\n", errBuf.String())
}
