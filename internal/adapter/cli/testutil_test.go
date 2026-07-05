package cli

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/platform/paths"
	registry "github.com/ilyachch/mnemonic/internal/store/registry"
	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type cmdResult struct {
	Stdout string
	Stderr string
	Err    error
}

// executeCommand runs a fresh CLI root with the given arguments and returns stdout, stderr, and error.
func executeCommand(args ...string) cmdResult {
	root := newRootForTest()
	return executeCommandWithRoot(root, args...)
}

func executeCommandWithBootstrap(boot *app.Bootstrap, args ...string) cmdResult {
	root := NewRootCommand(boot)
	return executeCommandWithRoot(root, args...)
}

func executeCommandWithRoot(root *cobra.Command, args ...string) cmdResult {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	root.SetOut(bufOut)
	root.SetErr(bufErr)
	root.SetArgs(args)

	err := root.Execute()
	if err != nil {
		fmt.Fprintln(bufErr, err)
	}

	return cmdResult{
		Stdout: bufOut.String(),
		Stderr: bufErr.String(),
		Err:    err,
	}
}

func newRootForTest() *cobra.Command {
	boot, err := newTestBootstrap()
	if err != nil {
		panic(err)
	}
	return NewRootCommand(boot)
}

func newTestBootstrap() (*app.Bootstrap, error) {
	configFile, err := os.CreateTemp("", "mnemonic-cli-config-*.toml")
	if err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintln(configFile, "version = 1"); err != nil {
		_ = configFile.Close()
		return nil, err
	}
	if err := configFile.Close(); err != nil {
		return nil, err
	}

	return app.New(app.Input{CLI: paths.CLIOverrides{ConfigFile: configFile.Name()}})
}

func newTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	return newRootForTest()
}

func setLocalProjectMemoriesHome(t *testing.T, projectRoot string) {
	t.Helper()

	t.Setenv("MNEMONIC_MEMORIES_HOME", filepath.Join(projectRoot, ".mnemonic-memories"))
}

func testIndexPath(root, projectID string) string {
	stateHome, err := resolveStateHome()
	if err != nil {
		return filepath.Join(root, "state", "mnemonic", "projects", projectID, "index.sqlite")
	}
	return filepath.Join(stateHome, "mnemonic", "projects", projectID, "index.sqlite")
}

func resolveStateHome() (string, error) {
	boot, err := newTestBootstrap()
	if err != nil {
		return "", err
	}
	return boot.Paths.StateHome, nil
}

func writeCentralProjectFixture(t *testing.T, memoriesHome, slug, projectID string, createdAt time.Time) string {
	t.Helper()

	projectDir := filepath.Join(memoriesHome, slug)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := manifestfmt.New()
	manifest.ProjectID = projectID
	manifest.Name = slug
	manifest.Slug = slug
	manifest.Description = "Central description"
	manifest.CustomInstructions = "Central instructions"
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = createdAt.Unix()
	manifest.UpdatedAt = createdAt.Unix()
	manifest.Generator.App = "mnemonic"

	if err := manifestfmt.WriteMnemonicManifest(filepath.Join(projectDir, "mnemonic.toml"), manifest); err != nil {
		t.Fatal(err)
	}

	return projectDir
}

func writeLocalProjectFixture(t *testing.T, cwd, slug string) error {
	t.Helper()

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)

	memoriesDir := filepath.Join(cwd, ".mnemonic-memories", slug)
	if err := os.MkdirAll(memoriesDir, 0o755); err != nil {
		return err
	}

	projectID := "550e8400-e29b-41d4-a716-446655440000"

	manifest := manifestfmt.New()
	manifest.ProjectID = projectID
	manifest.Name = slug
	manifest.Slug = slug
	manifest.Type = manifestfmt.ManifestTypeLocal
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = now.Unix()
	manifest.UpdatedAt = now.Unix()
	manifest.Generator.App = "mnemonic"

	if err := manifestfmt.WriteMnemonicManifest(filepath.Join(memoriesDir, "mnemonic.toml"), manifest); err != nil {
		return err
	}

	memHome, err := resolveMemoriesHome()
	if err != nil {
		return err
	}

	pointerPath := filepath.Join(memHome, slug+".toml")
	if err := os.MkdirAll(filepath.Dir(pointerPath), 0o755); err != nil {
		return err
	}
	return manifestfmt.WritePointerFile(pointerPath, &manifestfmt.PointerFile{
		ManifestPath: filepath.Join(memoriesDir, "mnemonic.toml"),
	})
}

func resolveMemoriesHome() (string, error) {
	boot, err := newTestBootstrap()
	if err != nil {
		return "", err
	}
	return boot.Paths.MemoriesHome, nil
}

func seedLocalProjectIndex(t *testing.T, projectRoot, projectID string, notes ...seededIndexNote) {
	t.Helper()

	idxPath := testIndexPath(projectRoot, projectID)
	require.NoError(t, os.MkdirAll(filepath.Dir(idxPath), 0o755))

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(idxPath)+"?mode=rwc")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	require.NoError(t, sqliteindex.ApplySchema(db))
	for _, note := range notes {
		relPath := note.RelPath
		if relPath == "" {
			relPath = note.Slug + ".md"
		}
		_, err := db.Exec(
			`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			note.NoteID,
			projectID,
			note.Slug,
			filepath.ToSlash(relPath),
			note.Title,
			note.ContentHash,
			note.CreatedAt.UTC().Format(time.RFC3339),
			note.UpdatedAt.UTC().Format(time.RFC3339),
		)
		require.NoError(t, err)
	}
}

type seededIndexNote struct {
	NoteID      string
	Slug        string
	Title       string
	RelPath     string
	ContentHash string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func testCatalogStore(memoriesHome string) registry.Store {
	return registry.New(memoriesHome, func(path string) (*manifestfmt.Manifest, error) {
		return manifestfmt.ParseMnemonicManifestFromFile(path)
	}, func(data []byte) (*manifestfmt.PointerFile, error) {
		return manifestfmt.ParsePointerFile(data)
	})
}
