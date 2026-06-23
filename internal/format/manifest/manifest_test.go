package manifest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixedTime(hour, min int) time.Time {
	return time.Date(2026, time.June, 23, hour, min, 0, 0, time.UTC)
}

func validManifest() *Manifest {
	return &Manifest{
		Version:               1,
		ProjectID:             "550e8400-e29b-41d4-a716-446655440000",
		Name:                  "Test Project",
		Slug:                  "test-project",
		Type:                  ManifestTypeLocal,
		MarkdownFormatVersion: 1,
		Description:           "A test project",
		CreatedAt:             fixedTime(10, 0),
		UpdatedAt:             fixedTime(10, 0),
		Layout: ManifestLayout{
			NotesGlob: []string{"**/*.md"},
			Ignore:    []string{"mnemonic.toml", ".trash/**"},
		},
		Generator: Generator{
			App:        "mnemonic",
			AppVersion: "1.0.0",
		},
	}
}

// ── New / NewMnemonicManifest ──────────────────────────────────────────

func TestNew(t *testing.T) {
	m := New()
	require.NotNil(t, m)
	assert.Equal(t, 1, m.Version)
	assert.Equal(t, []string{"**/*.md"}, m.Layout.NotesGlob)
	assert.Equal(t, []string{"mnemonic.toml", ".trash/**"}, m.Layout.Ignore)
	// CreatedAt and UpdatedAt are not set by New() — they are zero
	assert.True(t, m.CreatedAt.IsZero())
	assert.True(t, m.UpdatedAt.IsZero())
}

func TestNewMnemonicManifest(t *testing.T) {
	m := NewMnemonicManifest()
	require.NotNil(t, m)
	assert.Equal(t, 1, m.Version)
	assert.Equal(t, []string{"**/*.md"}, m.Layout.NotesGlob)
}

// ── ApplyDefaults ──────────────────────────────────────────────────────

func TestApplyDefaults_FillsMissing(t *testing.T) {
	m := &Manifest{}
	m.ApplyDefaults()
	assert.Equal(t, 1, m.Version)
	assert.Equal(t, []string{"**/*.md"}, m.Layout.NotesGlob)
	assert.Equal(t, []string{"mnemonic.toml", ".trash/**"}, m.Layout.Ignore)
}

func TestApplyDefaults_PreservesExisting(t *testing.T) {
	m := &Manifest{
		Version: 2,
		Layout: ManifestLayout{
			NotesGlob: []string{"docs/*.md"},
			Ignore:    []string{"secret.md"},
		},
	}
	m.ApplyDefaults()
	assert.Equal(t, 2, m.Version)
	assert.Equal(t, []string{"docs/*.md"}, m.Layout.NotesGlob)
	assert.Equal(t, []string{"secret.md"}, m.Layout.Ignore)
}

func TestApplyDefaults_NilSafe(t *testing.T) {
	var m *Manifest
	// Should not panic
	m.ApplyDefaults()
}

// ── IsLocal ────────────────────────────────────────────────────────────

func TestIsLocal_True(t *testing.T) {
	m := validManifest()
	m.Type = ManifestTypeLocal
	assert.True(t, m.IsLocal())
}

func TestIsLocal_FalseForEmpty(t *testing.T) {
	m := validManifest()
	m.Type = ""
	assert.False(t, m.IsLocal())
}

func TestIsLocal_NilSafe(t *testing.T) {
	var m *Manifest
	assert.False(t, m.IsLocal())
}

// ── Validate ───────────────────────────────────────────────────────────

func TestValidate_Valid(t *testing.T) {
	m := validManifest()
	require.NoError(t, m.Validate())
}

func TestValidate_ValidCentral(t *testing.T) {
	m := validManifest()
	m.Type = "" // central
	require.NoError(t, m.Validate())
}

func TestValidate_Nil(t *testing.T) {
	var m *Manifest
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestValidate_VersionZero(t *testing.T) {
	m := validManifest()
	m.Version = 0
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestValidate_VersionUnsupported(t *testing.T) {
	m := validManifest()
	m.Version = 99
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestValidate_MissingProjectID(t *testing.T) {
	m := validManifest()
	m.ProjectID = ""
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project_id")
}

func TestValidate_MissingName(t *testing.T) {
	m := validManifest()
	m.Name = ""
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestValidate_MissingSlug(t *testing.T) {
	m := validManifest()
	m.Slug = ""
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "slug")
}

func TestValidate_UnknownType(t *testing.T) {
	m := validManifest()
	m.Type = "remote"
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

func TestValidate_MissingMarkdownFormatVersion(t *testing.T) {
	m := validManifest()
	m.MarkdownFormatVersion = 0
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "markdown_format_version")
}

func TestValidate_MissingCreatedAt(t *testing.T) {
	m := validManifest()
	m.CreatedAt = time.Time{}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "created_at")
}

func TestValidate_MissingUpdatedAt(t *testing.T) {
	m := validManifest()
	m.UpdatedAt = time.Time{}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updated_at")
}

func TestValidate_EmptyNotesGlob(t *testing.T) {
	m := validManifest()
	m.Layout.NotesGlob = nil
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notes_glob")
}

func TestValidate_EmptyIgnore(t *testing.T) {
	m := validManifest()
	m.Layout.Ignore = nil
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ignore")
}

// ── MarshalTOML / Parse round-trip ─────────────────────────────────────

func TestMarshalTOML_RoundTrip(t *testing.T) {
	original := validManifest()

	data, err := original.MarshalTOML()
	require.NoError(t, err)

	parsed, err := ParseMnemonicManifest(data)
	require.NoError(t, err)

	assert.Equal(t, original.ProjectID, parsed.ProjectID)
	assert.Equal(t, original.Name, parsed.Name)
	assert.Equal(t, original.Slug, parsed.Slug)
	assert.Equal(t, original.Type, parsed.Type)
	assert.Equal(t, original.MarkdownFormatVersion, parsed.MarkdownFormatVersion)
	assert.Equal(t, original.Description, parsed.Description)
	assert.Equal(t, original.Generator.App, parsed.Generator.App)
	assert.Equal(t, original.Generator.AppVersion, parsed.Generator.AppVersion)
	// Times should match within the second (RFC3339 precision)
	assert.WithinDuration(t, original.CreatedAt, parsed.CreatedAt, time.Second)
	assert.WithinDuration(t, original.UpdatedAt, parsed.UpdatedAt, time.Second)
}

func TestMarshalTOML_InvalidManifest(t *testing.T) {
	m := &Manifest{}
	_, err := m.MarshalTOML()
	require.Error(t, err)
}

// ── ParseMnemonicManifest ─────────────────────────────────────────────

func TestParseMnemonicManifest_ValidTOML(t *testing.T) {
	tomlData := `version = 1
project_id = "550e8400-e29b-41d4-a716-446655440000"
name = "My Project"
slug = "my-project"
markdown_format_version = 1
created_at = "2026-06-23T10:00:00Z"
updated_at = "2026-06-23T10:00:00Z"
`
	m, err := ParseMnemonicManifest([]byte(tomlData))
	require.NoError(t, err)
	assert.Equal(t, "My Project", m.Name)
	assert.Equal(t, "my-project", m.Slug)
}

func TestParseMnemonicManifest_UnknownFields(t *testing.T) {
	tomlData := `version = 1
project_id = "550e8400-e29b-41d4-a716-446655440000"
name = "My Project"
slug = "my-project"
markdown_format_version = 1
created_at = "2026-06-23T10:00:00Z"
updated_at = "2026-06-23T10:00:00Z"
unknown_field = "should fail"
`
	_, err := ParseMnemonicManifest([]byte(tomlData))
	require.Error(t, err)
}

func TestParseMnemonicManifest_InvalidTOML(t *testing.T) {
	_, err := ParseMnemonicManifest([]byte("not valid toml {{{{{{"))
	require.Error(t, err)
}

func TestParseMnemonicManifest_FullTOML(t *testing.T) {
	tomlData := `version = 1
project_id = "550e8400-e29b-41d4-a716-446655440000"
name = "Full Project"
slug = "full-project"
type = "local"
markdown_format_version = 1
description = "A fully specified project"
created_at = "2026-06-23T10:00:00Z"
updated_at = "2026-06-23T10:00:00Z"

[layout]
notes_glob = ["docs/*.md", "blog/*.md"]
ignore = ["secret.md", ".trash/**"]

[generator]
app = "mnemonic"
app_version = "2.0.0"
`
	m, err := ParseMnemonicManifest([]byte(tomlData))
	require.NoError(t, err)
	assert.Equal(t, "Full Project", m.Name)
	assert.Equal(t, "full-project", m.Slug)
	assert.Equal(t, ManifestTypeLocal, m.Type)
	assert.Equal(t, "A fully specified project", m.Description)
	assert.Equal(t, []string{"docs/*.md", "blog/*.md"}, m.Layout.NotesGlob)
	assert.Equal(t, []string{"secret.md", ".trash/**"}, m.Layout.Ignore)
	assert.Equal(t, "mnemonic", m.Generator.App)
	assert.Equal(t, "2.0.0", m.Generator.AppVersion)
}

// ── WriteMnemonicManifest / ParseMnemonicManifestFromFile ─────────────

func TestWriteAndReadMnemonicManifest(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "mnemonic.toml")

	m := validManifest()
	err := WriteMnemonicManifest(path, m)
	require.NoError(t, err)

	read, err := ParseMnemonicManifestFromFile(path)
	require.NoError(t, err)

	assert.Equal(t, m.ProjectID, read.ProjectID)
	assert.Equal(t, m.Name, read.Name)
	assert.Equal(t, m.Slug, read.Slug)
}

func TestParseMnemonicManifestFile_Alias(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "mnemonic.toml")

	m := validManifest()
	err := WriteMnemonicManifest(path, m)
	require.NoError(t, err)

	read, err := ParseMnemonicManifestFile(path)
	require.NoError(t, err)

	assert.Equal(t, m.Name, read.Name)
}

func TestParseMnemonicManifestFromFile_NotFound(t *testing.T) {
	_, err := ParseMnemonicManifestFromFile("/nonexistent/path/mnemonic.toml")
	require.Error(t, err)
}

// ── WriteMnemonicManifest with invalid manifest ────────────────────────

func TestWriteMnemonicManifest_Invalid(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "mnemonic.toml")

	m := &Manifest{}
	err := WriteMnemonicManifest(path, m)
	require.Error(t, err)
}

// ── ParsePointerFile ───────────────────────────────────────────────────

func TestParsePointerFile_Valid(t *testing.T) {
	tomlData := `manifest_path = "/home/user/projects/my-project/mnemonic.toml"
`
	pf, err := ParsePointerFile([]byte(tomlData))
	require.NoError(t, err)
	assert.Equal(t, "/home/user/projects/my-project/mnemonic.toml", pf.ManifestPath)
}

func TestParsePointerFile_MissingPath(t *testing.T) {
	tomlData := `# empty
`
	_, err := ParsePointerFile([]byte(tomlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest_path")
}

func TestParsePointerFile_InvalidTOML(t *testing.T) {
	_, err := ParsePointerFile([]byte("{{{{"))
	require.Error(t, err)
}

func TestParsePointerFile_UnknownFields(t *testing.T) {
	tomlData := `manifest_path = "/some/path"
extra = true
`
	_, err := ParsePointerFile([]byte(tomlData))
	require.Error(t, err)
}

// ── WritePointerFile ───────────────────────────────────────────────────

func TestWriteAndReadPointerFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "pointer.toml")

	pf := &PointerFile{
		ManifestPath: "/home/user/projects/my-project/mnemonic.toml",
	}
	err := WritePointerFile(path, pf)
	require.NoError(t, err)

	// Read back and verify
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	read, err := ParsePointerFile(data)
	require.NoError(t, err)
	assert.Equal(t, pf.ManifestPath, read.ManifestPath)
}

func TestWritePointerFile_Nil(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "pointer.toml")

	err := WritePointerFile(path, nil)
	require.Error(t, err)
}

// ── tomlTime helpers ───────────────────────────────────────────────────

func TestTOMLTime_MarshalText(t *testing.T) {
	ts := fixedTime(14, 30)
	tt := newTOMLTime(ts)
	data, err := tt.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "2026-06-23T14:30:00Z", string(data))
}

func TestTOMLTime_UnmarshalText(t *testing.T) {
	var tt tomlTime
	err := tt.UnmarshalText([]byte("2026-06-23T14:30:00Z"))
	require.NoError(t, err)
	assert.Equal(t, fixedTime(14, 30), tt.Time())
}

func TestTOMLTime_UnmarshalText_Empty(t *testing.T) {
	var tt tomlTime
	err := tt.UnmarshalText([]byte(""))
	require.NoError(t, err)
	assert.True(t, tt.Time().IsZero())
}

func TestTOMLTime_UnmarshalText_Invalid(t *testing.T) {
	var tt tomlTime
	err := tt.UnmarshalText([]byte("not-a-date"))
	require.Error(t, err)
}

// ── MarshalTOML with defaults applied ─────────────────────────────────

func TestMarshalTOML_AppliesDefaults(t *testing.T) {
	m := &Manifest{
		Version:               1,
		ProjectID:             "id",
		Name:                  "name",
		Slug:                  "slug",
		MarkdownFormatVersion: 1,
		CreatedAt:             fixedTime(10, 0),
		UpdatedAt:             fixedTime(10, 0),
		// NotesGlob and Ignore are empty — defaults should be applied
	}

	data, err := m.MarshalTOML()
	require.NoError(t, err)

	// Parse to verify defaults were applied
	var raw map[string]any
	err = toml.Unmarshal(data, &raw)
	require.NoError(t, err)

	layout, ok := raw["layout"].(map[string]any)
	require.True(t, ok, "layout should be present")
	assert.NotEmpty(t, layout["notes_glob"])
	assert.NotEmpty(t, layout["ignore"])
}

func TestMarshalTOML_IncludesGenerator(t *testing.T) {
	m := validManifest()
	m.Generator = Generator{App: "test-runner", AppVersion: "2.0.0"}

	data, err := m.MarshalTOML()
	require.NoError(t, err)

	parsed, err := ParseMnemonicManifest(data)
	require.NoError(t, err)
	assert.Equal(t, "test-runner", parsed.Generator.App)
	assert.Equal(t, "2.0.0", parsed.Generator.AppVersion)
}
