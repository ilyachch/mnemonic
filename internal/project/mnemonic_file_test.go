package project

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMnemonicFileRoundTrip(t *testing.T) {
	original := &MnemonicFile{
		Version:   1,
		CreatedAt: time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.June, 2, 10, 5, 0, 0, time.UTC),
		Projects: []MnemonicProject{
			{
				ID:                    "550e8400-e29b-41d4-a716-446655440000",
				Name:                  "backend",
				Slug:                  "backend",
				Kind:                  ProjectKindLocal,
				MemoriesPath:          ".mnemonic-memories/backend",
				MarkdownFormatVersion: 1,
				CreatedAt:             time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
				UpdatedAt:             time.Date(2026, time.June, 2, 10, 1, 0, 0, time.UTC),
			},
			{
				ID:                    "7f0a6d73-c3ba-4f0e-85b8-27bccf4370f1",
				Name:                  "research",
				Slug:                  "research",
				Kind:                  ProjectKindRegular,
				MemoriesPath:          "research",
				MarkdownFormatVersion: 1,
				CreatedAt:             time.Date(2026, time.June, 2, 10, 5, 0, 0, time.UTC),
				UpdatedAt:             time.Date(2026, time.June, 2, 10, 6, 0, 0, time.UTC),
			},
		},
	}

	data, err := original.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML() error = %v", err)
	}

	parsed, err := ParseMnemonicFile(data)
	if err != nil {
		t.Fatalf("ParseMnemonicFile() error = %v", err)
	}

	if !reflect.DeepEqual(parsed, original) {
		t.Fatalf("roundtrip mismatch\noriginal: %#v\nparsed: %#v\ntext:\n%s", original, parsed, string(data))
	}
}

func TestMnemonicFileRejectsDetachedKind(t *testing.T) {
	_, err := ParseMnemonicFile([]byte(`version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:00:00Z"

[[projects]]
id = "550e8400-e29b-41d4-a716-446655440000"
name = "backend"
slug = "backend"
kind = "detached"
memories_path = ".mnemonic-memories/backend"
markdown_format_version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:00:00Z"
`))
	if err == nil {
		t.Fatal("ParseMnemonicFile() error = nil, want detached kind rejection")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("ParseMnemonicFile() error = %q, want detached rejection", err)
	}
}

func TestMnemonicFileRequiresProjectFields(t *testing.T) {
	_, err := ParseMnemonicFile([]byte(`version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:00:00Z"

[[projects]]
id = "550e8400-e29b-41d4-a716-446655440000"
slug = "backend"
kind = "local"
memories_path = ".mnemonic-memories/backend"
markdown_format_version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:00:00Z"
`))
	if err == nil {
		t.Fatal("ParseMnemonicFile() error = nil, want required-field rejection")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("ParseMnemonicFile() error = %q, want missing name rejection", err)
	}
}

func TestMnemonicFileRejectsUnsupportedVersion(t *testing.T) {
	_, err := ParseMnemonicFile([]byte(`version = 999
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:00:00Z"

[[projects]]
id = "550e8400-e29b-41d4-a716-446655440000"
name = "backend"
slug = "backend"
kind = "local"
memories_path = ".mnemonic-memories/backend"
markdown_format_version = 1
created_at = "2026-06-02T10:00:00Z"
updated_at = "2026-06-02T10:00:00Z"
`))
	if err == nil {
		t.Fatal("ParseMnemonicFile() error = nil, want unsupported version rejection")
	}
	if !strings.Contains(err.Error(), "version 999 is unsupported; expected 1") {
		t.Fatalf("ParseMnemonicFile() error = %q, want unsupported version rejection", err)
	}
}
