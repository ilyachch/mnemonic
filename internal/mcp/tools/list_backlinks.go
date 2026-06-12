package tools

import (
	"context"

	"github.com/ilyachch/mnemonic/internal/graph"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const listBacklinksDescription = `List notes linking to a target note.
Use to check dependencies and impact before editing or deleting important knowledge.`

type ListBacklinksInput struct {
	Identifier string `json:"identifier" jsonschema:"Note id, slug, relative path, or exact title."`
	Limit      int    `json:"limit,omitempty" jsonschema:"Maximum backlinks to return."`
}

type ListBacklinksOutput struct {
	Links []graph.Backlink `json:"links"`
}

func RegisterListBacklinks(s *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "list_backlinks",
		Description: listBacklinksDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListBacklinksInput) (*sdkmcp.CallToolResult, ListBacklinksOutput, error) {
		db, err := deps.GetIndexDB()
		if err != nil {
			return nil, ListBacklinksOutput{}, err
		}

		target, err := QueryIndexedNoteByIdentifier(db, input.Identifier)
		if err != nil {
			return nil, ListBacklinksOutput{}, err
		}

		links, err := graph.Backlinks(db, target.NoteID)
		if err != nil {
			return nil, ListBacklinksOutput{}, err
		}
		if input.Limit > 0 && len(links) > input.Limit {
			links = links[:input.Limit]
		}
		if links == nil {
			links = []graph.Backlink{}
		}

		_ = ctx
		return nil, ListBacklinksOutput{Links: links}, nil
	})
}