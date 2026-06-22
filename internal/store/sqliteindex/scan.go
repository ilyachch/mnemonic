package sqliteindex

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
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

// ScanNotes collects markdown notes from a project root.
func ScanNotes(root string) ([]NoteDoc, []error, error) {
	var docs []NoteDoc
	var errs []error

	paths, err := (markdownstore.Store{RootDir: root}).Walk()
	if err != nil {
		return nil, nil, err
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
		for _, rel := range markdown.ParseRelations(note.Body) {
			target := strings.TrimSpace(rel.Target.Target)
			row := linkRow{RelationType: rel.RelationType, RawTarget: target, Source: string(rel.Source), Line: rel.Line}
			info.Links = append(info.Links, row)
		}
		docs = append(docs, info)
	}
	return docs, errs, nil
}

func hashNoteBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizeTitleSlug(s string) string {
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
	return strings.Trim(b.String(), "-")
}
