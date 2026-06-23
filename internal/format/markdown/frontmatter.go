package markdown

import (
	"bytes"
	"errors"
	"fmt"

	frontmatter "github.com/adrg/frontmatter"
)

// Frontmatter splits a markdown document into raw YAML frontmatter and body.
//
// Metadata is returned as the exact byte slice between the opening and closing
// frontmatter delimiters. Body is returned as the untouched remainder of the
// input after the closing delimiter.
type Frontmatter struct {
	Present  bool
	Metadata []byte
	Body     []byte
}

// ParseFrontmatter splits a markdown document into raw YAML frontmatter and body.
//
// If the document does not start with a frontmatter delimiter, the full input
// is returned as Body and Metadata is empty. If an opening delimiter is present
// but no closing delimiter is found, ParseFrontmatter returns an error.
func ParseFrontmatter(data []byte) (Frontmatter, error) {
	if len(data) == 0 {
		return Frontmatter{}, nil
	}

	if !hasFrontmatterPrefix(data) {
		return Frontmatter{Body: data}, nil
	}

	var metadata []byte
	body, err := frontmatter.Parse(bytes.NewReader(data), &metadata, frontmatter.NewFormat("---", "---", func(raw []byte, target interface{}) error {
		dst, ok := target.(*[]byte)
		if !ok {
			return fmt.Errorf("parse frontmatter: unexpected metadata target %T", target)
		}
		*dst = append((*dst)[:0], raw...)
		return nil
	}))
	if err != nil || bytes.Equal(body, data) {
		return Frontmatter{}, errors.New("unclosed frontmatter")
	}

	return Frontmatter{
		Present:  true,
		Metadata: metadata,
		Body:     body,
	}, nil
}

func hasFrontmatterPrefix(data []byte) bool {
	lineEnd := bytes.IndexByte(data, '\n')
	if lineEnd < 0 {
		return bytes.Equal(trimCarriageReturn(data), []byte("---"))
	}
	return bytes.Equal(trimCarriageReturn(data[:lineEnd]), []byte("---"))
}

func trimCarriageReturn(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\r' {
		return line[:len(line)-1]
	}
	return line
}
