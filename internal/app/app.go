package app

import (
	"github.com/ilyachch/mnemonic/internal/config"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
)

// Input configures app container creation.
type Input struct {
	CLI paths.CLIOverrides
}

// Bootstrap is the application container that wires config, paths, and services.
type Bootstrap struct {
	Config   *config.Config
	Paths    paths.EffectivePaths
	Services Services
}

// App is a compatibility alias for Bootstrap.
type App = Bootstrap

// New builds the application container from environment and config discovery.
func New(input Input) (*Bootstrap, error) {
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

	return &Bootstrap{
		Config: cfg,
		Paths:  effective,
		Services: Services{
			Catalog: &catalogsvc.Service{
				MemoriesHome: effective.MemoriesHome,
				StateHome:    effective.StateHome,
			},
			ProjectResolver: &FileResolver{
				MemoriesHome: effective.MemoriesHome,
			},
		},
	}, nil
}

// Close shuts down app-owned resources.
func (b *Bootstrap) Close() error {
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
