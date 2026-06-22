package app

import (
	"context"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/platform/config"
	"github.com/ilyachch/mnemonic/internal/platform/paths"
	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
	"github.com/ilyachch/mnemonic/internal/service/maintsvc"
	registry "github.com/ilyachch/mnemonic/internal/store/registry"
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

	registryStore := registry.New(effective.MemoriesHome, func(path string) (registry.ManifestData, error) {
		m, err := registry.ParseMnemonicManifestFromFile(path)
		if err != nil {
			return registry.ManifestData{}, err
		}
		return m, nil
	}, func(data []byte) (string, error) {
		pf, err := registry.ParsePointerFile(data)
		if err != nil {
			return "", err
		}
		return pf.ManifestPath, nil
	})

	catalog := &catalogsvc.Service{
		MemoriesHome: effective.MemoriesHome,
		StateHome:    effective.StateHome,
		Registry:     registryStore,
	}

	return &Bootstrap{
		Config: cfg,
		Paths:  effective,
		Services: Services{
			Catalog: catalog,
			Maint: &maintsvc.Service{
				Catalog: catalog,
				RuntimeFactory: func(ctx context.Context, k kb.KnowledgeBase) (maintsvc.Runtime, error) {
					_ = ctx
					return NewRuntimeApp(RuntimeInput{Config: cfg, KB: k})
				},
			},
			ProjectResolver: &FileResolver{
				Registry: registryStore,
			},
		},
	}, nil
}

// Close shuts down app-owned resources.
func (b *Bootstrap) Close() error {
	return nil
}
