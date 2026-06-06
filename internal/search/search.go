package search

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"
)

// Result is a single FTS search hit.
type Result struct {
	NoteID      string  `json:"note_id"`
	Slug        string  `json:"slug"`
	Title       string  `json:"title"`
	Path        string  `json:"path"`
	Score       float64 `json:"score"`
	Snippet     string  `json:"snippet"`
	ContentHash string  `json:"content_hash"`
}

// Search runs an FTS5 query against the notes index.
func Search(db *sql.DB, query string, limit int, tag string) ([]Result, error) {
	if db == nil {
		return nil, fmt.Errorf("db is required")
	}
	query = sanitizeFTSQuery(query)
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	if limit <= 0 {
		limit = 20
	}
	tag = strings.TrimSpace(tag)

	sqlQuery := `
		SELECT n.note_id, n.slug, n.title, n.rel_path, bm25(notes_fts) AS score,
		       snippet(notes_fts, 2, '[', ']', '...', 12) AS snippet,
		       n.content_hash
		FROM notes_fts
		JOIN notes n ON n.note_id = notes_fts.note_id
	`
	args := []any{query}
	where := `WHERE notes_fts MATCH ?`
	if tag != "" {
		where += ` AND EXISTS (
			SELECT 1
			FROM note_tags nt
			WHERE nt.note_id = n.note_id
			  AND nt.tag LIKE ?
		)`
		args = append(args, "%:"+tag)
	}
	sqlQuery += "\n" + where + "\nORDER BY score ASC\nLIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(
		sqlQuery,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.NoteID, &r.Slug, &r.Title, &r.Path, &r.Score, &r.Snippet, &r.ContentHash); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}

	return results, nil
}

func sanitizeFTSQuery(query string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, query)
}
