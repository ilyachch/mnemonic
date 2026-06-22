package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// LoadConfig loads a TOML config file into a Config value.
//
// The loader intentionally supports only the schema used by the MVP. Unknown
// fields are rejected, version must be explicitly set to 1 when a file exists,
// and parse errors are reported with line context.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return DefaultConfig(), nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file %q does not exist", path)
		}
		return nil, fmt.Errorf("stat config file %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("config file %q is a directory", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	raw := configFile{
		Paths:   DefaultConfig().Paths,
		Notes:   DefaultConfig().Notes,
		Index:   DefaultConfig().Index,
		Output:  DefaultConfig().Output,
		Logging: DefaultConfig().Logging,
	}

	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return nil, configSyntaxError(path, err)
	}

	if raw.Version == nil {
		return nil, fmt.Errorf("config file %q must set version = 1", path)
	}
	if *raw.Version != 1 {
		return nil, fmt.Errorf("config syntax error in %q: unsupported config version %d", path, *raw.Version)
	}

	return &Config{
		Version: *raw.Version,
		Paths:   raw.Paths,
		Notes:   raw.Notes,
		Index:   raw.Index,
		Output:  raw.Output,
		Logging: raw.Logging,
	}, nil
}

func configSyntaxError(path string, err error) error {
	var strictErr *toml.StrictMissingError
	if errors.As(err, &strictErr) && len(strictErr.Errors) > 0 {
		key := strictErr.Errors[0].Key()
		if len(key) > 0 {
			return fmt.Errorf("config syntax error in %q: unknown key %q", path, strings.Join(key, "."))
		}
	}

	return fmt.Errorf("config syntax error in %q: %w", path, err)
}

type configFile struct {
	Version *int          `toml:"version"`
	Paths   PathsConfig   `toml:"paths"`
	Notes   NotesConfig   `toml:"notes"`
	Index   IndexConfig   `toml:"index"`
	Output  OutputConfig  `toml:"output"`
	Logging LoggingConfig `toml:"logging"`
}
