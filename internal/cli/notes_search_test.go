package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestNotesSearchCommandReturnsHits(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	if err := project.InitProject(project.InitInput{
		CWD:          projectRoot,
		MemoriesHome: filepath.Join(projectRoot, ".mnemonic-memories"),
		Name:         "personal",
		Mode:         project.InitModeLocal,
	}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if _, err := notes.Create(notes.CreateInput{
		RootDir: memoriesRoot,
		Title:   "Auth migration",
		Body:    []byte("Search this body.\nObservation queryterm.\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440001"
		},
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", filepath.Join(projectRoot, ".mnemonic-memories", "personal")); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	result := executeCommand("notes", "search", "auth", "--project", "personal", "--json")
	if result.Err != nil {
		t.Fatalf("notes search returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got struct {
		Hits []struct {
			NoteID      string  `json:"note_id"`
			Slug        string  `json:"slug"`
			Title       string  `json:"title"`
			Path        string  `json:"path"`
			Snippet     string  `json:"snippet"`
			Score       float64 `json:"score"`
			ContentHash string  `json:"content_hash"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, result.Stdout)
	}
	if len(got.Hits) == 0 {
		t.Fatal("hits are empty")
	}
	first := got.Hits[0]
	if first.NoteID == "" || first.Slug == "" || first.Title == "" || first.Path == "" || first.Snippet == "" || first.ContentHash == "" {
		t.Fatalf("unexpected empty hit: %#v", first)
	}
	if first.Slug != "auth-migration" {
		t.Fatalf("slug = %q", first.Slug)
	}
}

func TestNotesSearchCommandRespectsLimit(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	if err := project.InitProject(project.InitInput{
		CWD:          projectRoot,
		MemoriesHome: filepath.Join(projectRoot, ".mnemonic-memories"),
		Name:         "personal",
		Mode:         project.InitModeLocal,
	}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	for i := 0; i < 3; i++ {
		title := "Query Term " + string(rune('A'+i))
		uid := "550e8400-e29b-41d4-a716-44665544000" + string(rune('2'+i))
		if _, err := notes.Create(notes.CreateInput{
			RootDir: filepath.Join(projectRoot, ".mnemonic-memories", "personal"),
			Title:   title,
			Body:    []byte("queryterm queryterm\n"),
			UUID: func() string {
				return uid
			},
		}); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", filepath.Join(projectRoot, ".mnemonic-memories", "personal")); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	result := executeCommand("notes", "search", "queryterm", "--project", "personal", "--limit", "2", "--json")
	if result.Err != nil {
		t.Fatalf("notes search returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}
	var got struct {
		Hits []any `json:"hits"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, result.Stdout)
	}
	if len(got.Hits) != 2 {
		t.Fatalf("len(hits) = %d, want 2", len(got.Hits))
	}
}

func TestNotesSearchCommandFiltersByTag(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	if err := project.InitProject(project.InitInput{
		CWD:          projectRoot,
		MemoriesHome: filepath.Join(projectRoot, ".mnemonic-memories"),
		Name:         "personal",
		Mode:         project.InitModeLocal,
	}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "frontmatter-tag.md"), "550e8400-e29b-41d4-a716-446655440001", "Frontmatter tag", "frontmatter-tag", []string{"django"}, "auth queryterm\n")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "inline-tag.md"), "550e8400-e29b-41d4-a716-446655440002", "Inline tag", "inline-tag", nil, "auth queryterm #django\n")
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	result := executeCommand("notes", "search", "auth", "--project", "personal", "--tag", "django", "--json")
	if result.Err != nil {
		t.Fatalf("notes search returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}
	var got struct {
		Hits []struct {
			Slug string `json:"slug"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, result.Stdout)
	}
	if len(got.Hits) != 2 {
		t.Fatalf("len(hits) = %d, want 2", len(got.Hits))
	}
}

func TestNotesSearchCommandMissingIndexSuggestsReindex(t *testing.T) {
	projectRoot := testutil.CleanEnvForTest(t)

	restoreClock := project.SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	defer restoreClock()

	if err := project.InitProject(project.InitInput{
		CWD:          projectRoot,
		MemoriesHome: filepath.Join(projectRoot, ".mnemonic-memories"),
		Name:         "personal",
		Mode:         project.InitModeLocal,
	}); err != nil {
		t.Fatalf("InitProject() error = %v", err)
	}

	restoreWD := chdirForNotesTest(t, projectRoot)
	defer restoreWD()

	result := executeCommand("notes", "search", "auth", "--project", "personal", "--json")
	if result.Err == nil {
		t.Fatal("notes search error = nil, want missing index error")
	}
	if ExitCodeForError(result.Err) != 3 {
		t.Fatalf("exit code = %d, want 3", ExitCodeForError(result.Err))
	}
	if got := result.Stderr; !strings.Contains(got, "mnemonic project reindex") {
		t.Fatalf("stderr = %q, want reindex suggestion", got)
	}
}
