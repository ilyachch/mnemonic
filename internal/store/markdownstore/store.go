package markdownstore

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/domain/slug"
	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/platform/clock"
	mnemonicfs "github.com/ilyachch/mnemonic/internal/platform/fs"
	"github.com/ilyachch/mnemonic/internal/platform/idgen"
	"github.com/ilyachch/mnemonic/internal/platform/lock"
	"github.com/ilyachch/mnemonic/internal/platform/parallel"
	_ "modernc.org/sqlite"
)

const writeLockName = "write"
const trashTimestampLayout = "20060102T150405Z"
const sqliteDriverName = "sqlite"

// Store provides root-scoped markdown note persistence and lookup.
type Store struct {
	RootDir   string
	StateDir  string
	IndexPath string
}

// CreateInput configures note creation.
type CreateInput struct {
	Title string
	Body  []byte
	Tags  []string
	Now   func() time.Time
	UUID  func() string
}

// CreateResult describes the created note for CLI output.
type CreateResult struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
}

// EditInput configures note editing.
type EditInput struct {
	Selector string
	Append   []byte
	Body     []byte
	HasBody  bool
	Set      map[string]string
	IfMatch  string
	Now      func() time.Time
}

// EditResult describes the edited note.
type EditResult struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// DeleteInput configures note deletion.
type DeleteInput struct {
	Selector string
	DryRun   bool
	Hard     bool
	TrashDir string
	Yes      bool
	Now      func() time.Time
}

// DeleteResult describes the outcome of a note deletion.
type DeleteResult struct {
	Mode      string `json:"mode"`
	Path      string `json:"path,omitempty"`
	TrashPath string `json:"trash_path,omitempty"`
}

// NoteSummary is the note listing projection used by the CLI.
type NoteSummary struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Path        string `json:"path"`
	UpdatedAt   string `json:"updated_at"`
	ContentHash string `json:"content_hash"`
}

// ResolvedNote contains the matched note and its relative path.
type ResolvedNote struct {
	Note markdown.Note
	Path string
}

// ShowResult describes a note show response.
type ShowResult struct {
	ResolvedNote
	ContentHash string `json:"content_hash"`
	RawMarkdown []byte `json:"-"`
}

// Create writes a new markdown note beneath the store root.
func (s Store) Create(input CreateInput) (CreateResult, error) {
	root := s.rootDir()
	if root == "" {
		return CreateResult{}, errors.New("root directory is required")
	}
	if input.Title == "" {
		return CreateResult{}, apperr.CLIUsage("note title is required", nil)
	}

	guard, err := acquireWriteLock(root, s.StateDir)
	if err != nil {
		return CreateResult{}, err
	}
	defer func() { _ = guard.Release() }()

	slugValue, err := slug.Slugify(input.Title)
	if err != nil {
		return CreateResult{}, err
	}

	now := input.Now
	if now == nil {
		now = clock.NowUTC
	}
	uuidFn := input.UUID
	if uuidFn == nil {
		uuidFn = idgen.NewUUID
	}

	timestamp := now().UTC()
	note := markdown.Note{
		MnemonicNoteID: uuidFn(),
		Title:          input.Title,
		Slug:           slugValue,
		Tags:           dedupeTags(input.Tags),
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
		Body:           append([]byte(nil), input.Body...),
	}

	rendered, err := markdown.RenderNote(note)
	if err != nil {
		return CreateResult{}, err
	}

	relPath := slugValue + ".md"
	absPath := filepath.Join(root, relPath)
	if _, err := os.Stat(absPath); err == nil {
		return CreateResult{}, apperr.Ambiguous(fmt.Sprintf("note slug %q already exists", slugValue), nil)
	} else if !os.IsNotExist(err) {
		return CreateResult{}, fmt.Errorf("check note path %q: %w", absPath, err)
	}

	if err := mnemonicfs.AtomicWriteFile(absPath, rendered, 0o644); err != nil {
		return CreateResult{}, err
	}

	return CreateResult{
		NoteID:      note.MnemonicNoteID,
		Slug:        slugValue,
		Path:        filepath.ToSlash(relPath),
		ContentHash: HashBytes(rendered),
	}, nil
}

// Edit applies an append edit to a note.
func (s Store) Edit(input EditInput) (EditResult, error) {
	root := s.rootDir()
	if root == "" {
		return EditResult{}, errors.New("root directory is required")
	}
	if input.Selector == "" {
		return EditResult{}, apperr.NotFound("note selector is required", nil)
	}

	guard, err := acquireWriteLock(root, s.StateDir)
	if err != nil {
		return EditResult{}, err
	}
	defer func() { _ = guard.Release() }()

	resolved, err := s.Resolve(input.Selector)
	if err != nil {
		return EditResult{}, err
	}

	absPath := filepath.Join(root, filepath.FromSlash(resolved.Path))
	currentBytes, err := os.ReadFile(absPath)
	if err != nil {
		return EditResult{}, fmt.Errorf("read note: %w", err)
	}

	edited, err := markdown.ParseNote(currentBytes)
	if err != nil {
		return EditResult{}, fmt.Errorf("parse note: %w", err)
	}

	if input.IfMatch != "" {
		if HashBytes(currentBytes) != input.IfMatch {
			return EditResult{}, apperr.Unsafe("content hash mismatch", nil)
		}
	}

	if err = applyEditSet(&edited, input.Set); err != nil {
		return EditResult{}, err
	}

	now := input.Now
	if now == nil {
		now = clock.NowUTC
	}

	if input.HasBody {
		edited.Body = append([]byte(nil), input.Body...)
	} else {
		edited.Body = append(append([]byte(nil), edited.Body...), input.Append...)
	}
	edited.UpdatedAt = now().UTC()

	rendered, err := markdown.RenderNote(edited)
	if err != nil {
		return EditResult{}, err
	}

	if err := mnemonicfs.AtomicWriteFile(absPath, rendered, 0o644); err != nil {
		return EditResult{}, err
	}

	return EditResult{
		NoteID:      edited.MnemonicNoteID,
		Slug:        edited.EffectiveSlug(),
		Path:        resolved.Path,
		ContentHash: HashBytes(rendered),
		CreatedAt:   edited.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   edited.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// Delete removes or trashes a note.
func (s Store) Delete(input DeleteInput) (DeleteResult, error) {
	root := s.rootDir()
	if root == "" {
		return DeleteResult{}, errors.New("root directory is required")
	}
	if input.Selector == "" {
		return DeleteResult{}, apperr.NotFound("note selector is required", nil)
	}

	guard, err := acquireWriteLock(root, s.StateDir)
	if err != nil {
		return DeleteResult{}, err
	}
	defer func() { _ = guard.Release() }()

	resolved, err := s.Resolve(input.Selector)
	if err != nil {
		return DeleteResult{}, err
	}
	if input.Hard {
		return s.hardDelete(root, resolved, input)
	}
	return s.trashDelete(root, resolved, input)
}

func (s Store) hardDelete(root string, resolved ResolvedNote, input DeleteInput) (DeleteResult, error) {
	if !input.Yes {
		return DeleteResult{}, apperr.Unsafe("hard delete requires --yes", nil)
	}
	if input.DryRun {
		return DeleteResult{Mode: "hard", Path: resolved.Path}, nil
	}
	absPath := filepath.Join(root, filepath.FromSlash(resolved.Path))
	if err := os.Remove(absPath); err != nil {
		return DeleteResult{}, fmt.Errorf("remove note: %w", err)
	}
	return DeleteResult{Mode: "hard", Path: resolved.Path}, nil
}

func (s Store) trashDelete(root string, resolved ResolvedNote, input DeleteInput) (DeleteResult, error) {
	trashPath, err := ResolveTrashPath(TrashPathInput{
		RootDir:      root,
		OriginalPath: resolved.Path,
		TrashDirName: input.TrashDir,
		Now:          input.Now,
	})
	if err != nil {
		return DeleteResult{}, err
	}
	if input.DryRun {
		return DeleteResult{Mode: "trash", Path: resolved.Path, TrashPath: trashPath}, nil
	}
	absPath := filepath.Join(root, filepath.FromSlash(resolved.Path))
	data, err := os.ReadFile(absPath)
	if err != nil {
		return DeleteResult{}, fmt.Errorf("read note: %w", err)
	}
	if err = mnemonicfs.WriteFile(trashPath, data, 0o644); err != nil {
		return DeleteResult{}, fmt.Errorf("move note to trash: %w", err)
	}
	if err = os.Remove(absPath); err != nil {
		return DeleteResult{}, fmt.Errorf("remove source note: %w", err)
	}
	return DeleteResult{Mode: "trash", Path: resolved.Path, TrashPath: trashPath}, nil
}

// List returns all note summaries under the store root, excluding .trash notes.
func (s Store) List() ([]NoteSummary, error) {
	root := s.rootDir()
	if root == "" {
		return nil, errors.New("root directory is required")
	}

	paths, err := s.Walk()
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	summaries, errs := parallel.MapIndexed(paths, runtime.GOMAXPROCS(0), func(_ int, relPath string) (NoteSummary, error) {
		absPath := filepath.Join(root, filepath.FromSlash(relPath))
		data, err := os.ReadFile(absPath)
		if err != nil {
			return NoteSummary{}, fmt.Errorf("read note %q: %w", relPath, err)
		}

		note, err := markdown.ParseNote(data)
		if err != nil {
			return NoteSummary{}, fmt.Errorf("parse note %q: %w", relPath, err)
		}
		if note.MnemonicNoteID == "" {
			return NoteSummary{}, fmt.Errorf("note %q is missing mnemonic_note_id", relPath)
		}
		if note.UpdatedAt.IsZero() {
			return NoteSummary{}, fmt.Errorf("note %q is missing updated_at", relPath)
		}

		return NoteSummary{
			NoteID:      note.MnemonicNoteID,
			Slug:        note.EffectiveSlug(),
			Title:       note.Title,
			Path:        relPath,
			UpdatedAt:   note.UpdatedAt.UTC().Format(time.RFC3339),
			ContentHash: HashBytes(data),
		}, nil
	})
	if err := parallel.FirstError(errs); err != nil {
		return nil, err
	}

	notes := make([]NoteSummary, 0, len(paths))
	notes = append(notes, summaries...)

	return notes, nil
}

// Show resolves a note selector and returns the parsed note plus content hash.
func (s Store) Show(selector string) (ShowResult, error) {
	resolved, err := s.Resolve(selector)
	if err != nil {
		return ShowResult{}, err
	}

	data, err := os.ReadFile(filepath.Join(s.rootDir(), filepath.FromSlash(resolved.Path)))
	if err != nil {
		return ShowResult{}, fmt.Errorf("read note %q: %w", resolved.Path, err)
	}

	note, err := markdown.ParseNote(data)
	if err != nil {
		return ShowResult{}, fmt.Errorf("parse note %q: %w", resolved.Path, err)
	}

	return ShowResult{
		ResolvedNote: ResolvedNote{
			Note: note,
			Path: resolved.Path,
		},
		ContentHash: HashBytes(data),
		RawMarkdown: append([]byte(nil), data...),
	}, nil
}

// Resolve finds a note using the canonical selector precedence.
func (s Store) Resolve(selector string) (ResolvedNote, error) {
	if selector == "" {
		return ResolvedNote{}, apperr.NotFound("note selector is required", nil)
	}

	indexPath := strings.TrimSpace(s.IndexPath)
	if indexPath == "" {
		return ResolvedNote{}, errors.New("index path is required")
	}
	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			return ResolvedNote{}, apperr.NotFound("index missing; run `mnemonic project reindex` before resolving", nil)
		}
		return ResolvedNote{}, fmt.Errorf("stat index %q: %w", indexPath, err)
	}

	db, err := s.openIndexReadonly()
	if err != nil {
		return ResolvedNote{}, err
	}
	defer func() { _ = db.Close() }()

	stages := []func() (ResolvedNote, bool, error){
		func() (ResolvedNote, bool, error) {
			return s.lookupIndexedNote(db, `SELECT note_id, slug, title, rel_path FROM notes WHERE note_id = ? LIMIT 2`, selector)
		},
		func() (ResolvedNote, bool, error) {
			return s.lookupIndexedNote(db, `SELECT note_id, slug, title, rel_path FROM notes WHERE slug = ? LIMIT 2`, selector)
		},
		func() (ResolvedNote, bool, error) {
			normalized, err := normalizePathSelector(selector)
			if err != nil {
				return ResolvedNote{}, false, err
			}
			return s.lookupIndexedNote(db, `SELECT note_id, slug, title, rel_path FROM notes WHERE rel_path = ? LIMIT 2`, normalized)
		},
		func() (ResolvedNote, bool, error) { return s.lookupIndexedTitle(db, selector) },
	}

	for _, stage := range stages {
		match, ok, err := stage()
		if err != nil {
			return ResolvedNote{}, err
		}
		if ok {
			return match, nil
		}
	}

	return ResolvedNote{}, apperr.NotFound(fmt.Sprintf("note %q not found", selector), nil)
}

// Walk returns all Markdown note paths under the store root as slash-separated relative paths.
func (s Store) Walk() ([]string, error) {
	root := s.rootDir()
	if root == "" {
		return nil, errors.New("root directory is required")
	}

	return walkRoot(root)
}

func walkRoot(root string) ([]string, error) {
	var notes []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || path == root {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("rel %q: %w", path, err)
		}
		if !shouldIncludeEntry(entry, filepath.ToSlash(rel)) {
			if entry.IsDir() && entry.Name() == ".trash" {
				return filepath.SkipDir
			}
			return nil
		}
		notes = append(notes, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return notes, nil
}

func shouldIncludeEntry(entry fs.DirEntry, rel string) bool {
	if entry.IsDir() {
		return false
	}
	if entry.Name() == "mnemonic.toml" {
		return false
	}
	if !strings.HasSuffix(entry.Name(), ".md") {
		return false
	}
	if strings.Contains(rel, "/.trash/") || strings.HasPrefix(rel, ".trash/") || rel == ".trash" {
		return false
	}
	return true
}

// HashBytes returns a deterministic SHA-256 content hash for note bytes.
func HashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type TrashPathInput struct {
	RootDir      string
	OriginalPath string
	TrashDirName string
	Now          func() time.Time
}

// ResolveTrashPath returns a unique trash path for a note and creates parent directories under the configured trash root.
func ResolveTrashPath(input TrashPathInput) (string, error) {
	rootDir := input.RootDir
	if rootDir == "" {
		return "", errors.New("root directory is required")
	}

	trashDirName := input.TrashDirName
	if trashDirName == "" {
		trashDirName = ".trash"
	}
	trashDirName, err := cleanRelativePath(trashDirName)
	if err != nil {
		return "", fmt.Errorf("trash directory: %w", err)
	}

	now := input.Now
	if now == nil {
		now = time.Now
	}

	originalPath, err := cleanRelativePath(input.OriginalPath)
	if err != nil {
		return "", err
	}

	trashParent := filepath.Join(rootDir, trashDirName, filepath.Dir(originalPath))
	if filepath.Dir(originalPath) == "." {
		trashParent = filepath.Join(rootDir, trashDirName)
	}
	if err := os.MkdirAll(trashParent, 0o755); err != nil {
		return "", fmt.Errorf("create trash parent dirs: %w", err)
	}

	base := filepath.Base(originalPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	stamp := now().UTC().Format(trashTimestampLayout)

	for suffix := 0; ; suffix++ {
		name := fmt.Sprintf("%s.deleted-%s%s", stem, stamp, ext)
		if suffix > 0 {
			name = fmt.Sprintf("%s.deleted-%s-%d%s", stem, stamp, suffix, ext)
		}

		candidate := filepath.Join(trashParent, name)
		_, statErr := os.Stat(candidate)
		switch {
		case statErr == nil:
			continue
		case !os.IsNotExist(statErr):
			return "", fmt.Errorf("check trash path %q: %w", candidate, statErr)
		default:
			return candidate, nil
		}
	}
}

func cleanRelativePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("original path is required")
	}
	if filepath.IsAbs(path) {
		return "", errors.New("original path must be relative")
	}

	cleaned := filepath.Clean(path)
	if cleaned == "." {
		return "", errors.New("original path is required")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", errors.New("original path escapes root directory")
	}

	return cleaned, nil
}

func acquireWriteLock(root string, stateDir string) (*lock.Guard, error) {
	if strings.TrimSpace(root) == "" {
		return nil, apperr.CLIUsage("root directory is required", nil)
	}
	if strings.TrimSpace(stateDir) == "" {
		return nil, apperr.CLIUsage("state directory is required", nil)
	}

	guard, err := lock.Acquire(lock.AcquireInput{
		StateDir:     stateDir,
		Name:         writeLockName,
		Timeout:      150 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
	})
	if err == nil {
		return guard, nil
	}
	if errors.Is(err, lock.ErrBusy) {
		return nil, apperr.Unsafe("project write lock is busy", err)
	}
	return nil, err
}

func applyEditSet(note *markdown.Note, set map[string]string) error {
	if note == nil || len(set) == 0 {
		return nil
	}
	if note.Frontmatter == nil {
		note.Frontmatter = map[string]any{}
	}

	for key, value := range set {
		switch key {
		case "mnemonic_note_id", "created_at":
			return apperr.Unsafe(fmt.Sprintf("frontmatter %q is protected", key), nil)
		case "title":
			note.Title = value
			note.Frontmatter[key] = value
		case "slug":
			note.Slug = value
			note.Frontmatter[key] = value
		case "type":
			note.Type = value
			note.Frontmatter[key] = value
		case "permalink":
			note.Permalink = value
			note.Frontmatter[key] = value
		default:
			note.Frontmatter[key] = value
		}
	}

	return nil
}

func dedupeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}

	return out
}

func (s Store) rootDir() string {
	return strings.TrimSpace(s.RootDir)
}

func (s Store) openIndexReadonly() (*sql.DB, error) {
	indexPath := strings.TrimSpace(s.IndexPath)
	if indexPath == "" {
		return nil, errors.New("index path is required")
	}

	db, err := sql.Open(sqliteDriverName, "file:"+filepath.ToSlash(indexPath)+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open index database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping index database: %w", err)
	}
	return db, nil
}

func (s Store) lookupIndexedNote(db *sql.DB, query string, selector string) (ResolvedNote, bool, error) {
	rows, err := db.Query(query, selector)
	if err != nil {
		return ResolvedNote{}, false, fmt.Errorf("query note %q: %w", selector, err)
	}
	defer func() { _ = rows.Close() }()

	var note ResolvedNote
	count := 0
	for rows.Next() {
		count++
		if count > 1 {
			return ResolvedNote{}, false, apperr.Ambiguous(fmt.Sprintf("note selector %q matches multiple notes", selector), nil)
		}
		if err := rows.Scan(&note.Note.MnemonicNoteID, &note.Note.Slug, &note.Note.Title, &note.Path); err != nil {
			return ResolvedNote{}, false, fmt.Errorf("scan note %q: %w", selector, err)
		}
	}
	if err := rows.Err(); err != nil {
		return ResolvedNote{}, false, fmt.Errorf("iterate note %q: %w", selector, err)
	}
	if count == 0 {
		return ResolvedNote{}, false, nil
	}
	return note, true, nil
}

func (s Store) lookupIndexedTitle(db *sql.DB, selector string) (ResolvedNote, bool, error) {
	rows, err := db.Query(`SELECT note_id, slug, title, rel_path FROM notes WHERE title = ? LIMIT 2`, selector)
	if err != nil {
		return ResolvedNote{}, false, fmt.Errorf("query note %q: %w", selector, err)
	}
	defer func() { _ = rows.Close() }()

	var note ResolvedNote
	count := 0
	for rows.Next() {
		count++
		if count > 1 {
			return ResolvedNote{}, false, apperr.Ambiguous(fmt.Sprintf("note selector %q matches multiple notes", selector), nil)
		}
		if err := rows.Scan(&note.Note.MnemonicNoteID, &note.Note.Slug, &note.Note.Title, &note.Path); err != nil {
			return ResolvedNote{}, false, fmt.Errorf("scan note %q: %w", selector, err)
		}
	}
	if err := rows.Err(); err != nil {
		return ResolvedNote{}, false, fmt.Errorf("iterate note %q: %w", selector, err)
	}
	if count == 0 {
		return ResolvedNote{}, false, nil
	}
	return note, true, nil
}

func normalizePathSelector(selector string) (string, error) {
	cleaned := filepath.Clean(strings.ReplaceAll(selector, "\\", "/"))
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "", apperr.NotFound(fmt.Sprintf("note %q not found", selector), nil)
	}
	if filepath.IsAbs(cleaned) {
		return "", apperr.NotFound(fmt.Sprintf("note %q not found", selector), nil)
	}
	return filepath.ToSlash(cleaned), nil
}
