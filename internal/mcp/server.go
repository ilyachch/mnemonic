package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	mnemonicfs "github.com/ilyachch/mnemonic/internal/fs"
	"github.com/ilyachch/mnemonic/internal/graph"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/search"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server is a transport shell for the future MCP stdio adapter.
type Server struct {
	Project project.ResolvedProject
	Paths   paths.EffectivePaths

	indexDBMu sync.Mutex
	indexConn *sql.DB
}

// NewServer creates a new MCP shell server wrapper.
func NewServer(resolved project.ResolvedProject, effectivePaths paths.EffectivePaths) *Server {
	return &Server{Project: resolved, Paths: effectivePaths}
}

// Run keeps the shell boundary explicit without exposing any tools yet.
func (s *Server) Run(ctx context.Context) error {
	defer func() {
		_ = s.closeIndexDB()
	}()

	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "mnemonic", Version: "0.1.0-dev"},
		&sdkmcp.ServerOptions{
			Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
			Capabilities: &sdkmcp.ServerCapabilities{
				Tools: &sdkmcp.ToolCapabilities{ListChanged: true},
			},
		},
	)

	registerReadOnlyTools(sdkServer, s)
	registerWriteTools(sdkServer, s)

	return sdkServer.Run(ctx, &sdkmcp.StdioTransport{})
}

type listNotesInput struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum number of notes to return"`
	Cursor string `json:"cursor,omitempty" jsonschema:"pagination cursor"`
}

type listNotesOutput struct {
	Notes      []notes.NoteSummary `json:"notes"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type listTagsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of tags to return"`
}

type listTagsOutput struct {
	Tags []listTagsItem `json:"tags"`
}

type listTagsItem struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type searchNotesInput struct {
	Query string `json:"query" jsonschema:"full-text search query"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum number of hits to return"`
	Tag   string `json:"tag,omitempty" jsonschema:"optional tag filter"`
}

type searchNotesOutput struct {
	Hits []search.Result `json:"hits"`
}

type readNoteInput struct {
	Identifier string `json:"identifier" jsonschema:"note identifier"`
}

type readNoteOutput struct {
	Note readNoteItem `json:"note"`
}

type readNoteItem struct {
	NoteID      string         `json:"note_id"`
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Path        string         `json:"path"`
	Frontmatter map[string]any `json:"frontmatter"`
	Body        string         `json:"body"`
	ContentHash string         `json:"content_hash"`
	UpdatedAt   string         `json:"updated_at"`
}

type listBacklinksInput struct {
	Identifier string `json:"identifier" jsonschema:"note identifier"`
	Limit      int    `json:"limit,omitempty" jsonschema:"maximum number of backlinks to return"`
}

type listBacklinksOutput struct {
	Links []graph.Backlink `json:"links"`
}

type createNoteInput struct {
	Title string   `json:"title" jsonschema:"note title"`
	Body  string   `json:"body,omitempty" jsonschema:"optional note body"`
	Path  string   `json:"path,omitempty" jsonschema:"optional relative note path"`
	Tags  []string `json:"tags,omitempty" jsonschema:"optional note tags"`
	Type  string   `json:"type,omitempty" jsonschema:"optional note type"`
}

type createNoteOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
}

type editNoteInput struct {
	Identifier       string            `json:"identifier" jsonschema:"note identifier"`
	Append           string            `json:"append,omitempty" jsonschema:"append text to the note body"`
	ReplaceBody      string            `json:"replace_body,omitempty" jsonschema:"replace the note body"`
	MergeFrontmatter map[string]string `json:"merge_frontmatter,omitempty" jsonschema:"merge frontmatter fields"`
	IfMatchHash      string            `json:"if_match_hash,omitempty" jsonschema:"require the current content hash to match"`
}

type editNoteOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type deleteNoteInput struct {
	Identifier  string `json:"identifier" jsonschema:"note identifier"`
	HardDelete  bool   `json:"hard_delete,omitempty" jsonschema:"delete permanently instead of moving to trash"`
	IfMatchHash string `json:"if_match_hash,omitempty" jsonschema:"require the current content hash to match"`
}

type deleteNoteOutput struct {
	Deleted   bool   `json:"deleted"`
	Mode      string `json:"mode"`
	Path      string `json:"path,omitempty"`
	TrashPath string `json:"trash_path,omitempty"`
}

func registerReadOnlyTools(server *sdkmcp.Server, appServer *Server) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_notes",
		Description: "List notes in the current project",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input listNotesInput) (*sdkmcp.CallToolResult, listNotesOutput, error) {
		if appServer == nil {
			return nil, listNotesOutput{}, fmt.Errorf("server is nil")
		}
		root, err := appServer.resolveMemoriesRoot()
		if err != nil {
			return nil, listNotesOutput{}, err
		}
		allNotes, err := notes.List(root)
		if err != nil {
			if os.IsNotExist(err) {
				allNotes = nil
			} else {
				return nil, listNotesOutput{}, err
			}
		}

		limit := input.Limit
		if limit <= 0 {
			limit = 20
		}

		cursor := 0
		if input.Cursor != "" {
			cursor, err = strconv.Atoi(input.Cursor)
			if err != nil || cursor < 0 {
				return nil, listNotesOutput{}, fmt.Errorf("invalid cursor %q", input.Cursor)
			}
		}
		if cursor > len(allNotes) {
			cursor = len(allNotes)
		}
		end := cursor + limit
		if end > len(allNotes) {
			end = len(allNotes)
		}

		result := listNotesOutput{
			Notes: allNotes[cursor:end],
		}
		if end < len(allNotes) {
			result.NextCursor = strconv.Itoa(end)
		}
		_ = ctx
		return nil, result, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_tags",
		Description: "List tags in the current project",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input listTagsInput) (*sdkmcp.CallToolResult, listTagsOutput, error) {
		if appServer == nil {
			return nil, listTagsOutput{}, fmt.Errorf("server is nil")
		}

		tagsDB, err := appServer.indexDB()
		if err != nil {
			return nil, listTagsOutput{}, err
		}

		tags, err := listTags(tagsDB, input.Limit)
		if err != nil {
			return nil, listTagsOutput{}, err
		}
		if tags == nil {
			tags = []listTagsItem{}
		}

		_ = ctx
		return nil, listTagsOutput{Tags: tags}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "search_notes",
		Description: "Search notes in the current project",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input searchNotesInput) (*sdkmcp.CallToolResult, searchNotesOutput, error) {
		if appServer == nil {
			return nil, searchNotesOutput{}, fmt.Errorf("server is nil")
		}

		searchDB, err := appServer.indexDB()
		if err != nil {
			return nil, searchNotesOutput{}, err
		}

		hits, err := search.Search(searchDB, input.Query, input.Limit, input.Tag)
		if err != nil {
			return nil, searchNotesOutput{}, err
		}

		_ = ctx
		return nil, searchNotesOutput{Hits: hits}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "read_note",
		Description: "Read a note by identifier",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input readNoteInput) (*sdkmcp.CallToolResult, readNoteOutput, error) {
		if appServer == nil {
			return nil, readNoteOutput{}, fmt.Errorf("server is nil")
		}
		root, err := appServer.resolveMemoriesRoot()
		if err != nil {
			return nil, readNoteOutput{}, err
		}

		resolved, err := notes.Resolve(root, input.Identifier)
		if err != nil {
			return nil, readNoteOutput{}, err
		}

		absPath := filepath.Join(root, filepath.FromSlash(resolved.Path))
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, readNoteOutput{}, fmt.Errorf("read note %q: %w", resolved.Path, err)
		}

		result := readNoteOutput{
			Note: readNoteItem{
				NoteID:      resolved.Note.MnemonicNoteID,
				Slug:        resolved.Note.EffectiveSlug(),
				Title:       resolved.Note.Title,
				Path:        resolved.Path,
				Frontmatter: resolved.Note.Frontmatter,
				Body:        string(resolved.Note.Body),
				ContentHash: notes.HashBytes(data),
				UpdatedAt:   resolved.Note.UpdatedAt.UTC().Format(time.RFC3339),
			},
		}
		_ = ctx
		return nil, result, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_backlinks",
		Description: "List backlinks for a note",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input listBacklinksInput) (*sdkmcp.CallToolResult, listBacklinksOutput, error) {
		if appServer == nil {
			return nil, listBacklinksOutput{}, fmt.Errorf("server is nil")
		}

		linksDB, err := appServer.indexDB()
		if err != nil {
			return nil, listBacklinksOutput{}, err
		}

		target, err := queryIndexedNoteByIdentifier(linksDB, input.Identifier)
		if err != nil {
			return nil, listBacklinksOutput{}, err
		}

		links, err := graph.Backlinks(linksDB, target.NoteID)
		if err != nil {
			return nil, listBacklinksOutput{}, err
		}
		if input.Limit > 0 && len(links) > input.Limit {
			links = links[:input.Limit]
		}
		if links == nil {
			links = []graph.Backlink{}
		}

		_ = ctx
		return nil, listBacklinksOutput{Links: links}, nil
	})
}

func registerWriteTools(server *sdkmcp.Server, appServer *Server) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "create_note",
		Description: "Create a note in the current project",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input createNoteInput) (*sdkmcp.CallToolResult, createNoteOutput, error) {
		if appServer == nil {
			return nil, createNoteOutput{}, fmt.Errorf("server is nil")
		}

		root, err := appServer.resolveMemoriesRoot()
		if err != nil {
			return nil, createNoteOutput{}, err
		}

		created, err := createNote(root, input)
		if err != nil {
			return nil, createNoteOutput{}, err
		}

		_ = ctx
		return nil, created, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "edit_note",
		Description: "Edit a note in the current project",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input editNoteInput) (*sdkmcp.CallToolResult, editNoteOutput, error) {
		if appServer == nil {
			return nil, editNoteOutput{}, fmt.Errorf("server is nil")
		}

		root, err := appServer.resolveMemoriesRoot()
		if err != nil {
			return nil, editNoteOutput{}, err
		}

		edited, err := editNote(root, input)
		if err != nil {
			return nil, editNoteOutput{}, err
		}

		_ = ctx
		return nil, edited, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "delete_note",
		Description: "Delete a note in the current project",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input deleteNoteInput) (*sdkmcp.CallToolResult, deleteNoteOutput, error) {
		if appServer == nil {
			return nil, deleteNoteOutput{}, fmt.Errorf("server is nil")
		}

		root, err := appServer.resolveMemoriesRoot()
		if err != nil {
			return nil, deleteNoteOutput{}, err
		}

		deleted, err := deleteNote(root, input)
		if err != nil {
			return nil, deleteNoteOutput{}, err
		}

		_ = ctx
		return nil, deleted, nil
	})
}

func (s *Server) resolveMemoriesRoot() (string, error) {
	repoRoot := filepath.Dir(s.Project.MnemonicFilePath)
	return project.ResolveMemoriesRoot(project.MemoriesRootInput{
		Kind:         string(s.Project.Project.Kind),
		Slug:         s.Project.Project.Slug,
		MemoriesPath: s.Project.Project.MemoriesPath,
		MemoriesHome: s.Paths.MemoriesHome,
		RepoRoot:     repoRoot,
	})
}

func (s *Server) indexDB() (*sql.DB, error) {
	if s == nil {
		return nil, fmt.Errorf("server is nil")
	}

	s.indexDBMu.Lock()
	defer s.indexDBMu.Unlock()

	if s.indexConn != nil {
		return s.indexConn, nil
	}

	indexPath, err := index.Path(s.Project.Project.ID)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			return nil, app.NewNotFoundError("index missing; run `mnemonic project reindex`", nil)
		}
		return nil, fmt.Errorf("stat index %q: %w", indexPath, err)
	}

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(indexPath)+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open index database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping index database: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply PRAGMA busy_timeout: %w", err)
	}

	s.indexConn = db
	return s.indexConn, nil
}

func (s *Server) closeIndexDB() error {
	if s == nil {
		return nil
	}

	s.indexDBMu.Lock()
	defer s.indexDBMu.Unlock()

	if s.indexConn == nil {
		return nil
	}

	db := s.indexConn
	s.indexConn = nil
	if err := db.Close(); err != nil {
		return fmt.Errorf("close index database: %w", err)
	}
	return nil
}

func listTags(db *sql.DB, limit int) ([]listTagsItem, error) {
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

	tags := make([]listTagsItem, 0)
	for rows.Next() {
		var item listTagsItem
		if err := rows.Scan(&item.Tag, &item.Count); err != nil {
			return nil, fmt.Errorf("scan tag row: %w", err)
		}
		tags = append(tags, item)
		if limit > 0 && len(tags) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tag rows: %w", err)
	}

	return tags, nil
}

type indexedNote struct {
	NoteID string
}

func queryIndexedNoteByIdentifier(db *sql.DB, identifier string) (indexedNote, error) {
	row := db.QueryRow(
		`SELECT note_id
		 FROM notes
		 WHERE note_id = ? OR slug = ? OR rel_path = ? OR title = ?`,
		identifier, identifier, identifier, identifier,
	)
	var note indexedNote
	if err := row.Scan(&note.NoteID); err != nil {
		if err == sql.ErrNoRows {
			return indexedNote{}, app.NewNotFoundError(fmt.Sprintf("note %q not found", identifier), nil)
		}
		return indexedNote{}, fmt.Errorf("query note %q: %w", identifier, err)
	}
	return note, nil
}

func boolPtr(value bool) *bool {
	return &value
}

func createNote(root string, input createNoteInput) (createNoteOutput, error) {
	if input.Title == "" {
		return createNoteOutput{}, app.NewCLIUsageError("note title is required", nil)
	}

	slug, err := project.Slugify(input.Title)
	if err != nil {
		return createNoteOutput{}, err
	}

	relPath, err := createNotePath(slug, input.Path)
	if err != nil {
		return createNoteOutput{}, err
	}

	absPath := filepath.Join(root, filepath.FromSlash(relPath))
	if _, err := os.Stat(absPath); err == nil {
		return createNoteOutput{}, app.NewAmbiguousError(fmt.Sprintf("note path %q already exists", relPath), nil)
	} else if !os.IsNotExist(err) {
		return createNoteOutput{}, fmt.Errorf("check note path %q: %w", absPath, err)
	}

	timestamp := project.NowUTC().UTC()
	note := markdown.Note{
		MnemonicNoteID: project.NewUUID(),
		Title:          input.Title,
		Slug:           slug,
		Tags:           dedupeTags(input.Tags),
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
		Type:           input.Type,
		Body:           []byte(input.Body),
	}

	rendered, err := markdown.RenderNote(note)
	if err != nil {
		return createNoteOutput{}, err
	}
	if err := mnemonicfs.AtomicWriteFile(absPath, rendered, 0o644); err != nil {
		return createNoteOutput{}, err
	}

	return createNoteOutput{
		NoteID:      note.MnemonicNoteID,
		Slug:        slug,
		Path:        relPath,
		ContentHash: notes.HashBytes(rendered),
	}, nil
}

func createNotePath(slug, requestedPath string) (string, error) {
	if requestedPath == "" {
		return slug + ".md", nil
	}

	cleaned := filepath.ToSlash(filepath.Clean(requestedPath))
	switch {
	case cleaned == ".", cleaned == "":
		return "", app.NewCLIUsageError("note path is required", nil)
	case filepath.IsAbs(requestedPath), strings.HasPrefix(cleaned, "../"), cleaned == "..":
		return "", app.NewCLIUsageError("note path must be relative to the project memories root", nil)
	}
	if filepath.Ext(cleaned) == "" {
		cleaned += ".md"
	}
	return cleaned, nil
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

func editNote(root string, input editNoteInput) (editNoteOutput, error) {
	if input.Identifier == "" {
		return editNoteOutput{}, app.NewCLIUsageError("note identifier is required", nil)
	}

	modeCount := 0
	if input.Append != "" {
		modeCount++
	}
	if input.ReplaceBody != "" {
		modeCount++
	}
	if len(input.MergeFrontmatter) > 0 {
		modeCount++
	}
	if modeCount == 0 {
		return editNoteOutput{}, app.NewCLIUsageError("edit requires append, replace_body, or merge_frontmatter", nil)
	}
	if modeCount > 1 {
		return editNoteOutput{}, app.NewCLIUsageError("edit modes append, replace_body, and merge_frontmatter are mutually exclusive", nil)
	}

	editInput := notes.EditInput{
		RootDir:  root,
		Selector: input.Identifier,
		IfMatch:  input.IfMatchHash,
	}
	if input.ReplaceBody != "" {
		editInput.Body = []byte(input.ReplaceBody)
		editInput.HasBody = true
	} else if input.Append != "" {
		editInput.Append = []byte(input.Append)
	} else {
		editInput.Set = input.MergeFrontmatter
	}

	edited, err := notes.Edit(editInput)
	if err != nil {
		return editNoteOutput{}, err
	}

	return editNoteOutput{
		NoteID:      edited.NoteID,
		Slug:        edited.Slug,
		Path:        edited.Path,
		ContentHash: edited.ContentHash,
		CreatedAt:   edited.CreatedAt,
		UpdatedAt:   edited.UpdatedAt,
	}, nil
}

func deleteNote(root string, input deleteNoteInput) (deleteNoteOutput, error) {
	if input.Identifier == "" {
		return deleteNoteOutput{}, app.NewCLIUsageError("note identifier is required", nil)
	}

	if input.IfMatchHash != "" {
		resolved, err := notes.Resolve(root, input.Identifier)
		if err != nil {
			return deleteNoteOutput{}, err
		}
		absPath := filepath.Join(root, filepath.FromSlash(resolved.Path))
		current, err := os.ReadFile(absPath)
		if err != nil {
			return deleteNoteOutput{}, fmt.Errorf("read note: %w", err)
		}
		if notes.HashBytes(current) != input.IfMatchHash {
			return deleteNoteOutput{}, app.NewUnsafeError("content hash mismatch", nil)
		}
	}

	deleted, err := notes.Delete(notes.DeleteInput{
		RootDir:  root,
		Selector: input.Identifier,
		Hard:     input.HardDelete,
		Yes:      input.HardDelete,
	})
	if err != nil {
		return deleteNoteOutput{}, err
	}

	return deleteNoteOutput{
		Deleted:   true,
		Mode:      deleted.Mode,
		Path:      deleted.Path,
		TrashPath: deleted.TrashPath,
	}, nil
}
