package project

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMnemonicManifestDefaults(t *testing.T) {
	t.Parallel()

	manifest := NewMnemonicManifest()

	if manifest.Version != 1 {
		t.Fatalf("Version = %d, want 1", manifest.Version)
	}
	if got, want := manifest.Layout.NotesGlob, []string{"**/*.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Layout.NotesGlob = %#v, want %#v", got, want)
	}
	if got, want := manifest.Layout.Ignore, []string{"mnemonic.toml", ".trash/**"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Layout.Ignore = %#v, want %#v", got, want)
	}
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
	if err != nil {
		t.Fatalf("MarshalTOML() error = %v", err)
	}

	parsed, err := ParseMnemonicManifest(data)
	if err != nil {
		t.Fatalf("ParseMnemonicManifest() error = %v", err)
	}

	if !reflect.DeepEqual(parsed, original) {
		t.Fatalf("roundtrip mismatch\noriginal: %#v\nparsed: %#v\ntext:\n%s", original, parsed, string(data))
	}
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
	if err != nil {
		t.Fatalf("ParseMnemonicManifest() error = %v", err)
	}

	if got, want := parsed.Layout.NotesGlob, []string{"**/*.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Layout.NotesGlob = %#v, want %#v", got, want)
	}
	if got, want := parsed.Layout.Ignore, []string{"mnemonic.toml", ".trash/**"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Layout.Ignore = %#v, want %#v", got, want)
	}
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
	if err == nil {
		t.Fatal("ParseMnemonicManifest() error = nil, want unsupported version rejection")
	}
	if !strings.Contains(err.Error(), "version 999 is unsupported; expected 1") {
		t.Fatalf("ParseMnemonicManifest() error = %q, want unsupported version rejection", err)
	}
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

			if err := base(tt.kind).Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
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
	if err == nil {
		t.Fatal("Validate() error = nil, want local kind rejection")
	}
	if !strings.Contains(err.Error(), `kind "local" is not allowed`) {
		t.Fatalf("Validate() error = %q, want local kind rejection", err)
	}
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
			if err == nil {
				t.Fatalf("Validate() error = nil, want %s", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %q, want %q", err, tt.wantErr)
			}
		})
	}
}
