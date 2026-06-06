package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	mnemonicfs "github.com/ilyachch/mnemonic/internal/fs"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/project"
)

// EditInput configures note editing.
type EditInput struct {
	RootDir  string
	Selector string
	Append   []byte
	Body     []byte
	HasBody  bool
	Set      map[string]string
	IfMatch  string
	Now      func() time.Time
}

// EditResult describes the edited note.
type EditResult struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Edit applies an append edit to a note.
func Edit(input EditInput) (EditResult, error) {
	if input.RootDir == "" {
		return EditResult{}, fmt.Errorf("root directory is required")
	}
	if input.Selector == "" {
		return EditResult{}, app.NewNotFoundError("note selector is required", nil)
	}

	guard, err := acquireWriteLock(input.RootDir)
	if err != nil {
		return EditResult{}, err
	}
	defer func() { _ = guard.Release() }()

	resolved, err := Resolve(input.RootDir, input.Selector)
	if err != nil {
		return EditResult{}, err
	}

	absPath := filepath.Join(input.RootDir, filepath.FromSlash(resolved.Path))
	if input.IfMatch != "" {
		current, err := os.ReadFile(absPath)
		if err != nil {
			return EditResult{}, fmt.Errorf("read note: %w", err)
		}
		if HashBytes(current) != input.IfMatch {
			return EditResult{}, app.NewUnsafeError("content hash mismatch", nil)
		}
	}

	if err := applyEditSet(&resolved.Note, input.Set); err != nil {
		return EditResult{}, err
	}

	now := input.Now
	if now == nil {
		now = project.NowUTC
	}

	edited := resolved.Note
	if input.HasBody {
		edited.Body = append([]byte(nil), input.Body...)
	} else {
		edited.Body = append(append([]byte(nil), edited.Body...), input.Append...)
	}
	edited.UpdatedAt = now().UTC()

	rendered, err := markdown.RenderNote(edited)
	if err != nil {
		return EditResult{}, err
	}

	if err := mnemonicfs.AtomicWriteFile(absPath, rendered, 0o644); err != nil {
		return EditResult{}, err
	}

	return EditResult{
		NoteID:      edited.MnemonicNoteID,
		Slug:        edited.EffectiveSlug(),
		Path:        resolved.Path,
		ContentHash: HashBytes(rendered),
		CreatedAt:   edited.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   edited.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func applyEditSet(note *markdown.Note, set map[string]string) error {
	if note == nil || len(set) == 0 {
		return nil
	}
	if note.Frontmatter == nil {
		note.Frontmatter = map[string]any{}
	}

	for key, value := range set {
		switch key {
		case "mnemonic_note_id", "created_at":
			return app.NewUnsafeError(fmt.Sprintf("frontmatter %q is protected", key), nil)
		case "title":
			note.Title = value
			note.Frontmatter[key] = value
		case "slug":
			note.Slug = value
			note.Frontmatter[key] = value
		case "type":
			note.Type = value
			note.Frontmatter[key] = value
		case "permalink":
			note.Permalink = value
			note.Frontmatter[key] = value
		default:
			note.Frontmatter[key] = value
		}
	}

	return nil
}
