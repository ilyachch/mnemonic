package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExampleMnemonicFileParses(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "projects", "example", ".mnemonic"))
	require.NoError(t, err)

	parsed, err := ParseMnemonicFile(data)
	require.NoError(t, err)

	require.Len(t, parsed.Projects, 1)
}

func TestExampleMnemonicManifestParses(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "projects", "example", "mnemonic.toml"))
	require.NoError(t, err)

	parsed, err := ParseMnemonicManifest(data)
	require.NoError(t, err)

	require.NotEmpty(t, parsed.ProjectID)
}
