package indexsvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/stretchr/testify/require"
)

func TestNewBindsKnowledgeBase(t *testing.T) {
	resolved := kb.KnowledgeBase{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Name:         "Demo",
		Slug:         "demo",
		Kind:         "central",
		RootDir:      "/tmp/demo",
		RepoRootDir:  "/tmp",
		ManifestPath: "/tmp/demo/mnemonic.toml",
		StateDir:     "/tmp/state",
		IndexPath:    "/tmp/state/index.sqlite",
	}

	svc := New(resolved, nil)
	require.Equal(t, resolved, svc.KB)
	require.Equal(t, resolved.RootDir, svc.Notes.RootDir)
	require.Equal(t, resolved.StateDir, svc.Notes.StateDir)
	require.Equal(t, resolved.RootDir, svc.Index.RootDir)
	require.Equal(t, resolved.StateDir, svc.Index.StateDir)
	require.Equal(t, resolved.IndexPath, svc.Index.IndexPath)
	require.Equal(t, resolved.ID, svc.Index.KBID)
}

func TestRebuild(t *testing.T) {
	svc, kb := newTestService(t)

	result, err := svc.Rebuild(context.Background())
	require.NoError(t, err)
	require.Equal(t, kb.ID, result.KBID)
	require.Equal(t, 2, result.NotesSeen)
	require.Equal(t, 2, result.NotesIndexed)
	require.Equal(t, "ok", result.Status)

	db, err := svc.Index.OpenReadonly()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var noteCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&noteCount))
	require.Equal(t, 2, noteCount)
}

func TestDoctorHealthy(t *testing.T) {
	svc, _ := newTestService(t)

	result, err := svc.Doctor(context.Background())
	require.NoError(t, err)
	require.Equal(t, "ok", result.Status)
	require.Len(t, result.Checks, 10)

	checks := map[string]DoctorCheck{}
	for _, check := range result.Checks {
		checks[check.Name] = check
	}

	require.Equal(t, "ok", checks["project path exists"].Status)
	require.Equal(t, "ok", checks["mnemonic.toml"].Status)
	require.Equal(t, "ok", checks["index exists"].Status)
	require.Equal(t, "ok", checks["index quick_check"].Status)
	require.Equal(t, "ok", checks["index schema"].Status)
	require.Equal(t, "ok", checks["duplicate note UUIDs"].Status)
	require.Equal(t, "ok", checks["duplicate note slugs"].Status)
	require.Equal(t, "ok", checks["unresolved link count"].Status)
	require.Equal(t, "ok", checks[".trash ignored"].Status)
	require.Equal(t, "ok", checks["stale temp files"].Status)
}

func TestDoctorMissingIndexNeedsReindex(t *testing.T) {
	svc, _ := newTestService(t)
	require.NoError(t, os.Remove(svc.Index.IndexPath))

	result, err := svc.Doctor(context.Background())
	require.NoError(t, err)
	require.Equal(t, "needs_reindex", result.Status)
	require.Len(t, result.Checks, 3)
	require.Equal(t, "missing", result.Checks[2].Status)
	require.Equal(t, "index exists", result.Checks[2].Name)
}

func newTestService(t *testing.T) (*Service, kb.KnowledgeBase) {
	t.Helper()

	root := t.TempDir()
	state := t.TempDir()
	k := kb.KnowledgeBase{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Name:         "Demo",
		Slug:         "demo",
		Kind:         "central",
		RootDir:      root,
		ManifestPath: filepath.Join(root, "mnemonic.toml"),
		StateDir:     state,
		IndexPath:    filepath.Join(state, "index.sqlite"),
	}

	manifest := manifestfmt.New()
	manifest.ProjectID = k.ID
	manifest.Name = k.Name
	manifest.Slug = k.Slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC).Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, manifestfmt.WriteMnemonicManifest(k.ManifestPath, manifest))

	require.NoError(t, writeNote(root, markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440001",
		Title:          "Target Note",
		Slug:           "target-note",
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("target body\n"),
	}))
	require.NoError(t, writeNote(root, markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440002",
		Title:          "Source Note",
		Slug:           "source-note",
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("See [[target-note]].\n"),
	}))

	trashDir := filepath.Join(root, ".trash")
	require.NoError(t, os.MkdirAll(trashDir, 0o755))
	require.NoError(t, writeNote(trashDir, markdown.Note{
		MnemonicNoteID: "550e8400-e29b-41d4-a716-446655440003",
		Title:          "Trash Note",
		Slug:           "trash-note",
		CreatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.June, 2, 12, 34, 56, 0, time.UTC),
		Body:           []byte("ignored\n"),
	}))

	svc := New(k, nil)
	_, err := svc.Rebuild(context.Background())
	require.NoError(t, err)

	return svc, k
}

func writeNote(root string, note markdown.Note) error {
	rendered, err := markdown.RenderNote(note)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, note.EffectiveSlug()+".md"), rendered, 0o644)
}
