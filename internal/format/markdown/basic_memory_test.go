package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicMemoryFixturesParse(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(filepath.Join("testdata", "basic-memory", "*.md"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(path)
			require.NoError(t, err)

			note, err := ParseNote(data)
			require.NoError(t, err)
			require.NotEmpty(t, note.Body)

			switch {
			case strings.Contains(path, "permalink"):
				require.NotEmpty(t, note.Permalink)
				require.Equal(t, note.Permalink, note.Slug)
			case strings.Contains(path, "observations"):
				observations := ParseObservations(note.Body)
				require.NotEmpty(t, observations)
				require.NotEmpty(t, observations[0].Tags)
			case strings.Contains(path, "relations"):
				refs := ParseRelations(note.Body)
				require.NotEmpty(t, refs)
				var hasRelationsSection bool
				var hasWikiLink bool
				for _, ref := range refs {
					switch ref.Source {
					case RelationSourceRelationsSection:
						hasRelationsSection = true
					case RelationSourceWikiLink:
						hasWikiLink = true
					}
				}
				require.True(t, hasRelationsSection)
				require.True(t, hasWikiLink)
			}
		})
	}
}
