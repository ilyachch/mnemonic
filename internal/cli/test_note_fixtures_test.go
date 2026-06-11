package cli

import (
	"os"
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
		for _, tag := range tags {
			content += "  - " + tag + "\n"
		}
	}
	content += "created_at: 2026-06-02T12:34:56Z\n"
	content += "updated_at: 2026-06-02T12:34:56Z\n"
	content += "---\n"
	content += body

	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
}