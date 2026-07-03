package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTaggedNote(t *testing.T, path, noteID, title, slug string, tags []string, body string) {
	t.Helper()

	content := "---\n"
	content += "mnemonic_note_id: " + noteID + "\n"
	content += "title: " + title + "\n"
	content += "slug: " + slug + "\n"
	if len(tags) > 0 {
		content += "tags:\n"
		var contentSb19 strings.Builder
		for _, tag := range tags {
			contentSb19.WriteString("  - " + tag + "\n")
		}
		content += contentSb19.String()
	}
	content += "created_at: 1780403696\n"
	content += "updated_at: 1780403696\n"
	content += "---\n"
	content += body

	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
}
