package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseObservationsExtractsCategoryContentTagsAndLineNumbers(t *testing.T) {
	t.Parallel()

	input := []byte("Intro #outside\n" +
		"- [decision] Roll out #backend #release\n" +
		"- [note] Trim me #Tag_2\n")

	observations := ParseObservations(input)
	require.Len(t, observations, 2)

	require.Equal(t, "decision", observations[0].Category)
	require.Equal(t, "Roll out #backend #release", observations[0].Content)
	require.Equal(t, 2, observations[0].LineStart)
	require.Equal(t, 2, observations[0].LineEnd)
	require.Equal(t, []Tag{
		{Value: "backend", Line: 2},
		{Value: "release", Line: 2},
	}, observations[0].Tags)

	require.Equal(t, "note", observations[1].Category)
	require.Equal(t, "Trim me #Tag_2", observations[1].Content)
	require.Equal(t, 3, observations[1].LineStart)
	require.Equal(t, 3, observations[1].LineEnd)
	require.Equal(t, []Tag{{Value: "Tag_2", Line: 3}}, observations[1].Tags)
}

func TestParseObservationsIgnoresNonstandardBulletsAndFencedCode(t *testing.T) {
	t.Parallel()

	input := []byte("* [ignored] wrong bullet\n" +
		"- [missing]\n" +
		"- [decision]   Keep #one\n" +
		"```md\n" +
		"- [hidden] code #skip\n" +
		"```\n" +
		"- [note] Final\n")

	observations := ParseObservations(input)
	require.Len(t, observations, 2)

	require.Equal(t, "decision", observations[0].Category)
	require.Equal(t, "Keep #one", observations[0].Content)
	require.Equal(t, []Tag{{Value: "one", Line: 3}}, observations[0].Tags)

	require.Equal(t, "note", observations[1].Category)
	require.Equal(t, "Final", observations[1].Content)
}

func TestParseObservationsSkipsInlineCodeInTags(t *testing.T) {
	t.Parallel()

	input := []byte("- [decision] Roll out `#ignored` and #release\n")

	observations := ParseObservations(input)
	require.Len(t, observations, 1)

	require.Equal(t, "Roll out `#ignored` and #release", observations[0].Content)
	require.Equal(t, []Tag{{Value: "release", Line: 1}}, observations[0].Tags)
}
