package tools

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/ilyachch/mnemonic/internal/notes"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const listNotesDescription = `List all notes in this memory (paginated).
Use to browse available knowledge. Prefer 'search_notes' for specific queries.`

type ListNotesInput struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"Maximum notes to return."`
	Cursor string `json:"cursor,omitempty" jsonschema:"Pagination cursor from a previous list_notes call."`
}

type ListNotesOutput struct {
	Notes      []notes.NoteSummary `json:"notes"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

func RegisterListNotes(s *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "list_notes",
		Description: listNotesDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListNotesInput) (*sdkmcp.CallToolResult, ListNotesOutput, error) {
		root, err := deps.GetMemoriesRoot()
		if err != nil {
			return nil, ListNotesOutput{}, err
		}

		allNotes, err := notes.List(root)
		if err != nil {
			if os.IsNotExist(err) {
				allNotes = nil
			} else {
				return nil, ListNotesOutput{}, err
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
				return nil, ListNotesOutput{}, fmt.Errorf("invalid cursor %q", input.Cursor)
			}
		}
		if cursor > len(allNotes) {
			cursor = len(allNotes)
		}
		end := cursor + limit
		if end > len(allNotes) {
			end = len(allNotes)
		}

		result := ListNotesOutput{
			Notes: allNotes[cursor:end],
		}
		if end < len(allNotes) {
			result.NextCursor = strconv.Itoa(end)
		}
		_ = ctx
		return nil, result, nil
	})
}
