package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/buildinfo"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/mcp/tools"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestCommandTransportInitializeAndListTools(t *testing.T) {
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	session, stderr := connectToMCPServerWithEnv(t, repoRoot, projectRoot, writableMCPEnv(t))

	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, tools.Tools, 8)
	gotNames := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		require.NotContains(t, tool.Name, " ")
		require.NotNil(t, tool.Annotations)
		gotNames = append(gotNames, tool.Name)
	}
	wantNames := []string{"create_note", "delete_note", "edit_note", "list_backlinks", "list_notes", "list_tags", "read_note", "search_notes"}
	require.True(t, slices.Equal(gotNames, wantNames))
	assertToolAnnotations(t, tools.Tools, "create_note", false, false)
	assertToolAnnotations(t, tools.Tools, "edit_note", false, true)
	assertToolAnnotations(t, tools.Tools, "delete_note", false, true)
	assertToolAnnotations(t, tools.Tools, "list_backlinks", true, false)
	assertToolAnnotations(t, tools.Tools, "list_notes", true, false)
	assertToolAnnotations(t, tools.Tools, "list_tags", true, false)
	assertToolAnnotations(t, tools.Tools, "read_note", true, false)
	assertToolAnnotations(t, tools.Tools, "search_notes", true, false)

	gotSnapshot, err := json.MarshalIndent(tools.Tools, "", "  ")
	require.NoError(t, err)
	wantSnapshot, err := os.ReadFile(filepath.Join(repoRoot, "internal", "mcp", "testdata", "read_only_tools.snapshot.json"))
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(string(wantSnapshot)), strings.TrimSpace(string(gotSnapshot)))

	if got := stderr.String(); got != "" {
		t.Logf("stderr output: %s", got)
	}
}

func TestListNotesReturnsEmptyArrayForEmptyProject(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_notes"})
	require.NoError(t, err)

	out := decodeListNotesOutput(t, result)
	require.Empty(t, out.Notes)
}

func TestListNotesSupportsPagination(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	_, err := notes.Create(notes.CreateInput{
		RootDir: memoryRoot,
		Title:   "Alpha Note",
	})
	require.NoError(t, err)
	_, err = notes.Create(notes.CreateInput{
		RootDir: memoryRoot,
		Title:   "Beta Note",
	})
	require.NoError(t, err)

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	firstResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_notes",
		Arguments: map[string]any{"limit": 1},
	})
	require.NoError(t, err)
	first := decodeListNotesOutput(t, firstResult)
	require.Len(t, first.Notes, 1)
	require.NotEmpty(t, first.NextCursor)

	secondResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_notes",
		Arguments: map[string]any{"limit": 1, "cursor": first.NextCursor},
	})
	require.NoError(t, err)
	second := decodeListNotesOutput(t, secondResult)
	require.Len(t, second.Notes, 1)
	require.Empty(t, second.NextCursor)
}

func TestListTagsReturnsTagsAndRespectsLimit(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "auth-one.md"), "550e8400-e29b-41d4-a716-446655440001", "Auth One", "auth-one", []string{"auth", "ops"}, "auth one body\n")
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "auth-two.md"), "550e8400-e29b-41d4-a716-446655440002", "Auth Two", "auth-two", []string{"auth"}, "auth two body\n")
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "beta-one.md"), "550e8400-e29b-41d4-a716-446655440003", "Beta One", "beta-one", []string{"beta"}, "beta body\n")
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_tags",
		Arguments: map[string]any{"limit": 1},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	out := decodeListTagsOutput(t, result)
	require.Len(t, out.Tags, 1)
	require.Equal(t, "auth", out.Tags[0].Tag)
	require.Equal(t, 2, out.Tags[0].Count)
}

func TestServerIndexDBReusesSingleConnection(t *testing.T) {
	_ = writableMCPEnv(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "auth-one.md"), "550e8400-e29b-41d4-a716-446655440001", "Auth One", "auth-one", []string{"auth"}, "auth one body\n")
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	effectivePaths, err := paths.ResolveEffectivePaths(paths.EffectiveInput{})
	require.NoError(t, err)

	entry, err := registry.Resolve(effectivePaths.MemoriesHome, "personal")
	require.NoError(t, err)

	server := NewServer(toAppResolution(entry), effectivePaths)
	t.Cleanup(func() {
		_ = server.closeIndexDB()
	})

	firstDB, err := server.GetIndexDB()
	require.NoError(t, err)
	secondDB, err := server.GetIndexDB()
	require.NoError(t, err)
	require.Same(t, firstDB, secondDB)
}

func TestListBacklinksReturnsLinksAndRespectsLimit(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	writeWikiMCPNote(t, filepath.Join(memoryRoot, "target-note.md"), "550e8400-e29b-41d4-a716-446655440010", "Target Note", "target-note", "target body\n")
	writeWikiMCPNote(t, filepath.Join(memoryRoot, "alpha-note.md"), "550e8400-e29b-41d4-a716-446655440011", "Alpha Note", "alpha-note", "[[target-note]]\n")
	writeWikiMCPNote(t, filepath.Join(memoryRoot, "beta-note.md"), "550e8400-e29b-41d4-a716-446655440012", "Beta Note", "beta-note", "[[Target Note]]\n")
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_backlinks",
		Arguments: map[string]any{"identifier": "target-note", "limit": 1},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	out := decodeListBacklinksOutput(t, result)
	require.Len(t, out.Links, 1)
	require.Equal(t, "alpha-note", out.Links[0].Slug)
}

func TestListBacklinksReturnsToolErrorForMissingNote(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	writeWikiMCPNote(t, filepath.Join(memoryRoot, "target-note.md"), "550e8400-e29b-41d4-a716-446655440010", "Target Note", "target-note", "target body\n")
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_backlinks",
		Arguments: map[string]any{"identifier": "missing-note"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
	require.Contains(t, resultText(t, result), "not found")
}

func TestCreateNoteCreatesReadableNote(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Auth Migration",
			"body":  "## Summary\n\nPlan.\n",
			"path":  "plans/auth-migration",
			"tags":  []string{"auth", "ops", "auth"},
			"type":  "decision",
		},
	})
	require.NoError(t, err)
	require.False(t, createResult.IsError)

	created := decodeCreateNoteOutput(t, createResult)
	require.NotEmpty(t, created.NoteID)
	require.Equal(t, "auth-migration", created.Slug)
	require.Equal(t, "plans/auth-migration.md", created.Path)
	require.NotEmpty(t, created.ContentHash)

	readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	require.NoError(t, err)
	require.False(t, readResult.IsError)

	read := decodeReadNoteOutput(t, readResult)
	require.Equal(t, created.Path, read.Note.Path)
	require.Equal(t, "## Summary\n\nPlan.\n", read.Note.Body)
	require.Equal(t, "decision", read.Note.Frontmatter["type"])
	require.True(t, slices.Equal(anySliceToStrings(t, read.Note.Frontmatter["tags"]), []string{"auth", "ops"}))
}

func TestWriteToolsRefreshIndexForReadOnlyTools(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	beforeSearch, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"query": "mcp-smoke-token-20260606-224145", "limit": 10},
	})
	require.NoError(t, err)
	require.False(t, beforeSearch.IsError)

	alphaResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "MCP Smoke Alpha",
			"path":  "mcp-smoke-alpha.md",
			"body":  "Alpha smoke body.",
			"tags":  []string{"mcp", "smoke"},
		},
	})
	require.NoError(t, err)
	require.False(t, alphaResult.IsError)
	alpha := decodeCreateNoteOutput(t, alphaResult)

	betaResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "MCP Smoke Beta",
			"body":  "Links to [[mcp-smoke-alpha]].\nUnique token: mcp-smoke-token-20260606-224145.",
		},
	})
	require.NoError(t, err)
	require.False(t, betaResult.IsError)
	beta := decodeCreateNoteOutput(t, betaResult)

	searchResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"query": `"mcp-smoke-token-20260606-224145"`, "limit": 10},
	})
	require.NoError(t, err)
	require.False(t, searchResult.IsError)
	searchOut := decodeSearchNotesOutput(t, searchResult)
	require.Len(t, searchOut.Hits, 1)
	require.Equal(t, beta.NoteID, searchOut.Hits[0].NoteID)

	backlinksResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_backlinks",
		Arguments: map[string]any{"identifier": alpha.NoteID, "limit": 10},
	})
	require.NoError(t, err)
	require.False(t, backlinksResult.IsError)
	backlinksOut := decodeListBacklinksOutput(t, backlinksResult)
	require.Len(t, backlinksOut.Links, 1)
	require.Equal(t, beta.NoteID, backlinksOut.Links[0].NoteID)
}

func TestEditNoteSupportsModes(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)

	t.Run("append", func(t *testing.T) {
		projectRoot := t.TempDir()
		seedMCPProject(t, projectRoot)
		session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

		created := decodeCreateNoteOutput(t, callCreateNote(t, session, "Append Target", "", nil, ""))
		readOut := decodeReadNoteOutput(t, callReadNote(t, session, created.NoteID))

		editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "edit_note",
			Arguments: map[string]any{
				"identifier": created.NoteID,
				"append":     "\n\n## Appendix\n\nMore content.\n",
			},
		})
		require.NoError(t, err)
		require.False(t, editResult.IsError)

		updated := decodeReadNoteOutput(t, callReadNote(t, session, created.NoteID))
		expectedBody := readOut.Note.Body + "\n\n## Appendix\n\nMore content.\n"
		require.Equal(t, expectedBody, updated.Note.Body)
	})

	t.Run("replace_body", func(t *testing.T) {
		projectRoot := t.TempDir()
		seedMCPProject(t, projectRoot)
		session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

		created := decodeCreateNoteOutput(t, callCreateNote(t, session, "Replace Target", "original body\n", nil, ""))
		readOut := decodeReadNoteOutput(t, callReadNote(t, session, created.NoteID))

		editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "edit_note",
			Arguments: map[string]any{
				"identifier":    created.NoteID,
				"replace_body":  "replaced body\n",
				"if_match_hash": readOut.Note.ContentHash,
			},
		})
		require.NoError(t, err)
		require.False(t, editResult.IsError)

		updated := decodeReadNoteOutput(t, callReadNote(t, session, created.NoteID))
		require.Equal(t, "replaced body\n", updated.Note.Body)
	})

	t.Run("merge_frontmatter", func(t *testing.T) {
		projectRoot := t.TempDir()
		seedMCPProject(t, projectRoot)
		session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

		created := decodeCreateNoteOutput(t, callCreateNote(t, session, "FM Target", "", nil, "decision"))

		editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "edit_note",
			Arguments: map[string]any{
				"identifier": created.NoteID,
				"merge_frontmatter": map[string]string{
					"status": "done",
				},
			},
		})
		require.NoError(t, err)
		require.False(t, editResult.IsError)

		updated := decodeReadNoteOutput(t, callReadNote(t, session, created.NoteID))
		require.Equal(t, "decision", updated.Note.Frontmatter["type"])
		require.Equal(t, "done", updated.Note.Frontmatter["status"])
	})
}

func TestEditNoteRejectsStaleHashWithoutChangingFile(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)
	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	created := decodeCreateNoteOutput(t, callCreateNote(t, session, "Stale Hash Edit", "original\n", nil, ""))
	readOut := decodeReadNoteOutput(t, callReadNote(t, session, created.NoteID))

	editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier":    created.NoteID,
			"replace_body":  "new body\n",
			"if_match_hash": readOut.Note.ContentHash + "-stale",
		},
	})
	require.NoError(t, err)
	require.True(t, editResult.IsError)
	require.Contains(t, resultText(t, editResult), "hash mismatch")

	updated := decodeReadNoteOutput(t, callReadNote(t, session, created.NoteID))
	require.Equal(t, "original\n", updated.Note.Body)
}

func TestEditNoteReplaceBodyRequiresIfMatchHash(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)
	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	_ = decodeCreateNoteOutput(t, callCreateNote(t, session, "Hashless Replace", "body\n", nil, ""))

	editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier":   "hashless-replace",
			"replace_body": "new body\n",
		},
	})
	require.NoError(t, err)
	require.True(t, editResult.IsError)
	require.Contains(t, resultText(t, editResult), "requires if_match_hash")
}

func TestDeleteNoteDefaultsToTrash(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	created := decodeCreateNoteOutput(t, callCreateNote(t, session, "Trashable", "", nil, ""))

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	require.NoError(t, err)
	require.False(t, deleteResult.IsError)

	out := decodeDeleteNoteOutput(t, deleteResult)
	require.True(t, out.Deleted)
	require.Equal(t, "trash", out.Mode)
	require.NotEmpty(t, out.TrashPath)

	memoryRootContent, err := os.ReadDir(memoryRoot)
	require.NoError(t, err)
	for _, entry := range memoryRootContent {
		if !entry.IsDir() {
			t.Errorf("expected memory root to be empty after deletion, found %s", entry.Name())
		}
	}
}

func TestDeleteNoteHardDeleteRemovesFile(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	created := decodeCreateNoteOutput(t, callCreateNote(t, session, "HardDeletable", "body\n", nil, ""))
	readOut := decodeReadNoteOutput(t, callReadNote(t, session, created.NoteID))

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "delete_note",
		Arguments: map[string]any{
			"identifier":    created.NoteID,
			"hard_delete":   true,
			"if_match_hash": readOut.Note.ContentHash,
		},
	})
	require.NoError(t, err)
	require.False(t, deleteResult.IsError)

	out := decodeDeleteNoteOutput(t, deleteResult)
	require.True(t, out.Deleted)
	require.Equal(t, "hard", out.Mode)
	require.Empty(t, out.TrashPath)
}

func TestDeleteNoteHardDeleteRequiresIfMatchHash(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)
	_ = decodeCreateNoteOutput(t, callCreateNote(t, session, "Hashless Delete", "", nil, ""))

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "delete_note",
		Arguments: map[string]any{
			"identifier":  "hashless-delete",
			"hard_delete": true,
		},
	})
	require.NoError(t, err)
	require.True(t, deleteResult.IsError)
	require.Contains(t, resultText(t, deleteResult), "requires if_match_hash")
}

func TestEnsurePathInsideRoot(t *testing.T) {
	root := "/tmp/memories"

	// Valid: path inside root
	require.NoError(t, ensurePathInsideRootHelper(root, "/tmp/memories/sub/note.md"))
	// Valid: path exactly equal to root
	require.NoError(t, ensurePathInsideRootHelper(root, "/tmp/memories"))
	// Invalid: path outside root
	require.Error(t, ensurePathInsideRootHelper(root, "/tmp/other/note.md"))
	// Invalid: traversal via symlink pointing outside
	require.Error(t, ensurePathInsideRootHelper(root, "/tmp/memories/../../etc/passwd"))
}

func ensurePathInsideRootHelper(root, target string) error {
	return tools.EnsurePathInsideRoot(root, target)
}

func TestDeleteNoteRejectsStaleHashWithoutDeletingFile(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	created := decodeCreateNoteOutput(t, callCreateNote(t, session, "Stale Delete", "original\n", nil, ""))
	readOut := decodeReadNoteOutput(t, callReadNote(t, session, created.NoteID))

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "delete_note",
		Arguments: map[string]any{
			"identifier":    created.NoteID,
			"hard_delete":   true,
			"if_match_hash": readOut.Note.ContentHash + "-stale",
		},
	})
	require.NoError(t, err)
	require.True(t, deleteResult.IsError)
	require.Contains(t, resultText(t, deleteResult), "hash mismatch")

	updated := decodeReadNoteOutput(t, callReadNote(t, session, created.NoteID))
	require.Equal(t, "original\n", updated.Note.Body)
}

func TestReadNoteReturnsPayload(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "test-note.md"), "550e8400-e29b-41d4-a716-446655440042", "Test Note", "test-note", nil, "test body\n")
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	byID, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": "550e8400-e29b-41d4-a716-446655440042"},
	})
	require.NoError(t, err)
	require.False(t, byID.IsError)
	byIDOut := decodeReadNoteOutput(t, byID)
	require.Equal(t, "Test Note", byIDOut.Note.Title)
	require.Equal(t, "test body\n", byIDOut.Note.Body)

	bySlug, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": "test-note"},
	})
	require.NoError(t, err)
	require.False(t, bySlug.IsError)
	bySlugOut := decodeReadNoteOutput(t, bySlug)
	require.Equal(t, byIDOut.Note.NoteID, bySlugOut.Note.NoteID)

	byTitle, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": "Test Note"},
	})
	require.NoError(t, err)
	require.False(t, byTitle.IsError)
	byTitleOut := decodeReadNoteOutput(t, byTitle)
	require.Equal(t, byIDOut.Note.NoteID, byTitleOut.Note.NoteID)
}

func TestReadNoteMissingReturnsToolError(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": "nonexistent-note"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestSearchNotesReturnsHits(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "alpha.md"), "550e8400-e29b-41d4-a716-446655440050", "Alpha", "alpha", []string{"tag1"}, "alpha body\n")
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "beta.md"), "550e8400-e29b-41d4-a716-446655440051", "Beta", "beta", []string{"tag2"}, "beta body with searchable content\n")
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"query": "searchable", "limit": 10},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	out := decodeSearchNotesOutput(t, result)
	require.Len(t, out.Hits, 1)
	require.Equal(t, "beta", out.Hits[0].Slug)
}

func TestSearchNotesMissingIndexSuggestsReindex(t *testing.T) {
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"query": "nothing", "limit": 10},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
	require.Contains(t, resultText(t, result), "index missing")
}

func TestGetIndexDB_nilServer(t *testing.T) {
	var s *Server
	_, err := s.GetIndexDB()
	require.Error(t, err)
	require.Contains(t, err.Error(), "server is nil")
}

func TestRebuildIndex_nilServer(t *testing.T) {
	var s *Server
	err := s.RebuildIndex("/tmp")
	require.Error(t, err)
	require.Contains(t, err.Error(), "server is nil")
}

func TestCloseIndexDB_nilServer(t *testing.T) {
	var s *Server
	err := s.closeIndexDB()
	require.NoError(t, err)
}

func TestCloseIndexDB_noConnection(t *testing.T) {
	s := &Server{}
	err := s.closeIndexDB()
	require.NoError(t, err)
}

func TestGetIndexDB_missingIndex(t *testing.T) {
	s := &Server{
		Project: app.ProjectResolution{
			Project: app.ProjectRecord{ID: "non-existent-uuid"},
		},
	}
	_, err := s.GetIndexDB()
	require.Error(t, err)
	require.Contains(t, err.Error(), "index missing")
}

func TestGetIndexDB_statError(t *testing.T) {
	// Use a server with a project ID that causes the index path to be a real
	// path that exists but is not a valid index file.
	// This exercises the stat error that is not IsNotExist.
	s := &Server{
		Project: app.ProjectResolution{
			Project: app.ProjectRecord{ID: "bad-uuid"},
		},
	}
	_, err := s.GetIndexDB()
	require.Error(t, err)
}

func TestRebuildIndex_closeError(t *testing.T) {
	// Test that RebuildIndex returns the closeIndexDB failure.
	s := &Server{
		testCloseDBErr: fmt.Errorf("simulated close error"),
	}
	err := s.RebuildIndex("/tmp")
	require.Error(t, err)
	require.Contains(t, err.Error(), "simulated close error")
}

func TestRebuildIndex_rebuildError(t *testing.T) {
	// Rebuild with a valid project that has no notes directory
	s := &Server{
		Project: app.ProjectResolution{
			Project: app.ProjectRecord{ID: "test-rebuild-error"},
		},
	}
	err := s.RebuildIndex("/nonexistent-dir")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Handler error paths - these exercise the closures registered via AddTool
// by calling them through MCP sessions.
// ---------------------------------------------------------------------------

func TestReadNote_missingReturnsToolError(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": "nonexistent"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestListBacklinks_missingIndexReturnsError(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_backlinks",
		Arguments: map[string]any{"identifier": "any-note"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestListTags_missingIndexReturnsError(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "list_tags",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestCreateNote_rebuildIndexSucceeds(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	seedMCPProject(t, projectRoot)

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Test Note",
		},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func connectToMCPServer(t *testing.T, repoRoot, projectRoot string) (*mcp.ClientSession, *captureWriter) {
	t.Helper()

	return connectToMCPServerWithEnv(t, repoRoot, projectRoot, writableMCPEnv(t))
}

func connectToMCPServerWithEnv(t *testing.T, repoRoot, projectRoot string, env []string) (*mcp.ClientSession, *captureWriter) {
	t.Helper()

	stderr := &captureWriter{}
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)

	effectivePaths, err := paths.ResolveEffectivePaths(paths.EffectiveInput{})
	require.NoError(t, err)

	// Resolve project using file-based registry
	entry, err := registry.Resolve(effectivePaths.MemoriesHome, "personal")
	require.NoError(t, err)

	appResolution := app.ProjectResolution{
		RepoRootAbs: entry.RepoRootAbs,
		MemoriesAbs: entry.MemoriesAbs,
		ManifestAbs: entry.ManifestPath,
		Project: app.ProjectRecord{
			ID:           entry.ProjectID,
			Name:         entry.Name,
			Slug:         entry.Slug,
			Kind:         entry.Type,
			MemoriesPath: filepath.Base(entry.MemoriesAbs),
		},
	}

	sdkServer := mcp.NewServer(
		&mcp.Implementation{Name: "mnemonic", Version: buildinfo.Version()},
		&mcp.ServerOptions{
			Capabilities: &mcp.ServerCapabilities{
				Tools: &mcp.ToolCapabilities{ListChanged: true},
			},
		},
	)
	server := NewServer(appResolution, effectivePaths)
	tools.RegisterAll(sdkServer, server, "")

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := sdkServer.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	})

	_ = env
	return clientSession, stderr
}

// toAppResolution adapts a registry.Entry to the app.ProjectResolution
// shape expected by NewServer.
func toAppResolution(entry registry.Entry) app.ProjectResolution {
	return app.ProjectResolution{
		RepoRootAbs: entry.RepoRootAbs,
		MemoriesAbs: entry.MemoriesAbs,
		ManifestAbs: entry.ManifestPath,
		Project: app.ProjectRecord{
			ID:           entry.ProjectID,
			Name:         entry.Name,
			Slug:         entry.Slug,
			Kind:         entry.Type,
			MemoriesPath: filepath.Base(entry.MemoriesAbs),
		},
	}
}

func memoriesPathFromEntry(entry registry.Entry) string {
	return filepath.Base(entry.MemoriesAbs)
}

// Decode helpers

func decodeListNotesOutput(t *testing.T, result *mcp.CallToolResult) tools.ListNotesOutput {
	t.Helper()

	var out tools.ListNotesOutput
	require.NoError(t, decodeListNotesPayload(t, result, &out))
	return out
}

func decodeListNotesPayload(t *testing.T, result *mcp.CallToolResult, out *tools.ListNotesOutput) error {
	t.Helper()

	return decodeAny(t, result, out)
}

func decodeCreateNoteOutput(t *testing.T, result *mcp.CallToolResult) tools.CreateNoteOutput {
	t.Helper()

	var out tools.CreateNoteOutput
	require.NoError(t, decodeCreateNotePayload(t, result, &out))
	return out
}

func decodeCreateNotePayload(t *testing.T, result *mcp.CallToolResult, out *tools.CreateNoteOutput) error {
	t.Helper()

	return decodeAny(t, result, out)
}

func decodeDeleteNoteOutput(t *testing.T, result *mcp.CallToolResult) tools.DeleteNoteOutput {
	t.Helper()

	var out tools.DeleteNoteOutput
	require.NoError(t, decodeDeleteNotePayload(t, result, &out))
	return out
}

func decodeDeleteNotePayload(t *testing.T, result *mcp.CallToolResult, out *tools.DeleteNoteOutput) error {
	t.Helper()

	return decodeAny(t, result, out)
}

func decodeListTagsOutput(t *testing.T, result *mcp.CallToolResult) tools.ListTagsOutput {
	t.Helper()

	var out tools.ListTagsOutput
	require.NoError(t, decodeListTagsPayload(t, result, &out))
	return out
}

func decodeListTagsPayload(t *testing.T, result *mcp.CallToolResult, out *tools.ListTagsOutput) error {
	t.Helper()

	return decodeAny(t, result, out)
}

func decodeListBacklinksOutput(t *testing.T, result *mcp.CallToolResult) tools.ListBacklinksOutput {
	t.Helper()

	var out tools.ListBacklinksOutput
	require.NoError(t, decodeListBacklinksPayload(t, result, &out))
	return out
}

func decodeListBacklinksPayload(t *testing.T, result *mcp.CallToolResult, out *tools.ListBacklinksOutput) error {
	t.Helper()

	return decodeAny(t, result, out)
}

func decodeReadNoteOutput(t *testing.T, result *mcp.CallToolResult) tools.ReadNoteOutput {
	t.Helper()

	var out tools.ReadNoteOutput
	require.NoError(t, decodeReadNotePayload(t, result, &out))
	return out
}

func decodeReadNotePayload(t *testing.T, result *mcp.CallToolResult, out *tools.ReadNoteOutput) error {
	t.Helper()

	return decodeAny(t, result, out)
}

func decodeSearchNotesOutput(t *testing.T, result *mcp.CallToolResult) tools.SearchNotesOutput {
	t.Helper()

	var out tools.SearchNotesOutput
	require.NoError(t, decodeSearchNotesPayload(t, result, &out))
	return out
}

func decodeSearchNotesPayload(t *testing.T, result *mcp.CallToolResult, out *tools.SearchNotesOutput) error {
	t.Helper()

	return decodeAny(t, result, out)
}

// decodeAny unmarshals StructuredContent or the first TextContent from a CallToolResult.
func decodeAny(t *testing.T, result *mcp.CallToolResult, out any) error {
	t.Helper()

	if result.StructuredContent != nil {
		data, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, out)
	}
	if len(result.Content) > 0 {
		raw, err := json.Marshal(result.Content[0])
		if err != nil {
			return err
		}
		var tc struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(raw, &tc); err != nil {
			return err
		}
		if tc.Text != "" {
			return json.Unmarshal([]byte(tc.Text), out)
		}
	}
	return nil
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if result == nil {
		return ""
	}

	if text := contentText(result.StructuredContent); text != "" {
		return text
	}
	for _, content := range result.Content {
		if text := contentString(content); text != "" {
			return text
		}
	}
	return ""
}

func contentText(v any) string {
	switch x := v.(type) {
	case json.RawMessage:
		return string(x)
	case map[string]any:
		raw, err := json.Marshal(x)
		if err != nil {
			return ""
		}
		return string(raw)
	default:
		return ""
	}
}

func contentString(content mcp.Content) string {
	switch v := content.(type) {
	case *mcp.TextContent:
		return v.Text
	case *mcp.ToolResultContent:
		if text := contentText(v.StructuredContent); text != "" {
			return text
		}
		for _, nested := range v.Content {
			if text := contentString(nested); text != "" {
				return text
			}
		}
	}
	return ""
}

func repoRootForTest(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func seedMCPProject(t *testing.T, projectRoot string) {
	t.Helper()

	_ = writableMCPEnv(t)

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)

	memoriesDir := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoriesDir, 0o755))

	// Write manifest
	manifest := project.NewMnemonicManifest()
	manifest.ProjectID = "550e8400-e29b-41d4-a716-446655440000"
	manifest.Name = "personal"
	manifest.Slug = "personal"
	manifest.Type = project.ManifestTypeLocal
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = now
	manifest.UpdatedAt = now
	manifest.Generator.App = "mnemonic"
	require.NoError(t, project.WriteMnemonicManifest(filepath.Join(memoriesDir, "mnemonic.toml"), manifest))

	// Write pointer file in memories home
	mnemonicPaths, err := paths.GetMnemonicPaths()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(mnemonicPaths.MemoriesHome, 0o755))
	require.NoError(t, project.WritePointerFile(filepath.Join(mnemonicPaths.MemoriesHome, "personal.toml"), &project.PointerFile{
		ManifestPath: filepath.Join(memoriesDir, "mnemonic.toml"),
	}))
}

// mcpTestBaseByName maps test names to their persistent base directory so the
// registry written during fixture setup remains visible to the connect helpers
// within a single test run.
var mcpTestBaseByName = map[string]string{}

func mcpTestBaseFor(t *testing.T) string {
	t.Helper()

	name := t.Name()
	if existing, ok := mcpTestBaseByName[name]; ok {
		return existing
	}
	base := t.TempDir()
	mcpTestBaseByName[name] = base
	return base
}

func TestMain(m *testing.M) {
	// Reset the per-test base map so the registry state is consistent across
	// tests in the same package run.
	mcpTestBaseByName = map[string]string{}
	os.Exit(m.Run())
}

func writableMCPEnv(t *testing.T) []string {
	t.Helper()

	base := mcpTestBaseFor(t)
	t.Setenv("HOME", base)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(base, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(base, "cache"))
	t.Setenv("GOCACHE", filepath.Join(base, "go-build"))
	t.Setenv("GOMODCACHE", "/home/ilyachch/.cache/go/pkg/mod")
	t.Setenv("GOSUMDB", "off")

	env := os.Environ()
	env = append(env,
		"HOME="+base,
		"XDG_CONFIG_HOME="+filepath.Join(base, "config"),
		"XDG_DATA_HOME="+filepath.Join(base, "data"),
		"XDG_STATE_HOME="+filepath.Join(base, "state"),
		"XDG_CACHE_HOME="+filepath.Join(base, "cache"),
		"GOCACHE="+filepath.Join(base, "go-build"),
		"GOMODCACHE=/home/ilyachch/.cache/go/pkg/mod",
		"GOSUMDB=off",
	)
	return env
}

type captureWriter struct {
	data []byte
}

func (w *captureWriter) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *captureWriter) String() string {
	return string(w.data)
}

// writeTaggedMCPNote writes a note with tags metadata.
func writeTaggedMCPNote(t *testing.T, path, noteID, title, slug string, tags []string, body string) {
	t.Helper()

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	fm := map[string]any{}
	if tags != nil {
		fm["tags"] = tags
	}
	note := markdown.Note{
		MnemonicNoteID: noteID,
		Title:          title,
		Slug:           slug,
		Tags:           tags,
		Frontmatter:    fm,
		CreatedAt:      now,
		UpdatedAt:      now,
		Body:           []byte(body),
	}
	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, rendered, 0o644))
}

func anySliceToStrings(t *testing.T, s any) []string {
	t.Helper()

	if s == nil {
		return nil
	}
	slice, ok := s.([]any)
	require.True(t, ok, "expected []any, got %T", s)
	out := make([]string, len(slice))
	for i, v := range slice {
		out[i] = v.(string)
	}
	return out
}

// writeWikiMCPNote writes a note with wiki-style body (for backlinks).
func writeWikiMCPNote(t *testing.T, path, noteID, title, slug, body string) {
	t.Helper()

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	note := markdown.Note{
		MnemonicNoteID: noteID,
		Title:          title,
		Slug:           slug,
		CreatedAt:      now,
		UpdatedAt:      now,
		Body:           []byte(body),
	}
	rendered, err := markdown.RenderNote(note)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, rendered, 0o644))
}

func assertToolAnnotations(t *testing.T, tools []*mcp.Tool, name string, readOnly, destructive bool) {
	t.Helper()

	for _, tool := range tools {
		if tool.Name == name {
			require.Equal(t, readOnly, tool.Annotations.ReadOnlyHint, "tool %s: readOnlyHint", name)
			if destructive {
				require.True(t, tool.Annotations.DestructiveHint != nil && *tool.Annotations.DestructiveHint,
					"tool %s: destructiveHint should be true", name)
			}
			return
		}
	}
	t.Errorf("tool %s not found", name)
}

// callCreateNote calls create_note with the given arguments.
func callCreateNote(t *testing.T, session *mcp.ClientSession, title, body string, tags []string, noteType string) *mcp.CallToolResult {
	t.Helper()

	args := map[string]any{"title": title}
	if body != "" {
		args["body"] = body
	}
	if tags != nil {
		args["tags"] = tags
	}
	if noteType != "" {
		args["type"] = noteType
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "create_note", Arguments: args})
	require.NoError(t, err)
	return result
}

// callReadNote calls read_note with the given identifier.
func callReadNote(t *testing.T, session *mcp.ClientSession, identifier string) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": identifier},
	})
	require.NoError(t, err)
	return result
}
