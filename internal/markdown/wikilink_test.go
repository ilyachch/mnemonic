package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWikiLinksExtractsTargetAliasAndLineNumber(t *testing.T) {
	t.Parallel()

	input := []byte("Intro [[Some Note]]\n" +
		"Refs [[some-note]] and [[folder/Some Note|Display Text]]\n" +
		"Tail [[Some Note|Display Text]]\n")

	links := ParseWikiLinks(input)
	require.Len(t, links, 4)

	assertWikiLink := func(idx int, want WikiLink) {
		t.Helper()
		require.Equal(t, want, links[idx])
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
	require.Len(t, links, 2)

	require.Equal(t, WikiLink{Target: "Valid Note", Line: 3}, links[0])
	require.Equal(t, WikiLink{Target: "Still Valid", Line: 4}, links[1])
}

func TestParseWikiLinksSkipsInlineCodeAndKeepsBlockquotes(t *testing.T) {
	t.Parallel()

	input := []byte("Inline `[[Ignored Note]]` and plain [[Visible Note]]\n" +
		"> quoted [[Quoted Note|Alias]]\n")

	links := ParseWikiLinks(input)
	require.Len(t, links, 2)

	require.Equal(t, WikiLink{Target: "Visible Note", Line: 1}, links[0])
	require.Equal(t, WikiLink{Target: "Quoted Note", Alias: "Alias", Line: 2}, links[1])
}

func TestParseWikiLinksHandlesEmptyInput(t *testing.T) {
	t.Parallel()

	links := ParseWikiLinks(nil)
	require.Empty(t, links)
}
