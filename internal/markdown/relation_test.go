package markdown

import "testing"

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
	if got, want := len(refs), 6; got != want {
		t.Fatalf("len(refs) = %d, want %d", got, want)
	}

	assertRelationRef := func(idx int, want RelationRef) {
		t.Helper()

		got := refs[idx]
		if got != want {
			t.Fatalf("refs[%d] = %#v, want %#v", idx, got, want)
		}
	}

	assertRelationRef(0, RelationRef{
		Target: WikiLink{Target: "General Note", Line: 1},
		Source: RelationSourceWikiLink,
		Line:   1,
	})
	assertRelationRef(1, RelationRef{
		RelationType: "depends_on",
		Target:       WikiLink{Target: "Session storage redesign", Line: 4},
		Source:       RelationSourceRelationsSection,
		Line:         4,
	})
	assertRelationRef(2, RelationRef{
		RelationType: "relates_to",
		Target:       WikiLink{Target: "Rollout checklist", Line: 8},
		Source:       RelationSourceRelationsSection,
		Line:         8,
	})
	assertRelationRef(3, RelationRef{
		Target: WikiLink{Target: "Section Note", Line: 9},
		Source: RelationSourceRelationsSection,
		Line:   9,
	})
	assertRelationRef(4, RelationRef{
		Target: WikiLink{Target: "Quoted Relation", Line: 10},
		Source: RelationSourceRelationsSection,
		Line:   10,
	})
	assertRelationRef(5, RelationRef{
		Target: WikiLink{Target: "Post Note", Line: 13},
		Source: RelationSourceWikiLink,
		Line:   13,
	})
}

func TestParseRelationsHandlesEmptyInput(t *testing.T) {
	t.Parallel()

	if refs := ParseRelations(nil); len(refs) != 0 {
		t.Fatalf("ParseRelations(nil) = %#v, want empty", refs)
	}
}
