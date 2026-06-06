package testutil

import (
	"path/filepath"
	"testing"
)

// CleanEnvForTest resets the common path-related environment variables to
// deterministic temporary locations and returns the temporary root.
func CleanEnvForTest(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	t.Setenv("MNEMONIC_CONFIG_HOME", "")
	t.Setenv("MNEMONIC_DATA_HOME", "")
	t.Setenv("MNEMONIC_STATE_HOME", "")
	t.Setenv("MNEMONIC_CACHE_HOME", "")
	t.Setenv("MNEMONIC_MEMORIES_HOME", "")
	t.Setenv("MNEMONIC_CONFIG_FILE", "")

	return tmp
}
