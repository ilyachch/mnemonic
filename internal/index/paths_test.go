package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestPathUsesStateHomeAndProjectID(t *testing.T) {
	stateHome := filepath.Join(testutil.CleanEnvForTest(t), "state")

	got, err := Path("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	want := filepath.Join(stateHome, "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "index.sqlite")
	require.Equal(t, want, got)
	require.NotEqual(t, filepath.Clean(filepath.Join(os.Getenv("HOME"), ".mnemonic")), filepath.Clean(filepath.Dir(got)))
}

func TestPathRejectsEmptyProjectID(t *testing.T) {
	_, err := Path("")
	require.Error(t, err)
}