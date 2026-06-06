package notes

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/registry"
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

	resolved, usedIndex, err := resolveFromIndex(root, selector)
	if err != nil {
		return ResolvedNote{}, err
	}
	if usedIndex {
		return resolved, nil
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

func resolveFromIndex(root string, selector string) (ResolvedNote, bool, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ResolvedNote{}, false, fmt.Errorf("resolve root %q: %w", root, err)
	}

	db, err := registry.OpenDB()
	if err != nil {
		return ResolvedNote{}, false, nil
	}
	defer func() { _ = db.Close() }()

	if err := registry.ApplySchema(db); err != nil {
		return ResolvedNote{}, false, nil
	}

	projectID, ok, err := lookupProjectIDByMemoriesRoot(db, absRoot)
	if err != nil {
		return ResolvedNote{}, false, nil
	}
	if !ok {
		return ResolvedNote{}, false, nil
	}

	indexPath, err := projectIndexPath(projectID)
	if err != nil {
		return ResolvedNote{}, false, err
	}
	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			return ResolvedNote{}, false, nil
		}
		return ResolvedNote{}, false, fmt.Errorf("stat index %q: %w", indexPath, err)
	}

	indexDB, err := sql.Open("sqlite", indexPath)
	if err != nil {
		return ResolvedNote{}, false, fmt.Errorf("open index database: %w", err)
	}
	defer func() { _ = indexDB.Close() }()

	if err := indexDB.Ping(); err != nil {
		return ResolvedNote{}, false, fmt.Errorf("ping index database: %w", err)
	}

	matches, err := matchResolvedNotesFromIndex(indexDB, selector)
	if err != nil {
		return ResolvedNote{}, true, err
	}
	switch len(matches) {
	case 0:
		return ResolvedNote{}, true, app.NewNotFoundError(fmt.Sprintf("note %q not found", selector), nil)
	case 1:
		return readResolvedNote(root, matches[0])
	default:
		return ResolvedNote{}, true, app.NewAmbiguousError(fmt.Sprintf("note selector %q matches multiple notes", selector), nil)
	}
}

func lookupProjectIDByMemoriesRoot(db *sql.DB, memoriesRoot string) (string, bool, error) {
	rows, err := db.Query(
		`SELECT p.project_id
		 FROM projects p
		 JOIN project_locations l ON l.project_id = p.project_id
		 WHERE p.removed_at IS NULL AND l.memories_abs = ?
		 LIMIT 2`,
		memoriesRoot,
	)
	if err != nil {
		return "", false, fmt.Errorf("query project by memories root %q: %w", memoriesRoot, err)
	}
	defer rows.Close()

	var projectID string
	count := 0
	for rows.Next() {
		if err := rows.Scan(&projectID); err != nil {
			return "", false, fmt.Errorf("scan project by memories root %q: %w", memoriesRoot, err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return "", false, fmt.Errorf("iterate project rows by memories root %q: %w", memoriesRoot, err)
	}
	switch count {
	case 0:
		return "", false, nil
	case 1:
		return projectID, true, nil
	default:
		return "", false, app.NewAmbiguousError(fmt.Sprintf("project root %q resolves to multiple projects", memoriesRoot), nil)
	}
}

func matchResolvedNotesFromIndex(db *sql.DB, selector string) ([]resolvedNote, error) {
	stageQueries := []func() ([]resolvedNote, error){
		func() ([]resolvedNote, error) {
			return queryResolvedNotes(db, `SELECT note_id, slug, rel_path, title FROM notes WHERE note_id = ? LIMIT 2`, selector)
		},
		func() ([]resolvedNote, error) {
			return queryResolvedNotes(db, `SELECT note_id, slug, rel_path, title FROM notes WHERE slug = ? LIMIT 2`, selector)
		},
		func() ([]resolvedNote, error) {
			return queryResolvedNotes(db, `SELECT note_id, slug, rel_path, title FROM notes WHERE rel_path = ? LIMIT 2`, normalizeResolvedPath(selector))
		},
		func() ([]resolvedNote, error) {
			return queryResolvedNotes(db, `SELECT note_id, slug, rel_path, title FROM notes WHERE title = ? LIMIT 2`, selector)
		},
		func() ([]resolvedNote, error) { return queryNotesByNormalizedTitle(db, selector) },
	}

	for _, stage := range stageQueries {
		matches, err := stage()
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			continue
		}
		return matches, nil
	}

	return nil, nil
}

func queryResolvedNotes(db *sql.DB, query string, arg string) ([]resolvedNote, error) {
	rows, err := db.Query(query, arg)
	if err != nil {
		return nil, fmt.Errorf("query note selector %q: %w", arg, err)
	}
	defer rows.Close()

	matches := make([]resolvedNote, 0, 2)
	for rows.Next() {
		var item resolvedNote
		if err := rows.Scan(&item.selectorData.UUID, &item.selectorData.Slug, &item.selectorData.Path, &item.selectorData.Title); err != nil {
			return nil, fmt.Errorf("scan note selector %q: %w", arg, err)
		}
		item.ResolvedNote.Path = item.selectorData.Path
		matches = append(matches, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate note selector %q: %w", arg, err)
	}
	return matches, nil
}

func queryNotesByNormalizedTitle(db *sql.DB, selector string) ([]resolvedNote, error) {
	rows, err := db.Query(`SELECT note_id, slug, rel_path, title FROM notes`)
	if err != nil {
		return nil, fmt.Errorf("query notes for normalized title %q: %w", selector, err)
	}
	defer rows.Close()

	matches := make([]resolvedNote, 0, 2)
	for rows.Next() {
		var item resolvedNote
		if err := rows.Scan(&item.selectorData.UUID, &item.selectorData.Slug, &item.selectorData.Path, &item.selectorData.Title); err != nil {
			return nil, fmt.Errorf("scan note for normalized title %q: %w", selector, err)
		}
		if normalizedTitleSlug(item.selectorData.Title) == selector {
			item.ResolvedNote.Path = item.selectorData.Path
			matches = append(matches, item)
			if len(matches) > 1 {
				break
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notes for normalized title %q: %w", selector, err)
	}
	return matches, nil
}

func normalizeResolvedPath(selector string) string {
	cleaned := filepath.ToSlash(filepath.Clean(selector))
	if cleaned == "." || strings.HasPrefix(cleaned, "..") || filepath.IsAbs(selector) {
		return selector
	}
	return cleaned
}

func readResolvedNote(root string, item resolvedNote) (ResolvedNote, bool, error) {
	absPath := filepath.Join(root, filepath.FromSlash(item.selectorData.Path))
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ResolvedNote{}, true, fmt.Errorf("read note %q: %w", item.selectorData.Path, err)
	}

	note, err := markdown.ParseNote(data)
	if err != nil {
		return ResolvedNote{}, true, fmt.Errorf("parse note %q: %w", item.selectorData.Path, err)
	}

	return ResolvedNote{
		Note: note,
		Path: item.selectorData.Path,
	}, true, nil
}

func projectIndexPath(projectID string) (string, error) {
	effective, err := paths.GetMnemonicPaths()
	if err != nil {
		return "", err
	}

	return filepath.Join(effective.StateHome, "mnemonic", "projects", projectID, "index.sqlite"), nil
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
