package config

import (
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/platform/paths"
)

const configFileName = "config.toml"

// DiscoverConfigFile returns the first existing config file following the
// precedence required by the MVP. When no config file exists, it returns an
// empty path and a nil error so callers can safely fall back to DefaultConfig.
func DiscoverConfigFile(explicitPath string) (string, error) {
	candidates, err := configCandidates(explicitPath)
	if err != nil {
		return "", err
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		info, err := os.Stat(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if info.IsDir() {
			continue
		}

		return candidate, nil
	}

	return "", nil
}

func configCandidates(explicitPath string) ([]string, error) {
	configFile, err := normalizedFilePath(explicitPath)
	if err != nil {
		return nil, err
	}

	mnemonicConfigHome, err := directoryFromEnv("MNEMONIC_CONFIG_HOME", "")
	if err != nil {
		return nil, err
	}

	xdgConfigHome, err := directoryFromEnv("XDG_CONFIG_HOME", defaultXDGConfigHome())
	if err != nil {
		return nil, err
	}

	envConfigFile, err := envFilePath("MNEMONIC_CONFIG_FILE")
	if err != nil {
		return nil, err
	}

	return []string{
		configFile,
		envConfigFile,
		joinConfigPath(mnemonicConfigHome),
		joinConfigPath(xdgConfigHome),
		joinConfigPath(defaultHomeConfigHome()),
	}, nil
}

func normalizedFilePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	return paths.ExpandPath(path)
}

func envFilePath(envKey string) (string, error) {
	value := os.Getenv(envKey)
	if value == "" {
		return "", nil
	}

	path, err := paths.ExpandPath(value)
	if err != nil {
		return "", err
	}

	return path, nil
}

func directoryFromEnv(envKey string, fallback string) (string, error) {
	value := os.Getenv(envKey)
	if value == "" {
		return fallback, nil
	}

	path, err := paths.ExpandPath(value)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(path) {
		return fallback, nil
	}

	return path, nil
}

func defaultXDGConfigHome() string {
	return filepath.Join(userHomeDirFallback(), ".config")
}

func defaultHomeConfigHome() string {
	return filepath.Join(userHomeDirFallback(), ".config")
}

func joinConfigPath(configHome string) string {
	if configHome == "" {
		return ""
	}

	return filepath.Join(configHome, "mnemonic", configFileName)
}

func userHomeDirFallback() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home
	}

	if home := os.Getenv("HOME"); home != "" {
		return home
	}

	return ""
}
