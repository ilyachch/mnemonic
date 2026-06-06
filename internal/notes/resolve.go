package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/project"
)

// ResolvedNote contains the matched note and its relative path.
type ResolvedNote struct {
	Note markdown.Note
	Path string
}

// Resolve finds a note using the canonical selector precedence.
func Resolve(root string, selector string) (ResolvedNote, error) {
	if root == "" {
		return ResolvedNote{}, fmt.Errorf("root directory is required")
	}
	if selector == "" {
		return ResolvedNote{}, app.NewNotFoundError("note selector is required", nil)
	}

	notes, err := loadResolvedNotes(root)
	if err != nil {
		return ResolvedNote{}, err
	}

	stages := []func(resolved resolvedNotes) []ResolvedNote{
		func(resolved resolvedNotes) []ResolvedNote { return resolved.matchUUID(selector) },
		func(resolved resolvedNotes) []ResolvedNote { return resolved.matchSlug(selector) },
		func(resolved resolvedNotes) []ResolvedNote { return resolved.matchPath(selector) },
		func(resolved resolvedNotes) []ResolvedNote { return resolved.matchTitle(selector) },
		func(resolved resolvedNotes) []ResolvedNote { return resolved.matchNormalizedTitle(selector) },
	}

	for _, stage := range stages {
		matches := stage(notes)
		switch len(matches) {
		case 0:
			continue
		case 1:
			return matches[0], nil
		default:
			return ResolvedNote{}, app.NewAmbiguousError(fmt.Sprintf("note selector %q matches multiple notes", selector), nil)
		}
	}

	return ResolvedNote{}, app.NewNotFoundError(fmt.Sprintf("note %q not found", selector), nil)
}

type resolvedNotes []resolvedNote

type resolvedNote struct {
	ResolvedNote
	selectorData selectorData
}

type selectorData struct {
	UUID        string
	Slug        string
	Path        string
	Title       string
	SlugByTitle string
}

func loadResolvedNotes(root string) (resolvedNotes, error) {
	paths, err := Walk(root)
	if err != nil {
		return nil, err
	}

	notes := make(resolvedNotes, 0, len(paths))
	for _, relPath := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
		if err != nil {
			return nil, fmt.Errorf("read note %q: %w", relPath, err)
		}

		note, err := markdown.ParseNote(data)
		if err != nil {
			return nil, fmt.Errorf("parse note %q: %w", relPath, err)
		}

		notes = append(notes, resolvedNote{
			ResolvedNote: ResolvedNote{
				Note: note,
				Path: relPath,
			},
			selectorData: selectorData{
				UUID:        note.MnemonicNoteID,
				Slug:        note.EffectiveSlug(),
				Path:        relPath,
				Title:       note.Title,
				SlugByTitle: normalizedTitleSlug(note.Title),
			},
		})
	}

	return notes, nil
}

func normalizedTitleSlug(title string) string {
	slug, err := project.Slugify(title)
	if err != nil {
		return ""
	}
	return slug
}

func (n resolvedNotes) matchUUID(selector string) []ResolvedNote {
	return n.match(func(item resolvedNote) bool { return item.selectorData.UUID == selector })
}

func (n resolvedNotes) matchSlug(selector string) []ResolvedNote {
	return n.match(func(item resolvedNote) bool { return item.selectorData.Slug == selector })
}

func (n resolvedNotes) matchPath(selector string) []ResolvedNote {
	cleaned := filepath.ToSlash(filepath.Clean(selector))
	if cleaned == "." || strings.HasPrefix(cleaned, "..") || filepath.IsAbs(selector) {
		return nil
	}
	return n.match(func(item resolvedNote) bool { return item.selectorData.Path == cleaned })
}

func (n resolvedNotes) matchTitle(selector string) []ResolvedNote {
	return n.match(func(item resolvedNote) bool { return item.selectorData.Title == selector })
}

func (n resolvedNotes) matchNormalizedTitle(selector string) []ResolvedNote {
	return n.match(func(item resolvedNote) bool { return item.selectorData.SlugByTitle == selector })
}

func (n resolvedNotes) match(pred func(resolvedNote) bool) []ResolvedNote {
	out := make([]ResolvedNote, 0, 1)
	for _, item := range n {
		if pred(item) {
			out = append(out, item.ResolvedNote)
		}
	}
	return out
}
