package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestOpenDBCreatesFileAndPragmas(t *testing.T) {
	stateHome := filepath.Join(testutil.CleanEnvForTest(t), "state")

	db, err := OpenDB("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	path := filepath.Join(stateHome, "mnemonic", "projects", "550e8400-e29b-41d4-a716-446655440000", "index.sqlite")
	_, err = os.Stat(path)
	require.NoError(t, err)

	var foreignKeys int
	require.NoError(t, db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys))
	require.Equal(t, 1, foreignKeys)

	var journalMode string
	require.NoError(t, db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode))
	require.Equal(t, "wal", journalMode)

	var busyTimeout int
	require.NoError(t, db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout))
	require.Equal(t, 5000, busyTimeout)

	var userVersion int
	require.NoError(t, db.QueryRow(`PRAGMA user_version`).Scan(&userVersion))
	require.Equal(t, 1, userVersion)
}
