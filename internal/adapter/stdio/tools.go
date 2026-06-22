package stdio

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/graph"
	"github.com/ilyachch/mnemonic/internal/search"
	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	listNotesDescription     = `List all notes in this knowledge base.`
	readNoteDescription      = `Read a note by note_id, slug, path, or title.`
	searchNotesDescription   = `Search this knowledge base.`
	listTagsDescription      = `List tags in this knowledge base.`
	listBacklinksDescription = `List backlinks for a note.`
	createNoteDescription    = `Create a new note.`
	editNoteDescription      = `Edit an existing note.`
	deleteNoteDescription    = `Delete a note.`
	rebuildIndexDescription  = `Rebuild the index.`
	doctorDescription        = `Run index and content health checks.`
)

type ListNotesInput struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type ListNotesOutput struct {
	Notes      []notesvc.NoteSummary `json:"notes"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

type ReadNoteInput struct {
	Identifier string `json:"identifier"`
}

type ReadNoteOutput struct {
	Note ReadNoteItem `json:"note"`
}

type ReadNoteItem struct {
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
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

type SearchNotesOutput struct {
	Hits []search.Result `json:"hits"`
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
	Links []graph.Backlink `json:"links"`
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
}

type DeleteNoteInput struct {
	Identifier  string `json:"identifier"`
	HardDelete  bool   `json:"hard_delete,omitempty"`
	IfMatchHash string `json:"if_match_hash,omitempty"`
}

type DeleteNoteOutput struct {
	Deleted   bool   `json:"deleted"`
	Mode      string `json:"mode"`
	Path      string `json:"path,omitempty"`
	TrashPath string `json:"trash_path,omitempty"`
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
	RegisterReadNote(server, deps)
	RegisterListBacklinks(server, deps)
	RegisterRebuildIndex(server, deps)
	RegisterDoctor(server, deps)
}

func RegisterWrite(server *sdkmcp.Server, deps Dependencies, description string) {
	RegisterCreateNote(server, deps, description)
	RegisterEditNote(server, deps)
	RegisterDeleteNote(server, deps)
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

		limit := input.Limit
		if limit <= 0 {
			limit = 20
		}

		cursor := 0
		if strings.TrimSpace(input.Cursor) != "" {
			cursor, err = strconv.Atoi(input.Cursor)
			if err != nil || cursor < 0 {
				return nil, ListNotesOutput{}, fmt.Errorf("invalid cursor %q", input.Cursor)
			}
		}
		if cursor > len(notes) {
			cursor = len(notes)
		}
		end := cursor + limit
		if end > len(notes) {
			end = len(notes)
		}

		result := ListNotesOutput{Notes: notes[cursor:end]}
		if end < len(notes) {
			result.NextCursor = strconv.Itoa(end)
		}
		return nil, result, nil
	})
}

func RegisterReadNote(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "read_note",
		Description: readNoteDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ReadNoteInput) (*sdkmcp.CallToolResult, ReadNoteOutput, error) {
		_ = ctx
		resolved, err := deps.Notes.Show(input.Identifier)
		if err != nil {
			return nil, ReadNoteOutput{}, err
		}
		result := ReadNoteOutput{
			Note: ReadNoteItem{
				NoteID:      resolved.Note.MnemonicNoteID,
				Slug:        resolved.Note.EffectiveSlug(),
				Title:       resolved.Note.Title,
				Path:        resolved.Path,
				Frontmatter: resolved.Note.Frontmatter,
				Body:        string(resolved.Note.Body),
				ContentHash: resolved.ContentHash,
				UpdatedAt:   resolved.Note.UpdatedAt.UTC().Format(time.RFC3339),
			},
		}
		return nil, result, nil
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
		hits, err := deps.Search.Search(ctx, searchsvc.SearchInput{Query: input.Query, Limit: input.Limit, Tag: input.Tag})
		if err != nil {
			return nil, SearchNotesOutput{}, err
		}
		return nil, SearchNotesOutput{Hits: hits}, nil
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
		if input.Identifier == "" {
			return nil, EditNoteOutput{}, apperr.CLIUsage("note identifier is required", nil)
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
			return nil, EditNoteOutput{}, apperr.CLIUsage("edit requires append, replace_body, or merge_frontmatter", nil)
		}
		if modeCount > 1 {
			return nil, EditNoteOutput{}, apperr.CLIUsage("edit modes append, replace_body, and merge_frontmatter are mutually exclusive", nil)
		}
		if input.ReplaceBody != "" && input.IfMatchHash == "" {
			return nil, EditNoteOutput{}, apperr.Unsafe("replace_body requires if_match_hash from read_note", nil)
		}

		editInput := notesvc.EditInput{
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
		}, nil
	})
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
			return nil, DeleteNoteOutput{}, apperr.Unsafe("hard delete requires if_match_hash from read_note", nil)
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
			Deleted:   true,
			Mode:      deleted.Mode,
			Path:      deleted.Path,
			TrashPath: deleted.TrashPath,
		}, nil
	})
}

func RegisterRebuildIndex(server *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "rebuild_index",
		Description: rebuildIndexDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
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
