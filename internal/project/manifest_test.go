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
		Kind:                  ManifestKindDetached,
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

func TestMnemonicManifestDefaultsWhenParsing(t *testing.T) {
	t.Parallel()

	parsed, err := ParseMnemonicManifest([]byte(`version = 1
project_id = "550e8400-e29b-41d4-a716-446655440000"
name = "personal"
slug = "personal"
kind = "regular"
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
}

func TestMnemonicManifestRejectsUnsupportedVersion(t *testing.T) {
	t.Parallel()

	_, err := ParseMnemonicManifest([]byte(`version = 999
project_id = "550e8400-e29b-41d4-a716-446655440000"
name = "personal"
slug = "personal"
kind = "regular"
markdown_format_version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:05:00Z"
`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "version 999 is unsupported; expected 1")
}

func TestMnemonicManifestValidateAllowsRegularAndDetached(t *testing.T) {
	t.Parallel()

	base := func(kind ManifestKind) *MnemonicManifest {
		manifest := NewMnemonicManifest()
		manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
		manifest.Name = "personal"
		manifest.Slug = "personal"
		manifest.Kind = kind
		manifest.MarkdownFormatVersion = 1
		manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
		manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
		manifest.Generator.App = "mnemonic"
		manifest.Generator.AppVersion = "0.1.0-dev"
		return manifest
	}

	for _, tt := range []struct {
		name string
		kind ManifestKind
	}{
		{name: "regular", kind: ManifestKindRegular},
		{name: "detached", kind: ManifestKindDetached},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, base(tt.kind).Validate())
		})
	}
}

func TestMnemonicManifestRejectsLocalKind(t *testing.T) {
	t.Parallel()

	manifest := NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "backend"
	manifest.Slug = "backend"
	manifest.Kind = ManifestKind("local")
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	manifest.UpdatedAt = time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)

	err := manifest.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), `kind "local" is not allowed`)
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
			name: "kind",
			mutate: func(manifest *MnemonicManifest) {
				manifest.Kind = ""
			},
			wantErr: "kind is required",
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
			manifest.Kind = ManifestKindRegular
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
