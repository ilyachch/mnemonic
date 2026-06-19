package tools

import (
	"context"

	"github.com/ilyachch/mnemonic/internal/search"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const searchNotesDescription = `Search this memory.
YOU MUST search proactively before acting or asking questions to find existing rules, workflows, or preferences. Never ask the user for facts already stored. Read found notes before proceeding.`

type SearchNotesInput struct {
	Query string `json:"query" jsonschema:"Natural language query or keywords. Include project entities such as features, bugs, decisions, files, modules, APIs, services, tickets, or architecture concepts."`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum hits to return. Use 5-10 for focused searches."`
	Tag   string `json:"tag,omitempty" jsonschema:"Optional tag filter when the topic is known."`
}

type SearchNotesOutput struct {
	Hits []search.Result `json:"hits"`
}

func RegisterSearchNotes(s *sdkmcp.Server, deps Dependencies, description string) {
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "search_notes",
		Description: buildToolDescription(description, searchNotesDescription),
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input SearchNotesInput) (*sdkmcp.CallToolResult, SearchNotesOutput, error) {
		db, err := deps.GetIndexDB()
		if err != nil {
			return nil, SearchNotesOutput{}, err
		}

		hits, err := search.Search(db, input.Query, input.Limit, input.Tag)
		if err != nil {
			return nil, SearchNotesOutput{}, err
		}

		_ = ctx
		return nil, SearchNotesOutput{Hits: hits}, nil
	})
}
