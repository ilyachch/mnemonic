package project

import "time"

// ProjectKind identifies the supported kinds of a project.
type ProjectKind string

const (
	// ProjectKindRegular stores a project that uses a regular memories directory.
	ProjectKindRegular ProjectKind = "regular"
	// ProjectKindLocal stores a project backed by a local `.mnemonic-memories` directory.
	ProjectKindLocal ProjectKind = "local"
)

// MnemonicProject captures the metadata needed by project helpers (init,
// resolve, paths). The authoritative copy lives in the registry; this type is
// only used during in-memory construction before registration.
type MnemonicProject struct {
	ID                    string
	Name                  string
	Slug                  string
	Kind                  ProjectKind
	MemoriesPath          string
	MarkdownFormatVersion int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
