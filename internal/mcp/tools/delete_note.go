package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/notes"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const deleteNoteDescription = `Delete permanently obsolete or duplicated notes.
Read first. Prefer 'edit_note' (e.g., adding a deprecation warning) unless deletion is strictly required.`

type DeleteNoteInput struct {
	Identifier  string `json:"identifier" jsonschema:"Existing note id, slug, path, or title."`
	HardDelete  bool   `json:"hard_delete,omitempty" jsonschema:"Delete permanently instead of moving to trash."`
	IfMatchHash string `json:"if_match_hash,omitempty" jsonschema:"Content hash from read_note for safe deletion."`
}

type DeleteNoteOutput struct {
	Deleted   bool   `json:"deleted"`
	Mode      string `json:"mode"`
	Path      string `json:"path,omitempty"`
	TrashPath string `json:"trash_path,omitempty"`
}

func RegisterDeleteNote(s *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "delete_note",
		Description: deleteNoteDescription,
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: BoolPtr(true),
			OpenWorldHint:   BoolPtr(false),
		},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input DeleteNoteInput) (*sdkmcp.CallToolResult, DeleteNoteOutput, error) {
		root, err := deps.GetMemoriesRoot()
		if err != nil {
			return nil, DeleteNoteOutput{}, err
		}

		deleted, err := deleteNote(root, input)
		if err != nil {
			return nil, DeleteNoteOutput{}, err
		}
		if err := deps.RebuildIndex(root); err != nil {
			return nil, DeleteNoteOutput{}, err
		}

		_ = ctx
		return nil, deleted, nil
	})
}

func deleteNote(root string, input DeleteNoteInput) (DeleteNoteOutput, error) {
	if input.Identifier == "" {
		return DeleteNoteOutput{}, apperr.CLIUsage("note identifier is required", nil)
	}
	if input.HardDelete && input.IfMatchHash == "" {
		return DeleteNoteOutput{}, apperr.Unsafe("hard delete requires if_match_hash from read_note", nil)
	}

	if input.IfMatchHash != "" {
		resolved, err := notes.Resolve(root, input.Identifier)
		if err != nil {
			return DeleteNoteOutput{}, err
		}
		absPath := filepath.Join(root, filepath.FromSlash(resolved.Path))
		current, err := os.ReadFile(absPath)
		if err != nil {
			return DeleteNoteOutput{}, fmt.Errorf("read note: %w", err)
		}
		if notes.HashBytes(current) != input.IfMatchHash {
			return DeleteNoteOutput{}, apperr.Unsafe("content hash mismatch", nil)
		}
	}

	deleted, err := notes.Delete(notes.DeleteInput{
		RootDir:  root,
		Selector: input.Identifier,
		Hard:     input.HardDelete,
		Yes:      input.HardDelete,
	})
	if err != nil {
		return DeleteNoteOutput{}, err
	}

	return DeleteNoteOutput{
		Deleted:   true,
		Mode:      deleted.Mode,
		Path:      deleted.Path,
		TrashPath: deleted.TrashPath,
	}, nil
}
