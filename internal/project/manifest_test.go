package project

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMnemonicManifestDefaults(t *testing.T) {
	t.Parallel()

	manifest := NewMnemonicManifest()

	require.Equal(t, 1, manifest.Version)
	require.True(t, reflect.DeepEqual(manifest.Layout.NotesGlob, []string{"**/*.md"}))
	require.True(t, reflect.DeepEqual(manifest.Layout.Ignore, []string{"mnemonic.toml", ".trash/**"}))
}

func TestMnemonicManifestRoundTrip(t *testing.T) {
	t.Parallel()

	original := &MnemonicManifest{
		Version:               1,
		ProjectID:             "550e8400-e29b-41d4-a716-446655440000",
		Name:                  "personal",
		Slug:                  "personal",
		Type:                  ManifestTypeLocal,
		MarkdownFormatVersion: 1,
		CreatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt:             time.Date(2026, time.June, 2, 10, 5, 0, 0, time.UTC),
		Layout: MnemonicManifestLayout{
			NotesGlob: []string{"**/*.md", "extra/**/*.md"},
			Ignore:    []string{"mnemonic.toml", ".trash/**", "*.tmp"},
		},
		Generator: MnemonicGenerator{
			App:        "mnemonic",
			AppVersion: "0.1.0-dev",
		},
	}

	data, err := original.MarshalTOML()
	require.NoError(t, err)

	parsed, err := ParseMnemonicManifest(data)
	require.NoError(t, err)

	require.True(t, reflect.DeepEqual(parsed, original))
}

func TestMnemonicManifestRoundTripWithDescription(t *testing.T) {
	t.Parallel()

	original := &MnemonicManifest{
		Version:               1,
		ProjectID:             "550e8400-e29b-41d4-a716-446655440000",
		Name:                  "backend",
		Slug:                  "backend",
		MarkdownFormatVersion: 1,
		Description:           "Backend architecture decisions, API contracts, and database schemas.",
		CreatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt:             time.Date(2026, time.June, 2, 10, 5, 0, 0, time.UTC),
		Layout: MnemonicManifestLayout{
			NotesGlob: []string{"**/*.md"},
			Ignore:    []string{"mnemonic.toml", ".trash/**"},
		},
		Generator: MnemonicGenerator{
			App:        "mnemonic",
			AppVersion: "0.1.0-dev",
		},
	}

	data, err := original.MarshalTOML()
	require.NoError(t, err)

	parsed, err := ParseMnemonicManifest(data)
	require.NoError(t, err)

	require.Equal(t, "Backend architecture decisions, API contracts, and database schemas.", parsed.Description)
	require.True(t, reflect.DeepEqual(parsed, original))
}

func TestMnemonicManifestDefaultsWhenParsing(t *testing.T) {
	t.Parallel()

	parsed, err := ParseMnemonicManifest([]byte(`version = 1
project_id = "550e8400-e29b-41d4-a716-446655440000"
name = "personal"
slug = "personal"
markdown_format_version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:05:00Z"

[generator]
app = "mnemonic"
app_version = "0.1.0-dev"
`))
	require.NoError(t, err)

	require.True(t, reflect.DeepEqual(parsed.Layout.NotesGlob, []string{"**/*.md"}))
	require.True(t, reflect.DeepEqual(parsed.Layout.Ignore, []string{"mnemonic.toml", ".trash/**"}))
	require.Equal(t, ManifestType(""), parsed.Type)
}

func TestMnemonicManifestRejectsUnsupportedVersion(t *testing.T) {
	t.Parallel()

	_, err := ParseMnemonicManifest([]byte(`version = 999
project_id = "550e8400-e29b-41d4-a716-446655440000"
name = "personal"
slug = "personal"
markdown_format_version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:05:00Z"
`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "version 999 is unsupported; expected 1")
}

func TestMnemonicManifestValidateAllowsCentralAndLocal(t *testing.T) {
	t.Parallel()

	base := func(typ ManifestType) *MnemonicManifest {
		manifest := NewMnemonicManifest()
		manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
		manifest.Name = "personal"
		manifest.Slug = "personal"
		manifest.Type = typ
		manifest.MarkdownFormatVersion = 1
		manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
		manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
		manifest.Generator.App = "mnemonic"
		manifest.Generator.AppVersion = "0.1.0-dev"
		return manifest
	}

	for _, tt := range []struct {
		name string
		typ  ManifestType
	}{
		{name: "central (empty)", typ: ""},
		{name: "local", typ: ManifestTypeLocal},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, base(tt.typ).Validate())
		})
	}
}

func TestMnemonicManifestRejectsUnknownType(t *testing.T) {
	t.Parallel()

	manifest := NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "backend"
	manifest.Slug = "backend"
	manifest.Type = ManifestType("detached")
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)

	err := manifest.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown type "detached"`)
}

func TestMnemonicManifestIsLocal(t *testing.T) {
	t.Parallel()

	local := &MnemonicManifest{Type: ManifestTypeLocal}
	require.True(t, local.IsLocal())

	central := &MnemonicManifest{Type: ""}
	require.False(t, central.IsLocal())

	nilManifest := (*MnemonicManifest)(nil)
	require.False(t, nilManifest.IsLocal())
}

func TestMnemonicManifestRequiresFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*MnemonicManifest)
		wantErr string
	}{
		{
			name: "project id",
			mutate: func(manifest *MnemonicManifest) {
				manifest.ProjectID = ""
			},
			wantErr: "project_id is required",
		},
		{
			name: "name",
			mutate: func(manifest *MnemonicManifest) {
				manifest.Name = ""
			},
			wantErr: "name is required",
		},
		{
			name: "slug",
			mutate: func(manifest *MnemonicManifest) {
				manifest.Slug = ""
			},
			wantErr: "slug is required",
		},
		{
			name: "markdown format version",
			mutate: func(manifest *MnemonicManifest) {
				manifest.MarkdownFormatVersion = 0
			},
			wantErr: "markdown_format_version must be positive",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manifest := NewMnemonicManifest()
			manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
			manifest.Name = "personal"
			manifest.Slug = "personal"
			manifest.MarkdownFormatVersion = 1
			manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
			manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
			tt.mutate(manifest)

			err := manifest.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
