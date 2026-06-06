package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestProjectDoctorReturnsOkForHealthyProject(t *testing.T) {
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
		Body:    []byte("[[Target Note]]\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440002"
		},
	}); err != nil {
		t.Fatalf("Create(source) error = %v", err)
	}
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	before := mustNoteSnapshot(t, memoriesRoot)
	result := executeCommand("project", "doctor", "personal", "--json")
	if result.Err != nil {
		t.Fatalf("project doctor returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `"status": "ok"`) {
		t.Fatalf("stdout = %s", result.Stdout)
	}
	after := mustNoteSnapshot(t, memoriesRoot)
	if before != after {
		t.Fatalf("markdown snapshot changed\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestProjectDoctorMissingIndexReturnsNeedsReindex(t *testing.T) {
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

	result := executeCommand("project", "doctor", "personal", "--json")
	if result.Err != nil {
		t.Fatalf("project doctor returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `"status": "needs_reindex"`) {
		t.Fatalf("stdout = %s", result.Stdout)
	}
}

func TestProjectDoctorCorruptedIndexReturnsExitSix(t *testing.T) {
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

	indexPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("index.Path() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("broken"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := executeCommand("project", "doctor", "personal", "--json")
	if result.Err == nil {
		t.Fatal("project doctor error = nil, want corrupted index")
	}
	if ExitCodeForError(result.Err) != 6 {
		t.Fatalf("exit code = %d, want 6", ExitCodeForError(result.Err))
	}
}

func TestProjectDoctorReportsStaleTempFileWithoutDeletingIt(t *testing.T) {
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
		Title:   "Healthy Note",
		Body:    []byte("body\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440001"
		},
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	tempPath := filepath.Join(projectRoot, ".tmp-test")
	if err := os.WriteFile(tempPath, []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := executeCommand("project", "doctor", "personal", "--json")
	if result.Err != nil {
		t.Fatalf("project doctor returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `"status": "warning"`) {
		t.Fatalf("stdout = %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `"name": "stale temp files"`) {
		t.Fatalf("stdout = %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `"count": 1`) {
		t.Fatalf("stdout = %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, tempPath) {
		t.Fatalf("stdout = %s", result.Stdout)
	}
	if _, err := os.Stat(tempPath); err != nil {
		t.Fatalf("Stat(%q) error = %v", tempPath, err)
	}
}

func mustNoteSnapshot(t *testing.T, root string) string {
	t.Helper()
	paths, err := notes.Walk(root)
	if err != nil {
		t.Fatalf("notes.Walk() error = %v", err)
	}
	var b strings.Builder
	for _, rel := range paths {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(abs)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", rel, err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", rel, err)
		}
		b.WriteString(rel)
		b.WriteByte('\n')
		b.WriteString(info.ModTime().UTC().Format(time.RFC3339Nano))
		b.WriteByte('\n')
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String()
}
