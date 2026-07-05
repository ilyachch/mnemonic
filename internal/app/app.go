package app

import (
	"context"
	"log/slog"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	manifest "github.com/ilyachch/mnemonic/internal/format/manifest"
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
	Logger   *slog.Logger
}

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

	registryStore := registry.New(effective.MemoriesHome, func(path string) (*manifest.Manifest, error) {
		return manifest.ParseMnemonicManifestFromFile(path)
	}, func(data []byte) (*manifest.PointerFile, error) {
		return manifest.ParsePointerFile(data)
	})

	catalog := &catalogsvc.Service{
		MemoriesHome: effective.MemoriesHome,
		StateHome:    effective.StateHome,
		Registry:     registryStore,
	}

	b := &Bootstrap{
		Config: cfg,
		Paths:  effective,
		Services: Services{
			Catalog: catalog,
			Maint: &maintsvc.Service{
				Catalog: catalog,
			},
		},
	}

	b.Services.Maint.RuntimeFactory = func(ctx context.Context, k kb.KnowledgeBase, logger *slog.Logger) (maintsvc.Runtime, error) {
		_ = ctx
		return NewRuntimeApp(RuntimeInput{Config: cfg, KB: k, Logger: logger})
	}

	return b, nil
}

// Close shuts down app-owned resources.
func (b *Bootstrap) Close() error {
	return nil
}
