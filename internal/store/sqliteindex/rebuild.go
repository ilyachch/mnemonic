package sqliteindex

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
)

// RebuildResult summarizes a rebuild.
type RebuildResult struct {
	KBID         string `json:"kb_id"`
	NotesSeen    int    `json:"notes_seen"`
	NotesIndexed int    `json:"notes_indexed"`
	Status       string `json:"status"`
}

// Rebuild builds a fresh index database and swaps it into place.
func (s Store) Rebuild() (RebuildResult, error) {
	if err := s.validateIndexPath(); err != nil {
		return RebuildResult{}, err
	}
	if strings.TrimSpace(s.RootDir) == "" {
		return RebuildResult{}, errors.New("root dir is required")
	}

	guard, err := acquireRebuildLock(s.IndexPath)
	if err != nil {
		return RebuildResult{}, err
	}
	defer func() { _ = guard.Unlock() }()
	defer func() { _ = guard.Close() }()

	docs, errs, err := ScanNotes(s.RootDir)
	if err != nil {
		return RebuildResult{}, err
	}
	if len(errs) > 0 {
		return RebuildResult{}, errs[0]
	}

	tempPath := tempIndexPath(s.IndexPath)
	_ = os.Remove(tempPath)
	if err = os.MkdirAll(filepath.Dir(tempPath), 0o755); err != nil {
		return RebuildResult{}, err
	}

	db, err := openDB(tempPath)
	if err != nil {
		return RebuildResult{}, err
	}
	if err := ApplySchema(db); err != nil {
		_ = db.Close()
		_ = os.Remove(tempPath)
		return RebuildResult{}, err
	}
	if err := insertDocs(db, docs, s.KBID); err != nil {
		_ = db.Close()
		_ = os.Remove(tempPath)
		return RebuildResult{}, err
	}
	if err := quickCheckFile(tempPath); err != nil {
		_ = db.Close()
		_ = os.Remove(tempPath)
		return RebuildResult{}, err
	}
	_ = db.Close()

	if err := os.MkdirAll(filepath.Dir(s.IndexPath), 0o755); err != nil {
		return RebuildResult{}, err
	}
	if err := removeIndexFiles(s.IndexPath); err != nil {
		return RebuildResult{}, err
	}
	if err := os.Rename(tempPath, s.IndexPath); err != nil {
		return RebuildResult{}, err
	}
	return RebuildResult{
		KBID:         s.KBID,
		NotesSeen:    len(docs),
		NotesIndexed: len(docs),
		Status:       "ok",
	}, nil
}

func tempIndexPath(path string) string {
	return strings.TrimSuffix(path, ".sqlite") + ".new.sqlite"
}

func insertDocs(db *sql.DB, docs []NoteDoc, kbid string) error {
	_, _, seenNorm, err := validateDocs(docs)
	if err != nil {
		return err
	}
	for _, doc := range docs {
		if err := insertNoteDoc(db, doc, kbid); err != nil {
			return err
		}
	}
	for _, doc := range docs {
		insertDocLinks(db, doc, docs, seenNorm)
	}
	return nil
}

func validateDocs(docs []NoteDoc) (seenNoteIDs, seenSlugs map[string]struct{}, seenNorm map[string]int, err error) {
	seenNoteIDs = make(map[string]struct{})
	seenSlugs = make(map[string]struct{})
	seenNorm = make(map[string]int)
	for _, doc := range docs {
		if _, ok := seenNoteIDs[doc.NoteID]; ok {
			return nil, nil, nil, fmt.Errorf("duplicate note_id %s", doc.NoteID)
		}
		seenNoteIDs[doc.NoteID] = struct{}{}
		if _, ok := seenSlugs[doc.Slug]; ok {
			return nil, nil, nil, fmt.Errorf("duplicate slug %s", doc.Slug)
		}
		seenSlugs[doc.Slug] = struct{}{}
		seenNorm[normalizeTitleSlug(doc.Title)]++
	}
	return
}

func insertNoteDoc(db *sql.DB, doc NoteDoc, kbid string) error {
	if _, err := db.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		doc.NoteID, kbid, doc.Slug, doc.RelPath, doc.Title, doc.ContentHash); err != nil {
		return err
	}
	_, _ = db.Exec(`INSERT INTO notes_fts(rowid, note_id, title, body) VALUES ((SELECT rowid FROM notes WHERE note_id = ?), ?, ?, ?)`,
		doc.NoteID, doc.NoteID, doc.Title, doc.SearchText)
	for _, tag := range doc.Tags {
		if tag.Value == "" {
			continue
		}
		_, _ = db.Exec(`INSERT INTO note_tags(note_id, tag) VALUES (?, ?)`, doc.NoteID, tag.Source+":"+tag.Value)
	}
	for _, ob := range doc.Observations {
		_, _ = db.Exec(`INSERT INTO observations(observation_id, note_id, kind, value) VALUES (?, ?, ?, ?)`,
			markdownstore.HashBytes([]byte(doc.NoteID+ob.Content)), doc.NoteID, ob.Category, ob.Content)
	}
	return nil
}

func insertDocLinks(db *sql.DB, doc NoteDoc, docs []NoteDoc, seenNorm map[string]int) {
	for _, link := range doc.Links {
		toID := sql.NullString{}
		if resolved, ok := resolveLinkTarget(docs, seenNorm, link.RawTarget); ok {
			toID.Valid = true
			toID.String = resolved
		}
		_, _ = db.Exec(`INSERT INTO links(link_id, note_id, to_note_id, target, relation_type, source_line) VALUES (?, ?, ?, ?, ?, ?)`,
			markdownstore.HashBytes([]byte(doc.NoteID+link.RawTarget+link.Source+strconv.Itoa(link.Line))), doc.NoteID, toID, link.RawTarget, link.RelationType, link.Line)
	}
}

func resolveLinkTarget(docs []NoteDoc, norms map[string]int, target string) (string, bool) {
	for _, doc := range docs {
		if doc.NoteID == target || doc.Slug == target || doc.RelPath == target || doc.Title == target {
			return doc.NoteID, true
		}
	}
	norm := normalizeTitleSlug(target)
	if norms[norm] != 1 {
		return "", false
	}
	for _, doc := range docs {
		if n := normalizeTitleSlug(doc.Title); n == norm {
			return doc.NoteID, true
		}
	}
	return "", false
}

func quickCheckFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("path is required")
	}

	db, err := sql.Open(sqliteDriverName, path)
	if err != nil {
		return apperr.Corrupted("index database is corrupted", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return classifyCorruption(err)
	}

	var result string
	if err := db.QueryRow(`PRAGMA quick_check`).Scan(&result); err != nil {
		return classifyCorruption(err)
	}
	if result != "ok" {
		return apperr.Corrupted("index database is corrupted", fmt.Errorf("quick_check = %s", result))
	}
	return nil
}
