package sqliteindex

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/platform/parallel"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
)

// NoteDoc is a scanned markdown note with metadata used for indexing.
type NoteDoc struct {
	NoteID       string
	Slug         string
	Title        string
	Summary      string
	Aliases      []string
	RelPath      string
	Frontmatter  map[string]any
	BodyMarkdown string
	BodyText     string
	ContentHash  string
	FileMTimeNS  int64
	FileSize     int64
	CreatedAt    int64
	UpdatedAt    int64
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
	Label       string
	LinkStyle   string
	SourceKind  string
	RawTarget   string
	ToNoteID    sql.NullString
	IsResolved  int
	IsAmbiguous int
	Line        int
}

// ScanNotes collects markdown notes from a project root.
func ScanNotes(root string) ([]NoteDoc, []error, error) {
	paths, err := (markdownstore.Store{RootDir: root}).Walk()
	if err != nil {
		return nil, nil, err
	}
	if len(paths) == 0 {
		return nil, nil, nil
	}

	scanned, errs := parallel.MapIndexed(paths, runtime.GOMAXPROCS(0), func(_ int, relPath string) (NoteDoc, error) {
		return scanOneNote(root, relPath)
	})

	docs := make([]NoteDoc, 0, len(paths))
	outErrs := make([]error, 0)
	for i := range paths {
		if errs[i] != nil {
			outErrs = append(outErrs, errs[i])
			continue
		}
		docs = append(docs, scanned[i])
	}
	return docs, outErrs, nil
}

func scanOneNote(root, relPath string) (NoteDoc, error) {
	abs := filepath.Join(root, filepath.FromSlash(relPath))
	data, err := os.ReadFile(abs)
	if err != nil {
		return NoteDoc{}, fmt.Errorf("%s: read note: %w", relPath, err)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return NoteDoc{}, fmt.Errorf("%s: stat note: %w", relPath, err)
	}
	note, err := markdown.ParseNote(data)
	if err != nil {
		return NoteDoc{}, fmt.Errorf("%s: parse note: %w", relPath, err)
	}
	info := NoteDoc{
		NoteID:       note.MnemonicNoteID,
		Slug:         note.EffectiveSlug(),
		Title:        note.Title,
		RelPath:      relPath,
		Frontmatter:  note.Frontmatter,
		BodyMarkdown: string(note.Body),
		BodyText:     string(note.Body),
		ContentHash:  markdownstore.HashBytes(data),
		FileMTimeNS:  st.ModTime().UnixNano(),
		FileSize:     st.Size(),
	}
	populateNoteDocData(&info, note, data)
	return info, nil
}

func populateNoteDocData(info *NoteDoc, note markdown.Note, data []byte) {
	info.Summary = note.Summary
	info.Aliases = note.Aliases
	if !note.CreatedAt.IsZero() {
		info.CreatedAt = note.CreatedAt.Unix()
	}
	if !note.UpdatedAt.IsZero() {
		info.UpdatedAt = note.UpdatedAt.Unix()
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
			ContentHash: markdownstore.HashBytes([]byte(ob.Category + "\n" + ob.Content)),
		})
		info.Tags = append(info.Tags, tagRow{Source: "observation", Value: ob.Category})
	}
	searchable := make([]string, 0, 4+len(note.Tags)+2*len(info.Observations)+len(note.Aliases))
	searchable = append(searchable, note.Title, note.Summary, string(note.Body))
	searchable = append(searchable, note.Tags...)
	searchable = append(searchable, note.Aliases...)
	for _, ob := range info.Observations {
		searchable = append(searchable, ob.Category, ob.Content)
	}
	info.SearchText = strings.Join(searchable, "\n")
	for _, rel := range markdown.ParseRelations(note.Body) {
		target := strings.TrimSpace(rel.Target.Target)
		info.Links = append(info.Links, linkRow{
			RawTarget:  target,
			Label:      rel.Target.Alias,
			LinkStyle:  rel.LinkStyle,
			SourceKind: string(rel.Source),
			Line:       rel.Line,
		})
	}
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
