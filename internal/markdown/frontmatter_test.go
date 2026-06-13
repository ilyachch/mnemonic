package markdown

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	wantMetadata := []byte("title: Example\r\ncount: 1\r\n")
	require.True(t, bytes.Equal(parsed.Metadata, wantMetadata))

	wantBody := []byte("# Heading\r\nBody\r\n")
	require.True(t, bytes.Equal(parsed.Body, wantBody))
}

func TestParseFrontmatterWithoutFrontmatter(t *testing.T) {
	t.Parallel()

	input := []byte("# Heading\n---\nnot frontmatter\n")

	parsed, err := ParseFrontmatter(input)
	require.NoError(t, err)

	require.Empty(t, parsed.Metadata)
	require.True(t, bytes.Equal(parsed.Body, input))
}

func TestParseFrontmatterIgnoresLeadingBlankLines(t *testing.T) {
	t.Parallel()

	input := []byte("\n---\n" +
		"title: Example\n" +
		"---\n" +
		"Body\n")

	parsed, err := ParseFrontmatter(input)
	require.NoError(t, err)

	require.Empty(t, parsed.Metadata)
	require.True(t, bytes.Equal(parsed.Body, input))
}

func TestParseFrontmatterRejectsUnclosedFrontmatter(t *testing.T) {
	t.Parallel()

	_, err := ParseFrontmatter([]byte("---\n" +
		"title: Example\n" +
		"# Heading\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unclosed frontmatter")
}
