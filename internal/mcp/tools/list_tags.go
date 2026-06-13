package tools

import (
	"context"
	"database/sql"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const listTagsDescription = `List available tags.
Use to understand categorization, ensure consistent tagging for new notes, or refine searches.`

type ListTagsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Maximum tags to return."`
}

type ListTagsOutput struct {
	Tags []ListTagsItem `json:"tags"`
}

type ListTagsItem struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func RegisterListTags(s *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "list_tags",
		Description: listTagsDescription,
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListTagsInput) (*sdkmcp.CallToolResult, ListTagsOutput, error) {
		db, err := deps.GetIndexDB()
		if err != nil {
			return nil, ListTagsOutput{}, err
		}

		tags, err := listTags(db, input.Limit)
		if err != nil {
			return nil, ListTagsOutput{}, err
		}
		if tags == nil {
			tags = []ListTagsItem{}
		}

		_ = ctx
		return nil, ListTagsOutput{Tags: tags}, nil
	})
}

func listTags(db *sql.DB, limit int) ([]ListTagsItem, error) {
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

	tags := make([]ListTagsItem, 0)
	for rows.Next() {
		var item ListTagsItem
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
