package markdown

import "testing"

func TestParseWikiLinksExtractsTargetAliasAndLineNumber(t *testing.T) {
	t.Parallel()

	input := []byte("Intro [[Some Note]]\n" +
		"Refs [[some-note]] and [[folder/Some Note|Display Text]]\n" +
		"Tail [[Some Note|Display Text]]\n")

	links := ParseWikiLinks(input)
	if got, want := len(links), 4; got != want {
		t.Fatalf("len(links) = %d, want %d", got, want)
	}

	assertWikiLink := func(idx int, want WikiLink) {
		t.Helper()

		got := links[idx]
		if got != want {
			t.Fatalf("links[%d] = %#v, want %#v", idx, got, want)
		}
	}

	assertWikiLink(0, WikiLink{Target: "Some Note", Line: 1})
	assertWikiLink(1, WikiLink{Target: "some-note", Line: 2})
	assertWikiLink(2, WikiLink{Target: "folder/Some Note", Alias: "Display Text", Line: 2})
	assertWikiLink(3, WikiLink{Target: "Some Note", Alias: "Display Text", Line: 3})
}

func TestParseWikiLinksIgnoresEscapedAndMalformedLinks(t *testing.T) {
	t.Parallel()

	input := []byte("\\[[Escaped Note]]\n" +
		"[[Unclosed Note\n" +
		"[[ ]] and [[Valid Note]]\n" +
		"prefix \\\\[[Still Valid]]\n")

	links := ParseWikiLinks(input)
	if got, want := len(links), 2; got != want {
		t.Fatalf("len(links) = %d, want %d", got, want)
	}

	if links[0] != (WikiLink{Target: "Valid Note", Line: 3}) {
		t.Fatalf("links[0] = %#v, want valid link on line 3", links[0])
	}
	if links[1] != (WikiLink{Target: "Still Valid", Line: 4}) {
		t.Fatalf("links[1] = %#v, want valid link on line 4", links[1])
	}
}

func TestParseWikiLinksSkipsInlineCodeAndKeepsBlockquotes(t *testing.T) {
	t.Parallel()

	input := []byte("Inline `[[Ignored Note]]` and plain [[Visible Note]]\n" +
		"> quoted [[Quoted Note|Alias]]\n")

	links := ParseWikiLinks(input)
	if got, want := len(links), 2; got != want {
		t.Fatalf("len(links) = %d, want %d", got, want)
	}

	if links[0] != (WikiLink{Target: "Visible Note", Line: 1}) {
		t.Fatalf("links[0] = %#v, want visible note on line 1", links[0])
	}
	if links[1] != (WikiLink{Target: "Quoted Note", Alias: "Alias", Line: 2}) {
		t.Fatalf("links[1] = %#v, want quoted note on line 2", links[1])
	}
}

func TestParseWikiLinksHandlesEmptyInput(t *testing.T) {
	t.Parallel()

	if links := ParseWikiLinks(nil); len(links) != 0 {
		t.Fatalf("ParseWikiLinks(nil) = %#v, want empty", links)
	}
}
