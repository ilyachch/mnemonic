package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	mnemonicfs "github.com/ilyachch/mnemonic/internal/platform/fs"
	toml "github.com/pelletier/go-toml/v2"
)

// PointerFile is the TOML schema for a local project pointer file
// stored at ~/.mnemonic/<slug>.toml.
type PointerFile struct {
	ManifestPath string `toml:"manifest_path"`
}

// ParsePointerFile reads a pointer file from raw TOML data.
func ParsePointerFile(data []byte) (*PointerFile, error) {
	var pf PointerFile
	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&pf); err != nil {
		return nil, fmt.Errorf("pointer file syntax error: %w", err)
	}
	if pf.ManifestPath == "" {
		return nil, fmt.Errorf("manifest_path is required")
	}
	return &pf, nil
}

// MarshalTOML serializes the pointer file to TOML.
func (p *PointerFile) MarshalTOML() ([]byte, error) {
	if p.ManifestPath == "" {
		return nil, fmt.Errorf("manifest_path is required")
	}
	return toml.Marshal(p)
}

// WritePointerFile writes a pointer file to disk.
func WritePointerFile(path string, pf *PointerFile) error {
	data, err := pf.MarshalTOML()
	if err != nil {
		return err
	}

	return writeFile(path, data, 0o644)
}

// writeFile writes data to a file, creating parent directories as needed.
func writeFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	return mnemonicfs.WriteFile(path, data, perm)
}
