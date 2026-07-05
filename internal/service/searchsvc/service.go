package searchsvc

import (
	"context"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/platform/clock"
	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
)

// Service owns runtime search operations for one selected knowledge base.
type Service struct {
	Index  sqliteindex.Store
	Logger *slog.Logger
}

// ListTagsOutput wraps a tag listing result.
type ListTagsOutput struct {
	Tags []ListTagsItem `json:"tags"`
}

// ListTagsItem is a grouped tag row.
type ListTagsItem struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// BacklinksInput configures backlink lookup for one note.
type BacklinksInput struct {
	Identifier string
	Limit      int
}

// Backlink wraps a backlink reference.
type Backlink struct {
	LinkID       string `json:"link_id"`
	NoteID       string `json:"note_id"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Path         string `json:"path"`
	RelationType string `json:"relation_type"`
	SourceKind   string `json:"source_kind"`
	SourceLine   int    `json:"source_line"`
}

// AdvancedSearchInput configures an advanced search with multi-query, time
// filters, tag filters, and related-note inclusion.
type AdvancedSearchInput struct {
	Queries        []string
	Tags           []string
	CreatedBefore  *int64
	CreatedAfter   *int64
	UpdatedBefore  *int64
	UpdatedAfter   *int64
	CreatedSince   string
	UpdatedSince   string
	Limit          int
	IncludeRelated bool
}

// AdvancedSearchResult mirrors an advanced search hit with optional related
// notes.
type AdvancedSearchResult struct {
	NoteID         string            `json:"note_id"`
	Slug           string            `json:"slug"`
	Title          string            `json:"title"`
	Path           string            `json:"path"`
	Score          float64           `json:"score"`
	Snippet        string            `json:"snippet"`
	ContentHash    string            `json:"content_hash"`
	Summary        string            `json:"summary"`
	Tags           []string          `json:"tags,omitempty"`
	MatchedQueries []string          `json:"matched_queries,omitempty"`
	RelatedNotes   []RelatedNoteItem `json:"related_notes,omitempty"`
}

// RelatedNoteItem is a short linked-note reference for the service layer.
type RelatedNoteItem struct {
	NoteID       string `json:"note_id"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Path         string `json:"path"`
	RelationType string `json:"relation_type"`
	SourceKind   string `json:"source_kind"`
	Direction    string `json:"direction"`
}

// New constructs the runtime search service for one knowledge base.
func New(k kb.KnowledgeBase, logger *slog.Logger) *Service {
	return &Service{
		Index: sqliteindex.Store{
			IndexPath: k.IndexPath,
			RootDir:   k.RootDir,
			StateDir:  k.StateDir,
			KBID:      k.ID,
		},
		Logger: logger,
	}
}

// ListTags returns tag counts from the bound index.
func (s Service) ListTags(ctx context.Context) (ListTagsOutput, error) {
	_ = ctx
	db, err := s.Index.OpenReadonly()
	if err != nil {
		return ListTagsOutput{}, err
	}
	defer func() { _ = db.Close() }()

	tags, err := s.Index.ListTags(db)
	if err != nil {
		return ListTagsOutput{}, err
	}

	out := ListTagsOutput{Tags: make([]ListTagsItem, 0, len(tags))}
	for _, tag := range tags {
		out.Tags = append(out.Tags, ListTagsItem{Tag: tag.Tag, Count: tag.Count})
	}
	return out, nil
}

// Backlinks resolves a note identifier and returns inbound links.
func (s Service) Backlinks(ctx context.Context, input BacklinksInput) ([]Backlink, error) {
	_ = ctx
	db, err := s.Index.OpenReadonly()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	target, err := s.Index.LookupNoteByIdentifier(db, input.Identifier)
	if err != nil {
		if strings.HasPrefix(err.Error(), "note ") && strings.HasSuffix(err.Error(), " not found") {
			return nil, apperr.NotFound(err.Error(), nil)
		}
		return nil, err
	}

	links, err := s.Index.Backlinks(db, target.NoteID, input.Limit)
	if err != nil {
		return nil, err
	}

	out := make([]Backlink, 0, len(links))
	for _, link := range links {
		out = append(out, Backlink{
			LinkID:       link.LinkID,
			NoteID:       link.NoteID,
			Slug:         link.Slug,
			Title:        link.Title,
			Path:         link.Path,
			RelationType: link.RelationType,
			SourceKind:   link.SourceKind,
			SourceLine:   link.SourceLine,
		})
	}
	return out, nil
}

// AdvancedSearch runs an advanced search with multi-query, time filters, tag
// filters, graph-aware reranking, and optional related notes.
func (s Service) AdvancedSearch(ctx context.Context, input AdvancedSearchInput) ([]AdvancedSearchResult, error) {
	_ = ctx

	if input.Limit < 0 {
		return nil, apperr.CLIUsage("limit must be >= 0", nil)
	}
	if input.Limit == 0 {
		input.Limit = 10
	}
	if err := validateAdvancedSearchInput(input); err != nil {
		return nil, err
	}

	db, err := s.Index.OpenReadonly()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	opts := sqliteindex.SearchOptions{
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

	if s.Logger != nil {
		s.Logger.Debug("advanced search executed",
			"queries", input.Queries,
			"tags", input.Tags,
			"created_since", input.CreatedSince,
			"updated_since", input.UpdatedSince,
			"include_related", input.IncludeRelated,
		)
	}

	hits, err := s.Index.SearchAdvanced(db, opts, clock.NowUTC())
	if err != nil {
		return nil, err
	}

	if s.Logger != nil {
		s.Logger.Info("advanced search completed",
			"count", len(hits),
		)
	}

	out := make([]AdvancedSearchResult, 0, len(hits))
	for _, hit := range hits {
		related := make([]RelatedNoteItem, 0, len(hit.RelatedNotes))
		for _, rn := range hit.RelatedNotes {
			related = append(related, RelatedNoteItem{
				NoteID:       rn.NoteID,
				Slug:         rn.Slug,
				Title:        rn.Title,
				Path:         rn.Path,
				RelationType: rn.RelationType,
				SourceKind:   rn.SourceKind,
				Direction:    rn.Direction,
			})
		}
		out = append(out, AdvancedSearchResult{
			NoteID:         hit.NoteID,
			Slug:           hit.Slug,
			Title:          hit.Title,
			Path:           hit.Path,
			Score:          hit.Score,
			Snippet:        hit.Snippet,
			ContentHash:    hit.ContentHash,
			Summary:        hit.Summary,
			Tags:           hit.Tags,
			MatchedQueries: hit.MatchedQueries,
			RelatedNotes:   related,
		})
	}
	return out, nil
}

func validateAdvancedSearchInput(input AdvancedSearchInput) error {
	if input.Limit < 1 || input.Limit > 100 {
		return apperr.CLIUsage("limit must be between 1 and 100", nil)
	}
	if len(input.Queries) > 8 {
		return apperr.CLIUsage("too many queries", nil)
	}
	for _, q := range input.Queries {
		if utf8.RuneCountInString(q) > 500 {
			return apperr.CLIUsage("query too long", nil)
		}
	}
	return nil
}
