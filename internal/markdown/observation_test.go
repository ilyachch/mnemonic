package markdown

import "testing"

func TestParseObservationsExtractsCategoryContentTagsAndLineNumbers(t *testing.T) {
	t.Parallel()

	input := []byte("Intro #outside\n" +
		"- [decision] Roll out #backend #release\n" +
		"- [note] Trim me #Tag_2\n")

	observations := ParseObservations(input)
	if got, want := len(observations), 2; got != want {
		t.Fatalf("len(observations) = %d, want %d", got, want)
	}

	if observations[0].Category != "decision" {
		t.Fatalf("observations[0].Category = %q, want %q", observations[0].Category, "decision")
	}
	if observations[0].Content != "Roll out #backend #release" {
		t.Fatalf("observations[0].Content = %q", observations[0].Content)
	}
	if observations[0].LineStart != 2 || observations[0].LineEnd != 2 {
		t.Fatalf("observations[0].LineStart/LineEnd = %d/%d, want 2/2", observations[0].LineStart, observations[0].LineEnd)
	}
	if got, want := observations[0].Tags, []Tag{
		{Value: "backend", Line: 2},
		{Value: "release", Line: 2},
	}; len(got) != len(want) {
		t.Fatalf("observations[0].Tags = %#v, want %#v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("observations[0].Tags = %#v, want %#v", got, want)
			}
		}
	}

	if observations[1].Category != "note" {
		t.Fatalf("observations[1].Category = %q, want %q", observations[1].Category, "note")
	}
	if observations[1].Content != "Trim me #Tag_2" {
		t.Fatalf("observations[1].Content = %q", observations[1].Content)
	}
	if observations[1].LineStart != 3 || observations[1].LineEnd != 3 {
		t.Fatalf("observations[1].LineStart/LineEnd = %d/%d, want 3/3", observations[1].LineStart, observations[1].LineEnd)
	}
	if got, want := observations[1].Tags, []Tag{{Value: "Tag_2", Line: 3}}; len(got) != len(want) {
		t.Fatalf("observations[1].Tags = %#v, want %#v", got, want)
	} else if got[0] != want[0] {
		t.Fatalf("observations[1].Tags = %#v, want %#v", got, want)
	}
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
	if got, want := len(observations), 2; got != want {
		t.Fatalf("len(observations) = %d, want %d", got, want)
	}

	if observations[0].Category != "decision" {
		t.Fatalf("observations[0].Category = %q", observations[0].Category)
	}
	if observations[0].Content != "Keep #one" {
		t.Fatalf("observations[0].Content = %q", observations[0].Content)
	}
	if got, want := observations[0].Tags, []Tag{{Value: "one", Line: 3}}; len(got) != len(want) {
		t.Fatalf("observations[0].Tags = %#v, want %#v", got, want)
	} else if got[0] != want[0] {
		t.Fatalf("observations[0].Tags = %#v, want %#v", got, want)
	}

	if observations[1].Category != "note" {
		t.Fatalf("observations[1].Category = %q", observations[1].Category)
	}
	if observations[1].Content != "Final" {
		t.Fatalf("observations[1].Content = %q", observations[1].Content)
	}
}

func TestParseObservationsSkipsInlineCodeInTags(t *testing.T) {
	t.Parallel()

	input := []byte("- [decision] Roll out `#ignored` and #release\n")

	observations := ParseObservations(input)
	if got, want := len(observations), 1; got != want {
		t.Fatalf("len(observations) = %d, want %d", got, want)
	}

	if observations[0].Content != "Roll out `#ignored` and #release" {
		t.Fatalf("observations[0].Content = %q", observations[0].Content)
	}
	if got, want := observations[0].Tags, []Tag{{Value: "release", Line: 1}}; len(got) != len(want) {
		t.Fatalf("observations[0].Tags = %#v, want %#v", got, want)
	} else if got[0] != want[0] {
		t.Fatalf("observations[0].Tags = %#v, want %#v", got, want)
	}
}
