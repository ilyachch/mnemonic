package markdown

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExampleNoteParses(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "markdown", "example-note.md"))
	require.NoError(t, err)

	note, err := ParseNote(data)
	require.NoError(t, err)

	require.NotEmpty(t, note.Title)
	require.NotEmpty(t, note.Body)
}
