package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRelationsExtractsRelationTypesWikiLinksAndSources(t *testing.T) {
	t.Parallel()

	input := []byte("Intro [[General Note]]\n\n" +
		"## Relations\n" +
		"- depends_on `[[Ignored]]` and [[Session storage redesign]]\n" +
		"```go\n" +
		"- relates_to [[Ignored inside fence]]\n" +
		"```\n" +
		"- relates_to [[Rollout checklist]]\n" +
		"See also [[Section Note]]\n" +
		"> quoted [[Quoted Relation]]\n\n" +
		"## Other\n" +
		"Outside [[Post Note]]\n")

	refs := ParseRelations(input)
	require.Len(t, refs, 6)

	assertRelationRef := func(idx int, want RelationRef) {
		t.Helper()
		require.Equal(t, want, refs[idx])
	}

	assertRelationRef(0, RelationRef{
		Target:    WikiLink{Target: "General Note", Line: 1},
		Source:    RelationSourceWikiLink,
		Line:      1,
		LinkStyle: "wiki",
	})
	assertRelationRef(1, RelationRef{
		RelationType: "depends_on",
		Target:       WikiLink{Target: "Session storage redesign", Line: 4},
		Source:       RelationSourceRelationsSection,
		Line:         4,
		LinkStyle:    "wiki",
	})
	assertRelationRef(2, RelationRef{
		RelationType: "relates_to",
		Target:       WikiLink{Target: "Rollout checklist", Line: 8},
		Source:       RelationSourceRelationsSection,
		Line:         8,
		LinkStyle:    "wiki",
	})
	assertRelationRef(3, RelationRef{
		Target:    WikiLink{Target: "Section Note", Line: 9},
		Source:    RelationSourceRelationsSection,
		Line:      9,
		LinkStyle: "wiki",
	})
	assertRelationRef(4, RelationRef{
		Target:    WikiLink{Target: "Quoted Relation", Line: 10},
		Source:    RelationSourceRelationsSection,
		Line:      10,
		LinkStyle: "wiki",
	})
	assertRelationRef(5, RelationRef{
		Target:    WikiLink{Target: "Post Note", Line: 13},
		Source:    RelationSourceWikiLink,
		Line:      13,
		LinkStyle: "wiki",
	})
}

func TestParseRelationsHandlesEmptyInput(t *testing.T) {
	t.Parallel()

	refs := ParseRelations(nil)
	require.Empty(t, refs)
}

func TestParseRelationsExtractsStandardLinks(t *testing.T) {
	t.Parallel()

	input := []byte("Text [Label](target-slug.md) and [[Wiki Target]]\n" +
		"[Details](../sub/notes-slug.md)\n" +
		"[NoExtension](path/to/note)\n")

	refs := ParseRelations(input)
	require.Len(t, refs, 4)

	assertRelationRef := func(idx int, want RelationRef) {
		t.Helper()
		require.Equal(t, want, refs[idx])
	}

	assertRelationRef(0, RelationRef{
		Target:    WikiLink{Target: "Wiki Target", Line: 1},
		Source:    RelationSourceWikiLink,
		Line:      1,
		LinkStyle: "wiki",
	})
	assertRelationRef(1, RelationRef{
		Target:    WikiLink{Target: "target-slug", Alias: "Label", Line: 1},
		Source:    RelationSourceWikiLink,
		Line:      1,
		LinkStyle: "regular",
	})
	assertRelationRef(2, RelationRef{
		Target:    WikiLink{Target: "notes-slug", Alias: "Details", Line: 2},
		Source:    RelationSourceWikiLink,
		Line:      2,
		LinkStyle: "regular",
	})
	assertRelationRef(3, RelationRef{
		Target:    WikiLink{Target: "note", Alias: "NoExtension", Line: 3},
		Source:    RelationSourceWikiLink,
		Line:      3,
		LinkStyle: "regular",
	})
}

func TestParseRelationsIgnoresExternalAndAnchorLinks(t *testing.T) {
	t.Parallel()

	input := []byte("[Website](https://example.com)\n" +
		"[Mail](mailto:foo@bar.com)\n" +
		"[FTP](ftp://files.com)\n" +
		"[Tel](tel:+1234567890)\n" +
		"[Top](#header)\n" +
		"[Valid](keep-me.md)\n")

	refs := ParseRelations(input)
	require.Len(t, refs, 1)

	require.Equal(t, RelationRef{
		Target:    WikiLink{Target: "keep-me", Alias: "Valid", Line: 6},
		Source:    RelationSourceWikiLink,
		Line:      6,
		LinkStyle: "regular",
	}, refs[0])
}

func TestParseRelationsStandardLinksInRelationsSection(t *testing.T) {
	t.Parallel()

	input := []byte("## Relations\n" +
		"- depends_on [Related Note](related-note.md)\n")

	refs := ParseRelations(input)
	require.Len(t, refs, 1)

	require.Equal(t, RelationRef{
		RelationType: "depends_on",
		Target:       WikiLink{Target: "related-note", Alias: "Related Note", Line: 2},
		Source:       RelationSourceRelationsSection,
		Line:         2,
		LinkStyle:    "regular",
	}, refs[0])
}
