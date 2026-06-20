package app

import (
	"github.com/ilyachch/mnemonic/internal/config"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/registry"
)

// Input configures app container creation.
type Input struct {
	CLI paths.CLIOverrides
}

// App is the application container that wires config, paths, and services.
type App struct {
	Config   *config.Config
	Paths    paths.EffectivePaths
	Services Services
}

// New builds the application container from environment and config discovery.
func New(input Input) (*App, error) {
	discoveredConfigPath, err := config.DiscoverConfigFile(input.CLI.ConfigFile)
	if err != nil {
		return nil, err
	}

	cfg := config.DefaultConfig()
	if discoveredConfigPath != "" {
		cfg, err = config.LoadConfig(discoveredConfigPath)
		if err != nil {
			return nil, err
		}
	}

	effective, err := paths.ResolveEffectivePaths(paths.EffectiveInput{
		CLI:                   input.CLI,
		ConfigFile:            discoveredConfigPath,
		RawConfigMemoriesHome: cfg.Paths.MemoriesHome,
	})
	if err != nil {
		return nil, err
	}

	// Wire the registry parsers using the project package.
	wireRegistryParsers()

	return &App{
		Config: cfg,
		Paths:  effective,
		Services: Services{
			ProjectResolver: &FileResolver{
				MemoriesHome: effective.MemoriesHome,
			},
		},
	}, nil
}

// Close shuts down app-owned resources.
func (a *App) Close() error {
	return nil
}

func wireRegistryParsers() {
	registry.DefaultManifestParser = func(path string) (registry.ManifestData, error) {
		m, err := project.ParseMnemonicManifestFromFile(path)
		if err != nil {
			return registry.ManifestData{}, err
		}
		typ := "central"
		if m.IsLocal() {
			typ = "local"
		}
		return registry.ManifestData{
			ProjectID: m.ProjectID,
			Name:      m.Name,
			Slug:      m.Slug,
			Type:      typ,
		}, nil
	}

	registry.DefaultPointerParser = func(data []byte) (string, error) {
		pf, err := project.ParsePointerFile(data)
		if err != nil {
			return "", err
		}
		return pf.ManifestPath, nil
	}
}
