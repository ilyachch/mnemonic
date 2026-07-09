package markdown

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/domain/slug"
	"gopkg.in/yaml.v3"
)

// Note holds parsed markdown note content and canonical frontmatter fields.
type Note struct {
	Frontmatter    map[string]any
	MnemonicNoteID string
	Title          string
	Slug           string
	Tags           []string
	Summary        string
	Aliases        []string
	Type           string
	Body           []byte
}

// EffectiveSlug returns the canonical slug.
func (n Note) EffectiveSlug() string {
	return n.Slug
}

// ParseNote parses a markdown note into canonical metadata, raw frontmatter, and body.
func ParseNote(data []byte) (Note, error) {
	split, err := ParseFrontmatter(data)
	if err != nil {
		return Note{}, err
	}

	note := Note{
		Frontmatter: map[string]any{},
		Body:        split.Body,
	}
	if !split.Present || len(split.Metadata) == 0 {
		return note, nil
	}

	var raw map[string]any
	if err := yaml.Unmarshal(split.Metadata, &raw); err != nil {
		return Note{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	if raw == nil {
		raw = map[string]any{}
	}
	note.Frontmatter = raw

	if err := note.populateFromRaw(raw); err != nil {
		return Note{}, err
	}

	return note, nil
}

func (n *Note) populateFromRaw(raw map[string]any) error {
	var errField error
	if n.MnemonicNoteID, errField = noteStringField(raw, "mnemonic_note_id"); errField != nil {
		return errField
	}
	if n.Title, errField = noteStringField(raw, "title"); errField != nil {
		return errField
	}
	if n.Slug, errField = noteStringField(raw, "slug"); errField != nil {
		return errField
	}
	if n.Tags, errField = noteStringSliceField(raw, "tags"); errField != nil {
		return errField
	}
	if n.Summary, errField = noteStringField(raw, "summary"); errField != nil {
		return errField
	}
	if n.Aliases, errField = noteStringSliceField(raw, "aliases"); errField != nil {
		return errField
	}
	if n.Type, errField = noteStringField(raw, "type"); errField != nil {
		return errField
	}

	return nil
}

func noteStringField(raw map[string]any, key string) (string, error) {
	value, ok := raw[key]
	if !ok || value == nil {
		return "", nil
	}

	s, ok := value.(string)
	if !ok {
		return "", newStringFieldError(key, fmt.Errorf("must be a string, got %T", value))
	}

	return s, nil
}

func noteStringSliceField(raw map[string]any, key string) ([]string, error) {
	value, ok := raw[key]
	if !ok || value == nil {
		return nil, nil
	}

	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			s, ok := item.(string)
			if !ok {
				return nil, newStringSliceFieldError(key, fmt.Errorf("items must be strings, got %T", item))
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, newStringSliceFieldError(key, fmt.Errorf("must be a list of strings, got %T", value))
	}
}

// GetOrDeriveTitle returns the note's title, deriving it dynamically when the
// YAML title is absent. Derivation order: explicit Title field, then the first
// H1 heading in the body, then the file's base name (without extension).
func (n Note) GetOrDeriveTitle(relPath string) string {
	if n.Title != "" {
		return n.Title
	}
	if title := ExtractH1Title(n.Body); title != "" {
		return title
	}
	return strings.TrimSuffix(filepath.Base(relPath), ".md")
}

// GetOrDeriveSlug returns the note's slug, deriving it dynamically when the
// YAML slug is absent. Derivation uses Slugify on the derived title; if that
// fails (e.g. non-ASCII input), a local cleanup fallback lowercases the
// filename and replaces any non-alphanumeric characters with "-".
func (n Note) GetOrDeriveSlug(relPath string) string {
	if n.Slug != "" {
		return n.Slug
	}
	derived, err := slug.Slugify(n.GetOrDeriveTitle(relPath))
	if err == nil && derived != "" {
		return derived
	}
	return slugFromBaseName(relPath)
}

// slugFromBaseName produces a best-effort slug from a file's base name by
// stripping the extension and lowercasing. Used when Slugify rejects the
// title (e.g. non-ASCII input).
func slugFromBaseName(relPath string) string {
	base := strings.TrimSuffix(filepath.Base(relPath), ".md")
	base = strings.ToLower(base)
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
