package stdio

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	maxSearchLimit     = 100
	maxQueryCount      = 8
	maxQueryLength     = 500
	maxReadIdentifiers = 50
	maxBodyChars       = 100000
	maxDiagnosticLimit = 200
)

const (
	listNotesDescription = `List all notes in this knowledge base.`
	readNotesDescription = `Read one or more notes by note_id, slug, path, or title.
Provide an array of identifiers to batch-read multiple notes in a single call.
note_id, slug, and title are always returned.
By default returns note_id, slug, title, summary, tags, and body.
Use the "fields" parameter to include path, frontmatter, content_hash, aliases, created_at, or updated_at.
Unresolved identifiers are listed in the "missing" array.
Per-selector errors (ambiguous, corrupted, io_error, internal) are reported in the "issues" array.

Parameters:
- identifiers ([]string, required): note IDs, slugs, file paths, or titles to resolve (max 50).
- fields ([]string, optional): select which optional fields to include. Valid values: summary, tags, body, path, frontmatter, content_hash, aliases, created_at, updated_at.
- max_body_chars (int, optional): truncate each note body to this many characters (max 100000).`
	searchNotesDescription = `Search this knowledge base with multi-query full-text search, time filters, tag filters, and graph-aware reranking.
Provide multiple distinct query variants via the "queries" array to improve recall — each query contributes to the combined ranking via Reciprocal Rank Fusion.
Results include note_id, slug, title, summary, tags, matched_queries, and a relevance snippet.
Set include_related to true to fetch linked notes for each hit (with relation_type, source_kind, direction).
Set debug to true to expose internal fields (path, score, content_hash).

Parameters:
- queries ([]string, optional): FTS5 query strings; submit several phrasing variants (max 8, 500 chars each).
- tags ([]string, optional): restrict results to notes tagged with every listed tag (AND).
- created_before / created_after (int64, optional): Unix timestamps for creation time range.
- updated_before / updated_after (int64, optional): Unix timestamps for update time range.
- created_since / updated_since (string, optional): relative duration (e.g. "24h", "7d").
- limit (int, optional): maximum number of results (default 10, max 100).
- include_related (bool, optional): return related notes (backlinks and forward links).
- debug (bool, optional): expose path, score, and content_hash for each hit.`
	listTagsDescription      = `List tags in this knowledge base.`
	listBacklinksDescription = `List backlinks for a note.`
	createNoteDescription    = `Create a new note.`
	editNoteDescription      = `Edit an existing note by appending to the body, replacing the body, merging frontmatter, or setting/clearing tags and aliases. Tags and aliases are presence-aware: absent=no change, empty array=clear, non-empty=replace. Must not set tags or aliases through merge_frontmatter; use the typed fields.`
	deleteNoteDescription    = `Delete a note.`
	rebuildIndexDescription  = `Rebuild the index.`
	doctorDescription        = `Run index and content health checks.`
	diagnoseNotesDescription = `Scan notes for metadata issues, broken links, and content problems.
Returns paginated diagnostic issues. Set include_suggestions to true to receive candidate targets for unresolved or ambiguous links.
Use this tool periodically to verify repository integrity after bulk changes.

Parameters:
- kinds ([]string, optional): filter by diagnostic kind. Valid values: "invalid_frontmatter", "missing_required_field", "missing_summary", "duplicate_slug", "duplicate_alias", "unresolved_link", "ambiguous_link", "empty_body".
- limit (int, optional): maximum issues per page (default 50, max 200).
- cursor (int, optional): zero-based page offset.
- include_suggestions (bool, optional): resolve broken links via search and include candidate notes.`
)

type ListNotesInput struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type ListNotesOutput struct {
	Notes      []notesvc.NoteSummary `json:"notes"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

type ReadNotesInput struct {
	Identifiers  []string `json:"identifiers"`
	Fields       []string `json:"fields,omitempty"`
	MaxBodyChars int      `json:"max_body_chars,omitempty"`
}

type ReadNotesOutput struct {
	Notes   []ReadNotesNote         `json:"notes"`
	Missing []string                `json:"missing,omitempty"`
	Issues  []notesvc.ReadManyIssue `json:"issues,omitempty"`
}

type ReadNotesNote struct {
	NoteID      string          `json:"note_id"`
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	Summary     *string         `json:"summary,omitempty"`
	Tags        *[]string       `json:"tags,omitempty"`
	Aliases     *[]string       `json:"aliases,omitempty"`
	Body        *string         `json:"body,omitempty"`
	Path        *string         `json:"path,omitempty"`
	Frontmatter *map[string]any `json:"frontmatter,omitempty"`
	ContentHash *string         `json:"content_hash,omitempty"`
	CreatedAt   *int64          `json:"created_at,omitempty"`
	UpdatedAt   *int64          `json:"updated_at,omitempty"`
}

type SearchNotesInput struct {
	Queries        []string `json:"queries,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	CreatedBefore  *int64   `json:"created_before,omitempty"`
	CreatedAfter   *int64   `json:"created_after,omitempty"`
	UpdatedBefore  *int64   `json:"updated_before,omitempty"`
	UpdatedAfter   *int64   `json:"updated_after,omitempty"`
	CreatedSince   string   `json:"created_since,omitempty"`
	UpdatedSince   string   `json:"updated_since,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	IncludeRelated bool     `json:"include_related,omitempty"`
	Debug          bool     `json:"debug,omitempty"`
}

type SearchNotesHit struct {
	NoteID         string                  `json:"note_id"`
	Slug           string                  `json:"slug"`
	Title          string                  `json:"title"`
	Snippet        string                  `json:"snippet"`
	Summary        string                  `json:"summary,omitempty"`
	Tags           []string                `json:"tags,omitempty"`
	MatchedQueries []string                `json:"matched_queries,omitempty"`
	Path           string                  `json:"path,omitempty"`
	Score          *float64                `json:"score,omitempty"`
	ContentHash    string                  `json:"content_hash,omitempty"`
	RelatedNotes   []searchNotesRelatedHit `json:"related_notes,omitempty"`
}

type searchNotesRelatedHit struct {
	NoteID       string `json:"note_id"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Path         string `json:"path,omitempty"`
	RelationType string `json:"relation_type"`
	SourceKind   string `json:"source_kind"`
	Direction    string `json:"direction"`
}

type SearchNotesOutput struct {
	Hits []SearchNotesHit `json:"hits"`
}

type ListTagsInput struct {
	Limit int `json:"limit,omitempty"`
}

type ListTagsOutput struct {
	Tags []searchsvc.ListTagsItem `json:"tags"`
}

type ListBacklinksInput struct {
	Identifier string `json:"identifier"`
	Limit      int    `json:"limit,omitempty"`
}

type ListBacklinksOutput struct {
	Links []searchsvc.Backlink `json:"links"`
}

type CreateNoteInput struct {
	Title string   `json:"title"`
	Body  string   `json:"body,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

type CreateNoteOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

type EditNoteInput struct {
	Identifier       string            `json:"identifier"`
	Append           string            `json:"append,omitempty"`
	ReplaceBody      string            `json:"replace_body,omitempty"`
	MergeFrontmatter map[string]string `json:"merge_frontmatter,omitempty"`
	Tags             *[]string         `json:"tags,omitempty"`
	Aliases          *[]string         `json:"aliases,omitempty"`
	IfMatchHash      string            `json:"if_match_hash,omitempty"`
}

type EditNoteOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	UpdatedAt   string `json:"updated_at"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

type DeleteNoteInput struct {
	Identifier  string `json:"identifier"`
	HardDelete  bool   `json:"hard_delete,omitempty"`
	IfMatchHash string `json:"if_match_hash,omitempty"`
}

type DeleteNoteOutput struct {
	Deleted     bool   `json:"deleted"`
	Mode        string `json:"mode"`
	Path        string `json:"path,omitempty"`
	TrashPath   string `json:"trash_path,omitempty"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

type RebuildIndexOutput struct {
	KBID         string `json:"kb_id"`
	NotesSeen    int    `json:"notes_seen"`
	NotesIndexed int    `json:"notes_indexed"`
	Status       string `json:"status"`
}

type DoctorOutput struct {
	Status string        `json:"status"`
	Checks []DoctorCheck `json:"checks"`
}

type DoctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Count  int    `json:"count,omitempty"`
}

type DiagnoseNotesInput struct {
	Kinds              []indexsvc.DiagnosticKind `json:"kinds,omitempty"`
	Limit              int                       `json:"limit,omitempty"`
	Cursor             int                       `json:"cursor,omitempty"`
	IncludeSuggestions bool                      `json:"include_suggestions,omitempty"`
}

type DiagnoseNotesOutput struct {
	Issues     []DiagnoseNotesIssue `json:"issues"`
	TotalCount int                  `json:"total_count"`
	NextCursor int                  `json:"next_cursor,omitempty"`
}

type DiagnoseNotesIssue struct {
	Kind       indexsvc.DiagnosticKind        `json:"kind"`
	NoteID     string                         `json:"note_id,omitempty"`
	Slug       string                         `json:"slug,omitempty"`
	Path       string                         `json:"path,omitempty"`
	Field      string                         `json:"field,omitempty"`
	SourceLine int                            `json:"source_line,omitempty"`
	SourceKind string                         `json:"source_kind,omitempty"`
	LinkStyle  string                         `json:"link_style,omitempty"`
	Detail     string                         `json:"detail,omitempty"`
	Target     string                         `json:"target,omitempty"`
	Candidates []indexsvc.DiagnosticCandidate `json:"candidates,omitempty"`
}

func RegisterAll(server *sdkmcp.Server, deps Dependencies, description string, readOnly bool) {
	RegisterReadOnly(server, deps, description)
	if !readOnly {
		RegisterWrite(server, deps, description)
	}
}

func RegisterReadOnly(server *sdkmcp.Server, deps Dependencies, description string) {
	RegisterListNotes(server, deps)
	RegisterListTags(server, deps)
	RegisterSearchNotes(server, deps, description)
	RegisterReadNotes(server, deps)
	RegisterListBacklinks(server, deps)
	RegisterDoctor(server, deps)
	RegisterDiagnoseNotes(server, deps)
}

func RegisterWrite(server *sdkmcp.Server, deps Dependencies, description string) {
	RegisterCreateNote(server, deps, description)
	RegisterEditNote(server, deps)
	RegisterDeleteNote(server, deps)
	RegisterRebuildIndex(server, deps)
}

func RegisterListNotes(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_notes",
		Description: listNotesDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListNotesInput) (*sdkmcp.CallToolResult, ListNotesOutput, error) {
		_ = ctx
		if input.Limit < 0 {
			return nil, ListNotesOutput{}, apperr.CLIUsage("limit must be >= 0", nil)
		}
		notes, err := deps.Notes.List()
		if err != nil {
			return nil, ListNotesOutput{}, err
		}
		return nil, paginateNotes(notes, input), nil
	})
}

func paginateNotes(notes []notesvc.NoteSummary, input ListNotesInput) ListNotesOutput {
	limit := input.Limit
	if limit == 0 {
		limit = 20
	}
	if limit < 0 {
		return ListNotesOutput{}
	}
	cursor := parseCursor(input.Cursor, len(notes))
	end := cursor + limit
	if end > len(notes) {
		end = len(notes)
	}
	result := ListNotesOutput{Notes: notes[cursor:end]}
	if end < len(notes) {
		result.NextCursor = strconv.Itoa(end)
	}
	return result
}

func parseCursor(raw string, maxVal int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 || v > maxVal {
		return 0
	}
	return v
}

func RegisterReadNotes(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "read_notes",
		Description: readNotesDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ReadNotesInput) (*sdkmcp.CallToolResult, ReadNotesOutput, error) {
		_ = ctx
		if len(input.Identifiers) > maxReadIdentifiers {
			return nil, ReadNotesOutput{}, apperr.CLIUsage("too many identifiers", nil)
		}
		if input.MaxBodyChars > maxBodyChars {
			return nil, ReadNotesOutput{}, apperr.CLIUsage("max_body_chars exceeds maximum", nil)
		}
		fields, err := buildFieldSet(input.Fields)
		if err != nil {
			return nil, ReadNotesOutput{}, err
		}
		output, err := deps.Notes.ShowMany(notesvc.ReadManyInput{Selectors: input.Identifiers, MaxBodyChars: input.MaxBodyChars})
		if err != nil {
			return nil, ReadNotesOutput{}, err
		}
		var notes []ReadNotesNote
		for _, item := range output.Notes {
			notes = append(notes, buildReadNotesNote(fields, item.ShowResult, input.MaxBodyChars))
		}
		return nil, ReadNotesOutput{Notes: notes, Missing: output.Missing, Issues: output.Issues}, nil
	})
}

func validateSearchNotesInput(input *SearchNotesInput) error {
	if input.Limit < 0 {
		return apperr.CLIUsage("limit must be >= 0", nil)
	}
	if input.Limit == 0 {
		input.Limit = 10
	}
	if input.Limit > maxSearchLimit {
		return apperr.CLIUsage("limit exceeds maximum", nil)
	}
	if len(input.Queries) > maxQueryCount {
		return apperr.CLIUsage("too many queries", nil)
	}
	for _, q := range input.Queries {
		if utf8.RuneCountInString(q) > maxQueryLength {
			return apperr.CLIUsage("query too long", nil)
		}
	}
	return nil
}

func RegisterSearchNotes(server *sdkmcp.Server, deps Dependencies, description string) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "search_notes",
		Description: buildToolDescription(description, searchNotesDescription),
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input SearchNotesInput) (*sdkmcp.CallToolResult, SearchNotesOutput, error) {
		if err := validateSearchNotesInput(&input); err != nil {
			return nil, SearchNotesOutput{}, err
		}
		advancedInput := searchsvc.AdvancedSearchInput{
			Queries:        input.Queries,
			Tags:           input.Tags,
			CreatedBefore:  input.CreatedBefore,
			CreatedAfter:   input.CreatedAfter,
			UpdatedBefore:  input.UpdatedBefore,
			UpdatedAfter:   input.UpdatedAfter,
			CreatedSince:   input.CreatedSince,
			UpdatedSince:   input.UpdatedSince,
			Limit:          input.Limit,
			IncludeRelated: input.IncludeRelated,
		}
		hits, err := deps.Search.AdvancedSearch(ctx, advancedInput)
		if err != nil {
			return nil, SearchNotesOutput{}, err
		}
		out := make([]SearchNotesHit, 0, len(hits))
		for _, hit := range hits {
			out = append(out, toSearchNotesHit(hit, input.Debug))
		}
		return nil, SearchNotesOutput{Hits: out}, nil
	})
}

func RegisterListTags(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_tags",
		Description: listTagsDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListTagsInput) (*sdkmcp.CallToolResult, ListTagsOutput, error) {
		if input.Limit < 0 {
			return nil, ListTagsOutput{}, apperr.CLIUsage("limit must be >= 0", nil)
		}
		out, err := deps.Search.ListTags(ctx, searchsvc.ListTagsInput{Limit: input.Limit})
		if err != nil {
			return nil, ListTagsOutput{}, err
		}
		return nil, ListTagsOutput{Tags: out.Tags}, nil
	})
}

func RegisterListBacklinks(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_backlinks",
		Description: listBacklinksDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListBacklinksInput) (*sdkmcp.CallToolResult, ListBacklinksOutput, error) {
		if input.Limit < 0 {
			return nil, ListBacklinksOutput{}, apperr.CLIUsage("limit must be >= 0", nil)
		}
		links, err := deps.Search.Backlinks(ctx, searchsvc.BacklinksInput{Identifier: input.Identifier, Limit: input.Limit})
		if err != nil {
			return nil, ListBacklinksOutput{}, err
		}
		return nil, ListBacklinksOutput{Links: links}, nil
	})
}

func RegisterCreateNote(server *sdkmcp.Server, deps Dependencies, description string) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "create_note",
		Description: buildToolDescription(description, createNoteDescription),
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: BoolPtr(false),
			OpenWorldHint:   BoolPtr(false),
		},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input CreateNoteInput) (*sdkmcp.CallToolResult, CreateNoteOutput, error) {
		_ = ctx
		created, err := deps.Notes.Create(notesvc.CreateInput{
			Title: input.Title,
			Body:  []byte(input.Body),
			Tags:  input.Tags,
		})
		if err != nil {
			return nil, CreateNoteOutput{}, err
		}
		return nil, CreateNoteOutput{
			NoteID:      created.NoteID,
			Slug:        created.Slug,
			Path:        created.Path,
			ContentHash: created.ContentHash,
			IndexStatus: created.IndexStatus,
			IndexError:  created.IndexError,
		}, nil
	})
}

func RegisterEditNote(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "edit_note",
		Description: editNoteDescription,
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: BoolPtr(true),
			OpenWorldHint:   BoolPtr(false),
		},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input EditNoteInput) (*sdkmcp.CallToolResult, EditNoteOutput, error) {
		if err := validateEditInput(input); err != nil {
			return nil, EditNoteOutput{}, err
		}
		editInput := buildEditInput(input)
		edited, err := deps.Notes.Edit(editInput)
		if err != nil {
			return nil, EditNoteOutput{}, err
		}
		return nil, EditNoteOutput{
			NoteID:      edited.NoteID,
			Slug:        edited.Slug,
			Path:        edited.Path,
			ContentHash: edited.ContentHash,
			UpdatedAt:   edited.UpdatedAt,
			IndexStatus: edited.IndexStatus,
			IndexError:  edited.IndexError,
		}, nil
	})
}

func validateEditInput(input EditNoteInput) error {
	if input.Identifier == "" {
		return apperr.CLIUsage("note identifier is required", nil)
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
	if input.Tags != nil || input.Aliases != nil {
		modeCount++
	}
	if modeCount == 0 {
		return apperr.CLIUsage("edit requires append, replace_body, merge_frontmatter, tags, or aliases", nil)
	}
	if modeCount > 1 {
		return apperr.CLIUsage("edit modes append, replace_body, merge_frontmatter, tags, and aliases are mutually exclusive", nil)
	}
	if input.ReplaceBody != "" && input.IfMatchHash == "" {
		return apperr.Unsafe("replace_body requires if_match_hash from read_notes", nil)
	}
	return nil
}

func buildEditInput(input EditNoteInput) notesvc.EditInput {
	editInput := notesvc.EditInput{
		Selector: input.Identifier,
		IfMatch:  input.IfMatchHash,
	}
	switch {
	case input.ReplaceBody != "":
		editInput.Body = []byte(input.ReplaceBody)
		editInput.HasBody = true
	case input.Append != "":
		editInput.Append = []byte(input.Append)
	case input.Tags != nil || input.Aliases != nil:
		editInput.Tags = input.Tags
		editInput.Aliases = input.Aliases
	default:
		editInput.Set = input.MergeFrontmatter
	}
	return editInput
}

func RegisterDeleteNote(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "delete_note",
		Description: deleteNoteDescription,
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: BoolPtr(true),
			OpenWorldHint:   BoolPtr(false),
		},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input DeleteNoteInput) (*sdkmcp.CallToolResult, DeleteNoteOutput, error) {
		_ = ctx
		if input.Identifier == "" {
			return nil, DeleteNoteOutput{}, apperr.CLIUsage("note identifier is required", nil)
		}
		if input.HardDelete && input.IfMatchHash == "" {
			return nil, DeleteNoteOutput{}, apperr.Unsafe("hard delete requires if_match_hash from read_notes", nil)
		}
		if input.IfMatchHash != "" {
			resolved, err := deps.Notes.Show(input.Identifier)
			if err != nil {
				return nil, DeleteNoteOutput{}, err
			}
			if resolved.ContentHash != input.IfMatchHash {
				return nil, DeleteNoteOutput{}, apperr.Unsafe("content hash mismatch", nil)
			}
		}

		deleted, err := deps.Notes.Delete(notesvc.DeleteInput{
			Selector: input.Identifier,
			Hard:     input.HardDelete,
			Yes:      input.HardDelete,
		})
		if err != nil {
			return nil, DeleteNoteOutput{}, err
		}
		return nil, DeleteNoteOutput{
			Deleted:     true,
			Mode:        deleted.Mode,
			Path:        deleted.Path,
			TrashPath:   deleted.TrashPath,
			IndexStatus: deleted.IndexStatus,
			IndexError:  deleted.IndexError,
		}, nil
	})
}

func RegisterRebuildIndex(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "rebuild_index",
		Description: rebuildIndexDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: false},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, RebuildIndexOutput, error) {
		result, err := deps.Index.Rebuild(ctx)
		if err != nil {
			return nil, RebuildIndexOutput{}, err
		}
		return nil, RebuildIndexOutput{
			KBID:         result.KBID,
			NotesSeen:    result.NotesSeen,
			NotesIndexed: result.NotesIndexed,
			Status:       result.Status,
		}, nil
	})
}

func RegisterDoctor(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "doctor",
		Description: doctorDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, DoctorOutput, error) {
		result, err := deps.Index.Doctor(ctx)
		if err != nil {
			return nil, DoctorOutput{}, err
		}
		checks := make([]DoctorCheck, 0, len(result.Checks))
		for _, check := range result.Checks {
			checks = append(checks, DoctorCheck{
				Name:   check.Name,
				Status: check.Status,
				Detail: check.Detail,
				Count:  check.Count,
			})
		}
		return nil, DoctorOutput{Status: result.Status, Checks: checks}, nil
	})
}

func RegisterDiagnoseNotes(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "diagnose_notes",
		Description: diagnoseNotesDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input DiagnoseNotesInput) (*sdkmcp.CallToolResult, DiagnoseNotesOutput, error) {
		if input.Limit > maxDiagnosticLimit {
			return nil, DiagnoseNotesOutput{}, apperr.CLIUsage("limit exceeds maximum", nil)
		}
		result, err := deps.Index.Diagnose(ctx, indexsvc.DiagnoseInput{
			Kinds:              input.Kinds,
			Limit:              input.Limit,
			Cursor:             input.Cursor,
			IncludeSuggestions: input.IncludeSuggestions,
		})
		if err != nil {
			return nil, DiagnoseNotesOutput{}, err
		}

		issues := make([]DiagnoseNotesIssue, 0, len(result.Issues))
		for _, issue := range result.Issues {
			di := DiagnoseNotesIssue{
				Kind:       issue.Kind,
				NoteID:     issue.NoteID,
				Slug:       issue.Slug,
				Path:       issue.Path,
				Field:      issue.Field,
				SourceLine: issue.SourceLine,
				SourceKind: issue.SourceKind,
				LinkStyle:  issue.LinkStyle,
				Detail:     issue.Detail,
				Target:     issue.Target,
				Candidates: issue.Candidates,
			}
			issues = append(issues, di)
		}

		return nil, DiagnoseNotesOutput{
			Issues:     issues,
			TotalCount: result.TotalCount,
			NextCursor: result.NextCursor,
		}, nil
	})
}

func buildToolDescription(description, baseInstructions string) string {
	description = strings.TrimSpace(description)
	baseInstructions = strings.TrimSpace(baseInstructions)
	switch {
	case description == "":
		return baseInstructions
	case baseInstructions == "":
		return description
	default:
		return description + "\n\n" + baseInstructions
	}
}

var validReadFields = map[string]bool{
	"summary":      true,
	"tags":         true,
	"body":         true,
	"path":         true,
	"frontmatter":  true,
	"content_hash": true,
	"aliases":      true,
	"created_at":   true,
	"updated_at":   true,
}

var defaultReadFields = map[string]bool{
	"note_id": true,
	"slug":    true,
	"title":   true,
	"summary": true,
	"tags":    true,
	"body":    true,
}

func buildFieldSet(fields []string) (map[string]bool, error) {
	if len(fields) == 0 {
		return defaultReadFields, nil
	}
	set := make(map[string]bool, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "note_id" || f == "slug" || f == "title" {
			continue
		}
		if !validReadFields[f] {
			return nil, apperr.CLIUsage("unknown field: "+f, nil)
		}
		set[f] = true
	}
	return set, nil
}

func truncateRunes(s string, maxChars int) string {
	if maxChars <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return string(runes[:maxChars])
}

func BoolPtr(v bool) *bool {
	return &v
}

func toSearchNotesHit(hit searchsvc.AdvancedSearchResult, debug bool) SearchNotesHit {
	s := SearchNotesHit{
		NoteID:         hit.NoteID,
		Slug:           hit.Slug,
		Title:          hit.Title,
		Snippet:        hit.Snippet,
		Summary:        hit.Summary,
		Tags:           hit.Tags,
		MatchedQueries: hit.MatchedQueries,
	}
	if debug {
		s.Path = hit.Path
		s.Score = &hit.Score
		s.ContentHash = hit.ContentHash
	}
	for _, rn := range hit.RelatedNotes {
		sr := searchNotesRelatedHit{
			NoteID:       rn.NoteID,
			Slug:         rn.Slug,
			Title:        rn.Title,
			RelationType: rn.RelationType,
			SourceKind:   rn.SourceKind,
			Direction:    rn.Direction,
		}
		if debug {
			sr.Path = rn.Path
		}
		s.RelatedNotes = append(s.RelatedNotes, sr)
	}
	return s
}

func setOptionalSliceField(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func setReadNotesBasicFields(note *ReadNotesNote, fields map[string]bool, resolved notesvc.ShowResult, maxBodyChars int) {
	if fields["summary"] {
		s := resolved.Note.Summary
		note.Summary = &s
	}
	if fields["tags"] {
		tags := setOptionalSliceField(resolved.Note.Tags)
		note.Tags = &tags
	}
	if fields["aliases"] {
		aliases := setOptionalSliceField(resolved.Note.Aliases)
		note.Aliases = &aliases
	}
	if fields["body"] {
		body := string(resolved.Note.Body)
		if maxBodyChars > 0 {
			body = truncateRunes(body, maxBodyChars)
		}
		note.Body = &body
	}
	if fields["path"] {
		p := resolved.Path
		note.Path = &p
	}
}

func setReadNotesExtraFields(note *ReadNotesNote, fields map[string]bool, resolved notesvc.ShowResult) {
	if fields["frontmatter"] {
		fm := resolved.Note.Frontmatter
		note.Frontmatter = &fm
	}
	if fields["content_hash"] {
		h := resolved.ContentHash
		note.ContentHash = &h
	}
	if fields["created_at"] {
		if ca, ok := frontmatterUnixTime(resolved.Note.Frontmatter, "created_at"); ok {
			note.CreatedAt = &ca
		}
	}
	if fields["updated_at"] {
		if ua, ok := frontmatterUnixTime(resolved.Note.Frontmatter, "updated_at"); ok {
			note.UpdatedAt = &ua
		} else if !resolved.UpdatedAt.IsZero() {
			ua := resolved.UpdatedAt.Unix()
			note.UpdatedAt = &ua
		}
	}
}

// frontmatterUnixTime extracts a Unix epoch integer from raw frontmatter.
// YAML unmarshals integers as `int` (or `int64`); both are accepted.
func frontmatterUnixTime(fm map[string]any, key string) (int64, bool) {
	if fm == nil {
		return 0, false
	}
	value, ok := fm[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	}
	return 0, false
}

func buildReadNotesNote(fields map[string]bool, resolved notesvc.ShowResult, maxBodyChars int) ReadNotesNote {
	note := ReadNotesNote{
		NoteID: resolved.Note.MnemonicNoteID,
		Slug:   resolved.Note.EffectiveSlug(),
		Title:  resolved.Note.Title,
	}
	setReadNotesBasicFields(&note, fields, resolved, maxBodyChars)
	setReadNotesExtraFields(&note, fields, resolved)
	return note
}
