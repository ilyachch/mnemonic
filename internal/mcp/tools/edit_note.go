package tools

import (
	"context"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/notes"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const editNoteDescription = `Update existing notes when knowledge evolves.
YOU MUST keep memory accurate. Read the note first to get 'content_hash', then pass it as 'if_match_hash'.`

type EditNoteInput struct {
	Identifier       string            `json:"identifier" jsonschema:"Existing note id, slug, path, or title."`
	Append           string            `json:"append,omitempty" jsonschema:"Markdown text to append with new durable information."`
	ReplaceBody      string            `json:"replace_body,omitempty" jsonschema:"Full replacement body. Use only when safer than appending."`
	MergeFrontmatter map[string]string `json:"merge_frontmatter,omitempty" jsonschema:"Frontmatter fields to merge, such as status, type, or summary."`
	IfMatchHash      string            `json:"if_match_hash,omitempty" jsonschema:"Content hash from read_note for safe updates."`
}

type EditNoteOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func RegisterEditNote(s *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "edit_note",
		Description: editNoteDescription,
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: BoolPtr(true),
			OpenWorldHint:   BoolPtr(false),
		},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input EditNoteInput) (*sdkmcp.CallToolResult, EditNoteOutput, error) {
		root, err := deps.GetMemoriesRoot()
		if err != nil {
			return nil, EditNoteOutput{}, err
		}

		edited, err := editNote(root, input)
		if err != nil {
			return nil, EditNoteOutput{}, err
		}
		if err := deps.RebuildIndex(root); err != nil {
			return nil, EditNoteOutput{}, err
		}

		_ = ctx
		return nil, edited, nil
	})
}

func editNote(root string, input EditNoteInput) (EditNoteOutput, error) {
	if input.Identifier == "" {
		return EditNoteOutput{}, app.NewCLIUsageError("note identifier is required", nil)
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
		return EditNoteOutput{}, app.NewCLIUsageError("edit requires append, replace_body, or merge_frontmatter", nil)
	}
	if modeCount > 1 {
		return EditNoteOutput{}, app.NewCLIUsageError("edit modes append, replace_body, and merge_frontmatter are mutually exclusive", nil)
	}
	if input.ReplaceBody != "" && input.IfMatchHash == "" {
		return EditNoteOutput{}, app.NewUnsafeError("replace_body requires if_match_hash from read_note", nil)
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
		return EditNoteOutput{}, err
	}

	return EditNoteOutput{
		NoteID:      edited.NoteID,
		Slug:        edited.Slug,
		Path:        edited.Path,
		ContentHash: edited.ContentHash,
		CreatedAt:   edited.CreatedAt,
		UpdatedAt:   edited.UpdatedAt,
	}, nil
}
