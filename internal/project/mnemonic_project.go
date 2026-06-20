package project

import "time"

// ProjectKind identifies the supported kinds of a project.
type ProjectKind string

const (
	// ProjectKindCentral stores a central project rooted in the memories home.
	ProjectKindCentral ProjectKind = "central"
	// ProjectKindLocal stores a project backed by a local `.mnemonic-memories` directory.
	ProjectKindLocal ProjectKind = "local"
)

// MnemonicProject captures the metadata needed by project helpers (init,
// resolve, paths).
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
