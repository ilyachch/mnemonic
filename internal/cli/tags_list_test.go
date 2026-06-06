package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestTagsListCommandReturnsCountsAndSorts(t *testing.T) {
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
	writeTaggedNote(t, filepath.Join(memoriesRoot, "note-one.md"), "550e8400-e29b-41d4-a716-446655440001", "Note One", "note-one", []string{"django"}, "tagged twice #django\n")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "note-two.md"), "550e8400-e29b-41d4-a716-446655440002", "Note Two", "note-two", []string{"auth"}, "auth only\n")
	writeTaggedNote(t, filepath.Join(memoriesRoot, "note-three.md"), "550e8400-e29b-41d4-a716-446655440003", "Note Three", "note-three", []string{"auth"}, "mixed tags #django\n")
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	result := executeCommand("tags", "list", "--project", "personal", "--json")
	if result.Err != nil {
		t.Fatalf("tags list returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got struct {
		Tags []struct {
			Tag   string `json:"tag"`
			Count int    `json:"count"`
		} `json:"tags"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, result.Stdout)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(got.Tags))
	}
	if got.Tags[0].Tag != "auth" || got.Tags[0].Count != 2 {
		t.Fatalf("tags[0] = %#v, want auth count 2", got.Tags[0])
	}
	if got.Tags[1].Tag != "django" || got.Tags[1].Count != 2 {
		t.Fatalf("tags[1] = %#v, want django count 2", got.Tags[1])
	}
}

func TestTagsListCommandMissingIndexSuggestsReindex(t *testing.T) {
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

	result := executeCommand("tags", "list", "--project", "personal", "--json")
	if result.Err == nil {
		t.Fatal("tags list error = nil, want missing index error")
	}
	if ExitCodeForError(result.Err) != 3 {
		t.Fatalf("exit code = %d, want 3", ExitCodeForError(result.Err))
	}
	if got := result.Stderr; !strings.Contains(got, "mnemonic project reindex") {
		t.Fatalf("stderr = %q, want reindex suggestion", got)
	}
}
