package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTagsExtractsTagsAndLineNumbers(t *testing.T) {
	t.Parallel()

	input := []byte("Intro #django and #backend-api\n" +
		"See https://example.com/#fragment and [docs](https://example.com/path#section) #release\n" +
		"```go\n" +
		"#inside-code and #also-inside\n" +
		"```\n" +
		"Tail #final_tag\n")

	tags := ParseTags(input)
	require.Len(t, tags, 4)

	want := []Tag{
		{Value: "django", Line: 1},
		{Value: "backend-api", Line: 1},
		{Value: "release", Line: 2},
		{Value: "final_tag", Line: 6},
	}

	for i := range want {
		require.Equal(t, want[i], tags[i])
	}
}

func TestParseTagsIgnoresEscapedMalformedAndEmptyInput(t *testing.T) {
	t.Parallel()

	input := []byte("\\#escaped\n" +
		"Trailing #\n" +
		"Adjacentfoo#bar\n" +
		"prefix #123tag and #Tag_2\n")

	tags := ParseTags(input)
	require.Len(t, tags, 2)

	require.Equal(t, Tag{Value: "123tag", Line: 4}, tags[0])
	require.Equal(t, Tag{Value: "Tag_2", Line: 4}, tags[1])

	require.Empty(t, ParseTags(nil))
}

func TestParseTagsSkipsInlineCodeAndKeepsBlockquotes(t *testing.T) {
	t.Parallel()

	input := []byte("Inline `#ignored` and #plain\n" +
		"> quoted #visible\n")

	tags := ParseTags(input)
	require.Len(t, tags, 2)

	require.Equal(t, Tag{Value: "plain", Line: 1}, tags[0])
	require.Equal(t, Tag{Value: "visible", Line: 2}, tags[1])
}