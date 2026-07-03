package indexsvc

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
)

// DiagnosticKind enumerates known diagnostic categories.
type DiagnosticKind string

const (
	KindInvalidFrontmatter   DiagnosticKind = "invalid_frontmatter"
	KindMissingRequiredField DiagnosticKind = "missing_required_field"
	KindMissingSummary       DiagnosticKind = "missing_summary"
	KindInvalidTimestamp     DiagnosticKind = "invalid_timestamp"
	KindDuplicateSlug        DiagnosticKind = "duplicate_slug"
	KindDuplicateAlias       DiagnosticKind = "duplicate_alias"
	KindUnresolvedLink       DiagnosticKind = "unresolved_link"
	KindAmbiguousLink        DiagnosticKind = "ambiguous_link"
	KindEmptyBody            DiagnosticKind = "empty_body"
)

// DiagnoseInput configures a diagnostics run.
type DiagnoseInput struct {
	Kinds              []DiagnosticKind
	Limit              int
	Cursor             int
	IncludeSuggestions bool
}

// DiagnoseOutput is the paginated diagnostics payload.
type DiagnoseOutput struct {
	Issues     []DiagnosticIssue `json:"issues"`
	TotalCount int               `json:"total_count"`
	NextCursor int               `json:"next_cursor,omitempty"`
}

// DiagnosticIssue describes one detected problem on a note.
type DiagnosticIssue struct {
	Kind       DiagnosticKind        `json:"kind"`
	NoteID     string                `json:"note_id,omitempty"`
	Slug       string                `json:"slug,omitempty"`
	Path       string                `json:"path,omitempty"`
	Detail     string                `json:"detail,omitempty"`
	Candidates []DiagnosticCandidate `json:"candidates,omitempty"`
}

// DiagnosticCandidate is a suggested target for a broken link.
type DiagnosticCandidate struct {
	NoteID string `json:"note_id"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Path   string `json:"path"`
}

// Diagnose scans the knowledge base and returns per-note diagnostic issues.
func (s Service) Diagnose(ctx context.Context, input DiagnoseInput) (DiagnoseOutput, error) {
	_ = ctx

	kindFilter := buildKindFilter(input.Kinds)
	issues, err := s.collectIssues(kindFilter)
	if err != nil {
		return DiagnoseOutput{}, err
	}

	totalCount := len(issues)
	cursor := input.Cursor
	if cursor < 0 {
		cursor = 0
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}

	end := cursor + limit
	if end > len(issues) {
		end = len(issues)
	}

	out := DiagnoseOutput{
		Issues:     issues[cursor:end],
		TotalCount: totalCount,
	}
	if end < len(issues) {
		out.NextCursor = end
	}
	return out, nil
}

func buildKindFilter(kinds []DiagnosticKind) map[DiagnosticKind]bool {
	if len(kinds) == 0 {
		return nil
	}
	filter := make(map[DiagnosticKind]bool, len(kinds))
	for _, k := range kinds {
		filter[k] = true
	}
	return filter
}

type kindFilter map[DiagnosticKind]bool

func (f kindFilter) include(kind DiagnosticKind) bool {
	if f == nil {
		return true
	}
	return f[kind]
}

func (s Service) collectIssues(filter kindFilter) ([]DiagnosticIssue, error) {
	paths, err := (markdownstore.Store{RootDir: s.KB.RootDir}).Walk()
	if err != nil {
		return nil, err
	}

	issues, slugCounts, aliasCounts := s.scanNoteFiles(paths, filter)
	issues = append(issues, s.collectDuplicateSlugIssues(slugCounts, filter)...)
	issues = append(issues, s.collectDuplicateAliasIssues(aliasCounts, filter)...)

	linkIssues, err := s.collectLinkIssues(filter)
	if err != nil {
		return nil, err
	}
	issues = append(issues, linkIssues...)

	return issues, nil
}

func (s Service) scanNoteFiles(
	paths []string, filter kindFilter,
) (issues []DiagnosticIssue, slugCounts map[string][]string, aliasCounts map[string][]string) {
	root := s.KB.RootDir
	slugCounts = map[string][]string{}
	aliasCounts = map[string][]string{}

	for _, rel := range paths {
		if isTrashNote(rel) {
			continue
		}
		i, slug, aliases := s.checkOneNote(root, rel, filter)
		issues = append(issues, i...)
		for _, s := range slug {
			slugCounts[s] = append(slugCounts[s], rel)
		}
		for _, a := range aliases {
			aliasCounts[a] = append(aliasCounts[a], rel)
		}
	}
	return
}

func (s Service) checkOneNote(root, rel string, filter kindFilter) (issues []DiagnosticIssue, slugs []string, aliases []string) {
	absPath := filepath.Join(root, filepath.FromSlash(rel))
	data, readErr := os.ReadFile(absPath)
	if readErr != nil {
		return
	}

	note, parseErr := markdown.ParseNote(data)
	if parseErr != nil {
		if filter.include(KindInvalidFrontmatter) {
			issues = append(issues, DiagnosticIssue{
				Kind: KindInvalidFrontmatter, Path: rel, Detail: parseErr.Error(),
			})
		}
		return
	}

	issues = s.checkNoteFields(note, rel, filter)
	if s := note.EffectiveSlug(); s != "" {
		slugs = append(slugs, s)
	}
	for _, a := range note.Aliases {
		if a != "" {
			aliases = append(aliases, a)
		}
	}
	return
}

func (s Service) checkNoteFields(note markdown.Note, path string, filter kindFilter) []DiagnosticIssue {
	issues := make([]DiagnosticIssue, 0, 7)
	noteID := note.MnemonicNoteID
	slug := note.EffectiveSlug()

	issues = append(issues, s.checkRequiredFields(noteID, slug, note.Title, path, filter)...)
	issues = append(issues, s.checkContentFields(noteID, slug, note, path, filter)...)
	return issues
}

func (s Service) checkRequiredFields(noteID, slug, title, path string, filter kindFilter) []DiagnosticIssue {
	issues := make([]DiagnosticIssue, 0, 3)
	if noteID == "" && filter.include(KindMissingRequiredField) {
		issues = append(issues, DiagnosticIssue{
			Kind: KindMissingRequiredField, Slug: slug, Path: path,
			Detail: "missing mnemonic_note_id",
		})
	}
	if title == "" && filter.include(KindMissingRequiredField) {
		issues = append(issues, DiagnosticIssue{
			Kind: KindMissingRequiredField, NoteID: noteID, Slug: slug, Path: path,
			Detail: "missing title",
		})
	}
	if slug == "" && filter.include(KindMissingRequiredField) {
		issues = append(issues, DiagnosticIssue{
			Kind: KindMissingRequiredField, NoteID: noteID, Path: path,
			Detail: "missing slug",
		})
	}
	return issues
}

func (s Service) checkContentFields(noteID, slug string, note markdown.Note, path string, filter kindFilter) []DiagnosticIssue {
	issues := make([]DiagnosticIssue, 0, 4)
	if note.Summary == "" && filter.include(KindMissingSummary) {
		issues = append(issues, DiagnosticIssue{
			Kind: KindMissingSummary, NoteID: noteID, Slug: slug, Path: path,
		})
	}
	if !note.CreatedAt.IsZero() && note.CreatedAt.Unix() <= 0 && filter.include(KindInvalidTimestamp) {
		issues = append(issues, DiagnosticIssue{
			Kind: KindInvalidTimestamp, NoteID: noteID, Slug: slug, Path: path,
			Detail: "created_at <= 0",
		})
	}
	if !note.UpdatedAt.IsZero() && note.UpdatedAt.Unix() <= 0 && filter.include(KindInvalidTimestamp) {
		issues = append(issues, DiagnosticIssue{
			Kind: KindInvalidTimestamp, NoteID: noteID, Slug: slug, Path: path,
			Detail: "updated_at <= 0",
		})
	}
	if len(strings.TrimSpace(string(note.Body))) == 0 && filter.include(KindEmptyBody) {
		issues = append(issues, DiagnosticIssue{
			Kind: KindEmptyBody, NoteID: noteID, Slug: slug, Path: path,
		})
	}
	return issues
}

func (s Service) collectDuplicateSlugIssues(
	seenSlugs map[string][]string, filter kindFilter,
) []DiagnosticIssue {
	if !filter.include(KindDuplicateSlug) {
		return nil
	}
	var issues []DiagnosticIssue
	for slug, paths := range seenSlugs {
		if len(paths) > 1 {
			issues = append(issues, DiagnosticIssue{
				Kind:   KindDuplicateSlug,
				Slug:   slug,
				Detail: "duplicate slug across notes: " + strings.Join(paths, ", "),
			})
		}
	}
	return issues
}

func (s Service) collectDuplicateAliasIssues(
	seenAliases map[string][]string, filter kindFilter,
) []DiagnosticIssue {
	if !filter.include(KindDuplicateAlias) {
		return nil
	}
	var issues []DiagnosticIssue
	for alias, paths := range seenAliases {
		if len(paths) > 1 {
			issues = append(issues, DiagnosticIssue{
				Kind:   KindDuplicateAlias,
				Detail: "duplicate alias \"" + alias + "\" across notes: " + strings.Join(paths, ", "),
			})
		}
	}
	return issues
}

func (s Service) collectLinkIssues(filter kindFilter) ([]DiagnosticIssue, error) {
	if !filter.include(KindUnresolvedLink) && !filter.include(KindAmbiguousLink) {
		return nil, nil
	}

	exists, err := s.Index.Exists()
	if err != nil || !exists {
		return nil, nil
	}

	db, err := s.Index.OpenReadonly()
	if err != nil {
		return nil, nil
	}
	defer func() { _ = db.Close() }()

	linkIssues, err := s.Index.ListLinkIssues(db)
	if err != nil {
		return nil, err
	}

	var issues []DiagnosticIssue
	for _, li := range linkIssues {
		if li.IsAmbiguous && filter.include(KindAmbiguousLink) {
			issues = append(issues, DiagnosticIssue{
				Kind:   KindAmbiguousLink,
				NoteID: li.NoteID, Slug: li.Slug, Path: li.Path,
				Detail: "ambiguous link target \"" + li.Target + "\" at line " + strconv.Itoa(li.SourceLine),
			})
		}
		if !li.IsAmbiguous && filter.include(KindUnresolvedLink) {
			issues = append(issues, DiagnosticIssue{
				Kind:   KindUnresolvedLink,
				NoteID: li.NoteID, Slug: li.Slug, Path: li.Path,
				Detail: "unresolved link target \"" + li.Target + "\" at line " + strconv.Itoa(li.SourceLine),
			})
		}
	}
	return issues, nil
}
