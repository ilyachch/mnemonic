package project

import (
	"fmt"
	"time"

	"bytes"

	toml "github.com/pelletier/go-toml/v2"
)

// ProjectKind identifies the supported kinds in .mnemonic files.
type ProjectKind string

const (
	// ProjectKindRegular stores a project that uses a regular memories directory.
	ProjectKindRegular ProjectKind = "regular"
	// ProjectKindLocal stores a project backed by a local `.mnemonic-memories` directory.
	ProjectKindLocal ProjectKind = "local"
)

// MnemonicFile is the TOML schema stored in `.mnemonic`.
type MnemonicFile struct {
	Version   int               `toml:"version"`
	CreatedAt time.Time         `toml:"created_at"`
	UpdatedAt time.Time         `toml:"updated_at"`
	Projects  []MnemonicProject `toml:"projects"`
}

// MnemonicProject is a single `[[projects]]` entry in `.mnemonic`.
type MnemonicProject struct {
	ID                    string      `toml:"id"`
	Name                  string      `toml:"name"`
	Slug                  string      `toml:"slug"`
	Kind                  ProjectKind `toml:"kind"`
	MemoriesPath          string      `toml:"memories_path"`
	MarkdownFormatVersion int         `toml:"markdown_format_version"`
	CreatedAt             time.Time   `toml:"created_at"`
	UpdatedAt             time.Time   `toml:"updated_at"`
}

// NewMnemonicFile returns a schema-populated file with UTC timestamps.
func NewMnemonicFile() *MnemonicFile {
	now := NowUTC()
	return &MnemonicFile{
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// MarshalTOML serializes the .mnemonic schema into TOML.
func (f *MnemonicFile) MarshalTOML() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}

	raw := mnemonicFileTOML{
		Version:   &f.Version,
		CreatedAt: newTOMLTime(f.CreatedAt),
		UpdatedAt: newTOMLTime(f.UpdatedAt),
		Projects:  make([]mnemonicProjectTOML, len(f.Projects)),
	}
	for i := range f.Projects {
		raw.Projects[i] = mnemonicProjectTOML{
			ID:                    f.Projects[i].ID,
			Name:                  f.Projects[i].Name,
			Slug:                  f.Projects[i].Slug,
			Kind:                  f.Projects[i].Kind,
			MemoriesPath:          f.Projects[i].MemoriesPath,
			MarkdownFormatVersion: f.Projects[i].MarkdownFormatVersion,
			CreatedAt:             newTOMLTime(f.Projects[i].CreatedAt),
			UpdatedAt:             newTOMLTime(f.Projects[i].UpdatedAt),
		}
	}

	return toml.Marshal(raw)
}

// ParseMnemonicFile parses a `.mnemonic` TOML document.
func ParseMnemonicFile(data []byte) (*MnemonicFile, error) {
	raw := mnemonicFileTOML{}
	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf(".mnemonic syntax error: %w", err)
	}

	if raw.Version == nil {
		return nil, fmt.Errorf("mnemonic file version is required")
	}

	result := &MnemonicFile{
		Version:   *raw.Version,
		CreatedAt: raw.CreatedAt.Time(),
		UpdatedAt: raw.UpdatedAt.Time(),
		Projects:  make([]MnemonicProject, len(raw.Projects)),
	}
	for i := range raw.Projects {
		result.Projects[i] = MnemonicProject{
			ID:                    raw.Projects[i].ID,
			Name:                  raw.Projects[i].Name,
			Slug:                  raw.Projects[i].Slug,
			Kind:                  raw.Projects[i].Kind,
			MemoriesPath:          raw.Projects[i].MemoriesPath,
			MarkdownFormatVersion: raw.Projects[i].MarkdownFormatVersion,
			CreatedAt:             raw.Projects[i].CreatedAt.Time(),
			UpdatedAt:             raw.Projects[i].UpdatedAt.Time(),
		}
	}

	if err := result.Validate(); err != nil {
		return nil, err
	}

	return result, nil
}

// Validate checks the schema invariants expected for .mnemonic files.
func (f *MnemonicFile) Validate() error {
	if f == nil {
		return fmt.Errorf("mnemonic file is nil")
	}
	if f.Version == 0 {
		return fmt.Errorf("mnemonic file version is required")
	}
	if f.Version != 1 {
		return fmt.Errorf("mnemonic file version %d is unsupported; expected 1", f.Version)
	}
	if f.CreatedAt.IsZero() {
		return fmt.Errorf("mnemonic file created_at is required")
	}
	if f.UpdatedAt.IsZero() {
		return fmt.Errorf("mnemonic file updated_at is required")
	}

	for i := range f.Projects {
		if err := f.Projects[i].Validate(); err != nil {
			return fmt.Errorf("projects[%d]: %w", i, err)
		}
	}

	return nil
}

// Validate checks the schema invariants for a single project entry.
func (p *MnemonicProject) Validate() error {
	if p == nil {
		return fmt.Errorf("project is nil")
	}
	if p.ID == "" {
		return fmt.Errorf("id is required")
	}
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if p.Slug == "" {
		return fmt.Errorf("slug is required")
	}
	if p.MemoriesPath == "" {
		return fmt.Errorf("memories_path is required")
	}
	if p.MarkdownFormatVersion <= 0 {
		return fmt.Errorf("markdown_format_version must be positive")
	}
	if p.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	if p.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}

	switch p.Kind {
	case ProjectKindRegular, ProjectKindLocal:
		return nil
	case "detached":
		return fmt.Errorf("kind %q is not allowed in .mnemonic", p.Kind)
	case "":
		return fmt.Errorf("kind is required")
	default:
		return fmt.Errorf("unknown kind %q", p.Kind)
	}
}

type mnemonicFileTOML struct {
	Version   *int                  `toml:"version"`
	CreatedAt tomlTime              `toml:"created_at"`
	UpdatedAt tomlTime              `toml:"updated_at"`
	Projects  []mnemonicProjectTOML `toml:"projects"`
}

type mnemonicProjectTOML struct {
	ID                    string      `toml:"id"`
	Name                  string      `toml:"name"`
	Slug                  string      `toml:"slug"`
	Kind                  ProjectKind `toml:"kind"`
	MemoriesPath          string      `toml:"memories_path"`
	MarkdownFormatVersion int         `toml:"markdown_format_version"`
	CreatedAt             tomlTime    `toml:"created_at"`
	UpdatedAt             tomlTime    `toml:"updated_at"`
}
