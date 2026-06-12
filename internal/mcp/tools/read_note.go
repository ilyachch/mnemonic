package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyachch/mnemonic/internal/notes"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const readNoteDescription = `Read a note by note_id, slug, path, or title.
Always read before using, editing, or deleting a note. Returns 'content_hash', which is REQUIRED for safe edits/deletions.`

type ReadNoteInput struct {
	Identifier string `json:"identifier" jsonschema:"Note id, slug, relative path, or exact title."`
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

func RegisterReadNote(s *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "read_note",
		Description: readNoteDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ReadNoteInput) (*sdkmcp.CallToolResult, ReadNoteOutput, error) {
		root, err := deps.GetMemoriesRoot()
		if err != nil {
			return nil, ReadNoteOutput{}, err
		}

		resolved, err := notes.Resolve(root, input.Identifier)
		if err != nil {
			return nil, ReadNoteOutput{}, err
		}

		absPath := filepath.Join(root, filepath.FromSlash(resolved.Path))
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, ReadNoteOutput{}, fmt.Errorf("read note %q: %w", resolved.Path, err)
		}

		result := ReadNoteOutput{
			Note: ReadNoteItem{
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
}