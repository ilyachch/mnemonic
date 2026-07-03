package markdown

import (
	"bytes"
)

// WikiLink describes a raw wiki-link reference extracted from markdown text.
type WikiLink struct {
	Target string
	Alias  string
	Line   int
}

// ParseWikiLinks extracts wiki-links from markdown text.
//
// The parser returns raw targets, optional aliases, and 1-based line numbers.
// It skips malformed or escaped openings without panicking.
func ParseWikiLinks(data []byte) []WikiLink {
	doc := parseDocument(data)
	if doc == nil {
		return nil
	}

	var links []WikiLink
	for _, line := range collectVisibleLines(doc.root, doc.source, doc.lineForOffset) {
		links = append(links, parseWikiLinksInLine([]byte(line.Text), line.Line)...)
	}

	return links
}

func parseWikiLinksInLine(line []byte, lineNumber int) []WikiLink {
	var links []WikiLink

	for i := 0; i < len(line)-1; {
		if line[i] != '[' || line[i+1] != '[' || isEscapedWikiLinkStart(line, i) {
			i++
			continue
		}

		closing := bytes.Index(line[i+2:], []byte("]]"))
		if closing < 0 {
			i += 2
			continue
		}

		link, ok := parseWikiLinkContent(line[i+2:i+2+closing], lineNumber)
		if ok {
			links = append(links, link)
		}

		i += closing + 4
	}

	return links
}

func isEscapedWikiLinkStart(line []byte, pos int) bool {
	return isEscapedChar(line, pos)
}

func isEscapedChar(line []byte, pos int) bool {
	backslashes := 0
	for i := pos - 1; i >= 0 && line[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func parseWikiLinkContent(content []byte, lineNumber int) (WikiLink, bool) {
	target := content
	alias := []byte(nil)

	if separator := bytes.IndexByte(content, '|'); separator >= 0 {
		target = content[:separator]
		alias = content[separator+1:]
	}

	target = bytes.TrimSpace(target)
	if len(target) == 0 {
		return WikiLink{}, false
	}

	link := WikiLink{
		Target: string(target),
		Line:   lineNumber,
	}

	alias = bytes.TrimSpace(alias)
	if len(alias) > 0 {
		link.Alias = string(alias)
	}

	return link, true
}

// FormatLink renders a slug-based link in the given style.
//
//   - "wiki": [[target-slug|Label]] or [[target-slug]]
//   - "regular": [Label](target-slug.md) or [](target-slug.md)
func FormatLink(targetSlug, label, style string) string {
	switch style {
	case "wiki":
		if label == "" {
			return "[[" + targetSlug + "]]"
		}
		return "[[" + targetSlug + "|" + label + "]]"
	case "regular":
		if label == "" {
			return "[](" + targetSlug + ".md)"
		}
		return "[" + label + "](" + targetSlug + ".md)"
	default:
		return ""
	}
}
