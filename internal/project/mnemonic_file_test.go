package project

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	parsed, err := ParseMnemonicFile(data)
	require.NoError(t, err)

	require.True(t, reflect.DeepEqual(parsed, original))
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
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
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
	require.Error(t, err)
	require.Contains(t, err.Error(), "name is required")
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
	require.Error(t, err)
	require.Contains(t, err.Error(), "version 999 is unsupported; expected 1")
}
