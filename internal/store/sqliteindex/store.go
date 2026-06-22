package sqliteindex

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	SourceLine   int    `json:"source_line"`
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

// Search runs an FTS query against the index database.
func (s Store) Search(db *sql.DB, query string, limit int, tag string) ([]SearchHit, error) {
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

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

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

// ListTags returns grouped tags and note counts.
func (s Store) ListTags(db *sql.DB) ([]TagCount, error) {
	if db == nil {
		return nil, fmt.Errorf("db is required")
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
	defer rows.Close()

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
		return IndexedNote{}, fmt.Errorf("db is required")
	}
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return IndexedNote{}, fmt.Errorf("identifier is required")
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
		return nil, fmt.Errorf("db is required")
	}
	targetNoteID = strings.TrimSpace(targetNoteID)
	if targetNoteID == "" {
		return nil, fmt.Errorf("target note id is required")
	}

	sqlQuery := `SELECT l.link_id, l.note_id, n.slug, n.title, n.rel_path, l.relation_type, l.source_line
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

func (s Store) validateIndexPath() error {
	if strings.TrimSpace(s.IndexPath) == "" {
		return fmt.Errorf("index path is required")
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
		`PRAGMA user_version = 1`,
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
