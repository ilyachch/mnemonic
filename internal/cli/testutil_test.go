package cli

import (
	"bytes"
	"path/filepath"
	"testing"
)

type cmdResult struct {
	Stdout string
	Stderr string
	Err    error
}

// executeCommand runs RootCmd with the given arguments and returns stdout, stderr, and error.
func executeCommand(args ...string) cmdResult {
	closeAppContainer()
	defer closeAppContainer()

	jsonFlag = false
	projectFlag = ""
	mcpReadOnlyFlag = false
	_ = initCmd.Flags().Set("local", "false")
	_ = initCmd.Flags().Set("description", "")
	_ = projectImportCmd.Flags().Set("dry-run", "false")
	_ = notesCreateCmd.Flags().Set("title", "")
	_ = notesCreateCmd.Flags().Set("stdin", "false")
	_ = notesCreateCmd.Flags().Set("body-file", "")
	_ = notesCreateCmd.Flags().Set("tag", "")
	_ = notesEditCmd.Flags().Set("append", "")
	_ = notesEditCmd.Flags().Set("body-file", "")
	_ = notesEditCmd.Flags().Set("if-match", "")
	_ = notesEditCmd.Flags().Set("set", "")
	if flag := notesEditCmd.Flags().Lookup("set"); flag != nil {
		flag.Changed = false
	}
	_ = notesSearchCmd.Flags().Set("limit", "20")
	_ = notesDeleteCmd.Flags().Set("dry-run", "false")
	_ = notesDeleteCmd.Flags().Set("hard", "false")
	_ = notesDeleteCmd.Flags().Set("yes", "false")
	_ = projectRemoveCmd.Flags().Set("wipe", "false")
	_ = projectReindexCmd.Flags().Set("all", "false")
	if flag := projectReindexCmd.Flags().Lookup("all"); flag != nil {
		flag.Changed = false
	}
	_ = projectDoctorCmd.Flags().Set("all", "false")
	if flag := projectDoctorCmd.Flags().Lookup("all"); flag != nil {
		flag.Changed = false
	}
	_ = mcpCmd.Flags().Set("help", "false")
	_ = mcpCmd.Flags().Set("read-only", "false")
	if flag := mcpCmd.Flags().Lookup("read-only"); flag != nil {
		flag.Changed = false
	}
	if flag := mcpCmd.Flags().Lookup("help"); flag != nil {
		flag.Changed = false
	}
	_ = webServeCmd.Flags().Set("addr", "")
	_ = webServeCmd.Flags().Set("port", "")
	_ = webServeCmd.Flags().Set("help", "false")
	if flag := webServeCmd.Flags().Lookup("help"); flag != nil {
		flag.Changed = false
	}
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	RootCmd.SetOut(bufOut)
	RootCmd.SetErr(bufErr)
	RootCmd.SetArgs(args)

	err := RootCmd.Execute()

	return cmdResult{
		Stdout: bufOut.String(),
		Stderr: bufErr.String(),
		Err:    err,
	}
}

func setLocalProjectMemoriesHome(t *testing.T, projectRoot string) {
	t.Helper()

	t.Setenv("MNEMONIC_MEMORIES_HOME", filepath.Join(projectRoot, ".mnemonic-memories"))
}
