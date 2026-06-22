package markdown

import "bytes"

// Tag describes a raw inline tag extracted from markdown text.
type Tag struct {
	Value string
	Line  int
}

// ParseTags extracts inline tags from markdown text.
//
// Tags are raw hashtag tokens such as `#django` and `#backend-api`. The parser
// ignores escaped hashes, URL fragments, and fenced code blocks.
func ParseTags(data []byte) []Tag {
	doc := parseDocument(data)
	if doc == nil {
		return nil
	}

	var tags []Tag
	for _, line := range collectVisibleLines(doc.root, doc.source, doc.lineForOffset) {
		tags = append(tags, parseTagsInLine([]byte(line.Text), line.Line)...)
	}

	return tags
}

func parseTagsInLine(line []byte, lineNumber int) []Tag {
	var tags []Tag

	for i := 0; i < len(line); i++ {
		if line[i] != '#' || isEscapedTagStart(line, i) || !isTagBoundary(line, i) || isURLFragment(line, i) {
			continue
		}

		end := i + 1
		if end >= len(line) || !isTagNameStart(line[end]) {
			continue
		}
		end++
		for end < len(line) && isTagNameChar(line[end]) {
			end++
		}

		tags = append(tags, Tag{
			Value: string(line[i+1 : end]),
			Line:  lineNumber,
		})
		i = end - 1
	}

	return tags
}

func isEscapedTagStart(line []byte, pos int) bool {
	backslashes := 0
	for i := pos - 1; i >= 0 && line[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func isTagBoundary(line []byte, pos int) bool {
	if pos == 0 {
		return true
	}

	prev := line[pos-1]
	return !isTagNameChar(prev)
}

func isURLFragment(line []byte, pos int) bool {
	start := pos - 1
	for start >= 0 && line[start] != ' ' && line[start] != '\t' {
		start--
	}
	start++
	if start >= pos {
		return false
	}

	return bytes.Contains(line[start:pos], []byte("://"))
}

func isTagNameStart(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

func isTagNameChar(ch byte) bool {
	return isTagNameStart(ch) || ch == '-' || ch == '_'
}
