package cli

import (
	"strconv"
	"strings"

	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
	"github.com/spf13/cobra"
)

func newNotesSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search notes",
		RunE:  runNotesSearch,
	}
	cmd.Flags().StringSliceP("query", "q", nil, "search query (repeatable)")
	cmd.Flags().StringSliceP("tag", "t", nil, "filter by tag (repeatable)")
	cmd.Flags().String("created-since", "", "filter by creation time (relative, e.g. 24h)")
	cmd.Flags().String("updated-since", "", "filter by update time (relative, e.g. 24h)")
	cmd.Flags().Int64("created-before", 0, "filter by creation time (Unix timestamp)")
	cmd.Flags().Int64("created-after", 0, "filter by creation time (Unix timestamp)")
	cmd.Flags().Int64("updated-before", 0, "filter by update time (Unix timestamp)")
	cmd.Flags().Int64("updated-after", 0, "filter by update time (Unix timestamp)")
	cmd.Flags().Bool("include-related", false, "include related notes in results")
	cmd.Flags().Bool("debug", false, "show debug fields (path, score, content_hash)")
	cmd.Flags().Int("limit", 10, "maximum number of results")
	return cmd
}

func runNotesSearch(cmd *cobra.Command, args []string) error {
	queries, err := cmd.Flags().GetStringSlice("query")
	if err != nil {
		return err
	}
	tags, err := cmd.Flags().GetStringSlice("tag")
	if err != nil {
		return err
	}
	limit, err := cmd.Flags().GetInt("limit")
	if err != nil {
		return err
	}
	createdSince, err := cmd.Flags().GetString("created-since")
	if err != nil {
		return err
	}
	updatedSince, err := cmd.Flags().GetString("updated-since")
	if err != nil {
		return err
	}
	includeRelated, err := cmd.Flags().GetBool("include-related")
	if err != nil {
		return err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return err
	}

	createdBefore, err := parseInt64Flag(cmd, "created-before")
	if err != nil {
		return err
	}
	createdAfter, err := parseInt64Flag(cmd, "created-after")
	if err != nil {
		return err
	}
	updatedBefore, err := parseInt64Flag(cmd, "updated-before")
	if err != nil {
		return err
	}
	updatedAfter, err := parseInt64Flag(cmd, "updated-after")
	if err != nil {
		return err
	}

	runtime, err := runtimeAppForSelectedProject(cmd)
	if err != nil {
		return err
	}

	if err = requireRuntimeSearchIndex(runtime); err != nil {
		return err
	}

	input := searchsvc.AdvancedSearchInput{
		Queries:        queries,
		Tags:           tags,
		CreatedBefore:  createdBefore,
		CreatedAfter:   createdAfter,
		UpdatedBefore:  updatedBefore,
		UpdatedAfter:   updatedAfter,
		CreatedSince:   createdSince,
		UpdatedSince:   updatedSince,
		Limit:          limit,
		IncludeRelated: includeRelated,
	}

	hits, err := runtime.Services.Search.AdvancedSearch(commandContext(cmd), input)
	if err != nil {
		return err
	}

	output := notesSearchOutputFromHits(hits, debug)
	human := formatNotesSearchHuman(hits, debug)
	return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
}

func parseInt64Flag(cmd *cobra.Command, name string) (*int64, error) {
	if !cmd.Flags().Changed(name) {
		return nil, nil
	}
	v, err := cmd.Flags().GetInt64(name)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

type notesSearchOutput struct {
	Hits []notesSearchHit `json:"hits"`
}

type notesSearchHit struct {
	NoteID         string               `json:"note_id"`
	Slug           string               `json:"slug"`
	Title          string               `json:"title"`
	Snippet        string               `json:"snippet"`
	Summary        string               `json:"summary,omitempty"`
	Tags           []string             `json:"tags,omitempty"`
	MatchedQueries []string             `json:"matched_queries,omitempty"`
	Path           string               `json:"path,omitempty"`
	Score          *float64             `json:"score,omitempty"`
	ContentHash    string               `json:"content_hash,omitempty"`
	RelatedNotes   []notesSearchRelated `json:"related_notes,omitempty"`
}

type notesSearchRelated struct {
	NoteID       string `json:"note_id"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Path         string `json:"path,omitempty"`
	RelationType string `json:"relation_type"`
	SourceKind   string `json:"source_kind"`
	Direction    string `json:"direction"`
}

func notesSearchOutputFromHits(hits []searchsvc.AdvancedSearchResult, debug bool) notesSearchOutput {
	out := notesSearchOutput{Hits: make([]notesSearchHit, 0, len(hits))}
	for _, hit := range hits {
		nh := notesSearchHit{
			NoteID:         hit.NoteID,
			Slug:           hit.Slug,
			Title:          hit.Title,
			Snippet:        hit.Snippet,
			Summary:        hit.Summary,
			Tags:           hit.Tags,
			MatchedQueries: hit.MatchedQueries,
		}
		if debug {
			nh.Path = hit.Path
			score := hit.Score
			nh.Score = &score
			nh.ContentHash = hit.ContentHash
		}
		for _, rn := range hit.RelatedNotes {
			nr := notesSearchRelated{
				NoteID:       rn.NoteID,
				Slug:         rn.Slug,
				Title:        rn.Title,
				RelationType: rn.RelationType,
				SourceKind:   rn.SourceKind,
				Direction:    rn.Direction,
			}
			if debug {
				nr.Path = rn.Path
			}
			nh.RelatedNotes = append(nh.RelatedNotes, nr)
		}
		out.Hits = append(out.Hits, nh)
	}
	return out
}

func formatNotesSearchHuman(hits []searchsvc.AdvancedSearchResult, debug bool) string {
	if len(hits) == 0 {
		return "0 hits\n"
	}
	var sb strings.Builder
	for i, hit := range hits {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(hit.Slug)
		sb.WriteString("  ")
		sb.WriteString(hit.Title)
		if hit.Snippet != "" {
			sb.WriteString("  |  ")
			sb.WriteString(hit.Snippet)
		}
		if debug {
			sb.WriteString("  |  path=")
			sb.WriteString(hit.Path)
			sb.WriteString("  score=")
			sb.WriteString(strconv.FormatFloat(hit.Score, 'f', 3, 64))
			sb.WriteString("  hash=")
			sb.WriteString(hit.ContentHash)
		}
		for _, rn := range hit.RelatedNotes {
			sb.WriteString("\n  -> ")
			sb.WriteString(rn.Slug)
			sb.WriteString(" [")
			sb.WriteString(rn.RelationType)
			sb.WriteString("]")
			if debug {
				sb.WriteString(" ")
				sb.WriteString(rn.Path)
			}
		}
	}
	sb.WriteString("\n")
	return sb.String()
}
