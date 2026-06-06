package cli

import (
	"database/sql"
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

func TestProjectReindexRebuildsIncompatibleIndexWithoutChangingMarkdown(t *testing.T) {
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
	noteResult, err := notes.Create(notes.CreateInput{
		RootDir: memoriesRoot,
		Title:   "Healthy Note",
		Body:    []byte("body\n"),
		UUID: func() string {
			return "550e8400-e29b-41d4-a716-446655440001"
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoriesRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	indexPath, err := index.Path("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("index.Path() error = %v", err)
	}
	db, err := sql.Open("sqlite", indexPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 999`); err != nil {
		_ = db.Close()
		t.Fatalf("set user_version=999: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close() error = %v", err)
	}

	notePath := filepath.Join(memoriesRoot, filepath.FromSlash(noteResult.Path))
	beforeInfo, err := os.Stat(notePath)
	if err != nil {
		t.Fatalf("Stat(before) error = %v", err)
	}
	beforeContent, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("ReadFile(before) error = %v", err)
	}

	doctorResult := executeCommand("project", "doctor", "personal", "--json")
	if doctorResult.Err != nil {
		t.Fatalf("project doctor returned error: %v\nstderr: %s", doctorResult.Err, doctorResult.Stderr)
	}
	if !strings.Contains(doctorResult.Stdout, `"status": "needs_reindex"`) {
		t.Fatalf("doctor stdout = %s", doctorResult.Stdout)
	}
	if !strings.Contains(doctorResult.Stdout, `"name": "index schema"`) {
		t.Fatalf("doctor stdout = %s", doctorResult.Stdout)
	}

	reindexResult := executeCommand("project", "reindex", "personal", "--json")
	if reindexResult.Err != nil {
		t.Fatalf("project reindex returned error: %v\nstderr: %s", reindexResult.Err, reindexResult.Stderr)
	}

	afterInfo, err := os.Stat(notePath)
	if err != nil {
		t.Fatalf("Stat(after) error = %v", err)
	}
	if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		t.Fatalf("markdown mtime changed: before=%s after=%s", beforeInfo.ModTime(), afterInfo.ModTime())
	}
	afterContent, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("ReadFile(after) error = %v", err)
	}
	if string(afterContent) != string(beforeContent) {
		t.Fatalf("markdown content changed\nbefore:\n%s\nafter:\n%s", beforeContent, afterContent)
	}

	checkDB, err := sql.Open("sqlite", "file:"+filepath.ToSlash(indexPath)+"?mode=ro")
	if err != nil {
		t.Fatalf("sql.Open(check) error = %v", err)
	}
	defer func() { _ = checkDB.Close() }()
	status, err := index.CheckSchemaStatus(checkDB)
	if err != nil {
		t.Fatalf("CheckSchemaStatus() error = %v", err)
	}
	if status != index.SchemaStatusOK {
		t.Fatalf("schema status = %q, want %q", status, index.SchemaStatusOK)
	}
}
