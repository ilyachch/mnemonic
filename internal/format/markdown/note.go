package markdown

import (
	"fmt"
	"time"

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
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Type           string
	Permalink      string
	Body           []byte
}

// EffectiveSlug returns the canonical slug to write back to frontmatter.
//
// Permalink remains a compatibility fallback for older notes that have not yet
// been rewritten to the canonical `slug` field.
func (n Note) EffectiveSlug() string {
	if n.Slug != "" {
		return n.Slug
	}
	return n.Permalink
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

	var errField error
	if note.MnemonicNoteID, errField = noteStringField(raw, "mnemonic_note_id"); errField != nil {
		return Note{}, errField
	}
	if note.Title, errField = noteStringField(raw, "title"); errField != nil {
		return Note{}, errField
	}
	if note.Slug, errField = noteStringField(raw, "slug"); errField != nil {
		return Note{}, errField
	}
	if note.Permalink, errField = noteStringField(raw, "permalink"); errField != nil {
		return Note{}, errField
	}
	if note.Slug == "" {
		note.Slug = note.Permalink
	}
	if note.Tags, errField = noteStringSliceField(raw, "tags"); errField != nil {
		return Note{}, errField
	}
	if note.Summary, errField = noteStringField(raw, "summary"); errField != nil {
		return Note{}, errField
	}
	if note.Aliases, errField = noteStringSliceField(raw, "aliases"); errField != nil {
		return Note{}, errField
	}
	if note.CreatedAt, errField = noteTimeField(raw, "created_at"); errField != nil {
		return Note{}, errField
	}
	if note.UpdatedAt, errField = noteTimeField(raw, "updated_at"); errField != nil {
		return Note{}, errField
	}
	if note.Type, errField = noteStringField(raw, "type"); errField != nil {
		return Note{}, errField
	}

	return note, nil
}

func noteStringField(raw map[string]any, key string) (string, error) {
	value, ok := raw[key]
	if !ok || value == nil {
		return "", nil
	}

	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("frontmatter %q must be a string", key)
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
				return nil, fmt.Errorf("frontmatter %q items must be strings", key)
			}
			out = append(out, s)
		}
		return out, nil
	case string:
		return []string{typed}, nil
	default:
		return nil, fmt.Errorf("frontmatter %q must be a string or list of strings", key)
	}
}

func noteTimeField(raw map[string]any, key string) (time.Time, error) {
	value, ok := raw[key]
	if !ok || value == nil {
		return time.Time{}, nil
	}

	switch typed := value.(type) {
	case int:
		return time.Unix(int64(typed), 0).UTC(), nil
	case int64:
		return time.Unix(typed, 0).UTC(), nil
	default:
		return time.Time{}, fmt.Errorf("frontmatter %q must be a Unix timestamp as an integer, got %T", key, value)
	}
}
