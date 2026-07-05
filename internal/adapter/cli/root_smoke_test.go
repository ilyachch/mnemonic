package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCmdWiringSmoke(t *testing.T) {
	root := newTestRoot(t)

	require.Equal(t, "mnemonic", root.Use)

	projectFlag := root.PersistentFlags().Lookup("project")
	require.NotNil(t, projectFlag)
	require.Equal(t, "", projectFlag.DefValue)

	jsonFlag := root.PersistentFlags().Lookup("json")
	require.NotNil(t, jsonFlag)
	require.Equal(t, "false", jsonFlag.DefValue)

	for _, path := range [][]string{
		{"config", "show"},
		{"stdio"},
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
		{"project", "init"},
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
		cmd, _, err := root.Find(path)
		require.NoErrorf(t, err, "missing command %v", path)
		require.Equal(t, path[len(path)-1], cmd.Name())
	}

	_, _, err := root.Find([]string{"init"})
	require.Error(t, err)

	for _, item := range []struct {
		path     []string
		flagName string
		defValue string
	}{
		{path: []string{"notes", "create"}, flagName: "title", defValue: ""},
		{path: []string{"notes", "create"}, flagName: "stdin", defValue: "false"},
		{path: []string{"notes", "create"}, flagName: "body-file", defValue: ""},
		{path: []string{"notes", "search"}, flagName: "limit", defValue: "10"},
		{path: []string{"project", "init"}, flagName: "local", defValue: "false"},
		{path: []string{"project", "init"}, flagName: "description", defValue: ""},
		{path: []string{"project", "import"}, flagName: "dry-run", defValue: "false"},
		{path: []string{"project", "remove"}, flagName: "wipe", defValue: "false"},
		{path: []string{"project", "reindex"}, flagName: "all", defValue: "false"},
		{path: []string{"stdio"}, flagName: "read-only", defValue: "false"},
		{path: []string{"web", "serve"}, flagName: "addr", defValue: ""},
		{path: []string{"web", "serve"}, flagName: "port", defValue: ""},
	} {
		cmd, _, err := root.Find(item.path)
		require.NoErrorf(t, err, "missing command %v", item.path)
		flag := cmd.Flags().Lookup(item.flagName)
		require.NotNil(t, flag)
		require.Equal(t, item.defValue, flag.DefValue)
	}
}
