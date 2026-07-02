package markdown

import (
	"testing"
)

func TestExtractH1TitleReturnsFirstH1(t *testing.T) {
	body := []byte("# My Note Title\n\nSome body text.\n")
	got := ExtractH1Title(body)
	want := "My Note Title"
	if got != want {
		t.Fatalf("ExtractH1Title = %q, want %q", got, want)
	}
}

func TestExtractH1TitleReturnsEmptyWhenNoH1(t *testing.T) {
	body := []byte("## Subheading\n\nNo level-1 heading here.\n")
	got := ExtractH1Title(body)
	if got != "" {
		t.Fatalf("ExtractH1Title = %q, want empty", got)
	}
}

func TestExtractH1TitleReturnsEmptyForEmptyBody(t *testing.T) {
	got := ExtractH1Title(nil)
	if got != "" {
		t.Fatalf("ExtractH1Title(nil) = %q, want empty", got)
	}
}

func TestExtractH1TitleFlattensInlineMarkup(t *testing.T) {
	body := []byte("# Hello *world* and `code`\n\nBody.\n")
	got := ExtractH1Title(body)
	want := "Hello world and code"
	if got != want {
		t.Fatalf("ExtractH1Title = %q, want %q", got, want)
	}
}

func TestExtractH1TitleFlattensLinks(t *testing.T) {
	body := []byte("# [Project Notes](https://example.com)\n\nBody.\n")
	got := ExtractH1Title(body)
	want := "Project Notes"
	if got != want {
		t.Fatalf("ExtractH1Title = %q, want %q", got, want)
	}
}

func TestExtractH1TitleReturnsFirstWhenMultipleH1(t *testing.T) {
	body := []byte("# First Title\n\n# Second Title\n")
	got := ExtractH1Title(body)
	want := "First Title"
	if got != want {
		t.Fatalf("ExtractH1Title = %q, want %q", got, want)
	}
}

func TestExtractH1TitleIgnoresH1InsideCodeBlock(t *testing.T) {
	body := []byte("```\n# Not a heading\n```\n\n# Real Heading\n")
	got := ExtractH1Title(body)
	want := "Real Heading"
	if got != want {
		t.Fatalf("ExtractH1Title = %q, want %q", got, want)
	}
}

func TestExtractH1TitleTrimsWhitespace(t *testing.T) {
	body := []byte("#   Spaced   Title   \n\nBody.\n")
	got := ExtractH1Title(body)
	want := "Spaced   Title"
	if got != want {
		t.Fatalf("ExtractH1Title = %q, want %q", got, want)
	}
}
