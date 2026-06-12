package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestOpenDBCreatesRegistryFileAndPragmas(t *testing.T) {
	dataHome := filepath.Join(testutil.CleanEnvForTest(t), "data")

	path, err := RegistryPath()
	require.NoError(t, err)
	wantPath := filepath.Join(dataHome, "mnemonic", "registry.sqlite")
	require.Equal(t, wantPath, path)

	db, err := OpenDB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = os.Stat(filepath.Dir(path))
	require.NoError(t, err, "parent directory missing")
	_, err = os.Stat(path)
	require.NoError(t, err, "registry file missing")

	var foreignKeys int
	err = db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys)
	require.NoError(t, err)
	require.Equal(t, 1, foreignKeys)

	var userVersion int
	err = db.QueryRow(`PRAGMA user_version`).Scan(&userVersion)
	require.NoError(t, err)
	require.Equal(t, 0, userVersion)
}

func TestNullString(t *testing.T) {
	assert.Nil(t, nullString(""))
	assert.Equal(t, "hello", nullString("hello"))
}
