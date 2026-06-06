package paths

import (
	"errors"
	"os"
	"path/filepath"
)

// MnemonicPaths holds resolved environment paths including XDG defaults and MNEMONIC_* overrides.
type MnemonicPaths struct {
	ConfigHome   string
	DataHome     string
	StateHome    string
	CacheHome    string
	MemoriesHome string
}

// GetMnemonicPaths resolves paths using XDG defaults and MNEMONIC_* environment overrides.
// It returns a validation error if MNEMONIC_MEMORIES_HOME is set to a relative path.
func GetMnemonicPaths() (MnemonicPaths, error) {
	xdg := GetXDGPaths()

	memoriesHome := os.Getenv("MNEMONIC_MEMORIES_HOME")
	if memoriesHome != "" && !filepath.IsAbs(memoriesHome) {
		return MnemonicPaths{}, errors.New("MNEMONIC_MEMORIES_HOME must be an absolute path")
	}

	if memoriesHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.Getenv("HOME")
		}
		memoriesHome = filepath.Join(home, ".mnemonic")
	}

	return MnemonicPaths{
		ConfigHome:   resolveMnemonicEnv("MNEMONIC_CONFIG_HOME", xdg.ConfigHome),
		DataHome:     resolveMnemonicEnv("MNEMONIC_DATA_HOME", xdg.DataHome),
		StateHome:    resolveMnemonicEnv("MNEMONIC_STATE_HOME", xdg.StateHome),
		CacheHome:    resolveMnemonicEnv("MNEMONIC_CACHE_HOME", xdg.CacheHome),
		MemoriesHome: memoriesHome,
	}, nil
}

func resolveMnemonicEnv(envKey string, fallback string) string {
	val := os.Getenv(envKey)
	if val != "" && filepath.IsAbs(val) {
		return val
	}
	return fallback
}
