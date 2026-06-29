package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// ExtractH1Title returns the text content of the first level-1 heading found
// in the supplied markdown body. Returns an empty string when no level-1
// heading is present.
//
// Inline markup (emphasis, code spans, links) is flattened to its textual
// content, mirroring the behaviour used by collectVisibleLines.
func ExtractH1Title(body []byte) string {
	doc := parseDocument(body)
	if doc == nil {
		return ""
	}

	var title string
	_ = ast.Walk(doc.root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if heading, ok := n.(*ast.Heading); ok {
				if heading.Level == 1 {
					title = collectHeadingText(heading, doc.source)
					return ast.WalkStop, nil
				}
				return ast.WalkSkipChildren, nil
			}
		}
		return ast.WalkContinue, nil
	})
	return title
}

// collectHeadingText concatenates the textual content of a heading node,
// flattening inline markup such as emphasis, links, and code spans.
func collectHeadingText(heading ast.Node, source []byte) string {
	var buf strings.Builder
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if t, ok := child.(*ast.Text); ok {
				buf.Write(t.Segment.Value(source))
				if t.SoftLineBreak() || t.HardLineBreak() {
					buf.WriteByte(' ')
				}
				continue
			}
			walk(child)
		}
	}
	walk(heading)
	return strings.TrimSpace(buf.String())
}
