package graph

import (
	"database/sql"
	"fmt"
)

// Backlink describes a resolved reference from another note to a target note.
type Backlink struct {
	LinkID       string `json:"link_id"`
	NoteID       string `json:"note_id"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Path         string `json:"path"`
	RelationType string `json:"relation_type"`
	SourceLine   int    `json:"source_line"`
}

// Backlinks returns resolved links pointing to targetNoteID.
func Backlinks(db *sql.DB, targetNoteID string) ([]Backlink, error) {
	if db == nil {
		return nil, fmt.Errorf("db is required")
	}
	if targetNoteID == "" {
		return nil, fmt.Errorf("target note id is required")
	}

	rows, err := db.Query(
		`SELECT l.link_id, l.note_id, n.slug, n.title, n.rel_path, l.relation_type, l.source_line
		 FROM links l
		 JOIN notes n ON n.note_id = l.note_id
		 WHERE l.to_note_id = ?
		 ORDER BY n.slug ASC, l.source_line ASC, l.link_id ASC`,
		targetNoteID,
	)
	if err != nil {
		return nil, fmt.Errorf("query backlinks: %w", err)
	}
	defer rows.Close()

	out := make([]Backlink, 0)
	for rows.Next() {
		var item Backlink
		if err := rows.Scan(&item.LinkID, &item.NoteID, &item.Slug, &item.Title, &item.Path, &item.RelationType, &item.SourceLine); err != nil {
			return nil, fmt.Errorf("scan backlink: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backlinks: %w", err)
	}

	return out, nil
}
