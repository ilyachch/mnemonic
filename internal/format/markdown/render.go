package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

var canonicalFrontmatterKeys = map[string]struct{}{
	"mnemonic_note_id": {},
	"title":            {},
	"slug":             {},
	"tags":             {},
	"summary":          {},
	"aliases":          {},
	"type":             {},
}

var removedFrontmatterKeys = map[string]struct{}{
	"permalink":  {},
	"created_at": {},
	"updated_at": {},
}

// RenderNote serializes a note into canonical YAML frontmatter plus body.
func RenderNote(note Note) ([]byte, error) {
	if note.MnemonicNoteID == "" {
		return nil, errors.New("mnemonic_note_id is required")
	}

	slug := note.EffectiveSlug()

	var buf bytes.Buffer
	buf.WriteString("---\n")
	if err := renderCanonicalFrontmatter(&buf, note, slug); err != nil {
		return nil, err
	}
	if err := renderExtraFrontmatter(&buf, note); err != nil {
		return nil, err
	}
	buf.WriteString("---\n")
	buf.Write(note.Body)

	return buf.Bytes(), nil
}

func renderCanonicalFrontmatter(buf *bytes.Buffer, note Note, slug string) error {
	pairs := []struct {
		key   string
		value any
	}{
		{"mnemonic_note_id", note.MnemonicNoteID},
	}
	if note.Title != "" {
		pairs = append(pairs, struct {
			key   string
			value any
		}{"title", note.Title})
	}
	if slug != "" {
		pairs = append(pairs, struct {
			key   string
			value any
		}{"slug", slug})
	}
	if len(note.Tags) > 0 {
		pairs = append(pairs, struct {
			key   string
			value any
		}{"tags", note.Tags})
	}
	if note.Summary != "" {
		pairs = append(pairs, struct {
			key   string
			value any
		}{"summary", note.Summary})
	}
	if len(note.Aliases) > 0 {
		pairs = append(pairs, struct {
			key   string
			value any
		}{"aliases", note.Aliases})
	}
	if note.Type != "" {
		pairs = append(pairs, struct {
			key   string
			value any
		}{"type", note.Type})
	}
	for _, p := range pairs {
		if err := renderFrontmatterField(buf, p.key, p.value); err != nil {
			return err
		}
	}
	return nil
}

func renderExtraFrontmatter(buf *bytes.Buffer, note Note) error {
	extraKeys := make([]string, 0, len(note.Frontmatter))
	for key := range note.Frontmatter {
		if _, ok := canonicalFrontmatterKeys[key]; ok {
			continue
		}
		if _, ok := removedFrontmatterKeys[key]; ok {
			continue
		}
		extraKeys = append(extraKeys, key)
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		if err := renderFrontmatterField(buf, key, note.Frontmatter[key]); err != nil {
			return err
		}
	}
	return nil
}

func renderFrontmatterField(buf *bytes.Buffer, key string, value any) error {
	rendered, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("render frontmatter %q: %w", key, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(rendered, &doc); err != nil {
		return fmt.Errorf("render frontmatter %q: inspect yaml: %w", key, err)
	}

	rendered = bytes.TrimSuffix(rendered, []byte("\n"))
	if shouldRenderFrontmatterBlock(&doc, rendered) {
		fmt.Fprintf(buf, "%s:\n", key)
		for _, line := range bytes.Split(rendered, []byte("\n")) {
			buf.WriteString("  ")
			buf.Write(line)
			buf.WriteByte('\n')
		}
		return nil
	}

	fmt.Fprintf(buf, "%s: %s\n", key, rendered)
	return nil
}

func shouldRenderFrontmatterBlock(doc *yaml.Node, rendered []byte) bool {
	if bytes.Contains(rendered, []byte("\n")) {
		return true
	}
	if len(doc.Content) == 0 {
		return false
	}

	switch doc.Content[0].Kind {
	case yaml.MappingNode, yaml.SequenceNode:
		return true
	default:
		return false
	}
}
