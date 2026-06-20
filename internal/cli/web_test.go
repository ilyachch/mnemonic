package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestWebServeFailsFastWhenProjectIsMissing(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(root, ".mnemonic")
	authDB := filepath.Join(root, "state", "mnemonic", "web_auth.sqlite")

	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)
	t.Setenv("MNEMONIC_SERVE_PROJECTS", "missing")
	t.Setenv("MNEMONIC_WEB_AUTH_DB", authDB)

	result := executeCommand("web", "serve")
	require.Error(t, result.Err)
	require.Equal(t, int(app.CodeNotFound), ExitCodeForError(result.Err))
	require.Contains(t, result.Stderr, "missing")

	_, err := os.Stat(authDB)
	require.True(t, os.IsNotExist(err), "auth db should not be initialized on boot failure")
}
