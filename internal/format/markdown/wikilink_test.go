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

func TestFormatLink(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		targetSlug string
		label      string
		style      string
		want       string
	}{
		{name: "wiki with label", targetSlug: "target-slug", label: "Label", style: "wiki", want: "[[target-slug|Label]]"},
		{name: "wiki without label", targetSlug: "target-slug", label: "", style: "wiki", want: "[[target-slug]]"},
		{name: "regular with label", targetSlug: "target-slug", label: "Label", style: "regular", want: "[Label](target-slug.md)"},
		{name: "regular without label", targetSlug: "target-slug", label: "", style: "regular", want: "[](target-slug.md)"},
		{name: "unknown style", targetSlug: "x", label: "y", style: "unknown", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatLink(tt.targetSlug, tt.label, tt.style)
			require.Equal(t, tt.want, got)
		})
	}
}
