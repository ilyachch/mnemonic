package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// ManifestType identifies the project type in mnemonic.toml.
type ManifestType string

const (
	// ManifestTypeLocal marks a project as local.
	ManifestTypeLocal ManifestType = "local"
)

// ManifestFormat holds format-related manifest settings.
type ManifestFormat struct {
	LinksStyle string `toml:"links_style"`
}

// Manifest is the TOML schema stored in mnemonic.toml.
type Manifest struct {
	Version               int            `toml:"version"`
	ProjectID             string         `toml:"project_id"`
	Name                  string         `toml:"name"`
	Slug                  string         `toml:"slug"`
	Type                  ManifestType   `toml:"type"`
	MarkdownFormatVersion int            `toml:"markdown_format_version"`
	Description           string         `toml:"description"`
	CustomInstructions    string         `toml:"custom_instructions"`
	CreatedAt             int64          `toml:"created_at"`
	UpdatedAt             int64          `toml:"updated_at"`
	Format                ManifestFormat `toml:"format"`
	Layout                ManifestLayout `toml:"layout"`
	Generator             Generator      `toml:"generator"`
}

// ManifestLayout holds layout-related manifest settings.
type ManifestLayout struct {
	NotesGlob []string `toml:"notes_glob"`
	Ignore    []string `toml:"ignore"`
}

// Generator holds metadata about the generator that wrote the manifest.
type Generator struct {
	App        string `toml:"app"`
	AppVersion string `toml:"app_version"`
}

// PointerFile is the TOML schema for a local project pointer file.
type PointerFile struct {
	ManifestPath string `toml:"manifest_path"`
}

// Backward-compatible aliases for existing callsites during the migration.
type MnemonicManifest = Manifest
type MnemonicManifestLayout = ManifestLayout
type MnemonicGenerator = Generator

type manifestTOML struct {
	Version               int            `toml:"version"`
	ProjectID             string         `toml:"project_id"`
	Name                  string         `toml:"name"`
	Slug                  string         `toml:"slug"`
	Type                  ManifestType   `toml:"type"`
	MarkdownFormatVersion int            `toml:"markdown_format_version"`
	Description           string         `toml:"description"`
	CustomInstructions    string         `toml:"custom_instructions"`
	CreatedAt             int64          `toml:"created_at"`
	UpdatedAt             int64          `toml:"updated_at"`
	Format                ManifestFormat `toml:"format"`
	Layout                ManifestLayout `toml:"layout"`
	Generator             Generator      `toml:"generator"`
}

// IsLocal returns true when the manifest explicitly declares itself as local.
func (m *Manifest) IsLocal() bool {
	return m != nil && m.Type == ManifestTypeLocal
}

// New returns a schema-populated manifest with layout defaults.
func New() *Manifest {
	m := &Manifest{
		Version: 1,
		Layout: ManifestLayout{
			NotesGlob: []string{"**/*.md"},
			Ignore:    []string{"mnemonic.toml", ".trash/**"},
		},
	}
	m.ApplyDefaults()
	return m
}

// NewMnemonicManifest returns a schema-populated manifest with layout defaults.
func NewMnemonicManifest() *Manifest { return New() }

// ApplyDefaults populates the manifest defaults for unset optional fields.
func (m *Manifest) ApplyDefaults() {
	if m == nil {
		return
	}

	if m.Version == 0 {
		m.Version = 1
	}
	if m.Format.LinksStyle == "" {
		m.Format.LinksStyle = "wiki"
	}
	if len(m.Layout.NotesGlob) == 0 {
		m.Layout.NotesGlob = []string{"**/*.md"}
	}
	if len(m.Layout.Ignore) == 0 {
		m.Layout.Ignore = []string{"mnemonic.toml", ".trash/**"}
	}
}

// Validate checks the schema invariants expected for mnemonic.toml files.
func (m *Manifest) Validate() error {
	if m == nil {
		return errors.New("mnemonic manifest is nil")
	}
	if m.Version == 0 {
		return errors.New("mnemonic manifest version is required")
	}
	if m.Version != 1 {
		return fmt.Errorf("mnemonic manifest version %d is unsupported; expected 1", m.Version)
	}
	if m.ProjectID == "" {
		return errors.New("project_id is required")
	}
	if m.Name == "" {
		return errors.New("name is required")
	}
	if m.Slug == "" {
		return errors.New("slug is required")
	}
	if m.MarkdownFormatVersion <= 0 {
		return errors.New("markdown_format_version must be positive")
	}
	if m.CreatedAt <= 0 {
		return errors.New("created_at is required")
	}
	if m.UpdatedAt <= 0 {
		return errors.New("updated_at is required")
	}
	if m.Format.LinksStyle != "" && m.Format.LinksStyle != "wiki" && m.Format.LinksStyle != "regular" {
		return fmt.Errorf("format.links_style must be %q or %q, got %q", "wiki", "regular", m.Format.LinksStyle)
	}
	if len(m.Layout.NotesGlob) == 0 {
		return errors.New("layout.notes_glob is required")
	}
	if len(m.Layout.Ignore) == 0 {
		return errors.New("layout.ignore is required")
	}

	switch m.Type {
	case "", ManifestTypeLocal:
		return nil
	default:
		return fmt.Errorf("unknown type %q; expected empty (central) or %q", m.Type, ManifestTypeLocal)
	}
}

// MarshalTOML serializes mnemonic.toml.
func (m *Manifest) MarshalTOML() ([]byte, error) {
	copy := *m
	copy.ApplyDefaults()
	if err := copy.Validate(); err != nil {
		return nil, err
	}

	raw := manifestTOML{
		Version:               copy.Version,
		ProjectID:             copy.ProjectID,
		Name:                  copy.Name,
		Slug:                  copy.Slug,
		Type:                  copy.Type,
		MarkdownFormatVersion: copy.MarkdownFormatVersion,
		Description:           copy.Description,
		CustomInstructions:    copy.CustomInstructions,
		CreatedAt:             copy.CreatedAt,
		UpdatedAt:             copy.UpdatedAt,
		Format:                copy.Format,
		Layout:                copy.Layout,
		Generator:             copy.Generator,
	}
	return toml.Marshal(raw)
}

// ParseMnemonicManifestFromFile reads and parses a mnemonic.toml file from disk.
func ParseMnemonicManifestFromFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mnemonic.toml: %w", err)
	}
	return ParseMnemonicManifest(data)
}

// ParseMnemonicManifestFile reads and parses a mnemonic.toml file from disk.
func ParseMnemonicManifestFile(path string) (*Manifest, error) {
	return ParseMnemonicManifestFromFile(path)
}

// ParseMnemonicManifest parses mnemonic.toml.
func ParseMnemonicManifest(data []byte) (*Manifest, error) {
	raw := manifestTOML{
		Version:   1,
		Layout:    ManifestLayout{NotesGlob: []string{"**/*.md"}, Ignore: []string{"mnemonic.toml", ".trash/**"}},
		Generator: Generator{},
	}

	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("mnemonic.toml syntax error: %w", err)
	}

	result := &Manifest{
		Version:               raw.Version,
		ProjectID:             raw.ProjectID,
		Name:                  raw.Name,
		Slug:                  raw.Slug,
		Type:                  raw.Type,
		MarkdownFormatVersion: raw.MarkdownFormatVersion,
		Description:           raw.Description,
		CustomInstructions:    raw.CustomInstructions,
		CreatedAt:             raw.CreatedAt,
		UpdatedAt:             raw.UpdatedAt,
		Format:                raw.Format,
		Layout:                raw.Layout,
		Generator:             raw.Generator,
	}

	result.ApplyDefaults()
	if err := result.Validate(); err != nil {
		return nil, err
	}

	return result, nil
}

// WriteMnemonicManifest writes a mnemonic.toml file.
func WriteMnemonicManifest(path string, manifest *Manifest) error {
	data, err := manifest.MarshalTOML()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
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
		return nil, errors.New("manifest_path is required")
	}
	return &pf, nil
}

// WritePointerFile writes a pointer file to disk.
func WritePointerFile(path string, pf *PointerFile) error {
	if pf == nil {
		return errors.New("pointer file is required")
	}
	data, err := toml.Marshal(pf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
