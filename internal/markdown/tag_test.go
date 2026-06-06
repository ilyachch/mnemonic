package markdown

import "testing"

func TestParseTagsExtractsTagsAndLineNumbers(t *testing.T) {
	t.Parallel()

	input := []byte("Intro #django and #backend-api\n" +
		"See https://example.com/#fragment and [docs](https://example.com/path#section) #release\n" +
		"```go\n" +
		"#inside-code and #also-inside\n" +
		"```\n" +
		"Tail #final_tag\n")

	tags := ParseTags(input)
	if got, want := len(tags), 4; got != want {
		t.Fatalf("len(tags) = %d, want %d", got, want)
	}

	want := []Tag{
		{Value: "django", Line: 1},
		{Value: "backend-api", Line: 1},
		{Value: "release", Line: 2},
		{Value: "final_tag", Line: 6},
	}

	for i := range want {
		if tags[i] != want[i] {
			t.Fatalf("tags[%d] = %#v, want %#v", i, tags[i], want[i])
		}
	}
}

func TestParseTagsIgnoresEscapedMalformedAndEmptyInput(t *testing.T) {
	t.Parallel()

	input := []byte("\\#escaped\n" +
		"Trailing #\n" +
		"Adjacentfoo#bar\n" +
		"prefix #123tag and #Tag_2\n")

	tags := ParseTags(input)
	if got, want := len(tags), 2; got != want {
		t.Fatalf("len(tags) = %d, want %d", got, want)
	}

	if tags[0] != (Tag{Value: "123tag", Line: 4}) {
		t.Fatalf("tags[0] = %#v, want numeric-start tag on line 4", tags[0])
	}
	if tags[1] != (Tag{Value: "Tag_2", Line: 4}) {
		t.Fatalf("tags[1] = %#v, want mixed-case tag on line 4", tags[1])
	}

	if tags := ParseTags(nil); len(tags) != 0 {
		t.Fatalf("ParseTags(nil) = %#v, want empty", tags)
	}
}

func TestParseTagsSkipsInlineCodeAndKeepsBlockquotes(t *testing.T) {
	t.Parallel()

	input := []byte("Inline `#ignored` and #plain\n" +
		"> quoted #visible\n")

	tags := ParseTags(input)
	if got, want := len(tags), 2; got != want {
		t.Fatalf("len(tags) = %d, want %d", got, want)
	}

	if tags[0] != (Tag{Value: "plain", Line: 1}) {
		t.Fatalf("tags[0] = %#v, want plain tag on line 1", tags[0])
	}
	if tags[1] != (Tag{Value: "visible", Line: 2}) {
		t.Fatalf("tags[1] = %#v, want quote tag on line 2", tags[1])
	}
}
