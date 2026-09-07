package cli

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

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
	// Dynamic timestamp: hardcoded values would eventually fall outside
	// relative time-filter windows (e.g. --created-since 1000h).
	now := time.Now().UTC().Unix()
	content += "created_at: " + strconv.FormatInt(now, 10) + "\n"
	content += "updated_at: " + strconv.FormatInt(now, 10) + "\n"
	content += "---\n"
	content += body

	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
}
