package app

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/config"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/registryschema"
	_ "modernc.org/sqlite"
)

// Input configures app container creation.
type Input struct {
	CLI paths.CLIOverrides
}

// App is the application container that wires config, paths, registry and services.
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

	db, err := openRegistryDB()
	if err != nil {
		return nil, err
	}
	if err := applyRegistrySchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &App{
		Config: cfg,
		Paths:  effective,
		Services: Services{
			Registry: db,
		},
	}, nil
}

// Close shuts down app-owned resources.
func (a *App) Close() error {
	if a == nil || a.Services.Registry == nil {
		return nil
	}
	if err := a.Services.Registry.Close(); err != nil {
		return fmt.Errorf("close registry: %w", err)
	}
	return nil
}

func openRegistryDB() (*sql.DB, error) {
	mnemonicPaths, err := paths.GetMnemonicPaths()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(mnemonicPaths.DataHome, "mnemonic", "registry.sqlite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create registry directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open registry database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping registry database: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return db, nil
}

func applyRegistrySchema(db *sql.DB) error {
	return registryschema.Apply(db)
}
