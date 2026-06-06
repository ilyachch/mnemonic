package markdown

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseFrontmatterWithBodyPreserved(t *testing.T) {
	t.Parallel()

	input := []byte("---\r\n" +
		"title: Example\r\n" +
		"count: 1\r\n" +
		"---\r\n" +
		"# Heading\r\n" +
		"Body\r\n")

	parsed, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}

	wantMetadata := []byte("title: Example\r\ncount: 1\r\n")
	if !bytes.Equal(parsed.Metadata, wantMetadata) {
		t.Fatalf("Metadata = %q, want %q", parsed.Metadata, wantMetadata)
	}

	wantBody := []byte("# Heading\r\nBody\r\n")
	if !bytes.Equal(parsed.Body, wantBody) {
		t.Fatalf("Body = %q, want %q", parsed.Body, wantBody)
	}
}

func TestParseFrontmatterWithoutFrontmatter(t *testing.T) {
	t.Parallel()

	input := []byte("# Heading\n---\nnot frontmatter\n")

	parsed, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}

	if len(parsed.Metadata) != 0 {
		t.Fatalf("Metadata = %q, want empty", parsed.Metadata)
	}
	if !bytes.Equal(parsed.Body, input) {
		t.Fatalf("Body = %q, want %q", parsed.Body, input)
	}
}

func TestParseFrontmatterIgnoresLeadingBlankLines(t *testing.T) {
	t.Parallel()

	input := []byte("\n---\n" +
		"title: Example\n" +
		"---\n" +
		"Body\n")

	parsed, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}

	if len(parsed.Metadata) != 0 {
		t.Fatalf("Metadata = %q, want empty", parsed.Metadata)
	}
	if !bytes.Equal(parsed.Body, input) {
		t.Fatalf("Body = %q, want %q", parsed.Body, input)
	}
}

func TestParseFrontmatterRejectsUnclosedFrontmatter(t *testing.T) {
	t.Parallel()

	_, err := ParseFrontmatter([]byte("---\n" +
		"title: Example\n" +
		"# Heading\n"))
	if err == nil {
		t.Fatal("ParseFrontmatter() error = nil, want unclosed frontmatter error")
	}
	if !strings.Contains(err.Error(), "unclosed frontmatter") {
		t.Fatalf("ParseFrontmatter() error = %q, want unclosed frontmatter error", err)
	}
}
