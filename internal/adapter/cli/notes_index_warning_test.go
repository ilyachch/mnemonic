package cli

import (
	"bytes"
	"testing"

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
