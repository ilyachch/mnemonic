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

func TestNotesBacklinksCommandReturnsLinks(t *testing.T) {
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
		Title:   "Target Note",
		Body:    []byte("target body\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440001"
		},
	}); err != nil {
		t.Fatalf("Create(target) error = %v", err)
	}
	if _, err := notes.Create(notes.CreateInput{
		RootDir: memoriesRoot,
		Title:   "Source Note",
		Body:    []byte("[[Target Note]]\n## Relations\n- depends_on [[Target Note]]\n- relates_to [[Missing Note]]\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440002"
		},
	}); err != nil {
		t.Fatalf("Create(source) error = %v", err)
	}

	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	result := executeCommand("notes", "backlinks", "target-note", "--project", "personal", "--json")
	if result.Err != nil {
		t.Fatalf("notes backlinks returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	var got struct {
		Links []struct {
			NoteID       string `json:"note_id"`
			Slug         string `json:"slug"`
			Title        string `json:"title"`
			Path         string `json:"path"`
			RelationType string `json:"relation_type"`
			SourceLine   int    `json:"source_line"`
		} `json:"links"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout: %s", err, result.Stdout)
	}
	if len(got.Links) != 2 {
		t.Fatalf("len(links) = %d, want 2", len(got.Links))
	}
	foundPlain := false
	foundRelation := false
	for _, link := range got.Links {
		if link.NoteID != "550e8400-e29b-41d4-a716-446655440002" {
			t.Fatalf("unexpected backlink note_id = %q", link.NoteID)
		}
		if link.SourceLine <= 0 {
			t.Fatalf("source_line not captured: %#v", link)
		}
		switch link.RelationType {
		case "":
			foundPlain = true
		case "depends_on":
			foundRelation = true
		default:
			t.Fatalf("unexpected relation_type = %q", link.RelationType)
		}
	}
	if !foundPlain || !foundRelation {
		t.Fatalf("missing backlink variants: plain=%v relation=%v links=%#v", foundPlain, foundRelation, got.Links)
	}
}

func TestNotesBacklinksCommandMissingNoteAndMissingIndex(t *testing.T) {
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

	result := executeCommand("notes", "backlinks", "missing-note", "--project", "personal", "--json")
	if result.Err == nil {
		t.Fatal("notes backlinks error = nil, want missing note")
	}
	if ExitCodeForError(result.Err) != 3 {
		t.Fatalf("exit code = %d, want 3", ExitCodeForError(result.Err))
	}

	memoriesRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if _, err := notes.Create(notes.CreateInput{
		RootDir: memoriesRoot,
		Title:   "Target Note",
		Body:    []byte("target body\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440001"
		},
	}); err != nil {
		t.Fatalf("Create(target) error = %v", err)
	}

	result = executeCommand("notes", "backlinks", "target-note", "--project", "personal", "--json")
	if result.Err == nil {
		t.Fatal("notes backlinks error = nil, want missing index")
	}
	if ExitCodeForError(result.Err) != 3 {
		t.Fatalf("exit code = %d, want 3", ExitCodeForError(result.Err))
	}
	if got := result.Stderr; !strings.Contains(got, "mnemonic project reindex") {
		t.Fatalf("stderr = %q, want reindex suggestion", got)
	}
}
