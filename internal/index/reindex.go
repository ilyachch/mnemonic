package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/notes"
)

// NoteDoc is a scanned markdown note with metadata used for indexing.
type NoteDoc struct {
	NoteID       string
	Slug         string
	Title        string
	RelPath      string
	Frontmatter  map[string]any
	BodyMarkdown string
	BodyText     string
	ContentHash  string
	FileMTimeNS  int64
	FileSize     int64
	Tags         []tagRow
	Observations []observationRow
	Links        []linkRow
	SearchText   string
}

type tagRow struct {
	Source string
	Value  string
}

type observationRow struct {
	Category    string
	Content     string
	LineStart   int
	ContentHash string
}

type linkRow struct {
	RelationType string
	RawTarget    string
	ToNoteID     sql.NullString
	Resolved     int
	Source       string
	Line         int
}

// ReindexResult summarizes a rebuild.
type ReindexResult struct {
	ProjectID    string `json:"project_id"`
	NotesSeen    int    `json:"notes_seen"`
	NotesIndexed int    `json:"notes_indexed"`
	Status       string `json:"status"`
}

// ScanNotes collects markdown notes from a project root.
func ScanNotes(root string) ([]NoteDoc, []error, error) {
	var docs []NoteDoc
	var errs []error
	paths, walkErr := notes.Walk(root)
	if walkErr != nil {
		return nil, nil, walkErr
	}
	for _, relPath := range paths {
		if filepath.Base(relPath) == "mnemonic.toml" {
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(relPath))
		data, readErr := os.ReadFile(abs)
		if readErr != nil {
			errs = append(errs, fmt.Errorf("%s: read note: %w", relPath, readErr))
			continue
		}
		st, statErr := os.Stat(abs)
		if statErr != nil {
			errs = append(errs, fmt.Errorf("%s: stat note: %w", relPath, statErr))
			continue
		}
		note, parseErr := markdown.ParseNote(data)
		if parseErr != nil {
			errs = append(errs, fmt.Errorf("%s: parse note: %w", relPath, parseErr))
			continue
		}
		info := NoteDoc{
			NoteID:       note.MnemonicNoteID,
			Slug:         note.EffectiveSlug(),
			Title:        note.Title,
			RelPath:      relPath,
			Frontmatter:  note.Frontmatter,
			BodyMarkdown: string(note.Body),
			BodyText:     string(note.Body),
			ContentHash:  hashNoteBytes(data),
			FileMTimeNS:  st.ModTime().UnixNano(),
			FileSize:     st.Size(),
		}
		info.Tags = append(info.Tags, tagRow{Source: "frontmatter"})
		for _, t := range note.Tags {
			info.Tags = append(info.Tags, tagRow{Source: "frontmatter", Value: t})
		}
		for _, t := range markdown.ParseTags(data) {
			info.Tags = append(info.Tags, tagRow{Source: "inline", Value: t.Value})
		}
		for _, ob := range markdown.ParseObservations(data) {
			info.Observations = append(info.Observations, observationRow{
				Category:    ob.Category,
				Content:     ob.Content,
				LineStart:   ob.LineStart,
				ContentHash: hashString(ob.Category + "\n" + ob.Content),
			})
			info.Tags = append(info.Tags, tagRow{Source: "observation", Value: ob.Category})
		}
		var searchable []string
		searchable = append(searchable, note.Title, string(note.Body))
		searchable = append(searchable, note.Tags...)
		for _, ob := range info.Observations {
			searchable = append(searchable, ob.Category, ob.Content)
		}
		info.SearchText = strings.Join(searchable, "\n")
		for _, rel := range markdown.ParseRelations(data) {
			target := strings.TrimSpace(rel.Target.Target)
			row := linkRow{RelationType: rel.RelationType, RawTarget: target, Source: string(rel.Source), Line: rel.Line}
			info.Links = append(info.Links, row)
		}
		docs = append(docs, info)
	}
	return docs, errs, nil
}

// RebuildProjectIndex builds a fresh index database and swaps it into place.
func RebuildProjectIndex(projectID string, root string) (ReindexResult, error) {
	guard, err := acquireReindexLock(projectID)
	if err != nil {
		return ReindexResult{}, err
	}
	defer func() { _ = guard.Release() }()

	notesSeen, errs, err := ScanNotes(root)
	if err != nil {
		return ReindexResult{}, err
	}
	if len(errs) > 0 {
		return ReindexResult{}, errs[0]
	}
	tempPath, err := tempIndexPath(projectID)
	if err != nil {
		return ReindexResult{}, err
	}
	_ = os.Remove(tempPath)
	if err := os.MkdirAll(filepath.Dir(tempPath), 0o755); err != nil {
		return ReindexResult{}, err
	}
	db, err := sql.Open(sqliteDriverName, tempPath)
	if err != nil {
		return ReindexResult{}, err
	}
	if err := ApplySchema(db); err != nil {
		_ = db.Close()
		return ReindexResult{}, err
	}
	if err := insertDocs(db, notesSeen); err != nil {
		_ = db.Close()
		_ = os.Remove(tempPath)
		return ReindexResult{}, err
	}
	if err := QuickCheck(tempPath); err != nil {
		_ = db.Close()
		_ = os.Remove(tempPath)
		return ReindexResult{}, err
	}
	_ = db.Close()

	finalPath, err := Path(projectID)
	if err != nil {
		return ReindexResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return ReindexResult{}, err
	}
	if err := removeExistingIndexFiles(finalPath); err != nil {
		return ReindexResult{}, err
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return ReindexResult{}, err
	}
	return ReindexResult{ProjectID: projectID, NotesSeen: len(notesSeen), NotesIndexed: len(notesSeen), Status: "ok"}, nil
}

func removeExistingIndexFiles(path string) error {
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		if err := os.Remove(candidate); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func insertDocs(db *sql.DB, docs []NoteDoc) error {
	seenNoteIDs := make(map[string]struct{})
	seenSlugs := make(map[string]struct{})
	seenNorm := make(map[string]int)
	for _, doc := range docs {
		if _, ok := seenNoteIDs[doc.NoteID]; ok {
			return fmt.Errorf("duplicate note_id %s", doc.NoteID)
		}
		seenNoteIDs[doc.NoteID] = struct{}{}
		if _, ok := seenSlugs[doc.Slug]; ok {
			return fmt.Errorf("duplicate slug %s", doc.Slug)
		}
		seenSlugs[doc.Slug] = struct{}{}
		norm, _ := normalizeTitleSlug(doc.Title)
		seenNorm[norm]++
	}
	for _, doc := range docs {
		if _, err := db.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
			doc.NoteID, "project", doc.Slug, doc.RelPath, doc.Title, doc.ContentHash); err != nil {
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
				hashString(doc.NoteID+ob.Content), doc.NoteID, ob.Category, ob.Content)
		}
		for _, link := range doc.Links {
			toID := sql.NullString{}
			if resolved, ok := resolveLinkTarget(docs, seenNorm, link.RawTarget); ok {
				toID.Valid = true
				toID.String = resolved
			}
			_, _ = db.Exec(`INSERT INTO links(link_id, note_id, to_note_id, target, relation_type, source_line) VALUES (?, ?, ?, ?, ?, ?)`,
				hashString(doc.NoteID+link.RawTarget+link.Source+fmt.Sprint(link.Line)), doc.NoteID, toID, link.RawTarget, link.RelationType, link.Line)
		}
	}
	return nil
}

func resolveLinkTarget(docs []NoteDoc, norms map[string]int, target string) (string, bool) {
	for _, doc := range docs {
		if doc.NoteID == target || doc.Slug == target || doc.RelPath == target || doc.Title == target {
			return doc.NoteID, true
		}
	}
	norm, _ := normalizeTitleSlug(target)
	if norms[norm] != 1 {
		return "", false
	}
	for _, doc := range docs {
		if n, _ := normalizeTitleSlug(doc.Title); n == norm {
			return doc.NoteID, true
		}
	}
	return "", false
}

func normalizeTitleSlug(s string) (string, error) {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ', r == '-', r == '_':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-"), nil
}

func hashNoteBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func hashString(s string) string { return hashNoteBytes([]byte(s)) }

func tempIndexPath(projectID string) (string, error) {
	p, err := Path(projectID)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(p, ".sqlite") + ".new.sqlite", nil
}

// UpdateProjectIndexStatus is a tiny helper used by CLI commands.
func UpdateProjectIndexStatus(db *sql.DB, projectID string, present bool, needsReindex bool) error {
	_, err := db.Exec(`UPDATE project_status SET index_present = ?, needs_reindex = ?, last_seen_at = ? WHERE project_id = ?`,
		boolToInt(present), boolToInt(needsReindex), time.Now().UTC().Format(time.RFC3339), projectID)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
