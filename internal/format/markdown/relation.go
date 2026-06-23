package markdown

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// RelationSource describes where a parsed link came from.
type RelationSource string

const (
	// RelationSourceWikiLink marks a plain wiki-link outside the Relations section.
	RelationSourceWikiLink RelationSource = "wikilink"
	// RelationSourceRelationsSection marks a link extracted from the Relations section.
	RelationSourceRelationsSection RelationSource = "relations_section"
)

// RelationRef describes a parsed wiki-link with optional relation metadata.
type RelationRef struct {
	RelationType string
	Target       WikiLink
	Source       RelationSource
	Line         int
}

// ParseRelations extracts relation references and wiki-links from markdown text.
//
// Links inside a `## Relations` section are marked with source
// `relations_section`. Links outside that section are marked with source
// `wikilink`. Bullet lines in the Relations section may prefix the target link
// with a relation type such as `depends_on` or `relates_to`.
func ParseRelations(data []byte) []RelationRef {
	doc := parseDocument(data)
	if doc == nil {
		return nil
	}

	var refs []RelationRef
	inRelationsSection := false
	relationsSectionLevel := 0

	for child := doc.root.FirstChild(); child != nil; child = child.NextSibling() {
		if heading, ok := child.(*ast.Heading); ok {
			if isRelationsHeading(heading, doc) {
				inRelationsSection = true
				relationsSectionLevel = heading.Level
			} else if inRelationsSection && heading.Level <= relationsSectionLevel {
				inRelationsSection = false
				relationsSectionLevel = 0
			}
			continue
		}

		if inRelationsSection {
			appendRelationsFromNode(child, doc, &refs)
			continue
		}

		appendWikiLinksFromNode(child, doc, RelationSourceWikiLink, "", &refs)
	}

	return refs
}

func isRelationsHeading(heading *ast.Heading, doc *parsedDocument) bool {
	if heading.Level != 2 || heading.Lines().Len() == 0 {
		return false
	}

	line := heading.Lines().At(0)
	return bytes.Equal(bytes.TrimSpace(doc.lineBytes(doc.lineForOffset(line.Start))), []byte("## Relations"))
}

func appendRelationsFromNode(node ast.Node, doc *parsedDocument, refs *[]RelationRef) {
	switch n := node.(type) {
	case *ast.Heading:
		return
	case *ast.List:
		if n.IsOrdered() || n.Marker != '-' {
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				appendRelationsFromNode(child, doc, refs)
			}
			return
		}

		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			appendRelationsFromListItem(item.(*ast.ListItem), doc, refs)
		}
		return
	case *ast.ListItem:
		appendRelationsFromListItem(n, doc, refs)
		return
	case *ast.TextBlock, *ast.Paragraph:
		for _, line := range collectVisibleLines(n, doc.source, doc.lineForOffset) {
			appendWikiLinksFromLine(line.Text, line.Line, RelationSourceRelationsSection, "", refs)
		}
		return
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		appendRelationsFromNode(child, doc, refs)
	}
}

func appendRelationsFromListItem(item *ast.ListItem, doc *parsedDocument, refs *[]RelationRef) {
	lines := collectVisibleLines(item, doc.source, doc.lineForOffset)
	if len(lines) == 0 {
		return
	}

	relationType := parseRelationTypeLine(lines[0].Text)
	for i, line := range lines {
		currentRelationType := ""
		if i == 0 {
			currentRelationType = relationType
		}
		appendWikiLinksFromLine(line.Text, line.Line, RelationSourceRelationsSection, currentRelationType, refs)
	}
}

func appendWikiLinksFromNode(node ast.Node, doc *parsedDocument, source RelationSource, relationType string, refs *[]RelationRef) {
	switch n := node.(type) {
	case *ast.Heading:
		return
	case *ast.List:
		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			appendWikiLinksFromNode(item, doc, source, relationType, refs)
		}
		return
	case *ast.ListItem:
		for _, line := range collectVisibleLines(n, doc.source, doc.lineForOffset) {
			appendWikiLinksFromLine(line.Text, line.Line, source, relationType, refs)
		}
		return
	case *ast.TextBlock, *ast.Paragraph:
		for _, line := range collectVisibleLines(n, doc.source, doc.lineForOffset) {
			appendWikiLinksFromLine(line.Text, line.Line, source, relationType, refs)
		}
		return
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		appendWikiLinksFromNode(child, doc, source, relationType, refs)
	}
}

func appendWikiLinksFromLine(line string, lineNumber int, source RelationSource, relationType string, refs *[]RelationRef) {
	for _, link := range parseWikiLinksInLine([]byte(line), lineNumber) {
		*refs = append(*refs, RelationRef{
			RelationType: relationType,
			Target:       link,
			Source:       source,
			Line:         lineNumber,
		})
	}
}

func parseRelationTypeLine(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
