package paths

import (
	"os"
	"path/filepath"
)

// XDGPaths holds the resolved standard XDG base directories.
type XDGPaths struct {
	ConfigHome string
	DataHome   string
	StateHome  string
	CacheHome  string
}

// GetXDGPaths calculates base directories according to the XDG Base Directory Specification.
// It falls back to default locations under the user's home directory if environment variables
// are unset or contain relative paths.
func GetXDGPaths() XDGPaths {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}

	return XDGPaths{
		ConfigHome: resolveEnvPath("XDG_CONFIG_HOME", filepath.Join(home, ".config")),
		DataHome:   resolveEnvPath("XDG_DATA_HOME", filepath.Join(home, ".local", "share")),
		StateHome:  resolveEnvPath("XDG_STATE_HOME", filepath.Join(home, ".local", "state")),
		CacheHome:  resolveEnvPath("XDG_CACHE_HOME", filepath.Join(home, ".cache")),
	}
}

func resolveEnvPath(envKey string, defaultPath string) string {
	val := os.Getenv(envKey)
	if val != "" && filepath.IsAbs(val) {
		return val
	}
	return defaultPath
}
