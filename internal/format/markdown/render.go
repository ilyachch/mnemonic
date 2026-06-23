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
	"created_at":       {},
	"updated_at":       {},
	"type":             {},
	"permalink":        {},
}

// RenderNote serializes a note into canonical YAML frontmatter plus body.
func RenderNote(note Note) ([]byte, error) {
	if note.MnemonicNoteID == "" {
		return nil, errors.New("mnemonic_note_id is required")
	}

	slug := note.EffectiveSlug()
	if slug == "" {
		return nil, errors.New("slug is required")
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")

	if err := renderFrontmatterField(&buf, "mnemonic_note_id", note.MnemonicNoteID); err != nil {
		return nil, err
	}
	if note.Title != "" {
		if err := renderFrontmatterField(&buf, "title", note.Title); err != nil {
			return nil, err
		}
	}
	if err := renderFrontmatterField(&buf, "slug", slug); err != nil {
		return nil, err
	}
	if len(note.Tags) > 0 {
		if err := renderFrontmatterField(&buf, "tags", note.Tags); err != nil {
			return nil, err
		}
	}
	if !note.CreatedAt.IsZero() {
		if err := renderFrontmatterField(&buf, "created_at", note.CreatedAt.UTC()); err != nil {
			return nil, err
		}
	}
	if !note.UpdatedAt.IsZero() {
		if err := renderFrontmatterField(&buf, "updated_at", note.UpdatedAt.UTC()); err != nil {
			return nil, err
		}
	}
	if note.Type != "" {
		if err := renderFrontmatterField(&buf, "type", note.Type); err != nil {
			return nil, err
		}
	}

	extraKeys := make([]string, 0, len(note.Frontmatter))
	for key := range note.Frontmatter {
		if _, ok := canonicalFrontmatterKeys[key]; ok {
			continue
		}
		extraKeys = append(extraKeys, key)
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		if err := renderFrontmatterField(&buf, key, note.Frontmatter[key]); err != nil {
			return nil, err
		}
	}

	buf.WriteString("---\n")
	buf.Write(note.Body)

	return buf.Bytes(), nil
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
