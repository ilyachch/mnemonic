package indexsvc

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ilyachch/mnemonic/internal/apperr"
	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
)

// DiagnosticKind enumerates known diagnostic categories.
type DiagnosticKind string

const (
	KindInvalidFrontmatter   DiagnosticKind = "invalid_frontmatter"
	KindMissingRequiredField DiagnosticKind = "missing_required_field"
	KindMissingSummary       DiagnosticKind = "missing_summary"
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
	Field      string                `json:"field,omitempty"`
	SourceLine int                   `json:"source_line,omitempty"`
	SourceKind string                `json:"source_kind,omitempty"`
	LinkStyle  string                `json:"link_style,omitempty"`
	Detail     string                `json:"detail,omitempty"`
	Target     string                `json:"target,omitempty"`
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

	if input.Limit < 0 {
		return DiagnoseOutput{}, apperr.CLIUsage("limit must be >= 0", nil)
	}
	if input.Limit == 0 {
		input.Limit = 50
	}
	if err := validateDiagnoseInput(input); err != nil {
		return DiagnoseOutput{}, err
	}

	kindFilter := buildKindFilter(input.Kinds)
	issues, err := s.collectIssues(kindFilter)
	if err != nil {
		return DiagnoseOutput{}, err
	}

	totalCount := len(issues)
	cursor := input.Cursor
	if cursor >= len(issues) {
		return DiagnoseOutput{TotalCount: totalCount}, nil
	}
	limit := input.Limit
	if limit == 0 {
		limit = 50
	}

	end := cursor + limit
	if end > len(issues) {
		end = len(issues)
	}

	paged := issues[cursor:end]
	if input.IncludeSuggestions {
		paged = s.populateSuggestions(paged)
	}

	out := DiagnoseOutput{
		Issues:     paged,
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

var validDiagnosticKinds = map[DiagnosticKind]bool{
	KindInvalidFrontmatter:   true,
	KindMissingRequiredField: true,
	KindMissingSummary:       true,
	KindDuplicateSlug:        true,
	KindDuplicateAlias:       true,
	KindUnresolvedLink:       true,
	KindAmbiguousLink:        true,
	KindEmptyBody:            true,
}

func validateDiagnoseInput(input DiagnoseInput) error {
	if input.Limit < 1 || input.Limit > 200 {
		return apperr.CLIUsage("limit must be between 1 and 200", nil)
	}
	if input.Cursor < 0 {
		return apperr.CLIUsage("cursor must be >= 0", nil)
	}
	if len(input.Kinds) > len(validDiagnosticKinds) {
		return apperr.CLIUsage("too many diagnostic kinds", nil)
	}
	for _, k := range input.Kinds {
		if !validDiagnosticKinds[k] {
			return apperr.CLIUsage("unknown diagnostic kind: "+string(k), nil)
		}
	}
	return nil
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
		issues = s.classifyParseError(parseErr, rel, filter)
		return
	}

	issues = s.checkNoteFields(note, rel, filter)
	if s := note.GetOrDeriveSlug(rel); s != "" {
		slugs = append(slugs, s)
	}
	for _, a := range note.Aliases {
		if a != "" {
			aliases = append(aliases, a)
		}
	}
	return
}

func (s Service) classifyParseError(parseErr error, rel string, filter kindFilter) []DiagnosticIssue {
	var issues []DiagnosticIssue
	var fieldErr *markdown.FrontmatterFieldError
	if errors.As(parseErr, &fieldErr) {
		if filter.include(KindInvalidFrontmatter) {
			issues = append(issues, DiagnosticIssue{
				Kind: KindInvalidFrontmatter, Path: rel, Field: fieldErr.Field, Detail: parseErr.Error(),
			})
		}
	} else {
		if filter.include(KindInvalidFrontmatter) {
			issues = append(issues, DiagnosticIssue{
				Kind: KindInvalidFrontmatter, Path: rel, Detail: parseErr.Error(),
			})
		}
	}
	return issues
}

func (s Service) checkNoteFields(note markdown.Note, path string, filter kindFilter) []DiagnosticIssue {
	issues := make([]DiagnosticIssue, 0, 7)
	noteID := note.MnemonicNoteID
	slug := note.GetOrDeriveSlug(path)
	title := note.GetOrDeriveTitle(path)

	issues = append(issues, s.checkRequiredFields(noteID, slug, title, path, filter)...)
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
	issues := make([]DiagnosticIssue, 0, 6)
	if note.Summary == "" && filter.include(KindMissingSummary) {
		issues = append(issues, DiagnosticIssue{
			Kind: KindMissingSummary, NoteID: noteID, Slug: slug, Path: path,
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
				Target:     li.Target,
				SourceLine: li.SourceLine,
				SourceKind: li.SourceKind,
				LinkStyle:  li.LinkStyle,
				Detail:     "ambiguous link target \"" + li.Target + "\" at line " + strconv.Itoa(li.SourceLine),
			})
		}
		if !li.IsAmbiguous && filter.include(KindUnresolvedLink) {
			issues = append(issues, DiagnosticIssue{
				Kind:   KindUnresolvedLink,
				NoteID: li.NoteID, Slug: li.Slug, Path: li.Path,
				Target:     li.Target,
				SourceLine: li.SourceLine,
				SourceKind: li.SourceKind,
				LinkStyle:  li.LinkStyle,
				Detail:     "unresolved link target \"" + li.Target + "\" at line " + strconv.Itoa(li.SourceLine),
			})
		}
	}
	return issues, nil
}

func (s Service) populateSuggestions(issues []DiagnosticIssue) []DiagnosticIssue {
	targets := collectLinkIssueTargets(issues)
	if len(targets) == 0 {
		return issues
	}

	db, err := s.Index.OpenReadonly()
	if err != nil {
		return issues
	}
	defer func() { _ = db.Close() }()

	candidatesByTarget := searchCandidatesByTarget(s.Index, db, targets)

	for i := range issues {
		if issues[i].Target != "" && (issues[i].Kind == KindUnresolvedLink || issues[i].Kind == KindAmbiguousLink) {
			issues[i].Candidates = candidatesByTarget[issues[i].Target]
		}
	}
	return issues
}

func collectLinkIssueTargets(issues []DiagnosticIssue) map[string]bool {
	targets := make(map[string]bool)
	for _, issue := range issues {
		if issue.Target != "" && (issue.Kind == KindUnresolvedLink || issue.Kind == KindAmbiguousLink) {
			targets[issue.Target] = true
		}
	}
	return targets
}

func searchCandidatesByTarget(store sqliteindex.Store, db *sql.DB, targets map[string]bool) map[string][]DiagnosticCandidate {
	candidatesByTarget := make(map[string][]DiagnosticCandidate, len(targets))

	targetList := make([]string, 0, len(targets))
	for target := range targets {
		targetList = append(targetList, target)
	}

	hitsByTarget, err := store.SearchCandidatesByTargets(db, targetList, 3)
	if err != nil {
		return candidatesByTarget
	}

	for target, hits := range hitsByTarget {
		candidates := make([]DiagnosticCandidate, 0, len(hits))
		for _, hit := range hits {
			candidates = append(candidates, DiagnosticCandidate{
				NoteID: hit.NoteID,
				Slug:   hit.Slug,
				Title:  hit.Title,
				Path:   hit.Path,
			})
		}
		candidatesByTarget[target] = candidates
	}

	return candidatesByTarget
}
