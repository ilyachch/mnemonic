package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// Observation describes a raw markdown observation extracted from text.
type Observation struct {
	Category  string
	Content   string
	LineStart int
	LineEnd   int
	Tags      []Tag
}

// ParseObservations extracts markdown observations from text.
//
// Observations use the form `- [category] content`. The parser is intentionally
// forgiving: nonstandard bullets are skipped without error, and inline tags
// inside the observation content are extracted separately.
func ParseObservations(data []byte) []Observation {
	doc := parseDocument(data)
	if doc == nil {
		return nil
	}

	var observations []Observation
	for child := doc.root.FirstChild(); child != nil; child = child.NextSibling() {
		list, ok := child.(*ast.List)
		if !ok || list.IsOrdered() || list.Marker != '-' {
			continue
		}

		for item := list.FirstChild(); item != nil; item = item.NextSibling() {
			observation, ok := parseObservationListItem(item.(*ast.ListItem), doc)
			if ok {
				observations = append(observations, observation)
			}
		}
	}

	return observations
}

func parseObservationListItem(item *ast.ListItem, doc *parsedDocument) (Observation, bool) {
	firstBlock := item.FirstChild()
	if firstBlock == nil {
		return Observation{}, false
	}

	textBlock, ok := firstBlock.(*ast.TextBlock)
	if !ok {
		return Observation{}, false
	}
	if textBlock.Lines().Len() == 0 {
		return Observation{}, false
	}

	seg := textBlock.Lines().At(0)
	line := string(seg.Value(doc.source))
	firstLine := line
	if index := strings.IndexByte(firstLine, '\n'); index >= 0 {
		firstLine = firstLine[:index]
	}
	firstLine = strings.TrimSpace(firstLine)
	if len(firstLine) == 0 {
		return Observation{}, false
	}

	lineNumber := doc.lineForOffset(textBlock.Lines().At(0).Start)
	observation, ok := parseObservationVisibleLine(firstLine, lineNumber)
	if !ok {
		return Observation{}, false
	}

	lines := collectVisibleLines(textBlock, doc.source, doc.lineForOffset)
	if len(lines) > 0 {
		observation.Tags = parseTagsInLine([]byte(lines[0].Text), lines[0].Line)
	}

	return observation, true
}

func parseObservationVisibleLine(line string, lineNumber int) (Observation, bool) {
	if len(line) < 4 || line[0] != '[' {
		return Observation{}, false
	}

	closing := strings.IndexByte(line[1:], ']')
	if closing < 0 {
		return Observation{}, false
	}
	closing++

	category := strings.TrimSpace(line[1:closing])
	if len(category) == 0 {
		return Observation{}, false
	}
	if closing+1 >= len(line) || (line[closing+1] != ' ' && line[closing+1] != '\t') {
		return Observation{}, false
	}

	content := strings.TrimSpace(line[closing+1:])
	if len(content) == 0 {
		return Observation{}, false
	}

	return Observation{
		Category:  category,
		Content:   content,
		LineStart: lineNumber,
		LineEnd:   lineNumber,
	}, true
}
