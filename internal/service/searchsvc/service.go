package searchsvc

import (
	"context"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/graph"
	"github.com/ilyachch/mnemonic/internal/search"
	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
)

// Service owns runtime search operations for one selected knowledge base.
type Service struct {
	Index sqliteindex.Store
}

// SearchInput configures a runtime search query.
type SearchInput struct {
	Query string
	Limit int
	Tag   string
}

// ListTagsOutput mirrors the legacy tag listing payload.
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

// New constructs the runtime search service for one knowledge base.
func New(k kb.KnowledgeBase) *Service {
	return &Service{
		Index: sqliteindex.Store{
			IndexPath: k.IndexPath,
			RootDir:   k.RootDir,
			KBID:      k.ID,
		},
	}
}

// Search runs a full-text query against the bound index.
func (s Service) Search(ctx context.Context, input SearchInput) ([]search.Result, error) {
	_ = ctx
	db, err := s.Index.OpenReadonly()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	hits, err := s.Index.Search(db, input.Query, input.Limit, input.Tag)
	if err != nil {
		return nil, err
	}

	out := make([]search.Result, 0, len(hits))
	for _, hit := range hits {
		out = append(out, search.Result{
			NoteID:      hit.NoteID,
			Slug:        hit.Slug,
			Title:       hit.Title,
			Path:        hit.Path,
			Score:       hit.Score,
			Snippet:     hit.Snippet,
			ContentHash: hit.ContentHash,
		})
	}
	return out, nil
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
func (s Service) Backlinks(ctx context.Context, input BacklinksInput) ([]graph.Backlink, error) {
	_ = ctx
	db, err := s.Index.OpenReadonly()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	target, err := s.Index.LookupNoteByIdentifier(db, input.Identifier)
	if err != nil {
		return nil, err
	}

	links, err := s.Index.Backlinks(db, target.NoteID, input.Limit)
	if err != nil {
		return nil, err
	}

	out := make([]graph.Backlink, 0, len(links))
	for _, link := range links {
		out = append(out, graph.Backlink{
			LinkID:       link.LinkID,
			NoteID:       link.NoteID,
			Slug:         link.Slug,
			Title:        link.Title,
			Path:         link.Path,
			RelationType: link.RelationType,
			SourceLine:   link.SourceLine,
		})
	}
	return out, nil
}
