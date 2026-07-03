package stdio

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	listNotesDescription = `List all notes in this knowledge base.`
	readNotesDescription = `Read one or more notes by note_id, slug, path, or title.
Provide an array of identifiers to batch-read multiple notes in a single call.
Resolved notes are returned with their note_id, slug, title, path, frontmatter, body, content_hash, and updated_at (RFC 3339).
Unresolved identifiers are listed in the "missing" array.

Parameters:
- identifiers ([]string, required): note IDs, slugs, file paths, or titles to resolve.
- fields ([]string, optional): limit output to the specified fields.
- max_body_chars (int, optional): truncate each note body to this many characters.`
	searchNotesDescription = `Search this knowledge base with multi-query full-text search, time filters, tag filters, and graph-aware reranking.
Provide multiple distinct query variants via the "queries" array to improve recall — each query contributes to the combined ranking.
Results include note_id, slug, title, and a relevance snippet. Set include_related to true to fetch linked notes for each hit.
Set debug to true to expose internal fields (path, score, content_hash).

Parameters:
- queries ([]string, optional): FTS5 query strings; submit several phrasing variants.
- tags ([]string, optional): restrict results to notes tagged with every listed tag (AND).
- created_before / created_after (int64, optional): Unix timestamps for creation time range.
- updated_before / updated_after (int64, optional): Unix timestamps for update time range.
- created_since / updated_since (string, optional): relative duration (e.g. "24h", "7d").
- limit (int, optional): maximum number of results (default 20).
- include_related (bool, optional): return related notes (backlinks and forward links) with their relation_type.
- debug (bool, optional): expose path, score, and content_hash for each hit.`
	listTagsDescription      = `List tags in this knowledge base.`
	listBacklinksDescription = `List backlinks for a note.`
	createNoteDescription    = `Create a new note.`
	editNoteDescription      = `Edit an existing note.`
	deleteNoteDescription    = `Delete a note.`
	rebuildIndexDescription  = `Rebuild the index.`
	doctorDescription        = `Run index and content health checks.`
	diagnoseNotesDescription = `Scan notes for metadata issues, broken links, and content problems.
Returns paginated diagnostic issues. Set include_suggestions to true to receive candidate targets for unresolved or ambiguous links.
Use this tool periodically to verify repository integrity after bulk changes.

Parameters:
- kinds ([]string, optional): filter by diagnostic kind. Valid values: "invalid_frontmatter", "missing_required_field", "missing_summary", "invalid_timestamp", "duplicate_slug", "duplicate_alias", "unresolved_link", "ambiguous_link", "empty_body".
- limit (int, optional): maximum issues per page (default 50).
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
	Notes   []ReadNotesNote `json:"notes"`
	Missing []string        `json:"missing,omitempty"`
}

type ReadNotesNote struct {
	NoteID      string         `json:"note_id"`
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Path        string         `json:"path"`
	Frontmatter map[string]any `json:"frontmatter"`
	Body        string         `json:"body"`
	ContentHash string         `json:"content_hash"`
	UpdatedAt   string         `json:"updated_at"`
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
	NoteID       string                  `json:"note_id"`
	Slug         string                  `json:"slug"`
	Title        string                  `json:"title"`
	Snippet      string                  `json:"snippet"`
	Path         string                  `json:"path,omitempty"`
	Score        float64                 `json:"score,omitempty"`
	ContentHash  string                  `json:"content_hash,omitempty"`
	RelatedNotes []searchNotesRelatedHit `json:"related_notes,omitempty"`
}

type searchNotesRelatedHit struct {
	NoteID       string `json:"note_id"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Path         string `json:"path"`
	RelationType string `json:"relation_type"`
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
	IfMatchHash      string            `json:"if_match_hash,omitempty"`
}

type EditNoteOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
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
	Detail     string                         `json:"detail,omitempty"`
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
		notes, err := deps.Notes.List()
		if err != nil {
			return nil, ListNotesOutput{}, err
		}
		return nil, paginateNotes(notes, input), nil
	})
}

func paginateNotes(notes []notesvc.NoteSummary, input ListNotesInput) ListNotesOutput {
	limit := input.Limit
	if limit <= 0 {
		limit = 20
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
		var notes []ReadNotesNote
		var missing []string
		for _, identifier := range input.Identifiers {
			resolved, err := deps.Notes.Show(identifier)
			if err != nil {
				missing = append(missing, identifier)
				continue
			}
			body := string(resolved.Note.Body)
			if input.MaxBodyChars > 0 && len(body) > input.MaxBodyChars {
				body = body[:input.MaxBodyChars]
			}
			notes = append(notes, ReadNotesNote{
				NoteID:      resolved.Note.MnemonicNoteID,
				Slug:        resolved.Note.EffectiveSlug(),
				Title:       resolved.Note.Title,
				Path:        resolved.Path,
				Frontmatter: resolved.Note.Frontmatter,
				Body:        body,
				ContentHash: resolved.ContentHash,
				UpdatedAt:   resolved.Note.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}
		return nil, ReadNotesOutput{Notes: notes, Missing: missing}, nil
	})
}

func RegisterSearchNotes(server *sdkmcp.Server, deps Dependencies, description string) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "search_notes",
		Description: buildToolDescription(description, searchNotesDescription),
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input SearchNotesInput) (*sdkmcp.CallToolResult, SearchNotesOutput, error) {
		if input.Limit <= 0 {
			input.Limit = 20
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
			s := SearchNotesHit{
				NoteID:  hit.NoteID,
				Slug:    hit.Slug,
				Title:   hit.Title,
				Snippet: hit.Snippet,
			}
			if input.Debug {
				s.Path = hit.Path
				s.Score = hit.Score
				s.ContentHash = hit.ContentHash
			}
			for _, rn := range hit.RelatedNotes {
				s.RelatedNotes = append(s.RelatedNotes, searchNotesRelatedHit{
					NoteID:       rn.NoteID,
					Slug:         rn.Slug,
					Title:        rn.Title,
					Path:         rn.Path,
					RelationType: rn.RelationType,
				})
			}
			out = append(out, s)
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
		out, err := deps.Search.ListTags(ctx)
		if err != nil {
			return nil, ListTagsOutput{}, err
		}
		tags := out.Tags
		if input.Limit > 0 && len(tags) > input.Limit {
			tags = tags[:input.Limit]
		}
		return nil, ListTagsOutput{Tags: tags}, nil
	})
}

func RegisterListBacklinks(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_backlinks",
		Description: listBacklinksDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListBacklinksInput) (*sdkmcp.CallToolResult, ListBacklinksOutput, error) {
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
			CreatedAt:   edited.CreatedAt,
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
	if modeCount == 0 {
		return apperr.CLIUsage("edit requires append, replace_body, or merge_frontmatter", nil)
	}
	if modeCount > 1 {
		return apperr.CLIUsage("edit modes append, replace_body, and merge_frontmatter are mutually exclusive", nil)
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
		result, err := deps.Index.Diagnose(ctx, indexsvc.DiagnoseInput{
			Kinds:  input.Kinds,
			Limit:  input.Limit,
			Cursor: input.Cursor,
		})
		if err != nil {
			return nil, DiagnoseNotesOutput{}, err
		}

		issues := make([]DiagnoseNotesIssue, 0, len(result.Issues))
		for _, issue := range result.Issues {
			di := DiagnoseNotesIssue{
				Kind:   issue.Kind,
				NoteID: issue.NoteID,
				Slug:   issue.Slug,
				Path:   issue.Path,
				Detail: issue.Detail,
			}
			if input.IncludeSuggestions && (issue.Kind == indexsvc.KindUnresolvedLink || issue.Kind == indexsvc.KindAmbiguousLink) {
				di.Candidates = findLinkCandidates(ctx, deps, issue)
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

func findLinkCandidates(ctx context.Context, deps Dependencies, issue indexsvc.DiagnosticIssue) []indexsvc.DiagnosticCandidate {
	target := extractLinkTarget(issue.Detail)
	if target == "" {
		return nil
	}
	hits, err := deps.Search.Search(ctx, searchsvc.SearchInput{Query: target, Limit: 3})
	if err != nil {
		return nil
	}
	candidates := make([]indexsvc.DiagnosticCandidate, 0, len(hits))
	for _, hit := range hits {
		candidates = append(candidates, indexsvc.DiagnosticCandidate{
			NoteID: hit.NoteID,
			Slug:   hit.Slug,
			Title:  hit.Title,
			Path:   hit.Path,
		})
	}
	return candidates
}

func extractLinkTarget(detail string) string {
	const prefix = `target "`
	start := strings.Index(detail, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)
	end := strings.Index(detail[start:], `"`)
	if end == -1 {
		return ""
	}
	return detail[start : start+end]
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

func BoolPtr(v bool) *bool {
	return &v
}
