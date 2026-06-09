package cli

import (
	"bytes"
)

type cmdResult struct {
	Stdout string
	Stderr string
	Err    error
}

// executeCommand runs RootCmd with the given arguments and returns stdout, stderr, and error.
func executeCommand(args ...string) cmdResult {
	defer closeAppContainer()

	jsonFlag = false
	_ = initCmd.Flags().Set("local", "false")
	_ = initCmd.Flags().Set("detached", "false")
	_ = projectDiscoverCmd.Flags().Set("dry-run", "false")
	_ = projectImportCmd.Flags().Set("dry-run", "false")
	_ = notesCreateCmd.Flags().Set("project", "")
	_ = notesCreateCmd.Flags().Set("title", "")
	_ = notesCreateCmd.Flags().Set("stdin", "false")
	_ = notesCreateCmd.Flags().Set("body-file", "")
	_ = notesCreateCmd.Flags().Set("tag", "")
	_ = notesListCmd.Flags().Set("project", "")
	_ = notesShowCmd.Flags().Set("project", "")
	_ = notesEditCmd.Flags().Set("project", "")
	_ = notesEditCmd.Flags().Set("append", "")
	_ = notesEditCmd.Flags().Set("body-file", "")
	_ = notesEditCmd.Flags().Set("if-match", "")
	_ = notesEditCmd.Flags().Set("set", "")
	if flag := notesEditCmd.Flags().Lookup("set"); flag != nil {
		flag.Changed = false
	}
	_ = notesSearchCmd.Flags().Set("project", "")
	_ = notesSearchCmd.Flags().Set("limit", "20")
	_ = notesDeleteCmd.Flags().Set("project", "")
	_ = notesDeleteCmd.Flags().Set("dry-run", "false")
	_ = notesDeleteCmd.Flags().Set("hard", "false")
	_ = notesDeleteCmd.Flags().Set("yes", "false")
	_ = projectRemoveCmd.Flags().Set("hard", "false")
	_ = projectRemoveCmd.Flags().Set("delete-markdown", "false")
	_ = projectRemoveCmd.Flags().Set("wipe", "false")
	_ = mcpCmd.Flags().Set("project", "")
	_ = mcpCmd.Flags().Set("help", "false")
	if flag := mcpCmd.Flags().Lookup("help"); flag != nil {
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
