package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ilyachch/mnemonic/internal/buildinfo"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCommandTransportInitializeAndListTools(t *testing.T) {
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

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
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

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
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

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
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

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
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "auth-one.md"), "550e8400-e29b-41d4-a716-446655440001", "Auth One", "auth-one", []string{"auth"}, "auth one body\n")
	_, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	resolvedProject, err := project.ResolveProject(project.ResolveProjectInput{
		CWD:             projectRoot,
		ProjectSelector: "personal",
	})
	require.NoError(t, err)
	effectivePaths, err := paths.ResolveEffectivePaths(paths.EffectiveInput{})
	require.NoError(t, err)

	server := NewServer(resolvedProject, effectivePaths)
	t.Cleanup(func() {
		_ = server.closeIndexDB()
	})

	firstDB, err := server.indexDB()
	require.NoError(t, err)
	secondDB, err := server.indexDB()
	require.NoError(t, err)
	require.Same(t, firstDB, secondDB)
}

func TestListBacklinksReturnsLinksAndRespectsLimit(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

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
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

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
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

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
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

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
		writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))
		session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

		createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "create_note",
			Arguments: map[string]any{
				"title": "Auth Migration",
				"body":  "## Summary\n",
			},
		})
		require.NoError(t, err)
		created := decodeCreateNoteOutput(t, createResult)

		editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "edit_note",
			Arguments: map[string]any{
				"identifier": created.NoteID,
				"append":     "Next step",
			},
		})
		require.NoError(t, err)
		require.False(t, editResult.IsError)

		readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "read_note",
			Arguments: map[string]any{"identifier": created.NoteID},
		})
		require.NoError(t, err)
		read := decodeReadNoteOutput(t, readResult)
		require.Equal(t, "## Summary\nNext step", read.Note.Body)
	})

	t.Run("replace_body", func(t *testing.T) {
		projectRoot := t.TempDir()
		writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))
		session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

		createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "create_note",
			Arguments: map[string]any{
				"title": "Body Replace",
				"body":  "Old body\n",
			},
		})
		require.NoError(t, err)
		created := decodeCreateNoteOutput(t, createResult)

		readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "read_note",
			Arguments: map[string]any{"identifier": created.NoteID},
		})
		require.NoError(t, err)
		initialRead := decodeReadNoteOutput(t, readResult)

		editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "edit_note",
			Arguments: map[string]any{
				"identifier":    created.NoteID,
				"replace_body":  "Replacement body\n",
				"if_match_hash": initialRead.Note.ContentHash,
			},
		})
		require.NoError(t, err)
		require.False(t, editResult.IsError)

		readResult, err = session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "read_note",
			Arguments: map[string]any{"identifier": created.NoteID},
		})
		require.NoError(t, err)
		finalRead := decodeReadNoteOutput(t, readResult)
		require.Equal(t, "Replacement body\n", finalRead.Note.Body)
	})

	t.Run("merge_frontmatter", func(t *testing.T) {
		projectRoot := t.TempDir()
		writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))
		session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

		createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "create_note",
			Arguments: map[string]any{
				"title": "Frontmatter Merge",
			},
		})
		require.NoError(t, err)
		created := decodeCreateNoteOutput(t, createResult)

		editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "edit_note",
			Arguments: map[string]any{
				"identifier": created.NoteID,
				"merge_frontmatter": map[string]string{
					"type":       "decision",
					"permalink":  "/decisions/frontmatter-merge",
					"custom_key": "custom-value",
				},
			},
		})
		require.NoError(t, err)
		require.False(t, editResult.IsError)

		readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "read_note",
			Arguments: map[string]any{"identifier": created.NoteID},
		})
		require.NoError(t, err)
		read := decodeReadNoteOutput(t, readResult)
		require.Equal(t, "decision", read.Note.Frontmatter["type"])
		require.Equal(t, "custom-value", read.Note.Frontmatter["custom_key"])
	})
}

func TestEditNoteRejectsStaleHashWithoutChangingFile(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Auth Migration",
			"body":  "## Summary\n",
		},
	})
	require.NoError(t, err)
	created := decodeCreateNoteOutput(t, createResult)

	firstEdit, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier":    created.NoteID,
			"if_match_hash": created.ContentHash,
			"append":        "A",
		},
	})
	require.NoError(t, err)
	require.False(t, firstEdit.IsError)

	secondEdit, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier":    created.NoteID,
			"if_match_hash": created.ContentHash,
			"append":        "B",
		},
	})
	require.NoError(t, err)
	require.True(t, secondEdit.IsError)
	require.Contains(t, resultText(t, secondEdit), "content hash mismatch")

	readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	require.NoError(t, err)
	read := decodeReadNoteOutput(t, readResult)
	require.Equal(t, "## Summary\nA", read.Note.Body)
}

func TestEditNoteReplaceBodyRequiresIfMatchHash(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Replace Protected",
			"body":  "Original body\n",
		},
	})
	require.NoError(t, err)
	created := decodeCreateNoteOutput(t, createResult)

	editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier":   created.NoteID,
			"replace_body": "Replacement body\n",
		},
	})
	require.NoError(t, err)
	require.True(t, editResult.IsError)
	require.Contains(t, resultText(t, editResult), "replace_body requires if_match_hash from read_note")

	readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	require.NoError(t, err)
	read := decodeReadNoteOutput(t, readResult)
	require.Equal(t, "Original body\n", read.Note.Body)
}

func TestDeleteNoteDefaultsToTrash(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Trash Me",
			"body":  "body\n",
		},
	})
	require.NoError(t, err)
	created := decodeCreateNoteOutput(t, createResult)

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	require.NoError(t, err)
	require.False(t, deleteResult.IsError)

	deleted := decodeDeleteNoteOutput(t, deleteResult)
	require.True(t, deleted.Deleted)
	require.Equal(t, "trash", deleted.Mode)
	require.Equal(t, "trash-me.md", deleted.Path)
	require.NotEmpty(t, deleted.TrashPath)
	_, err = os.Stat(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "trash-me.md"))
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(deleted.TrashPath)
	require.NoError(t, err)
}

func TestDeleteNoteHardDeleteRemovesFile(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Hard Delete",
			"body":  "body\n",
		},
	})
	require.NoError(t, err)
	created := decodeCreateNoteOutput(t, createResult)

	readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	require.NoError(t, err)
	read := decodeReadNoteOutput(t, readResult)

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "delete_note",
		Arguments: map[string]any{
			"identifier":    created.NoteID,
			"hard_delete":   true,
			"if_match_hash": read.Note.ContentHash,
		},
	})
	require.NoError(t, err)
	require.False(t, deleteResult.IsError)

	deleted := decodeDeleteNoteOutput(t, deleteResult)
	require.Equal(t, "hard", deleted.Mode)
	require.Empty(t, deleted.TrashPath)
	_, err = os.Stat(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "hard-delete.md"))
	require.True(t, os.IsNotExist(err))
}

func TestDeleteNoteHardDeleteRequiresIfMatchHash(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Hard Delete Protected",
			"body":  "body\n",
		},
	})
	require.NoError(t, err)
	created := decodeCreateNoteOutput(t, createResult)

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "delete_note",
		Arguments: map[string]any{
			"identifier":  created.NoteID,
			"hard_delete": true,
		},
	})
	require.NoError(t, err)
	require.True(t, deleteResult.IsError)
	require.Contains(t, resultText(t, deleteResult), "hard delete requires if_match_hash from read_note")
	_, err = os.Stat(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "hard-delete-protected.md"))
	require.NoError(t, err)
}

func TestEnsurePathInsideRoot(t *testing.T) {
	root := t.TempDir()

	inside := filepath.Join(root, "nested", "note.md")
	require.NoError(t, ensurePathInsideRoot(root, inside))

	outside := filepath.Join(root, "..", "outside.md")
	err := ensurePathInsideRoot(root, outside)
	require.Error(t, err)
	require.Contains(t, err.Error(), "note path must stay inside the project memories root")
}

func TestDeleteNoteRejectsStaleHashWithoutDeletingFile(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	createResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Delete Protected",
			"body":  "body\n",
		},
	})
	require.NoError(t, err)
	created := decodeCreateNoteOutput(t, createResult)

	editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier": created.NoteID,
			"append":     "updated",
		},
	})
	require.NoError(t, err)
	require.False(t, editResult.IsError)

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "delete_note",
		Arguments: map[string]any{
			"identifier":    created.NoteID,
			"if_match_hash": created.ContentHash,
		},
	})
	require.NoError(t, err)
	require.True(t, deleteResult.IsError)
	require.Contains(t, resultText(t, deleteResult), "content hash mismatch")
	_, err = os.Stat(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "delete-protected.md"))
	require.NoError(t, err)
}

func writeTaggedMCPNote(t *testing.T, path, noteID, title, slug string, tags []string, body string) {
	t.Helper()

	content := "---\n"
	content += "mnemonic_note_id: " + noteID + "\n"
	content += "title: " + title + "\n"
	content += "slug: " + slug + "\n"
	if len(tags) > 0 {
		content += "tags:\n"
		for _, tag := range tags {
			content += "  - " + tag + "\n"
		}
	}
	content += "created_at: 2026-06-02T12:34:56Z\n"
	content += "updated_at: 2026-06-02T12:34:56Z\n"
	content += "---\n"
	content += body

	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func anySliceToStrings(t *testing.T, value any) []string {
	t.Helper()

	switch items := value.(type) {
	case []string:
		return append([]string(nil), items...)
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			s, ok := item.(string)
			require.True(t, ok, "item %#v is not string", item)
			out = append(out, s)
		}
		return out
	}

	require.FailNow(t, "value %#v is not []string or []any", value)
	return nil
}

func writeWikiMCPNote(t *testing.T, path, noteID, title, slug, body string) {
	t.Helper()

	content := "---\n"
	content += "mnemonic_note_id: " + noteID + "\n"
	content += "title: " + title + "\n"
	content += "slug: " + slug + "\n"
	content += "created_at: 2026-06-02T12:34:56Z\n"
	content += "updated_at: 2026-06-02T12:34:56Z\n"
	content += "---\n"
	content += body

	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func assertToolAnnotations(t *testing.T, tools []*mcp.Tool, name string, readOnly bool, destructive bool) {
	t.Helper()

	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		require.NotNil(t, tool.Annotations)
		require.Equal(t, readOnly, tool.Annotations.ReadOnlyHint)
		if !readOnly {
			require.NotNil(t, tool.Annotations.DestructiveHint)
			require.Equal(t, destructive, *tool.Annotations.DestructiveHint)
		}
		return
	}

	require.FailNow(t, "tool %q not found", name)
}

func TestReadNoteReturnsPayload(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	created, err := notes.Create(notes.CreateInput{
		RootDir: memoryRoot,
		Title:   "Alpha Note",
		Body:    []byte("alpha body\n"),
	})
	require.NoError(t, err)

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	out := decodeReadNoteOutput(t, result)
	require.Equal(t, "alpha body\n", out.Note.Body)
	require.NotEmpty(t, out.Note.ContentHash)
	require.Equal(t, "Alpha Note", out.Note.Frontmatter["title"])
}

func TestReadNoteMissingReturnsToolError(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": "missing-note"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestSearchNotesReturnsHits(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	require.NoError(t, os.MkdirAll(memoryRoot, 0o755))
	_, err := notes.Create(notes.CreateInput{
		RootDir: memoryRoot,
		Title:   "Searchable Note",
		Body:    []byte("alpha beta gamma\n"),
	})
	require.NoError(t, err)
	_, err = index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot)
	require.NoError(t, err)

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"query": "alpha", "limit": 10},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	out := decodeSearchNotesOutput(t, result)
	require.Len(t, out.Hits, 1)
	hit := out.Hits[0]
	require.NotEmpty(t, hit.NoteID)
	require.NotEmpty(t, hit.Slug)
	require.NotEmpty(t, hit.Title)
	require.NotEmpty(t, hit.Path)
	require.NotEmpty(t, hit.Snippet)
	require.NotEmpty(t, hit.ContentHash)
	require.Equal(t, "searchable-note", hit.Slug)
}

func TestSearchNotesMissingIndexSuggestsReindex(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"query": "alpha"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
	require.Contains(t, resultText(t, result), "mnemonic project reindex")
}

func connectToMCPServer(t *testing.T, repoRoot, projectRoot string) (*mcp.ClientSession, *captureWriter) {
	t.Helper()

	return connectToMCPServerWithEnv(t, repoRoot, projectRoot, writableMCPEnv(t))
}

func connectToMCPServerWithEnv(t *testing.T, repoRoot, projectRoot string, env []string) (*mcp.ClientSession, *captureWriter) {
	t.Helper()

	stderr := &captureWriter{}
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	resolvedProject, err := project.ResolveProject(project.ResolveProjectInput{
		CWD:             projectRoot,
		ProjectSelector: "personal",
	})
	require.NoError(t, err)
	effectivePaths, err := paths.ResolveEffectivePaths(paths.EffectiveInput{})
	require.NoError(t, err)

	sdkServer := mcp.NewServer(
		&mcp.Implementation{Name: "mnemonic", Version: buildinfo.Version()},
		&mcp.ServerOptions{
			Capabilities: &mcp.ServerCapabilities{
				Tools: &mcp.ToolCapabilities{ListChanged: true},
			},
		},
	)
	server := NewServer(resolvedProject, effectivePaths)
	registerReadOnlyTools(sdkServer, server)
	registerWriteTools(sdkServer, server)

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

func decodeListNotesOutput(t *testing.T, result *mcp.CallToolResult) listNotesOutput {
	t.Helper()

	require.NotNil(t, result)

	if out, ok := decodeListNotesPayload(result.StructuredContent); ok {
		return out
	}

	for _, content := range result.Content {
		if out, ok := decodeListNotesContent(t, content); ok {
			return out
		}
	}

	require.FailNow(t, "result has no decodable list_notes payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return listNotesOutput{}
}

func decodeCreateNoteOutput(t *testing.T, result *mcp.CallToolResult) createNoteOutput {
	t.Helper()

	require.NotNil(t, result)

	if structured, ok := decodeCreateNotePayload(t, result.StructuredContent); ok {
		return structured
	}
	for _, content := range result.Content {
		if structured, ok := decodeCreateNoteContent(t, content); ok {
			return structured
		}
	}

	require.FailNow(t, "result has no decodable create_note payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return createNoteOutput{}
}

func decodeEditNoteOutput(t *testing.T, result *mcp.CallToolResult) editNoteOutput {
	t.Helper()

	require.NotNil(t, result)

	if structured, ok := decodeEditNotePayload(t, result.StructuredContent); ok {
		return structured
	}
	for _, content := range result.Content {
		if structured, ok := decodeEditNoteContent(t, content); ok {
			return structured
		}
	}

	require.FailNow(t, "result has no decodable edit_note payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return editNoteOutput{}
}

func decodeDeleteNoteOutput(t *testing.T, result *mcp.CallToolResult) deleteNoteOutput {
	t.Helper()

	require.NotNil(t, result)

	if structured, ok := decodeDeleteNotePayload(t, result.StructuredContent); ok {
		return structured
	}
	for _, content := range result.Content {
		if structured, ok := decodeDeleteNoteContent(t, content); ok {
			return structured
		}
	}

	require.FailNow(t, "result has no decodable delete_note payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return deleteNoteOutput{}
}

func decodeListNotesPayload(structured any) (listNotesOutput, bool) {
	switch v := structured.(type) {
	case json.RawMessage:
		var out listNotesOutput
		if err := json.Unmarshal(v, &out); err != nil {
			return listNotesOutput{}, false
		}
		return out, true
	case map[string]any:
		raw, err := json.Marshal(v)
		if err != nil {
			return listNotesOutput{}, false
		}
		var out listNotesOutput
		if err := json.Unmarshal(raw, &out); err != nil {
			return listNotesOutput{}, false
		}
		return out, true
	default:
		return listNotesOutput{}, false
	}
}

func decodeListNotesContent(t *testing.T, content mcp.Content) (listNotesOutput, bool) {
	switch v := content.(type) {
	case *mcp.TextContent:
		var out listNotesOutput
		if err := json.Unmarshal([]byte(v.Text), &out); err != nil {
			t.Logf("list_notes text payload: %q", v.Text)
			return listNotesOutput{}, false
		}
		return out, true
	case *mcp.ToolResultContent:
		if out, ok := decodeListNotesPayload(v.StructuredContent); ok {
			return out, true
		}
		for _, nested := range v.Content {
			if out, ok := decodeListNotesContent(t, nested); ok {
				return out, true
			}
		}
		return listNotesOutput{}, false
	default:
		return listNotesOutput{}, false
	}
}

func decodeCreateNotePayload(t *testing.T, structured any) (createNoteOutput, bool) {
	t.Helper()

	switch v := structured.(type) {
	case createNoteOutput:
		return v, true
	case *createNoteOutput:
		if v != nil {
			return *v, true
		}
	case map[string]any:
		data, err := json.Marshal(v)
		require.NoError(t, err)
		var out createNoteOutput
		require.NoError(t, json.Unmarshal(data, &out))
		return out, true
	}

	return createNoteOutput{}, false
}

func decodeCreateNoteContent(t *testing.T, content mcp.Content) (createNoteOutput, bool) {
	t.Helper()

	switch v := content.(type) {
	case *mcp.TextContent:
		var out createNoteOutput
		if err := json.Unmarshal([]byte(v.Text), &out); err != nil {
			t.Logf("create_note text payload: %q", v.Text)
			require.NoError(t, err)
		}
		return out, true
	case *mcp.ToolResultContent:
		return decodeCreateNotePayload(t, v.StructuredContent)
	}

	return createNoteOutput{}, false
}

func decodeEditNotePayload(t *testing.T, structured any) (editNoteOutput, bool) {
	t.Helper()

	switch v := structured.(type) {
	case editNoteOutput:
		return v, true
	case *editNoteOutput:
		if v != nil {
			return *v, true
		}
	case map[string]any:
		data, err := json.Marshal(v)
		require.NoError(t, err)
		var out editNoteOutput
		require.NoError(t, json.Unmarshal(data, &out))
		return out, true
	}

	return editNoteOutput{}, false
}

func decodeEditNoteContent(t *testing.T, content mcp.Content) (editNoteOutput, bool) {
	t.Helper()

	switch v := content.(type) {
	case *mcp.TextContent:
		var out editNoteOutput
		if err := json.Unmarshal([]byte(v.Text), &out); err != nil {
			t.Logf("edit_note text payload: %q", v.Text)
			require.NoError(t, err)
		}
		return out, true
	case *mcp.ToolResultContent:
		return decodeEditNotePayload(t, v.StructuredContent)
	}

	return editNoteOutput{}, false
}

func decodeDeleteNotePayload(t *testing.T, structured any) (deleteNoteOutput, bool) {
	t.Helper()

	switch v := structured.(type) {
	case deleteNoteOutput:
		return v, true
	case *deleteNoteOutput:
		if v != nil {
			return *v, true
		}
	case map[string]any:
		data, err := json.Marshal(v)
		require.NoError(t, err)
		var out deleteNoteOutput
		require.NoError(t, json.Unmarshal(data, &out))
		return out, true
	}

	return deleteNoteOutput{}, false
}

func decodeDeleteNoteContent(t *testing.T, content mcp.Content) (deleteNoteOutput, bool) {
	t.Helper()

	switch v := content.(type) {
	case *mcp.TextContent:
		var out deleteNoteOutput
		if err := json.Unmarshal([]byte(v.Text), &out); err != nil {
			t.Logf("delete_note text payload: %q", v.Text)
			require.NoError(t, err)
		}
		return out, true
	case *mcp.ToolResultContent:
		return decodeDeleteNotePayload(t, v.StructuredContent)
	}

	return deleteNoteOutput{}, false
}

func decodeListTagsOutput(t *testing.T, result *mcp.CallToolResult) listTagsOutput {
	t.Helper()

	require.NotNil(t, result)

	if out, ok := decodeListTagsPayload(result.StructuredContent); ok {
		return out
	}

	for _, content := range result.Content {
		if out, ok := decodeListTagsContent(t, content); ok {
			return out
		}
	}

	require.FailNow(t, "result has no decodable list_tags payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return listTagsOutput{}
}

func decodeListBacklinksOutput(t *testing.T, result *mcp.CallToolResult) listBacklinksOutput {
	t.Helper()

	if out, ok := decodeListBacklinksPayload(result.StructuredContent); ok {
		return out
	}
	for _, content := range result.Content {
		if out, ok := decodeListBacklinksContent(t, content); ok {
			return out
		}
	}
	require.FailNow(t, "result has no decodable list_backlinks payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return listBacklinksOutput{}
}

func decodeListBacklinksPayload(structured any) (listBacklinksOutput, bool) {
	if structured == nil {
		return listBacklinksOutput{}, false
	}
	data, err := json.Marshal(structured)
	if err != nil {
		return listBacklinksOutput{}, false
	}
	var out listBacklinksOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return listBacklinksOutput{}, false
	}
	return out, true
}

func decodeListBacklinksContent(t *testing.T, content mcp.Content) (listBacklinksOutput, bool) {
	t.Helper()

	switch v := content.(type) {
	case *mcp.TextContent:
		var out listBacklinksOutput
		if err := json.Unmarshal([]byte(v.Text), &out); err == nil {
			return out, true
		}
	case *mcp.ToolResultContent:
		if out, ok := decodeListBacklinksPayload(v.StructuredContent); ok {
			return out, true
		}
		for _, nested := range v.Content {
			if out, ok := decodeListBacklinksContent(t, nested); ok {
				return out, true
			}
		}
	}
	return listBacklinksOutput{}, false
}

func decodeListTagsPayload(structured any) (listTagsOutput, bool) {
	switch v := structured.(type) {
	case json.RawMessage:
		var out listTagsOutput
		if err := json.Unmarshal(v, &out); err != nil {
			return listTagsOutput{}, false
		}
		return out, true
	case map[string]any:
		raw, err := json.Marshal(v)
		if err != nil {
			return listTagsOutput{}, false
		}
		var out listTagsOutput
		if err := json.Unmarshal(raw, &out); err != nil {
			return listTagsOutput{}, false
		}
		return out, true
	default:
		return listTagsOutput{}, false
	}
}

func decodeListTagsContent(t *testing.T, content mcp.Content) (listTagsOutput, bool) {
	switch v := content.(type) {
	case *mcp.TextContent:
		var out listTagsOutput
		if err := json.Unmarshal([]byte(v.Text), &out); err != nil {
			t.Logf("list_tags text payload: %q", v.Text)
			return listTagsOutput{}, false
		}
		return out, true
	case *mcp.ToolResultContent:
		if out, ok := decodeListTagsPayload(v.StructuredContent); ok {
			return out, true
		}
		for _, nested := range v.Content {
			if out, ok := decodeListTagsContent(t, nested); ok {
				return out, true
			}
		}
		return listTagsOutput{}, false
	default:
		return listTagsOutput{}, false
	}
}

func decodeReadNoteOutput(t *testing.T, result *mcp.CallToolResult) readNoteOutput {
	t.Helper()

	require.NotNil(t, result)

	if out, ok := decodeReadNotePayload(result.StructuredContent); ok {
		return out
	}

	for _, content := range result.Content {
		if out, ok := decodeReadNoteContent(t, content); ok {
			return out
		}
	}

	require.FailNow(t, "result has no decodable read_note payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return readNoteOutput{}
}

func decodeReadNotePayload(structured any) (readNoteOutput, bool) {
	switch v := structured.(type) {
	case json.RawMessage:
		var out readNoteOutput
		if err := json.Unmarshal(v, &out); err != nil {
			return readNoteOutput{}, false
		}
		return out, true
	case map[string]any:
		raw, err := json.Marshal(v)
		if err != nil {
			return readNoteOutput{}, false
		}
		var out readNoteOutput
		if err := json.Unmarshal(raw, &out); err != nil {
			return readNoteOutput{}, false
		}
		return out, true
	default:
		return readNoteOutput{}, false
	}
}

func decodeReadNoteContent(t *testing.T, content mcp.Content) (readNoteOutput, bool) {
	switch v := content.(type) {
	case *mcp.TextContent:
		var out readNoteOutput
		if err := json.Unmarshal([]byte(v.Text), &out); err != nil {
			t.Logf("read_note text payload: %q", v.Text)
			return readNoteOutput{}, false
		}
		return out, true
	case *mcp.ToolResultContent:
		if out, ok := decodeReadNotePayload(v.StructuredContent); ok {
			return out, true
		}
		for _, nested := range v.Content {
			if out, ok := decodeReadNoteContent(t, nested); ok {
				return out, true
			}
		}
		return readNoteOutput{}, false
	default:
		return readNoteOutput{}, false
	}
}

func decodeSearchNotesOutput(t *testing.T, result *mcp.CallToolResult) searchNotesOutput {
	t.Helper()

	require.NotNil(t, result)

	if out, ok := decodeSearchNotesPayload(result.StructuredContent); ok {
		return out
	}

	for _, content := range result.Content {
		if out, ok := decodeSearchNotesContent(t, content); ok {
			return out
		}
	}

	require.FailNow(t, "result has no decodable search_notes payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return searchNotesOutput{}
}

func decodeSearchNotesPayload(structured any) (searchNotesOutput, bool) {
	switch v := structured.(type) {
	case json.RawMessage:
		var out searchNotesOutput
		if err := json.Unmarshal(v, &out); err != nil {
			return searchNotesOutput{}, false
		}
		return out, true
	case map[string]any:
		raw, err := json.Marshal(v)
		if err != nil {
			return searchNotesOutput{}, false
		}
		var out searchNotesOutput
		if err := json.Unmarshal(raw, &out); err != nil {
			return searchNotesOutput{}, false
		}
		return out, true
	default:
		return searchNotesOutput{}, false
	}
}

func decodeSearchNotesContent(t *testing.T, content mcp.Content) (searchNotesOutput, bool) {
	switch v := content.(type) {
	case *mcp.TextContent:
		var out searchNotesOutput
		if err := json.Unmarshal([]byte(v.Text), &out); err != nil {
			t.Logf("search_notes text payload: %q", v.Text)
			return searchNotesOutput{}, false
		}
		return out, true
	case *mcp.ToolResultContent:
		if out, ok := decodeSearchNotesPayload(v.StructuredContent); ok {
			return out, true
		}
		for _, nested := range v.Content {
			if out, ok := decodeSearchNotesContent(t, nested); ok {
				return out, true
			}
		}
		return searchNotesOutput{}, false
	default:
		return searchNotesOutput{}, false
	}
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

func writeMCPMnemonicFile(t *testing.T, path string) {
	t.Helper()

	now := time.Date(2026, time.June, 2, 10, 0, 0, 0, time.UTC)
	file := project.NewMnemonicFile()
	file.CreatedAt = now
	file.UpdatedAt = now
	file.Projects = []project.MnemonicProject{
		{
			ID:                    "550e8400-e29b-41d4-a716-446655440000",
			Name:                  "personal",
			Slug:                  "personal",
			Kind:                  project.ProjectKindLocal,
			MemoriesPath:          ".mnemonic-memories/personal",
			MarkdownFormatVersion: 1,
			CreatedAt:             now,
			UpdatedAt:             now,
		},
	}
	require.NoError(t, project.WriteMnemonicFile(path, file))
}

func writableMCPEnv(t *testing.T) []string {
	t.Helper()

	base := t.TempDir()
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