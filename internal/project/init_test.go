package project

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/ilyachch/mnemonic/internal/testutil"
)

func TestInitRegularProject(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	dataHome := filepath.Join(root, "data")

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	if err := InitRegularProject(InitRegularInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "my-app",
	}); err != nil {
		t.Fatalf("InitRegularProject() error = %v", err)
	}

	projectPath := filepath.Join(cwd, ".mnemonic")
	manifestPath := filepath.Join(memoriesHome, "my-app", "mnemonic.toml")

	if _, err := os.Stat(projectPath); err != nil {
		t.Fatalf("project file missing: %v", err)
	}
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(memoriesHome, "my-app", "index.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("index.sqlite exists or stat failed unexpectedly: %v", err)
	}

	projectData, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("ReadFile(project) error = %v", err)
	}
	parsedProject, err := ParseMnemonicFile(projectData)
	if err != nil {
		t.Fatalf("ParseMnemonicFile() error = %v", err)
	}
	if got, want := len(parsedProject.Projects), 1; got != want {
		t.Fatalf("len(projects) = %d, want %d", got, want)
	}

	projectEntry := parsedProject.Projects[0]
	if projectEntry.ID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("project id = %q, want %q", projectEntry.ID, "550e8400-e29b-41d4-a716-446655440000")
	}
	if projectEntry.Name != "my-app" {
		t.Fatalf("project name = %q, want %q", projectEntry.Name, "my-app")
	}
	if projectEntry.Slug != "my-app" {
		t.Fatalf("project slug = %q, want %q", projectEntry.Slug, "my-app")
	}
	if projectEntry.Kind != ProjectKindRegular {
		t.Fatalf("project kind = %q, want %q", projectEntry.Kind, ProjectKindRegular)
	}
	if projectEntry.MemoriesPath != "my-app" {
		t.Fatalf("project memories_path = %q, want %q", projectEntry.MemoriesPath, "my-app")
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	parsedManifest, err := ParseMnemonicManifest(manifestData)
	if err != nil {
		t.Fatalf("ParseMnemonicManifest() error = %v", err)
	}
	if parsedManifest.Kind != ManifestKindRegular {
		t.Fatalf("manifest kind = %q, want %q", parsedManifest.Kind, ManifestKindRegular)
	}
	if parsedManifest.Name != "my-app" {
		t.Fatalf("manifest name = %q, want %q", parsedManifest.Name, "my-app")
	}
	if parsedManifest.Slug != "my-app" {
		t.Fatalf("manifest slug = %q, want %q", parsedManifest.Slug, "my-app")
	}
	if parsedManifest.ProjectID != projectEntry.ID {
		t.Fatalf("manifest project_id = %q, want %q", parsedManifest.ProjectID, projectEntry.ID)
	}

	assertProjectRegistered(t, dataHome, projectEntry.ID, filepath.Join(memoriesHome, projectEntry.MemoriesPath), filepath.Join(memoriesHome, "my-app", "mnemonic.toml"))
}

func TestInitLocalProject(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	dataHome := filepath.Join(root, "data")

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	if err := InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "backend",
		Mode:         InitModeLocal,
	}); err != nil {
		t.Fatalf("InitProject(local) error = %v", err)
	}

	projectPath := filepath.Join(cwd, ".mnemonic")
	localPath := filepath.Join(cwd, ".mnemonic-memories", "backend")
	manifestPath := filepath.Join(localPath, "mnemonic.toml")

	if _, err := os.Stat(projectPath); err != nil {
		t.Fatalf("project file missing: %v", err)
	}
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("local memories directory missing: %v", err)
	}
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatalf("local manifest exists or stat failed unexpectedly: %v", err)
	}

	projectData, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("ReadFile(project) error = %v", err)
	}
	parsedProject, err := ParseMnemonicFile(projectData)
	if err != nil {
		t.Fatalf("ParseMnemonicFile() error = %v", err)
	}
	if got, want := len(parsedProject.Projects), 1; got != want {
		t.Fatalf("len(projects) = %d, want %d", got, want)
	}
	projectEntry := parsedProject.Projects[0]
	if projectEntry.Kind != ProjectKindLocal {
		t.Fatalf("project kind = %q, want %q", projectEntry.Kind, ProjectKindLocal)
	}
	if projectEntry.MemoriesPath != filepath.Join(".mnemonic-memories", "backend") {
		t.Fatalf("project memories_path = %q, want %q", projectEntry.MemoriesPath, filepath.Join(".mnemonic-memories", "backend"))
	}

	assertProjectRegistered(t, dataHome, projectEntry.ID, filepath.Join(cwd, projectEntry.MemoriesPath), "")
}

func TestInitDetachedProject(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	dataHome := filepath.Join(root, "data")

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	if err := InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "personal",
		Mode:         InitModeDetached,
	}); err != nil {
		t.Fatalf("InitProject(detached) error = %v", err)
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
	parsedManifest, err := ParseMnemonicManifest(manifestData)
	if err != nil {
		t.Fatalf("ParseMnemonicManifest() error = %v", err)
	}
	if parsedManifest.Kind != ManifestKindDetached {
		t.Fatalf("manifest kind = %q, want %q", parsedManifest.Kind, ManifestKindDetached)
	}
	if parsedManifest.Slug != "personal" {
		t.Fatalf("manifest slug = %q, want %q", parsedManifest.Slug, "personal")
	}

	assertProjectRegistered(t, dataHome, "550e8400-e29b-41d4-a716-446655440000", filepath.Join(memoriesHome, "personal"), manifestPath)
}

func TestInitLocalProjectAppendsExistingMnemonicFile(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	dataHome := filepath.Join(root, "data")

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	if err := InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "backend",
		Mode:         InitModeLocal,
	}); err != nil {
		t.Fatalf("first InitProject(local) error = %v", err)
	}

	restore()
	restore = SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC),
		"7f0a6d73-c3ba-4f0e-85b8-27bccf4370f1",
	))
	t.Cleanup(restore)

	if err := InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "frontend",
		Mode:         InitModeLocal,
	}); err != nil {
		t.Fatalf("second InitProject(local) error = %v", err)
	}

	projectData, err := os.ReadFile(filepath.Join(cwd, ".mnemonic"))
	if err != nil {
		t.Fatalf("ReadFile(project) error = %v", err)
	}
	parsedProject, err := ParseMnemonicFile(projectData)
	if err != nil {
		t.Fatalf("ParseMnemonicFile() error = %v", err)
	}
	if got, want := len(parsedProject.Projects), 2; got != want {
		t.Fatalf("len(projects) = %d, want %d", got, want)
	}

	if got, want := parsedProject.CreatedAt, time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("root created_at = %v, want %v", got, want)
	}
	if got, want := parsedProject.UpdatedAt, time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("root updated_at = %v, want %v", got, want)
	}

	first := parsedProject.Projects[0]
	if first.Name != "backend" {
		t.Fatalf("first project name = %q, want %q", first.Name, "backend")
	}
	if got, want := first.CreatedAt, time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("first project created_at = %v, want %v", got, want)
	}

	second := parsedProject.Projects[1]
	if second.Name != "frontend" {
		t.Fatalf("second project name = %q, want %q", second.Name, "frontend")
	}
	if got, want := second.CreatedAt, time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("second project created_at = %v, want %v", got, want)
	}

	assertProjectRegistered(t, dataHome, first.ID, filepath.Join(cwd, first.MemoriesPath), "")
	assertProjectRegistered(t, dataHome, second.ID, filepath.Join(cwd, second.MemoriesPath), "")
}

func TestInitLocalProjectRejectsDuplicateSlug(t *testing.T) {
	root := testutil.CleanEnvForTest(t)
	cwd := t.TempDir()
	memoriesHome := t.TempDir()
	dataHome := filepath.Join(root, "data")

	restore := SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC),
		"550e8400-e29b-41d4-a716-446655440000",
	))
	t.Cleanup(restore)

	if err := InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "backend",
		Mode:         InitModeLocal,
	}); err != nil {
		t.Fatalf("first InitProject(local) error = %v", err)
	}

	restore()
	restore = SetClock(testutil.NewClock(
		time.Date(2026, time.June, 2, 11, 0, 0, 0, time.UTC),
		"7f0a6d73-c3ba-4f0e-85b8-27bccf4370f1",
	))
	t.Cleanup(restore)

	err := InitProject(InitInput{
		CWD:          cwd,
		MemoriesHome: memoriesHome,
		Name:         "Backend",
		Mode:         InitModeLocal,
	})
	if err == nil {
		t.Fatal("second InitProject(local) error = nil, want duplicate slug rejection")
	}
	if got := err.Error(); got != `project slug "backend" already exists` {
		t.Fatalf("second InitProject(local) error = %q, want duplicate slug rejection", got)
	}

	projectData, err := os.ReadFile(filepath.Join(cwd, ".mnemonic"))
	if err != nil {
		t.Fatalf("ReadFile(project) error = %v", err)
	}
	parsedProject, err := ParseMnemonicFile(projectData)
	if err != nil {
		t.Fatalf("ParseMnemonicFile() error = %v", err)
	}
	if got, want := len(parsedProject.Projects), 1; got != want {
		t.Fatalf("len(projects) = %d, want %d", got, want)
	}

	assertProjectRegistered(t, dataHome, "550e8400-e29b-41d4-a716-446655440000", filepath.Join(cwd, parsedProject.Projects[0].MemoriesPath), "")
}

func assertProjectRegistered(t *testing.T, dataHome, projectID, memoriesAbs, manifestAbs string) {
	t.Helper()

	db, err := registry.OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if err := registry.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema() error = %v", err)
	}

	var gotMemoriesAbs, gotManifestAbs sql.NullString
	if err := db.QueryRow(`SELECT memories_abs, manifest_abs FROM project_locations WHERE project_id = ?`, projectID).Scan(&gotMemoriesAbs, &gotManifestAbs); err != nil {
		t.Fatalf("query project_locations failed: %v", err)
	}
	if gotMemoriesAbs.String != memoriesAbs {
		t.Fatalf("memories_abs = %q, want %q", gotMemoriesAbs.String, memoriesAbs)
	}
	if gotManifestAbs.String != manifestAbs {
		t.Fatalf("manifest_abs = %q, want %q", gotManifestAbs.String, manifestAbs)
	}

	var needsReindex, indexPresent int
	if err := db.QueryRow(`SELECT needs_reindex, index_present FROM project_status WHERE project_id = ?`, projectID).Scan(&needsReindex, &indexPresent); err != nil {
		t.Fatalf("query project_status failed: %v", err)
	}
	if needsReindex != 1 {
		t.Fatalf("needs_reindex = %d, want 1", needsReindex)
	}
	if indexPresent != 0 {
		t.Fatalf("index_present = %d, want 0", indexPresent)
	}
}
