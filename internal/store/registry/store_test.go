package registry

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeValidManifest(projectID, name, slug string) *manifestfmt.Manifest {
	m := manifestfmt.New()
	m.ProjectID = projectID
	m.Name = name
	m.Slug = slug
	m.Type = manifestfmt.ManifestTypeLocal
	m.MarkdownFormatVersion = 1
	m.CreatedAt = time.Date(2026, time.June, 23, 10, 0, 0, 0, time.UTC)
	m.UpdatedAt = m.CreatedAt
	return m
}

func writeManifest(t *testing.T, dir string, m *manifestfmt.Manifest) {
	t.Helper()
	err := manifestfmt.WriteMnemonicManifest(filepath.Join(dir, "mnemonic.toml"), m)
	require.NoError(t, err)
}

func realManifestParser(path string) (*manifestfmt.Manifest, error) {
	return manifestfmt.ParseMnemonicManifestFromFile(path)
}

func realPointerParser(data []byte) (*manifestfmt.PointerFile, error) {
	return manifestfmt.ParsePointerFile(data)
}

func newTestStore(memoriesHome string) Store {
	return New(memoriesHome, realManifestParser, realPointerParser)
}

// ── New ────────────────────────────────────────────────────────────────

func TestNew(t *testing.T) {
	s := New("/tmp/home", realManifestParser, realPointerParser)
	assert.Equal(t, "/tmp/home", s.MemoriesHome)
	assert.NotNil(t, s.ManifestParser)
	assert.NotNil(t, s.PointerParser)
}

func TestNew_NilParsers(t *testing.T) {
	s := New("/tmp/home", nil, nil)
	assert.Equal(t, "/tmp/home", s.MemoriesHome)
	assert.Nil(t, s.ManifestParser)
	assert.Nil(t, s.PointerParser)
}

// ── Scan ───────────────────────────────────────────────────────────────

func TestScan_EmptyMemoriesHome(t *testing.T) {
	s := Store{MemoriesHome: ""}
	entries, issues, err := s.Scan()
	require.Error(t, err)
	assert.Nil(t, entries)
	assert.Nil(t, issues)
}

func TestScan_NonexistentDirectory(t *testing.T) {
	s := newTestStore("/nonexistent/path/12345")
	entries, issues, err := s.Scan()
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.Empty(t, issues)
}

func TestScan_EmptyDirectory(t *testing.T) {
	tmp := t.TempDir()
	s := newTestStore(tmp)

	entries, issues, err := s.Scan()
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.Empty(t, issues)
}

func TestScan_CentralProject(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "my-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	m := makeValidManifest("proj-1", "My Project", "my-project")
	writeManifest(t, projectDir, m)

	s := newTestStore(tmp)
	entries, issues, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Empty(t, issues)

	e := entries[0]
	assert.Equal(t, "my-project", e.Slug)
	assert.Equal(t, "proj-1", e.ProjectID)
	assert.Equal(t, "My Project", e.Name)
	assert.Equal(t, "central", e.Type)
	assert.Equal(t, filepath.Join(tmp, "my-project", "mnemonic.toml"), e.ManifestPath)
	assert.Equal(t, projectDir, e.MemoriesAbs)
	assert.Equal(t, projectDir, e.RepoRootAbs)
}

func TestScan_CorruptManifest(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "bad-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	// Write an invalid manifest
	err := os.WriteFile(filepath.Join(projectDir, "mnemonic.toml"), []byte("not valid toml {{{{{"), 0o644)
	require.NoError(t, err)

	s := newTestStore(tmp)
	entries, issues, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Len(t, issues, 1)

	e := entries[0]
	assert.Equal(t, "bad-project", e.Slug)
	assert.Equal(t, "central", e.Type)

	issue := issues[0]
	assert.Equal(t, "bad-project", issue.Slug)
	assert.True(t, issue.Corrupt)
}

func TestScan_SlugMismatch(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "dir-slug")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	// Manifest has a different slug than the directory name
	m := makeValidManifest("proj-1", "Different Name", "manifest-slug")
	writeManifest(t, projectDir, m)

	s := newTestStore(tmp)
	entries, issues, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Len(t, issues, 1)

	assert.Equal(t, "dir-slug", entries[0].Slug)
	assert.True(t, issues[0].Corrupt)
	assert.Contains(t, issues[0].Error, "mismatch")
}

func TestScan_LocalProject(t *testing.T) {
	tmp := t.TempDir()

	// Create the actual project directory elsewhere
	projectDir := filepath.Join(tmp, "real-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	m := makeValidManifest("proj-local", "Local Project", "local-project")
	writeManifest(t, projectDir, m)

	memoriesHome := filepath.Join(tmp, "memories")
	require.NoError(t, os.MkdirAll(memoriesHome, 0o755))

	// Create pointer file
	pf := &manifestfmt.PointerFile{ManifestPath: filepath.Join(projectDir, "mnemonic.toml")}
	err := manifestfmt.WritePointerFile(filepath.Join(memoriesHome, "local-project.toml"), pf)
	require.NoError(t, err)

	s := newTestStore(memoriesHome)
	entries, issues, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Empty(t, issues)

	e := entries[0]
	assert.Equal(t, "local-project", e.Slug)
	assert.Equal(t, "proj-local", e.ProjectID)
	assert.Equal(t, "Local Project", e.Name)
	assert.Equal(t, "local", e.Type)
}

// TestScan_MultipleProjects verifies Scan returns multiple entries correctly.
func TestScan_MultipleProjects(t *testing.T) {
	tmp := t.TempDir()

	for _, slug := range []string{"proj-a", "proj-b", "proj-c"} {
		projectDir := filepath.Join(tmp, slug)
		require.NoError(t, os.MkdirAll(projectDir, 0o755))
		m := makeValidManifest("id-"+slug, slug, slug)
		writeManifest(t, projectDir, m)
	}

	s := newTestStore(tmp)
	entries, issues, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Empty(t, issues)

	slugs := make([]string, 0, 3)
	for _, e := range entries {
		slugs = append(slugs, e.Slug)
	}
	assert.ElementsMatch(t, []string{"proj-a", "proj-b", "proj-c"}, slugs)
}

func TestScan_OrphanPointer(t *testing.T) {
	tmp := t.TempDir()
	memoriesHome := filepath.Join(tmp, "memories")
	require.NoError(t, os.MkdirAll(memoriesHome, 0o755))

	// Pointer points to a nonexistent manifest
	pf := &manifestfmt.PointerFile{ManifestPath: "/nonexistent/mnemonic.toml"}
	err := manifestfmt.WritePointerFile(filepath.Join(memoriesHome, "orphan.toml"), pf)
	require.NoError(t, err)

	s := newTestStore(memoriesHome)
	entries, issues, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Len(t, issues, 1)

	assert.Equal(t, "orphan", entries[0].Slug)
	assert.True(t, issues[0].Orphan)
}

func TestScan_IgnoresHiddenAndNonTOML(t *testing.T) {
	tmp := t.TempDir()

	// Create a regular file (not .toml, not directory)
	err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("not a project"), 0o644)
	require.NoError(t, err)

	s := newTestStore(tmp)
	entries, issues, err := s.Scan()
	require.NoError(t, err)
	assert.Empty(t, entries, "non-toml and non-directory files should be ignored")
	assert.Empty(t, issues)
}

func TestScan_NilManifestParser(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "my-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	m := makeValidManifest("proj-1", "My Project", "my-project")
	writeManifest(t, projectDir, m)

	s := New(tmp, nil, realPointerParser)
	entries, issues, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Empty(t, issues)

	// With nil parser, we only get the slug and type
	assert.Equal(t, "my-project", entries[0].Slug)
	assert.Equal(t, "central", entries[0].Type)
	assert.Empty(t, entries[0].ProjectID)
}

func TestScan_NilPointerParser(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "real-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	m := makeValidManifest("proj-1", "Real Project", "real-project")
	writeManifest(t, projectDir, m)

	memoriesHome := filepath.Join(tmp, "memories")
	require.NoError(t, os.MkdirAll(memoriesHome, 0o755))

	pf := &manifestfmt.PointerFile{ManifestPath: filepath.Join(projectDir, "mnemonic.toml")}
	err := manifestfmt.WritePointerFile(filepath.Join(memoriesHome, "local-test.toml"), pf)
	require.NoError(t, err)

	s := New(memoriesHome, realManifestParser, nil)
	entries, issues, err := s.Scan()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Empty(t, issues)

	assert.Equal(t, "local-test", entries[0].Slug)
	assert.Equal(t, "local", entries[0].Type)
}

// ── Resolve ────────────────────────────────────────────────────────────

func TestResolve_Found(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "my-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	m := makeValidManifest("proj-1", "My Project", "my-project")
	writeManifest(t, projectDir, m)

	s := newTestStore(tmp)
	entry, err := s.Resolve("my-project")
	require.NoError(t, err)
	assert.Equal(t, "my-project", entry.Slug)
	assert.Equal(t, "proj-1", entry.ProjectID)
}

func TestResolve_NotFound(t *testing.T) {
	tmp := t.TempDir()
	s := newTestStore(tmp)

	_, err := s.Resolve("nonexistent")
	require.Error(t, err)

	var notFound ErrNotFound
	require.True(t, errors.As(err, &notFound))
	assert.Equal(t, "nonexistent", notFound.Slug)
}

func TestResolve_ErrNotFound_Error(t *testing.T) {
	err := ErrNotFound{Slug: "test-slug"}
	assert.Contains(t, err.Error(), "test-slug")
	assert.Contains(t, err.Error(), "not found")
}

func TestResolve_CorruptProject(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "bad-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	err := os.WriteFile(filepath.Join(projectDir, "mnemonic.toml"), []byte("garbage"), 0o644)
	require.NoError(t, err)

	s := newTestStore(tmp)
	_, err = s.Resolve("bad-project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORRUPTED")
}

func TestResolve_OrphanProject(t *testing.T) {
	tmp := t.TempDir()
	memoriesHome := filepath.Join(tmp, "memories")
	require.NoError(t, os.MkdirAll(memoriesHome, 0o755))

	pf := &manifestfmt.PointerFile{ManifestPath: "/nonexistent/mnemonic.toml"}
	err := manifestfmt.WritePointerFile(filepath.Join(memoriesHome, "orphan.toml"), pf)
	require.NoError(t, err)

	s := newTestStore(memoriesHome)
	_, err = s.Resolve("orphan")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ORPHANED")
}

// ── FindByMemoriesRoot ─────────────────────────────────────────────────

func TestFindByMemoriesRoot_Found(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "my-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	m := makeValidManifest("proj-1", "My Project", "my-project")
	writeManifest(t, projectDir, m)

	s := newTestStore(tmp)
	entry, found, err := s.FindByMemoriesRoot(projectDir)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "my-project", entry.Slug)
}

func TestFindByMemoriesRoot_NotFound(t *testing.T) {
	tmp := t.TempDir()
	s := newTestStore(tmp)

	_, found, err := s.FindByMemoriesRoot("/nonexistent/path")
	require.NoError(t, err)
	assert.False(t, found)
}

// ── Exists ─────────────────────────────────────────────────────────────

func TestExists_Directory(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "my-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	s := newTestStore(tmp)
	exists, err := s.Exists("my-project")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestExists_PointerFile(t *testing.T) {
	tmp := t.TempDir()
	err := os.WriteFile(filepath.Join(tmp, "pointer.toml"), []byte("manifest_path = \"/path\""), 0o644)
	require.NoError(t, err)

	s := newTestStore(tmp)
	exists, err := s.Exists("pointer")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestExists_NotFound(t *testing.T) {
	tmp := t.TempDir()
	s := newTestStore(tmp)

	exists, err := s.Exists("nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

// ── Slugs ──────────────────────────────────────────────────────────────

func TestSlugs(t *testing.T) {
	tmp := t.TempDir()

	// Create two projects
	for _, slug := range []string{"project-a", "project-b"} {
		projectDir := filepath.Join(tmp, slug)
		require.NoError(t, os.MkdirAll(projectDir, 0o755))
		m := makeValidManifest("id-"+slug, slug, slug)
		writeManifest(t, projectDir, m)
	}

	s := newTestStore(tmp)
	slugs, err := s.Slugs()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"project-a", "project-b"}, slugs)
}

func TestSlugs_Empty(t *testing.T) {
	tmp := t.TempDir()
	s := newTestStore(tmp)

	slugs, err := s.Slugs()
	require.NoError(t, err)
	assert.Empty(t, slugs)
}
