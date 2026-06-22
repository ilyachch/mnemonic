package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/spf13/cobra"
)

type cmdResult struct {
	Stdout string
	Stderr string
	Err    error
}

// executeCommand runs a fresh CLI root with the given arguments and returns stdout, stderr, and error.
func executeCommand(args ...string) cmdResult {
	root := newRootForTest()
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	root.SetOut(bufOut)
	root.SetErr(bufErr)
	root.SetArgs(args)

	err := root.Execute()
	if err != nil {
		fmt.Fprintln(bufErr, err)
	}

	return cmdResult{
		Stdout: bufOut.String(),
		Stderr: bufErr.String(),
		Err:    err,
	}
}

func newRootForTest() *cobra.Command {
	boot, err := newTestBootstrap()
	if err != nil {
		panic(err)
	}
	return NewRootCommand(boot)
}

func newTestBootstrap() (*app.Bootstrap, error) {
	configFile, err := os.CreateTemp("", "mnemonic-cli-config-*.toml")
	if err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintln(configFile, "version = 1"); err != nil {
		_ = configFile.Close()
		return nil, err
	}
	if err := configFile.Close(); err != nil {
		return nil, err
	}

	return app.New(app.Input{CLI: paths.CLIOverrides{ConfigFile: configFile.Name()}})
}

func newTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	return newRootForTest()
}

func setLocalProjectMemoriesHome(t *testing.T, projectRoot string) {
	t.Helper()

	t.Setenv("MNEMONIC_MEMORIES_HOME", filepath.Join(projectRoot, ".mnemonic-memories"))
}
