package project

import (
	"bytes"
	"fmt"
	"os"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

// ManifestType identifies the project type in mnemonic.toml.
// If omitted (empty), the project is treated as central.
// Only "local" is a valid explicit value.
type ManifestType string

const (
	// ManifestTypeLocal marks a project as local (backed by .mnemonic-memories in workspace).
	ManifestTypeLocal ManifestType = "local"
)

// MnemonicManifest is the TOML schema stored in mnemonic.toml.
type MnemonicManifest struct {
	Version               int                    `toml:"version"`
	ProjectID             string                 `toml:"project_id"`
	Name                  string                 `toml:"name"`
	Slug                  string                 `toml:"slug"`
	Type                  ManifestType           `toml:"type"`
	MarkdownFormatVersion int                    `toml:"markdown_format_version"`
	Description           string                 `toml:"description,omitempty"`
	CreatedAt             time.Time              `toml:"created_at"`
	UpdatedAt             time.Time              `toml:"updated_at"`
	Layout                MnemonicManifestLayout `toml:"layout"`
	Generator             MnemonicGenerator      `toml:"generator"`
}

// IsLocal returns true when the manifest explicitly declares itself as local.
func (m *MnemonicManifest) IsLocal() bool {
	return m != nil && m.Type == ManifestTypeLocal
}

// MnemonicManifestLayout holds layout-related manifest settings.
type MnemonicManifestLayout struct {
	NotesGlob []string `toml:"notes_glob"`
	Ignore    []string `toml:"ignore"`
}

// MnemonicGenerator holds metadata about the generator that wrote the manifest.
type MnemonicGenerator struct {
	App        string `toml:"app"`
	AppVersion string `toml:"app_version"`
}

// NewMnemonicManifest returns a schema-populated manifest with layout defaults.
func NewMnemonicManifest() *MnemonicManifest {
	m := &MnemonicManifest{
		Version: 1,
		Layout: MnemonicManifestLayout{
			NotesGlob: []string{"**/*.md"},
			Ignore:    []string{"mnemonic.toml", ".trash/**"},
		},
	}
	m.ApplyDefaults()
	return m
}

// ApplyDefaults populates the manifest defaults for unset optional fields.
func (m *MnemonicManifest) ApplyDefaults() {
	if m == nil {
		return
	}

	if m.Version == 0 {
		m.Version = 1
	}
	if len(m.Layout.NotesGlob) == 0 {
		m.Layout.NotesGlob = []string{"**/*.md"}
	}
	if len(m.Layout.Ignore) == 0 {
		m.Layout.Ignore = []string{"mnemonic.toml", ".trash/**"}
	}
}

// Validate checks the schema invariants expected for mnemonic.toml files.
func (m *MnemonicManifest) Validate() error {
	if m == nil {
		return fmt.Errorf("mnemonic manifest is nil")
	}
	if m.Version == 0 {
		return fmt.Errorf("mnemonic manifest version is required")
	}
	if m.Version != 1 {
		return fmt.Errorf("mnemonic manifest version %d is unsupported; expected 1", m.Version)
	}
	if m.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Slug == "" {
		return fmt.Errorf("slug is required")
	}
	if m.MarkdownFormatVersion <= 0 {
		return fmt.Errorf("markdown_format_version must be positive")
	}
	if m.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	if m.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}
	if len(m.Layout.NotesGlob) == 0 {
		return fmt.Errorf("layout.notes_glob is required")
	}
	if len(m.Layout.Ignore) == 0 {
		return fmt.Errorf("layout.ignore is required")
	}

	switch m.Type {
	case "", ManifestTypeLocal:
		return nil
	default:
		return fmt.Errorf("unknown type %q; expected empty (central) or %q", m.Type, ManifestTypeLocal)
	}
}

// MarshalTOML serializes mnemonic.toml.
func (m *MnemonicManifest) MarshalTOML() ([]byte, error) {
	copy := *m
	copy.ApplyDefaults()

	if err := copy.Validate(); err != nil {
		return nil, err
	}

	raw := mnemonicManifestTOML{
		Version:               copy.Version,
		ProjectID:             copy.ProjectID,
		Name:                  copy.Name,
		Slug:                  copy.Slug,
		Type:                  copy.Type,
		MarkdownFormatVersion: copy.MarkdownFormatVersion,
		Description:           copy.Description,
		CreatedAt:             newTOMLTime(copy.CreatedAt),
		UpdatedAt:             newTOMLTime(copy.UpdatedAt),
		Layout:                copy.Layout,
		Generator:             copy.Generator,
	}

	return toml.Marshal(raw)
}

// ParseMnemonicManifestFromFile reads and parses a mnemonic.toml file from disk.
func ParseMnemonicManifestFromFile(path string) (*MnemonicManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mnemonic.toml: %w", err)
	}
	return ParseMnemonicManifest(data)
}

// ParseMnemonicManifest parses mnemonic.toml.
func ParseMnemonicManifest(data []byte) (*MnemonicManifest, error) {
	raw := mnemonicManifestTOML{
		Version:   1,
		Layout:    MnemonicManifestLayout{NotesGlob: []string{"**/*.md"}, Ignore: []string{"mnemonic.toml", ".trash/**"}},
		Generator: MnemonicGenerator{},
	}

	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("mnemonic.toml syntax error: %w", err)
	}

	result := &MnemonicManifest{
		Version:               raw.Version,
		ProjectID:             raw.ProjectID,
		Name:                  raw.Name,
		Slug:                  raw.Slug,
		Type:                  raw.Type,
		MarkdownFormatVersion: raw.MarkdownFormatVersion,
		Description:           raw.Description,
		CreatedAt:             raw.CreatedAt.Time(),
		UpdatedAt:             raw.UpdatedAt.Time(),
		Layout:                raw.Layout,
		Generator:             raw.Generator,
	}

	result.ApplyDefaults()
	if err := result.Validate(); err != nil {
		return nil, err
	}

	return result, nil
}

type mnemonicManifestTOML struct {
	Version               int                    `toml:"version"`
	ProjectID             string                 `toml:"project_id"`
	Name                  string                 `toml:"name"`
	Slug                  string                 `toml:"slug"`
	Type                  ManifestType           `toml:"type"`
	MarkdownFormatVersion int                    `toml:"markdown_format_version"`
	Description           string                 `toml:"description,omitempty"`
	CreatedAt             tomlTime               `toml:"created_at"`
	UpdatedAt             tomlTime               `toml:"updated_at"`
	Layout                MnemonicManifestLayout `toml:"layout"`
	Generator             MnemonicGenerator      `toml:"generator"`
}
