package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestInitCommandRequiresName(t *testing.T) {
	result := executeCommand("init")
	if result.Err == nil {
		t.Fatalf("expected init without NAME to fail")
	}
	if got := ExitCodeForError(result.Err); got != 2 {
		t.Fatalf("exit code = %d, want 2", got)
	}
}

func TestInitCommandRejectsMultipleNames(t *testing.T) {
	result := executeCommand("init", "one", "two")
	if result.Err == nil {
		t.Fatalf("expected init with multiple NAME args to fail")
	}
	if got := ExitCodeForError(result.Err); got != 2 {
		t.Fatalf("exit code = %d, want 2", got)
	}
}

func TestInitCommandRejectsLocalAndDetachedTogether(t *testing.T) {
	result := executeCommand("init", "demo", "--local", "--detached")
	if result.Err == nil {
		t.Fatalf("expected init with incompatible flags to fail")
	}
	if got := ExitCodeForError(result.Err); got != 2 {
		t.Fatalf("exit code = %d, want 2", got)
	}
}

func TestInitCommandLocalCreatesLocalProject(t *testing.T) {
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := project.SetClock(projectClockForCLI())
	t.Cleanup(restore)

	result := executeCommand("init", "backend", "--local")
	if result.Err != nil {
		t.Fatalf("init returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	projectPath := filepath.Join(cwd, ".mnemonic")
	localPath := filepath.Join(cwd, ".mnemonic-memories", "backend")
	manifestPath := filepath.Join(memoriesHome, "backend", "mnemonic.toml")

	if _, err := os.Stat(projectPath); err != nil {
		t.Fatalf("project file missing: %v", err)
	}
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("local memories directory missing: %v", err)
	}
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatalf("unexpected detached-style manifest: %v", err)
	}

	projectData, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("ReadFile(project) error = %v", err)
	}
	parsedProject, err := project.ParseMnemonicFile(projectData)
	if err != nil {
		t.Fatalf("ParseMnemonicFile() error = %v", err)
	}
	if got, want := len(parsedProject.Projects), 1; got != want {
		t.Fatalf("len(projects) = %d, want %d", got, want)
	}
	if parsedProject.Projects[0].Kind != project.ProjectKindLocal {
		t.Fatalf("project kind = %q, want %q", parsedProject.Projects[0].Kind, project.ProjectKindLocal)
	}
	indexPath, err := index.Path(parsedProject.Projects[0].ID)
	if err != nil {
		t.Fatalf("index.Path() error = %v", err)
	}
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index file missing: %v", err)
	}
}

func TestInitCommandDetachedCreatesDetachedProject(t *testing.T) {
	cwd := t.TempDir()
	memoriesHome := t.TempDir()

	testutil.CleanEnvForTest(t)
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := project.SetClock(projectClockForCLI())
	t.Cleanup(restore)

	result := executeCommand("init", "personal", "--detached")
	if result.Err != nil {
		t.Fatalf("init returned error: %v\nstderr: %s", result.Err, result.Stderr)
	}

	projectPath := filepath.Join(cwd, ".mnemonic")
	manifestPath := filepath.Join(memoriesHome, "personal", "mnemonic.toml")

	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		t.Fatalf(".mnemonic exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest file missing: %v", err)
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	parsedManifest, err := project.ParseMnemonicManifest(manifestData)
	if err != nil {
		t.Fatalf("ParseMnemonicManifest() error = %v", err)
	}
	if parsedManifest.Kind != project.ManifestKindDetached {
		t.Fatalf("manifest kind = %q, want %q", parsedManifest.Kind, project.ManifestKindDetached)
	}
	indexPath, err := index.Path(parsedManifest.ProjectID)
	if err != nil {
		t.Fatalf("index.Path() error = %v", err)
	}
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index file missing: %v", err)
	}
}

func TestInitCommandRejectsDuplicateSlug(t *testing.T) {
	testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	restore := project.SetClock(projectClockForCLI())
	t.Cleanup(restore)

	first := executeCommand("init", "backend", "--local")
	if first.Err != nil {
		t.Fatalf("first init returned error: %v\nstderr: %s", first.Err, first.Stderr)
	}

	result := executeCommand("init", "Backend", "--local")
	if result.Err == nil {
		t.Fatal("second init error = nil, want duplicate slug rejection")
	}
	if got := ExitCodeForError(result.Err); got != 4 {
		t.Fatalf("exit code = %d, want 4", got)
	}
	if !strings.Contains(result.Stderr, `project slug "backend" already exists`) {
		t.Fatalf("stderr = %q, want duplicate slug message", result.Stderr)
	}
}

func projectClockForCLI() project.Clock {
	return projectClock{
		now:   time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		uuids: []string{"550e8400-e29b-41d4-a716-446655440000"},
	}
}

type projectClock struct {
	now   time.Time
	uuids []string
}

func (c projectClock) Now() time.Time {
	return c.now
}

func (c projectClock) UUID() string {
	if len(c.uuids) == 0 {
		return ""
	}
	return c.uuids[0]
}
