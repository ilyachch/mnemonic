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
	if n.CreatedAt, errField = noteTimeField(raw, "created_at"); errField != nil {
		return errField
	}
	if n.UpdatedAt, errField = noteTimeField(raw, "updated_at"); errField != nil {
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
	case string:
		return []string{typed}, nil
	default:
		return nil, newStringSliceFieldError(key, fmt.Errorf("must be a string or list of strings, got %T", value))
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
		return time.Time{}, newTimeFieldError(key, fmt.Errorf("must be a Unix timestamp as an integer, got %T", value))
	}
}
