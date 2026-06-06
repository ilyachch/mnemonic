package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBasicMemoryFixturesParse(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(filepath.Join("testdata", "basic-memory", "*.md"))
	if err != nil {
		t.Fatalf("filepath.Glob() error = %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no Basic Memory fixtures found")
	}

	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("os.ReadFile(%q) error = %v", path, err)
			}

			note, err := ParseNote(data)
			if err != nil {
				t.Fatalf("ParseNote(%q) error = %v", path, err)
			}
			if len(note.Body) == 0 {
				t.Fatalf("ParseNote(%q) returned empty body", path)
			}

			switch {
			case strings.Contains(path, "permalink"):
				if note.Permalink == "" {
					t.Fatalf("ParseNote(%q) permalink = empty", path)
				}
				if note.Slug != note.Permalink {
					t.Fatalf("ParseNote(%q) Slug = %q, want permalink fallback %q", path, note.Slug, note.Permalink)
				}
			case strings.Contains(path, "observations"):
				observations := ParseObservations(note.Body)
				if len(observations) == 0 {
					t.Fatalf("ParseObservations(%q) returned no observations", path)
				}
				if len(observations[0].Tags) == 0 {
					t.Fatalf("ParseObservations(%q) did not extract inline tags from observations", path)
				}
			case strings.Contains(path, "relations"):
				refs := ParseRelations(note.Body)
				if len(refs) == 0 {
					t.Fatalf("ParseRelations(%q) returned no relations", path)
				}
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
				if !hasRelationsSection {
					t.Fatalf("ParseRelations(%q) did not mark relations-section links", path)
				}
				if !hasWikiLink {
					t.Fatalf("ParseRelations(%q) did not preserve plain wiki links", path)
				}
			}
		})
	}
}
