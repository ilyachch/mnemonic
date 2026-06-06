package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// CLIOverrides captures path-related CLI flags that should win over env and config.
type CLIOverrides struct {
	ConfigFile   string
	ConfigHome   string
	DataHome     string
	StateHome    string
	CacheHome    string
	MemoriesHome string
}

// EffectiveInput groups all layers that participate in path resolution.
type EffectiveInput struct {
	CLI                   CLIOverrides
	ConfigFile            string
	RawConfigMemoriesHome string
}

// EffectivePaths is the fully resolved path set used by higher-level commands.
type EffectivePaths struct {
	ConfigFile            string `json:"config_file"`
	ConfigHome            string `json:"config_home"`
	DataHome              string `json:"data_home"`
	StateHome             string `json:"state_home"`
	CacheHome             string `json:"cache_home"`
	MemoriesHome          string `json:"memories_home"`
	RawConfigMemoriesHome string `json:"-"`
}

// ResolveEffectivePaths merges CLI flags, MNEMONIC_* env, config.toml values,
// and XDG fallbacks into a single deterministic path set.
func ResolveEffectivePaths(input EffectiveInput) (EffectivePaths, error) {
	mnemonicEnv, err := GetMnemonicPaths()
	if err != nil {
		return EffectivePaths{}, err
	}

	configFile, err := chooseConfiguredPath(input.CLI.ConfigFile, input.ConfigFile)
	if err != nil {
		return EffectivePaths{}, err
	}

	configHome, err := choosePath(input.CLI.ConfigHome, mnemonicEnv.ConfigHome)
	if err != nil {
		return EffectivePaths{}, err
	}
	dataHome, err := choosePath(input.CLI.DataHome, mnemonicEnv.DataHome)
	if err != nil {
		return EffectivePaths{}, err
	}
	stateHome, err := choosePath(input.CLI.StateHome, mnemonicEnv.StateHome)
	if err != nil {
		return EffectivePaths{}, err
	}
	cacheHome, err := choosePath(input.CLI.CacheHome, mnemonicEnv.CacheHome)
	if err != nil {
		return EffectivePaths{}, err
	}

	memoriesHome, rawConfigMemoriesHome, err := chooseMemoriesHome(input.CLI.MemoriesHome, input.RawConfigMemoriesHome, mnemonicEnv.MemoriesHome)
	if err != nil {
		return EffectivePaths{}, err
	}

	return EffectivePaths{
		ConfigFile:            configFile,
		ConfigHome:            configHome,
		DataHome:              dataHome,
		StateHome:             stateHome,
		CacheHome:             cacheHome,
		MemoriesHome:          memoriesHome,
		RawConfigMemoriesHome: rawConfigMemoriesHome,
	}, nil
}

func chooseConfiguredPath(cliValue, configuredValue string) (string, error) {
	if cliValue != "" {
		return normalizeAbsolutePath(cliValue)
	}
	if configuredValue != "" {
		return normalizeAbsolutePath(configuredValue)
	}
	return "", nil
}

func choosePath(cliValue, envValue string) (string, error) {
	if cliValue != "" {
		return normalizeAbsolutePath(cliValue)
	}
	if envValue != "" {
		return normalizeAbsolutePath(envValue)
	}
	return "", nil
}

func chooseMemoriesHome(cliValue, configValue, envValue string) (string, string, error) {
	if cliValue != "" {
		path, err := normalizeAbsolutePath(cliValue)
		return path, "", err
	}

	if mnemonicValue := os.Getenv("MNEMONIC_MEMORIES_HOME"); mnemonicValue != "" {
		path, err := normalizeAbsolutePath(mnemonicValue)
		return path, "", err
	}

	if configValue != "" {
		path, err := normalizeAbsolutePath(configValue)
		if err != nil {
			return "", "", err
		}
		return path, configValue, nil
	}

	return envValue, "", nil
}

func normalizeAbsolutePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	expanded, err := ExpandPath(path)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}

	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path %q: %w", path, err)
	}

	return filepath.Clean(abs), nil
}
