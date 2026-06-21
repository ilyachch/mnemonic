package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCmdWiringSmoke(t *testing.T) {
	require.Equal(t, "mnemonic", RootCmd.Use)

	projectFlag := RootCmd.PersistentFlags().Lookup("project")
	require.NotNil(t, projectFlag)
	require.Equal(t, "", projectFlag.DefValue)

	jsonFlag := RootCmd.PersistentFlags().Lookup("json")
	require.NotNil(t, jsonFlag)
	require.Equal(t, "false", jsonFlag.DefValue)

	for _, path := range [][]string{
		{"config", "show"},
		{"hello"},
		{"init"},
		{"mcp"},
		{"notes"},
		{"notes", "backlinks"},
		{"notes", "create"},
		{"notes", "delete"},
		{"notes", "edit"},
		{"notes", "list"},
		{"notes", "search"},
		{"notes", "show"},
		{"project"},
		{"project", "doctor"},
		{"project", "import"},
		{"project", "list"},
		{"project", "reindex"},
		{"project", "remove"},
		{"project", "show"},
		{"tags"},
		{"tags", "list"},
		{"version"},
		{"web"},
		{"web", "serve"},
	} {
		cmd, _, err := RootCmd.Find(path)
		require.NoErrorf(t, err, "missing command %v", path)
		require.Equal(t, path[len(path)-1], cmd.Name())
	}

	require.Equal(t, "", notesCreateCmd.Flags().Lookup("title").DefValue)
	require.Equal(t, "false", notesCreateCmd.Flags().Lookup("stdin").DefValue)
	require.Equal(t, "", notesCreateCmd.Flags().Lookup("body-file").DefValue)
	require.Equal(t, "20", notesSearchCmd.Flags().Lookup("limit").DefValue)
	require.Equal(t, "false", projectImportCmd.Flags().Lookup("dry-run").DefValue)
	require.Equal(t, "false", projectRemoveCmd.Flags().Lookup("wipe").DefValue)
	require.Equal(t, "false", projectReindexCmd.Flags().Lookup("all").DefValue)
	require.Equal(t, "false", mcpCmd.Flags().Lookup("read-only").DefValue)
	require.Equal(t, "", webServeCmd.Flags().Lookup("addr").DefValue)
	require.Equal(t, "", webServeCmd.Flags().Lookup("port").DefValue)
}
