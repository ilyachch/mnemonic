package sqliteindex

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gofrs/flock"
	"github.com/ilyachch/mnemonic/internal/apperr"
	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

// Store provides path-explicit access to a SQLite index database.
type Store struct {
	IndexPath string
	RootDir   string
	StateDir  string
	KBID      string
}

// SchemaStatus describes whether an existing index can be reused.
type SchemaStatus string

const (
	// SchemaStatusOK indicates that the database schema is compatible.
	SchemaStatusOK SchemaStatus = "ok"
	// SchemaStatusNeedsRebuild indicates that the index must be rebuilt.
	SchemaStatusNeedsRebuild SchemaStatus = "needs_rebuild"
)

// SearchHit is a single FTS search result from the index database.
type SearchHit struct {
	NoteID      string  `json:"note_id"`
	Slug        string  `json:"slug"`
	Title       string  `json:"title"`
	Path        string  `json:"path"`
	Score       float64 `json:"score"`
	Snippet     string  `json:"snippet"`
	ContentHash string  `json:"content_hash"`
}

// TagCount is a grouped tag row from the index database.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// IndexedNote is the minimal note record used for index lookups.
type IndexedNote struct {
	NoteID string `json:"note_id"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Path   string `json:"path"`
}

// Backlink is a resolved reference from another note.
type Backlink struct {
	LinkID       string `json:"link_id"`
	NoteID       string `json:"note_id"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Path         string `json:"path"`
	RelationType string `json:"relation_type"`
	SourceKind   string `json:"source_kind"`
	SourceLine   int    `json:"source_line"`
}

// SearchOptions configures an advanced search with multi-query, time, tag, and
// graph-aware features.
type SearchOptions struct {
	Queries        []string
	Tags           []string
	CreatedBefore  *int64
	CreatedAfter   *int64
	UpdatedBefore  *int64
	UpdatedAfter   *int64
	CreatedSince   string
	UpdatedSince   string
	Limit          int
	IncludeRelated bool
}

// SearchResult is an extended search hit that may carry related notes.
type SearchResult struct {
	NoteID         string        `json:"note_id"`
	Slug           string        `json:"slug"`
	Title          string        `json:"title"`
	Path           string        `json:"path"`
	Score          float64       `json:"score"`
	Snippet        string        `json:"snippet"`
	ContentHash    string        `json:"content_hash"`
	Summary        string        `json:"summary"`
	Tags           []string      `json:"tags,omitempty"`
	RelatedNotes   []RelatedNote `json:"related_notes,omitempty"`
	MatchCount     int           `json:"-"`
	MatchedQueries []string      `json:"matched_queries,omitempty"`
}

// RelatedNote is a short linked-note reference.
type RelatedNote struct {
	NoteID       string `json:"note_id"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Path         string `json:"path"`
	RelationType string `json:"relation_type"`
	SourceKind   string `json:"source_kind"`
	Direction    string `json:"direction"`
}

// Open opens the index database, creating parent directories and applying the
// required connection pragmas.
func (s Store) Open() (*sql.DB, error) {
	if err := s.validateIndexPath(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(s.IndexPath), 0o755); err != nil {
		return nil, fmt.Errorf("create index directory: %w", err)
	}
	return openDB(s.IndexPath)
}

// OpenReadonly opens the index database in read-only mode.
func (s Store) OpenReadonly() (*sql.DB, error) {
	if err := s.validateIndexPath(); err != nil {
		return nil, err
	}
	db, err := sql.Open(sqliteDriverName, "file:"+filepath.ToSlash(s.IndexPath)+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open index database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping index database: %w", err)
	}
	return db, nil
}

// Exists reports whether the index database file exists.
func (s Store) Exists() (bool, error) {
	if err := s.validateIndexPath(); err != nil {
		return false, err
	}
	_, err := os.Stat(s.IndexPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Remove deletes the index database and SQLite sidecars.
func (s Store) Remove() error {
	if err := s.validateIndexPath(); err != nil {
		return err
	}
	return removeIndexFiles(s.IndexPath)
}

// QuickCheck runs SQLite integrity checks for the explicit index path.
func (s Store) QuickCheck() error {
	if err := s.validateIndexPath(); err != nil {
		return err
	}
	db, err := sql.Open(sqliteDriverName, "file:"+filepath.ToSlash(s.IndexPath)+"?mode=ro")
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

// CheckSchemaStatus reports whether the current DB schema is compatible.
func (s Store) CheckSchemaStatus(db *sql.DB) (SchemaStatus, error) {
	return CheckSchemaStatus(db)
}

// SchemaStatus opens the index read-only and reports whether the schema is compatible.
func (s Store) SchemaStatus() (SchemaStatus, error) {
	db, err := s.OpenReadonly()
	if err != nil {
		return "", err
	}
	defer func() { _ = db.Close() }()

	return CheckSchemaStatus(db)
}

// Search runs an FTS query against the index database.
func (s Store) Search(db *sql.DB, query string, limit int, tag string) ([]SearchHit, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}
	query = sanitizeFTSQuery(query)
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query is required")
	}
	if limit <= 0 {
		limit = 20
	}
	tag = strings.TrimSpace(tag)

	sqlQuery := `
		SELECT n.note_id, n.slug, n.title, n.rel_path, bm25(notes_fts, 10.0, 5.0, 5.0, 2.0, 1.0) AS score,
		       snippet(notes_fts, 5, '[', ']', '...', 12) AS snippet,
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

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	results := make([]SearchHit, 0)
	for rows.Next() {
		var r SearchHit
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

// SearchAdvanced runs an advanced search with multi-query, time filters, tag
// filters, graph-aware reranking, and optional related notes.
func (s Store) SearchAdvanced(db *sql.DB, opts SearchOptions, now time.Time) ([]SearchResult, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	ca, cb, ua, ub, err := resolveTimeBounds(opts, now)
	if err != nil {
		return nil, err
	}

	hasTimeFilter := ca != nil || cb != nil || ua != nil || ub != nil
	hasTagFilter := len(opts.Tags) > 0
	hasQuery := len(opts.Queries) > 0

	if !searchParamsValid(hasQuery, hasTimeFilter, hasTagFilter) {
		return nil, errors.New("at least one query, tag, or time filter is required")
	}

	results, err := s.runSearch(db, opts, hasQuery, ca, cb, ua, ub)
	if err != nil {
		return nil, err
	}

	if hasQuery && len(results) > 1 {
		results = s.rerankByGraphLinks(db, results)
	}

	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	if err := s.populateSearchTags(db, results); err != nil {
		return nil, err
	}
	s.applySnippetFallback(results)

	if opts.IncludeRelated {
		if err := s.populateRelatedNotes(db, results); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (s Store) runSearch(db *sql.DB, opts SearchOptions, hasQuery bool, ca, cb, ua, ub *int64) ([]SearchResult, error) {
	if hasQuery {
		return s.searchMultiQuery(db, opts, ca, cb, ua, ub)
	}
	return s.searchNotesByFilter(db, opts, ca, cb, ua, ub)
}

func resolveTimeBounds(opts SearchOptions, now time.Time) (*int64, *int64, *int64, *int64, error) {
	ca, err := resolveTimeBound(opts.CreatedAfter, opts.CreatedSince, now)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("created_since: %w", err)
	}
	ua, err := resolveTimeBound(opts.UpdatedAfter, opts.UpdatedSince, now)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("updated_since: %w", err)
	}
	return ca, opts.CreatedBefore, ua, opts.UpdatedBefore, nil
}

func searchParamsValid(hasQuery, hasTimeFilter, hasTagFilter bool) bool {
	return hasQuery || hasTimeFilter || hasTagFilter
}

const rrfK = 60.0

func (s Store) searchMultiQuery(db *sql.DB, opts SearchOptions, ca, cb, ua, ub *int64) ([]SearchResult, error) {
	timeClause, timeArgs := buildTimeFilterClause(ca, cb, ua, ub)
	tagClause, tagArgs := buildTagFilterClause(opts.Tags)

	type docEntry struct {
		result         SearchResult
		matchCount     int
		matchedQueries map[string]bool
	}
	docs := make(map[string]*docEntry)

	for _, q := range opts.Queries {
		clean := sanitizeFTSQuery(q)
		if strings.TrimSpace(clean) == "" {
			continue
		}
		hits, err := runFTSSearch(db, clean, opts.Limit, timeClause, timeArgs, tagClause, tagArgs)
		if err != nil {
			return nil, err
		}
		for rank, hit := range hits {
			entry, ok := docs[hit.NoteID]
			if !ok {
				r := hit
				r.Score = 0.0
				entry = &docEntry{result: r, matchedQueries: make(map[string]bool)}
				docs[hit.NoteID] = entry
			}
			entry.result.Score += 1.0 / (rrfK + float64(rank+1))
			entry.matchCount++
			entry.matchedQueries[q] = true
		}
	}

	results := make([]SearchResult, 0, len(docs))
	for _, e := range docs {
		e.result.MatchCount = e.matchCount
		queries := make([]string, 0, len(e.matchedQueries))
		for q := range e.matchedQueries {
			queries = append(queries, q)
		}
		e.result.MatchedQueries = queries
		results = append(results, e.result)
	}

	sortSearchResults(results)
	return results, nil
}

func (s Store) searchNotesByFilter(db *sql.DB, opts SearchOptions, ca, cb, ua, ub *int64) ([]SearchResult, error) {
	var w strings.Builder
	w.WriteString("WHERE 1=1")
	var args []any

	if ca != nil {
		w.WriteString(" AND n.created_at > ?")
		args = append(args, *ca)
	}
	if cb != nil {
		w.WriteString(" AND n.created_at < ?")
		args = append(args, *cb)
	}
	if ua != nil {
		w.WriteString(" AND n.updated_at > ?")
		args = append(args, *ua)
	}
	if ub != nil {
		w.WriteString(" AND n.updated_at < ?")
		args = append(args, *ub)
	}

	for _, t := range opts.Tags {
		w.WriteString(` AND EXISTS (
		SELECT 1 FROM note_tags nt
		WHERE nt.note_id = n.note_id
		  AND nt.tag LIKE ?)`)
		args = append(args, "%:"+strings.TrimSpace(t))
	}

	query := "SELECT n.note_id, n.slug, n.title, n.rel_path, 0.0, '', n.content_hash, n.summary FROM notes n " + w.String() + " ORDER BY n.created_at DESC LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("filter search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	results := make([]SearchResult, 0)
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.NoteID, &r.Slug, &r.Title, &r.Path, &r.Score, &r.Snippet, &r.ContentHash, &r.Summary); err != nil {
			return nil, fmt.Errorf("scan filter result: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate filter results: %w", err)
	}
	return results, nil
}

func (s Store) rerankByGraphLinks(db *sql.DB, results []SearchResult) []SearchResult {
	const topN = 5
	n := topN
	if len(results) < n {
		n = len(results)
	}
	if n == 0 {
		return results
	}

	topIDs := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		topIDs[results[i].NoteID] = true
	}

	idToIdx := make(map[string]int, len(results))
	allIDs := make([]string, len(results))
	for i, r := range results {
		allIDs[i] = r.NoteID
		idToIdx[r.NoteID] = i
	}

	topList := make([]string, 0, n)
	for id := range topIDs {
		topList = append(topList, id)
	}

	connections := s.countGraphConnections(db, allIDs, topList, idToIdx)

	for i := range results {
		if conn := connections[i]; conn > 0 {
			results[i].Score += 0.1 * float64(conn)
		}
	}

	sortSearchResults(results)
	return results
}

func (s Store) countGraphConnections(db *sql.DB, allIDs, topList []string, idToIdx map[string]int) []int {
	connections := make([]int, len(allIDs))
	s.countOutgoingLinks(db, allIDs, topList, idToIdx, connections)
	s.countIncomingLinks(db, topList, allIDs, idToIdx, connections)
	return connections
}

func (s Store) countOutgoingLinks(db *sql.DB, allIDs, topList []string, idToIdx map[string]int, connections []int) {
	args := append(stringSliceToAny(allIDs), stringSliceToAny(topList)...)
	query := "SELECT note_id FROM links WHERE note_id IN (" + placeholders(len(allIDs)) + ") AND to_note_id IN (" + placeholders(len(topList)) + ") AND to_note_id IS NOT NULL"
	rows, err := db.Query(query, args...)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var noteID string
		if rows.Scan(&noteID) == nil {
			if idx, ok := idToIdx[noteID]; ok {
				connections[idx]++
			}
		}
	}
	_ = rows.Err()
}

func (s Store) countIncomingLinks(db *sql.DB, topList, allIDs []string, idToIdx map[string]int, connections []int) {
	args := append(stringSliceToAny(topList), stringSliceToAny(allIDs)...)
	query := "SELECT to_note_id FROM links WHERE note_id IN (" + placeholders(len(topList)) + ") AND to_note_id IN (" + placeholders(len(allIDs)) + ") AND to_note_id IS NOT NULL"
	rows, err := db.Query(query, args...)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var toNoteID string
		if rows.Scan(&toNoteID) == nil {
			if idx, ok := idToIdx[toNoteID]; ok {
				connections[idx]++
			}
		}
	}
	_ = rows.Err()
}

func (s Store) populateSearchTags(db *sql.DB, results []SearchResult) error {
	if len(results) == 0 {
		return nil
	}
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.NoteID
	}
	rows, err := db.Query(
		`SELECT note_id, tag FROM note_tags WHERE note_id IN (`+placeholders(len(ids))+`) ORDER BY note_id, tag`,
		stringSliceToAny(ids)...,
	)
	if err != nil {
		return fmt.Errorf("query search tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tagsByID := make(map[string][]string, len(results))
	for rows.Next() {
		var noteID, tag string
		if err := rows.Scan(&noteID, &tag); err != nil {
			return fmt.Errorf("scan search tag: %w", err)
		}
		tagsByID[noteID] = append(tagsByID[noteID], tag)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate search tags: %w", err)
	}
	for i := range results {
		tags := tagsByID[results[i].NoteID]
		if tags == nil {
			tags = []string{}
		}
		results[i].Tags = tags
	}
	return nil
}

func (s Store) applySnippetFallback(results []SearchResult) {
	for i := range results {
		if strings.TrimSpace(results[i].Snippet) != "" {
			continue
		}
		if strings.TrimSpace(results[i].Summary) != "" {
			results[i].Snippet = results[i].Summary
		} else {
			results[i].Snippet = results[i].Title
		}
	}
}

func (s Store) populateRelatedNotes(db *sql.DB, results []SearchResult) error {
	if len(results) == 0 {
		return nil
	}

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.NoteID
	}

	query := "SELECT l.note_id, l.to_note_id, n.note_id, n.slug, n.title, n.rel_path, l.relation_type, l.source_kind, 'outgoing' " +
		"FROM links l " +
		"JOIN notes n ON n.note_id = l.to_note_id " +
		"WHERE l.note_id IN (" + placeholders(len(ids)) + ") AND l.to_note_id IS NOT NULL " +
		"UNION ALL " +
		"SELECT l.to_note_id, l.note_id, n.note_id, n.slug, n.title, n.rel_path, l.relation_type, l.source_kind, 'incoming' " +
		"FROM links l " +
		"JOIN notes n ON n.note_id = l.note_id " +
		"WHERE l.to_note_id IN (" + placeholders(len(ids)) + ") AND l.to_note_id IS NOT NULL"

	allIDs := append(stringSliceToAny(ids), stringSliceToAny(ids)...)

	rows, err := db.Query(query, allIDs...)
	if err != nil {
		return fmt.Errorf("query related notes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	relatedByNoteID := make(map[string][]RelatedNote, len(results))
	for rows.Next() {
		var sourceID, targetID, nID, slug, title, path, relType, sourceKind, direction string
		if err := rows.Scan(&sourceID, &targetID, &nID, &slug, &title, &path, &relType, &sourceKind, &direction); err != nil {
			return fmt.Errorf("scan related note: %w", err)
		}
		relatedByNoteID[sourceID] = append(relatedByNoteID[sourceID], RelatedNote{
			NoteID:       nID,
			Slug:         slug,
			Title:        title,
			Path:         path,
			RelationType: relType,
			SourceKind:   sourceKind,
			Direction:    direction,
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate related notes: %w", err)
	}

	for i := range results {
		rn := relatedByNoteID[results[i].NoteID]
		if len(rn) > 3 {
			rn = rn[:3]
		}
		if rn == nil {
			rn = []RelatedNote{}
		}
		results[i].RelatedNotes = rn
	}

	return nil
}

func runFTSSearch(db *sql.DB, query string, limit int, timeClause string, timeArgs []any, tagClause string, tagArgs []any) ([]SearchResult, error) {
	sqlQuery := `SELECT n.note_id, n.slug, n.title, n.rel_path, bm25(notes_fts, 10.0, 5.0, 5.0, 2.0, 1.0) AS score,
		       snippet(notes_fts, 5, '[', ']', '...', 12) AS snippet,
		       n.content_hash, n.summary
		FROM notes_fts
		JOIN notes n ON n.note_id = notes_fts.note_id
		WHERE notes_fts MATCH ?`

	args := []any{query}
	if timeClause != "" {
		sqlQuery += " AND " + timeClause
		args = append(args, timeArgs...)
	}
	if tagClause != "" {
		sqlQuery += " AND " + tagClause
		args = append(args, tagArgs...)
	}
	sqlQuery += " ORDER BY score ASC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	results := make([]SearchResult, 0)
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.NoteID, &r.Slug, &r.Title, &r.Path, &r.Score, &r.Snippet, &r.ContentHash, &r.Summary); err != nil {
			return nil, fmt.Errorf("scan fts result: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fts results: %w", err)
	}
	return results, nil
}

func resolveTimeBound(absolute *int64, relative string, now time.Time) (*int64, error) {
	if relative != "" {
		ts, err := parseRelativeDuration(relative, now)
		if err != nil {
			return nil, err
		}
		return &ts, nil
	}
	return absolute, nil
}

func parseRelativeDuration(s string, now time.Time) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("duration is empty")
	}
	if len(s) < 2 {
		return 0, apperr.CLIUsage("invalid relative duration: "+s, errors.New("expected format like 15m, 24h, or 30d"))
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		return 0, apperr.CLIUsage("invalid relative duration: "+s, errors.New("expected a positive integer followed by m, h, or d"))
	}
	var duration time.Duration
	switch unit {
	case 'm':
		duration = time.Duration(num) * time.Minute
	case 'h':
		duration = time.Duration(num) * time.Hour
	case 'd':
		duration = time.Duration(num) * 24 * time.Hour
	default:
		return 0, apperr.CLIUsage("unsupported duration unit in "+s, errors.New("use m (minutes), h (hours), or d (days)"))
	}
	return now.Add(-duration).Unix(), nil
}

func buildTimeFilterClause(ca, cb, ua, ub *int64) (string, []any) {
	var parts []string
	var args []any
	if ca != nil {
		parts = append(parts, "n.created_at > ?")
		args = append(args, *ca)
	}
	if cb != nil {
		parts = append(parts, "n.created_at < ?")
		args = append(args, *cb)
	}
	if ua != nil {
		parts = append(parts, "n.updated_at > ?")
		args = append(args, *ua)
	}
	if ub != nil {
		parts = append(parts, "n.updated_at < ?")
		args = append(args, *ub)
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, " AND "), args
}

func buildTagFilterClause(tags []string) (string, []any) {
	if len(tags) == 0 {
		return "", nil
	}
	parts := make([]string, len(tags))
	args := make([]any, len(tags))
	for i, t := range tags {
		alias := fmt.Sprintf("nt%d", i)
		parts[i] = fmt.Sprintf("EXISTS (SELECT 1 FROM note_tags %s WHERE %s.note_id = n.note_id AND %s.tag LIKE ?)", alias, alias, alias)
		args[i] = "%:" + strings.TrimSpace(t)
	}
	return strings.Join(parts, " AND "), args
}

func placeholders(n int) string {
	parts := make([]string, n)
	for i := 0; i < n; i++ {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func stringSliceToAny(s []string) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

func sortSearchResults(results []SearchResult) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score || (results[j].Score == results[i].Score && results[j].Title < results[i].Title) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// ListTags returns grouped tags and note counts.
func (s Store) ListTags(db *sql.DB) ([]TagCount, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}

	rows, err := db.Query(
		`SELECT
			CASE
				WHEN instr(tag, ':') > 0 THEN substr(tag, instr(tag, ':') + 1)
				ELSE tag
			END AS tag_name,
			COUNT(DISTINCT note_id) AS count
		 FROM note_tags
		 GROUP BY tag_name
		 ORDER BY count DESC, tag_name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query tag list: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tags := make([]TagCount, 0)
	for rows.Next() {
		var item TagCount
		if err := rows.Scan(&item.Tag, &item.Count); err != nil {
			return nil, fmt.Errorf("scan tag row: %w", err)
		}
		tags = append(tags, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tag rows: %w", err)
	}

	return tags, nil
}

// LookupNoteByIdentifier returns the note matching the selector.
func (s Store) LookupNoteByIdentifier(db *sql.DB, identifier string) (IndexedNote, error) {
	if db == nil {
		return IndexedNote{}, errors.New("db is required")
	}
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return IndexedNote{}, errors.New("identifier is required")
	}

	row := db.QueryRow(
		`SELECT note_id, slug, title, rel_path
		 FROM notes
		 WHERE note_id = ? OR slug = ? OR rel_path = ? OR title = ?`,
		identifier, identifier, identifier, identifier,
	)
	var note IndexedNote
	if err := row.Scan(&note.NoteID, &note.Slug, &note.Title, &note.Path); err != nil {
		if err == sql.ErrNoRows {
			return IndexedNote{}, fmt.Errorf("note %q not found", identifier)
		}
		return IndexedNote{}, fmt.Errorf("query note %q: %w", identifier, err)
	}
	return note, nil
}

// Backlinks returns resolved links pointing to a note.
func (s Store) Backlinks(db *sql.DB, targetNoteID string, limit int) ([]Backlink, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}
	targetNoteID = strings.TrimSpace(targetNoteID)
	if targetNoteID == "" {
		return nil, errors.New("target note id is required")
	}

	sqlQuery := `SELECT l.link_id, l.note_id, n.slug, n.title, n.rel_path, l.relation_type, l.source_kind, l.source_line
		 FROM links l
		 JOIN notes n ON n.note_id = l.note_id
		 WHERE l.to_note_id = ?
		 ORDER BY n.slug ASC, l.source_line ASC, l.link_id ASC`
	args := []any{targetNoteID}
	if limit > 0 {
		sqlQuery += "\nLIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query backlinks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]Backlink, 0)
	for rows.Next() {
		var item Backlink
		if err := rows.Scan(&item.LinkID, &item.NoteID, &item.Slug, &item.Title, &item.Path, &item.RelationType, &item.SourceKind, &item.SourceLine); err != nil {
			return nil, fmt.Errorf("scan backlink: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backlinks: %w", err)
	}

	return out, nil
}

// CountUnresolvedLinks returns the number of links without a resolved target note.
func (s Store) CountUnresolvedLinks(db *sql.DB) (int, error) {
	if db == nil {
		return 0, errors.New("db is required")
	}

	var unresolved int
	if err := db.QueryRow(`SELECT COUNT(*) FROM links WHERE to_note_id IS NULL`).Scan(&unresolved); err != nil {
		return 0, fmt.Errorf("count unresolved links: %w", err)
	}
	return unresolved, nil
}

// LinkIssue describes one unresolved or ambiguous link found during diagnostics.
type LinkIssue struct {
	LinkID      string `json:"link_id"`
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Path        string `json:"path"`
	Target      string `json:"target"`
	Label       string `json:"label"`
	LinkStyle   string `json:"link_style"`
	SourceKind  string `json:"source_kind"`
	SourceLine  int    `json:"source_line"`
	IsAmbiguous bool   `json:"is_ambiguous"`
}

// ListLinkIssues returns all unresolved and ambiguous links with their source
// note context.
func (s Store) ListLinkIssues(db *sql.DB) ([]LinkIssue, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}

	rows, err := db.Query(`
		SELECT l.link_id, l.note_id, n.slug, n.title, n.rel_path,
		       l.target, l.label, l.link_style, l.source_kind, l.source_line,
		       l.is_ambiguous
		FROM links l
		JOIN notes n ON n.note_id = l.note_id
		WHERE l.to_note_id IS NULL
		ORDER BY n.slug ASC, l.source_line ASC`)
	if err != nil {
		return nil, fmt.Errorf("list link issues: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []LinkIssue
	for rows.Next() {
		var item LinkIssue
		if err := rows.Scan(&item.LinkID, &item.NoteID, &item.Slug, &item.Title, &item.Path,
			&item.Target, &item.Label, &item.LinkStyle, &item.SourceKind, &item.SourceLine,
			&item.IsAmbiguous); err != nil {
			return nil, fmt.Errorf("scan link issue: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate link issues: %w", err)
	}
	return out, nil
}

// UnresolvedLinkCount opens the index read-only and counts unresolved links.
func (s Store) UnresolvedLinkCount() (int, error) {
	db, err := s.OpenReadonly()
	if err != nil {
		return 0, err
	}
	defer func() { _ = db.Close() }()

	return s.CountUnresolvedLinks(db)
}

func (s Store) validateIndexPath() error {
	if strings.TrimSpace(s.IndexPath) == "" {
		return errors.New("index path is required")
	}
	return nil
}

func sanitizeFTSQuery(query string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, query)
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open(sqliteDriverName, path)
	if err != nil {
		return nil, fmt.Errorf("open index database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping index database: %w", err)
	}

	pragmas := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA user_version = 2`,
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply %s: %w", pragma, err)
		}
	}

	return db, nil
}

func acquireRebuildLock(indexPath string) (*flock.Flock, error) {
	lockPath := rebuildLockPath(indexPath)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	fileLock := flock.New(lockPath, flock.SetPermissions(0o644))
	deadline := time.Now().Add(150 * time.Millisecond)
	for {
		ok, err := fileLock.TryLock()
		if err != nil {
			return nil, fmt.Errorf("acquire lock: %w", err)
		}
		if ok {
			return fileLock, nil
		}
		if !time.Now().Before(deadline) {
			return nil, apperr.Unsafe("index rebuild lock is busy", fmt.Errorf("%s", lockPath))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func rebuildLockPath(indexPath string) string {
	return filepath.Join(filepath.Dir(indexPath), "reindex.lock")
}

func removeIndexFiles(path string) error {
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		if err := os.Remove(candidate); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func classifyCorruption(err error) error {
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return apperr.NotFound("index database not found", err)
	}
	return apperr.Corrupted("index database is corrupted", err)
}
