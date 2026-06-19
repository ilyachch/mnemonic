package project

import (
	"bytes"
	"fmt"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

// ManifestKind identifies the supported kinds in mnemonic.toml.
type ManifestKind string

const (
	// ManifestKindRegular stores a regular non-local project.
	ManifestKindRegular ManifestKind = "regular"
	// ManifestKindDetached stores a detached project rooted in the memories home.
	ManifestKindDetached ManifestKind = "detached"
)

// MnemonicManifest is the TOML schema stored in mnemonic.toml for non-local projects.
type MnemonicManifest struct {
	Version               int                    `toml:"version"`
	ProjectID             string                 `toml:"project_id"`
	Name                  string                 `toml:"name"`
	Slug                  string                 `toml:"slug"`
	Kind                  ManifestKind           `toml:"kind"`
	MarkdownFormatVersion int                    `toml:"markdown_format_version"`
	Description           string                 `toml:"description,omitempty"`
	CreatedAt             time.Time              `toml:"created_at"`
	UpdatedAt             time.Time              `toml:"updated_at"`
	Layout                MnemonicManifestLayout `toml:"layout"`
	Generator             MnemonicGenerator      `toml:"generator"`
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

	switch m.Kind {
	case ManifestKindRegular, ManifestKindDetached:
		return nil
	case "":
		return fmt.Errorf("kind is required")
	case "local":
		return fmt.Errorf("kind %q is not allowed in mnemonic.toml", m.Kind)
	default:
		return fmt.Errorf("unknown kind %q", m.Kind)
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
		Kind:                  copy.Kind,
		MarkdownFormatVersion: copy.MarkdownFormatVersion,
		Description:           copy.Description,
		CreatedAt:             newTOMLTime(copy.CreatedAt),
		UpdatedAt:             newTOMLTime(copy.UpdatedAt),
		Layout:                copy.Layout,
		Generator:             copy.Generator,
	}

	return toml.Marshal(raw)
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
		Kind:                  raw.Kind,
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
	Kind                  ManifestKind           `toml:"kind"`
	MarkdownFormatVersion int                    `toml:"markdown_format_version"`
	Description           string                 `toml:"description,omitempty"`
	CreatedAt             tomlTime               `toml:"created_at"`
	UpdatedAt             tomlTime               `toml:"updated_at"`
	Layout                MnemonicManifestLayout `toml:"layout"`
	Generator             MnemonicGenerator      `toml:"generator"`
}
