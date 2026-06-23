package markdown

import (
	"bytes"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var markdownParser = goldmark.New()

type parsedDocument struct {
	source    []byte
	lineStart []int
	root      ast.Node
}

type visibleLine struct {
	Text string
	Line int
}

func parseDocument(data []byte) *parsedDocument {
	if len(data) == 0 {
		return nil
	}

	return &parsedDocument{
		source:    data,
		lineStart: buildLineStarts(data),
		root:      markdownParser.Parser().Parse(text.NewReader(data)),
	}
}

func buildLineStarts(data []byte) []int {
	starts := []int{0}
	for i, b := range data {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func (d *parsedDocument) lineForOffset(offset int) int {
	if offset < 0 {
		return 1
	}

	idx := sort.Search(len(d.lineStart), func(i int) bool {
		return d.lineStart[i] > offset
	})
	if idx == 0 {
		return 1
	}
	return idx
}

func (d *parsedDocument) lineBytes(line int) []byte {
	if line <= 0 || line > len(d.lineStart) {
		return nil
	}

	start := d.lineStart[line-1]
	end := len(d.source)
	if line < len(d.lineStart) {
		end = d.lineStart[line] - 1
	}

	return bytes.TrimRight(d.source[start:end], "\r\n")
}

func collectVisibleLines(node ast.Node, source []byte, lineForOffset func(int) int) []visibleLine {
	acc := newLineAccumulator(source, lineForOffset)
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		switch n := n.(type) {
		case *ast.CodeBlock, *ast.FencedCodeBlock, *ast.CodeSpan:
			return
		case *ast.Text:
			acc.appendText(n)
			return
		}
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walk(child)
			if isBlockBoundary(child) {
				acc.flush()
			}
		}
	}
	walk(node)
	acc.flush()
	return acc.lines
}

type lineAccumulator struct {
	source        []byte
	lineForOffset func(int) int
	lines         []visibleLine
	buf           strings.Builder
	currentLine   int
}

func newLineAccumulator(source []byte, lineForOffset func(int) int) *lineAccumulator {
	return &lineAccumulator{source: source, lineForOffset: lineForOffset}
}

func (a *lineAccumulator) appendText(n *ast.Text) {
	raw := n.Segment.Value(a.source)
	if len(raw) == 0 {
		return
	}
	offset := n.Segment.Start
	for len(raw) > 0 {
		if a.currentLine == 0 {
			a.currentLine = a.lineForOffset(offset)
		}
		newline := bytes.IndexByte(raw, '\n')
		if newline < 0 {
			a.buf.Write(raw)
			if n.SoftLineBreak() || n.HardLineBreak() {
				a.flush()
			}
			return
		}
		a.buf.Write(raw[:newline])
		a.flush()
		offset += newline + 1
		raw = raw[newline+1:]
	}
}

func (a *lineAccumulator) flush() {
	if a.buf.Len() == 0 {
		a.currentLine = 0
		return
	}
	a.lines = append(a.lines, visibleLine{
		Text: a.buf.String(),
		Line: a.currentLine,
	})
	a.buf.Reset()
	a.currentLine = 0
}

func isBlockBoundary(node ast.Node) bool {
	switch node.(type) {
	case *ast.Document, *ast.Blockquote, *ast.Heading, *ast.List, *ast.ListItem, *ast.Paragraph, *ast.TextBlock, *ast.FencedCodeBlock, *ast.CodeBlock:
		return true
	default:
		return false
	}
}
