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

// RelationRef describes a parsed link with optional relation metadata.
type RelationRef struct {
	RelationType string
	Target       WikiLink
	Source       RelationSource
	Line         int
	LinkStyle    string
}

// ParseRelations extracts relation references, wiki-links, and standard
// markdown links from markdown text.
//
// Links inside a `## Relations` section are marked with source
// `relations_section`. Links outside that section are marked with source
// `wikilink`. Bullet lines in the Relations section may prefix the target link
// with a relation type such as `depends_on` or `relates_to`.
//
// Wiki-links are tagged with LinkStyle "wiki" and standard markdown inline
// links with "regular".
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

		appendLinksFromNode(child, doc, RelationSourceWikiLink, "", &refs)
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
			appendLinksFromLine(line.Text, line.Line, RelationSourceRelationsSection, "", refs)
		}
		collectRegularLinksFromNode(n, doc, RelationSourceRelationsSection, "", refs)
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
		appendLinksFromLine(line.Text, line.Line, RelationSourceRelationsSection, currentRelationType, refs)
	}
	collectRegularLinksFromNode(item, doc, RelationSourceRelationsSection, relationType, refs)
}

func appendLinksFromNode(node ast.Node, doc *parsedDocument, source RelationSource, relationType string, refs *[]RelationRef) {
	switch n := node.(type) {
	case *ast.Heading:
		return
	case *ast.List:
		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			appendLinksFromNode(item, doc, source, relationType, refs)
		}
		return
	case *ast.ListItem:
		for _, line := range collectVisibleLines(n, doc.source, doc.lineForOffset) {
			appendLinksFromLine(line.Text, line.Line, source, relationType, refs)
		}
		collectRegularLinksFromNode(n, doc, source, relationType, refs)
		return
	case *ast.TextBlock, *ast.Paragraph:
		for _, line := range collectVisibleLines(n, doc.source, doc.lineForOffset) {
			appendLinksFromLine(line.Text, line.Line, source, relationType, refs)
		}
		collectRegularLinksFromNode(n, doc, source, relationType, refs)
		return
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		appendLinksFromNode(child, doc, source, relationType, refs)
	}
}

func appendLinksFromLine(line string, lineNumber int, source RelationSource, relationType string, refs *[]RelationRef) {
	for _, link := range parseWikiLinksInLine([]byte(line), lineNumber) {
		*refs = append(*refs, RelationRef{
			RelationType: relationType,
			Target:       link,
			Source:       source,
			Line:         lineNumber,
			LinkStyle:    "wiki",
		})
	}
}

func collectRegularLinksFromNode(node ast.Node, doc *parsedDocument, source RelationSource, relationType string, refs *[]RelationRef) {
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		switch n := n.(type) {
		case *ast.CodeBlock, *ast.FencedCodeBlock, *ast.CodeSpan:
			return
		case *ast.Link:
			dest := strings.TrimSpace(string(n.Destination))
			if len(dest) == 0 || isExternalOrAnchorString(dest) {
				return
			}
			label := collectLinkLabelText(n, doc.source)
			line := linkLineNumber(n, doc)
			*refs = append(*refs, RelationRef{
				RelationType: relationType,
				Target:       WikiLink{Target: extractSlugFromDestString(dest), Alias: label, Line: line},
				Source:       source,
				Line:         line,
				LinkStyle:    "regular",
			})
			return
		}
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walk(child)
		}
	}
	walk(node)
}

func linkLineNumber(link *ast.Link, doc *parsedDocument) int {
	for child := link.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			return doc.lineForOffset(text.Segment.Start)
		}
	}
	return 0
}

func collectLinkLabelText(link *ast.Link, source []byte) string {
	var buf strings.Builder
	for child := link.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			buf.Write(text.Segment.Value(source))
		}
	}
	return buf.String()
}

func parseRelationTypeLine(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func isExternalOrAnchorString(s string) bool {
	for _, scheme := range []string{"http://", "https://", "ftp://", "mailto:", "tel:"} {
		if strings.HasPrefix(s, scheme) {
			return true
		}
	}
	return len(s) > 0 && s[0] == '#'
}

func extractSlugFromDestString(s string) string {
	s = strings.TrimRight(s, "/")
	if idx := strings.LastIndexByte(s, '/'); idx >= 0 {
		s = s[idx+1:]
	}
	return strings.TrimSuffix(s, ".md")
}
