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
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) != 8 {
		t.Fatalf("ListTools() tools = %d, want 8", len(tools.Tools))
	}
	gotNames := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		if strings.Contains(tool.Name, " ") {
			t.Fatalf("tool name %q contains spaces", tool.Name)
		}
		if tool.Annotations == nil {
			t.Fatalf("tool %q is missing annotations", tool.Name)
		}
		gotNames = append(gotNames, tool.Name)
	}
	wantNames := []string{"create_note", "delete_note", "edit_note", "list_backlinks", "list_notes", "list_tags", "read_note", "search_notes"}
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("tool names = %v, want %v", gotNames, wantNames)
	}
	assertToolAnnotations(t, tools.Tools, "create_note", false, false)
	assertToolAnnotations(t, tools.Tools, "edit_note", false, true)
	assertToolAnnotations(t, tools.Tools, "delete_note", false, true)
	assertToolAnnotations(t, tools.Tools, "list_backlinks", true, false)
	assertToolAnnotations(t, tools.Tools, "list_notes", true, false)
	assertToolAnnotations(t, tools.Tools, "list_tags", true, false)
	assertToolAnnotations(t, tools.Tools, "read_note", true, false)
	assertToolAnnotations(t, tools.Tools, "search_notes", true, false)

	gotSnapshot, err := json.MarshalIndent(tools.Tools, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	wantSnapshot, err := os.ReadFile(filepath.Join(repoRoot, "internal", "mcp", "testdata", "read_only_tools.snapshot.json"))
	if err != nil {
		t.Fatalf("ReadFile(snapshot) error = %v\nactual:\n%s", err, gotSnapshot)
	}
	if strings.TrimSpace(string(gotSnapshot)) != strings.TrimSpace(string(wantSnapshot)) {
		t.Fatalf("tools snapshot mismatch (-want +got):\n--- want\n%s\n--- got\n%s", wantSnapshot, gotSnapshot)
	}

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
	if err != nil {
		t.Fatalf("CallTool(list_notes) error = %v", err)
	}

	out := decodeListNotesOutput(t, result)
	if len(out.Notes) != 0 {
		t.Fatalf("notes len = %d, want 0", len(out.Notes))
	}
}

func TestListNotesSupportsPagination(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if err := os.MkdirAll(memoryRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if _, err := notes.Create(notes.CreateInput{
		RootDir: memoryRoot,
		Title:   "Alpha Note",
	}); err != nil {
		t.Fatalf("Create(alpha) error = %v", err)
	}
	if _, err := notes.Create(notes.CreateInput{
		RootDir: memoryRoot,
		Title:   "Beta Note",
	}); err != nil {
		t.Fatalf("Create(beta) error = %v", err)
	}

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	firstResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_notes",
		Arguments: map[string]any{"limit": 1},
	})
	if err != nil {
		t.Fatalf("CallTool(list_notes, limit=1) error = %v", err)
	}
	first := decodeListNotesOutput(t, firstResult)
	if len(first.Notes) != 1 {
		t.Fatalf("first page notes len = %d, want 1", len(first.Notes))
	}
	if first.NextCursor == "" {
		t.Fatal("expected next_cursor on first page")
	}

	secondResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_notes",
		Arguments: map[string]any{"limit": 1, "cursor": first.NextCursor},
	})
	if err != nil {
		t.Fatalf("CallTool(list_notes, pagination) error = %v", err)
	}
	second := decodeListNotesOutput(t, secondResult)
	if len(second.Notes) != 1 {
		t.Fatalf("second page notes len = %d, want 1", len(second.Notes))
	}
	if second.NextCursor != "" {
		t.Fatalf("second page next_cursor = %q, want empty", second.NextCursor)
	}
}

func TestListTagsReturnsTagsAndRespectsLimit(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if err := os.MkdirAll(memoryRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "auth-one.md"), "550e8400-e29b-41d4-a716-446655440001", "Auth One", "auth-one", []string{"auth", "ops"}, "auth one body\n")
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "auth-two.md"), "550e8400-e29b-41d4-a716-446655440002", "Auth Two", "auth-two", []string{"auth"}, "auth two body\n")
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "beta-one.md"), "550e8400-e29b-41d4-a716-446655440003", "Beta One", "beta-one", []string{"beta"}, "beta body\n")
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_tags",
		Arguments: map[string]any{"limit": 1},
	})
	if err != nil {
		t.Fatalf("CallTool(list_tags) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("list_tags returned tool error: text=%q content=%#v", resultText(t, result), result.Content)
	}

	out := decodeListTagsOutput(t, result)
	if len(out.Tags) != 1 {
		t.Fatalf("tags len = %d, want 1", len(out.Tags))
	}
	if out.Tags[0].Tag != "auth" {
		t.Fatalf("tag = %q, want %q", out.Tags[0].Tag, "auth")
	}
	if out.Tags[0].Count != 2 {
		t.Fatalf("count = %d, want 2", out.Tags[0].Count)
	}
}

func TestServerIndexDBReusesSingleConnection(t *testing.T) {
	_ = writableMCPEnv(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if err := os.MkdirAll(memoryRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	writeTaggedMCPNote(t, filepath.Join(memoryRoot, "auth-one.md"), "550e8400-e29b-41d4-a716-446655440001", "Auth One", "auth-one", []string{"auth"}, "auth one body\n")
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	resolvedProject, err := project.ResolveProject(project.ResolveProjectInput{
		CWD:             projectRoot,
		ProjectSelector: "personal",
	})
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	effectivePaths, err := paths.ResolveEffectivePaths(paths.EffectiveInput{})
	if err != nil {
		t.Fatalf("ResolveEffectivePaths() error = %v", err)
	}

	server := NewServer(resolvedProject, effectivePaths)
	t.Cleanup(func() {
		_ = server.closeIndexDB()
	})

	firstDB, err := server.indexDB()
	if err != nil {
		t.Fatalf("indexDB() first error = %v", err)
	}
	secondDB, err := server.indexDB()
	if err != nil {
		t.Fatalf("indexDB() second error = %v", err)
	}
	if firstDB != secondDB {
		t.Fatal("indexDB() did not reuse the cached database handle")
	}
}

func TestListBacklinksReturnsLinksAndRespectsLimit(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if err := os.MkdirAll(memoryRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	writeWikiMCPNote(t, filepath.Join(memoryRoot, "target-note.md"), "550e8400-e29b-41d4-a716-446655440010", "Target Note", "target-note", "target body\n")
	writeWikiMCPNote(t, filepath.Join(memoryRoot, "alpha-note.md"), "550e8400-e29b-41d4-a716-446655440011", "Alpha Note", "alpha-note", "[[target-note]]\n")
	writeWikiMCPNote(t, filepath.Join(memoryRoot, "beta-note.md"), "550e8400-e29b-41d4-a716-446655440012", "Beta Note", "beta-note", "[[Target Note]]\n")
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_backlinks",
		Arguments: map[string]any{"identifier": "target-note", "limit": 1},
	})
	if err != nil {
		t.Fatalf("CallTool(list_backlinks) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("list_backlinks returned tool error: text=%q content=%#v", resultText(t, result), result.Content)
	}

	out := decodeListBacklinksOutput(t, result)
	if len(out.Links) != 1 {
		t.Fatalf("links len = %d, want 1", len(out.Links))
	}
	if out.Links[0].Slug != "alpha-note" {
		t.Fatalf("slug = %q, want %q", out.Links[0].Slug, "alpha-note")
	}
}

func TestListBacklinksReturnsToolErrorForMissingNote(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if err := os.MkdirAll(memoryRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	writeWikiMCPNote(t, filepath.Join(memoryRoot, "target-note.md"), "550e8400-e29b-41d4-a716-446655440010", "Target Note", "target-note", "target body\n")
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_backlinks",
		Arguments: map[string]any{"identifier": "missing-note"},
	})
	if err != nil {
		t.Fatalf("CallTool(list_backlinks missing) error = %v", err)
	}
	if !result.IsError {
		t.Fatal("list_backlinks missing note IsError = false, want true")
	}
	if got := resultText(t, result); !strings.Contains(got, "not found") {
		t.Fatalf("list_backlinks missing note text = %q, want not found", got)
	}
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
	if err != nil {
		t.Fatalf("CallTool(create_note) error = %v", err)
	}
	if createResult.IsError {
		t.Fatalf("create_note returned tool error: text=%q content=%#v", resultText(t, createResult), createResult.Content)
	}

	created := decodeCreateNoteOutput(t, createResult)
	if created.NoteID == "" {
		t.Fatal("created note_id is empty")
	}
	if created.Slug != "auth-migration" {
		t.Fatalf("slug = %q, want %q", created.Slug, "auth-migration")
	}
	if created.Path != "plans/auth-migration.md" {
		t.Fatalf("path = %q, want %q", created.Path, "plans/auth-migration.md")
	}
	if created.ContentHash == "" {
		t.Fatal("created content_hash is empty")
	}

	readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	if err != nil {
		t.Fatalf("CallTool(read_note) error = %v", err)
	}
	if readResult.IsError {
		t.Fatalf("read_note returned tool error: text=%q content=%#v", resultText(t, readResult), readResult.Content)
	}

	read := decodeReadNoteOutput(t, readResult)
	if read.Note.Path != created.Path {
		t.Fatalf("read path = %q, want %q", read.Note.Path, created.Path)
	}
	if read.Note.Body != "## Summary\n\nPlan.\n" {
		t.Fatalf("read body = %q", read.Note.Body)
	}
	if got := read.Note.Frontmatter["type"]; got != "decision" {
		t.Fatalf("frontmatter[type] = %#v, want %q", got, "decision")
	}
	if got := read.Note.Frontmatter["tags"]; !slices.Equal(anySliceToStrings(t, got), []string{"auth", "ops"}) {
		t.Fatalf("frontmatter[tags] = %#v, want %#v", got, []string{"auth", "ops"})
	}
}

func TestWriteToolsRefreshIndexForReadOnlyTools(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if err := os.MkdirAll(memoryRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	beforeSearch, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"query": "mcp-smoke-token-20260606-224145", "limit": 10},
	})
	if err != nil {
		t.Fatalf("CallTool(search_notes before) error = %v", err)
	}
	if beforeSearch.IsError {
		t.Fatalf("search_notes before returned tool error: text=%q content=%#v", resultText(t, beforeSearch), beforeSearch.Content)
	}

	alphaResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "MCP Smoke Alpha",
			"path":  "mcp-smoke-alpha.md",
			"body":  "Alpha smoke body.",
			"tags":  []string{"mcp", "smoke"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(create_note alpha) error = %v", err)
	}
	if alphaResult.IsError {
		t.Fatalf("create_note alpha returned tool error: text=%q content=%#v", resultText(t, alphaResult), alphaResult.Content)
	}
	alpha := decodeCreateNoteOutput(t, alphaResult)

	betaResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "MCP Smoke Beta",
			"body":  "Links to [[mcp-smoke-alpha]].\nUnique token: mcp-smoke-token-20260606-224145.",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(create_note beta) error = %v", err)
	}
	if betaResult.IsError {
		t.Fatalf("create_note beta returned tool error: text=%q content=%#v", resultText(t, betaResult), betaResult.Content)
	}
	beta := decodeCreateNoteOutput(t, betaResult)

	searchResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"query": `"mcp-smoke-token-20260606-224145"`, "limit": 10},
	})
	if err != nil {
		t.Fatalf("CallTool(search_notes after) error = %v", err)
	}
	if searchResult.IsError {
		t.Fatalf("search_notes after returned tool error: text=%q content=%#v", resultText(t, searchResult), searchResult.Content)
	}
	searchOut := decodeSearchNotesOutput(t, searchResult)
	if len(searchOut.Hits) != 1 || searchOut.Hits[0].NoteID != beta.NoteID {
		t.Fatalf("search hits = %#v, want beta note %q", searchOut.Hits, beta.NoteID)
	}

	backlinksResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_backlinks",
		Arguments: map[string]any{"identifier": alpha.NoteID, "limit": 10},
	})
	if err != nil {
		t.Fatalf("CallTool(list_backlinks) error = %v", err)
	}
	if backlinksResult.IsError {
		t.Fatalf("list_backlinks returned tool error: text=%q content=%#v", resultText(t, backlinksResult), backlinksResult.Content)
	}
	backlinksOut := decodeListBacklinksOutput(t, backlinksResult)
	if len(backlinksOut.Links) != 1 || backlinksOut.Links[0].NoteID != beta.NoteID {
		t.Fatalf("backlinks = %#v, want beta note %q", backlinksOut.Links, beta.NoteID)
	}
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
		if err != nil {
			t.Fatalf("CallTool(create_note) error = %v", err)
		}
		created := decodeCreateNoteOutput(t, createResult)

		editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "edit_note",
			Arguments: map[string]any{
				"identifier": created.NoteID,
				"append":     "Next step",
			},
		})
		if err != nil {
			t.Fatalf("CallTool(edit_note append) error = %v", err)
		}
		if editResult.IsError {
			t.Fatalf("edit_note append returned tool error: text=%q content=%#v", resultText(t, editResult), editResult.Content)
		}

		readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "read_note",
			Arguments: map[string]any{"identifier": created.NoteID},
		})
		if err != nil {
			t.Fatalf("CallTool(read_note) error = %v", err)
		}
		read := decodeReadNoteOutput(t, readResult)
		if read.Note.Body != "## Summary\nNext step" {
			t.Fatalf("body = %q, want %q", read.Note.Body, "## Summary\nNext step")
		}
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
		if err != nil {
			t.Fatalf("CallTool(create_note) error = %v", err)
		}
		created := decodeCreateNoteOutput(t, createResult)

		editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "edit_note",
			Arguments: map[string]any{
				"identifier":   created.NoteID,
				"replace_body": "Replacement body\n",
			},
		})
		if err != nil {
			t.Fatalf("CallTool(edit_note replace_body) error = %v", err)
		}
		if editResult.IsError {
			t.Fatalf("edit_note replace_body returned tool error: text=%q content=%#v", resultText(t, editResult), editResult.Content)
		}

		readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "read_note",
			Arguments: map[string]any{"identifier": created.NoteID},
		})
		if err != nil {
			t.Fatalf("CallTool(read_note) error = %v", err)
		}
		read := decodeReadNoteOutput(t, readResult)
		if read.Note.Body != "Replacement body\n" {
			t.Fatalf("body = %q, want %q", read.Note.Body, "Replacement body\n")
		}
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
		if err != nil {
			t.Fatalf("CallTool(create_note) error = %v", err)
		}
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
		if err != nil {
			t.Fatalf("CallTool(edit_note merge_frontmatter) error = %v", err)
		}
		if editResult.IsError {
			t.Fatalf("edit_note merge_frontmatter returned tool error: text=%q content=%#v", resultText(t, editResult), editResult.Content)
		}

		readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "read_note",
			Arguments: map[string]any{"identifier": created.NoteID},
		})
		if err != nil {
			t.Fatalf("CallTool(read_note) error = %v", err)
		}
		read := decodeReadNoteOutput(t, readResult)
		if got := read.Note.Frontmatter["type"]; got != "decision" {
			t.Fatalf("frontmatter[type] = %#v, want %q", got, "decision")
		}
		if got := read.Note.Frontmatter["custom_key"]; got != "custom-value" {
			t.Fatalf("frontmatter[custom_key] = %#v, want %q", got, "custom-value")
		}
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
	if err != nil {
		t.Fatalf("CallTool(create_note) error = %v", err)
	}
	created := decodeCreateNoteOutput(t, createResult)

	firstEdit, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier":    created.NoteID,
			"if_match_hash": created.ContentHash,
			"append":        "A",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(edit_note first) error = %v", err)
	}
	if firstEdit.IsError {
		t.Fatalf("edit_note first returned tool error: text=%q content=%#v", resultText(t, firstEdit), firstEdit.Content)
	}

	secondEdit, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier":    created.NoteID,
			"if_match_hash": created.ContentHash,
			"append":        "B",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(edit_note stale) error = %v", err)
	}
	if !secondEdit.IsError {
		t.Fatal("edit_note stale hash IsError = false, want true")
	}
	if got := resultText(t, secondEdit); !strings.Contains(got, "content hash mismatch") {
		t.Fatalf("stale hash error text = %q, want content hash mismatch", got)
	}

	readResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	if err != nil {
		t.Fatalf("CallTool(read_note) error = %v", err)
	}
	read := decodeReadNoteOutput(t, readResult)
	if read.Note.Body != "## Summary\nA" {
		t.Fatalf("body after stale write = %q, want %q", read.Note.Body, "## Summary\nA")
	}
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
	if err != nil {
		t.Fatalf("CallTool(create_note) error = %v", err)
	}
	created := decodeCreateNoteOutput(t, createResult)

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	if err != nil {
		t.Fatalf("CallTool(delete_note) error = %v", err)
	}
	if deleteResult.IsError {
		t.Fatalf("delete_note returned tool error: text=%q content=%#v", resultText(t, deleteResult), deleteResult.Content)
	}

	deleted := decodeDeleteNoteOutput(t, deleteResult)
	if !deleted.Deleted {
		t.Fatal("deleted = false, want true")
	}
	if deleted.Mode != "trash" {
		t.Fatalf("mode = %q, want %q", deleted.Mode, "trash")
	}
	if deleted.Path != "trash-me.md" {
		t.Fatalf("path = %q, want %q", deleted.Path, "trash-me.md")
	}
	if deleted.TrashPath == "" {
		t.Fatal("trash_path is empty")
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "trash-me.md")); !os.IsNotExist(err) {
		t.Fatalf("source note stat = %v, want not exist", err)
	}
	if _, err := os.Stat(deleted.TrashPath); err != nil {
		t.Fatalf("trash note stat = %v", err)
	}
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
	if err != nil {
		t.Fatalf("CallTool(create_note) error = %v", err)
	}
	created := decodeCreateNoteOutput(t, createResult)

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "delete_note",
		Arguments: map[string]any{
			"identifier":  created.NoteID,
			"hard_delete": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(delete_note hard) error = %v", err)
	}
	if deleteResult.IsError {
		t.Fatalf("delete_note hard returned tool error: text=%q content=%#v", resultText(t, deleteResult), deleteResult.Content)
	}

	deleted := decodeDeleteNoteOutput(t, deleteResult)
	if deleted.Mode != "hard" {
		t.Fatalf("mode = %q, want %q", deleted.Mode, "hard")
	}
	if deleted.TrashPath != "" {
		t.Fatalf("trash_path = %q, want empty", deleted.TrashPath)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "hard-delete.md")); !os.IsNotExist(err) {
		t.Fatalf("source note stat = %v, want not exist", err)
	}
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
	if err != nil {
		t.Fatalf("CallTool(create_note) error = %v", err)
	}
	created := decodeCreateNoteOutput(t, createResult)

	editResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier": created.NoteID,
			"append":     "updated",
		},
	})
	if err != nil {
		t.Fatalf("CallTool(edit_note) error = %v", err)
	}
	if editResult.IsError {
		t.Fatalf("edit_note returned tool error: text=%q content=%#v", resultText(t, editResult), editResult.Content)
	}

	deleteResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "delete_note",
		Arguments: map[string]any{
			"identifier":    created.NoteID,
			"if_match_hash": created.ContentHash,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(delete_note stale) error = %v", err)
	}
	if !deleteResult.IsError {
		t.Fatal("delete_note stale hash IsError = false, want true")
	}
	if got := resultText(t, deleteResult); !strings.Contains(got, "content hash mismatch") {
		t.Fatalf("stale hash error text = %q, want content hash mismatch", got)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".mnemonic-memories", "personal", "delete-protected.md")); err != nil {
		t.Fatalf("source note stat after stale delete = %v, want exists", err)
	}
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

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
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
			if !ok {
				t.Fatalf("item %#v is not string", item)
			}
			out = append(out, s)
		}
		return out
	}

	t.Fatalf("value %#v is not []string or []any", value)
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

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assertToolAnnotations(t *testing.T, tools []*mcp.Tool, name string, readOnly bool, destructive bool) {
	t.Helper()

	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		if tool.Annotations == nil {
			t.Fatalf("tool %q is missing annotations", name)
		}
		if tool.Annotations.ReadOnlyHint != readOnly {
			t.Fatalf("tool %q readOnlyHint = %v, want %v", name, tool.Annotations.ReadOnlyHint, readOnly)
		}
		if !readOnly {
			if tool.Annotations.DestructiveHint == nil {
				t.Fatalf("tool %q destructiveHint is nil", name)
			}
			if *tool.Annotations.DestructiveHint != destructive {
				t.Fatalf("tool %q destructiveHint = %v, want %v", name, *tool.Annotations.DestructiveHint, destructive)
			}
		}
		return
	}

	t.Fatalf("tool %q not found", name)
}

func TestReadNoteReturnsPayload(t *testing.T) {
	_ = writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if err := os.MkdirAll(memoryRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	created, err := notes.Create(notes.CreateInput{
		RootDir: memoryRoot,
		Title:   "Alpha Note",
		Body:    []byte("alpha body\n"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	session, _ := connectToMCPServer(t, repoRoot, projectRoot)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": created.NoteID},
	})
	if err != nil {
		t.Fatalf("CallTool(read_note) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("read_note returned tool error: %#v", result.Content)
	}

	out := decodeReadNoteOutput(t, result)
	if out.Note.Body != "alpha body\n" {
		t.Fatalf("note body = %q, want %q", out.Note.Body, "alpha body\n")
	}
	if out.Note.ContentHash == "" {
		t.Fatal("content_hash is empty")
	}
	if out.Note.Frontmatter["title"] != "Alpha Note" {
		t.Fatalf("frontmatter title = %#v, want %q", out.Note.Frontmatter["title"], "Alpha Note")
	}
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
	if err != nil {
		t.Fatalf("CallTool(read_note, missing) error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error, got %#v", result.Content)
	}
}

func TestSearchNotesReturnsHits(t *testing.T) {
	env := writableMCPEnv(t)
	repoRoot := repoRootForTest(t)
	projectRoot := t.TempDir()
	writeMCPMnemonicFile(t, filepath.Join(projectRoot, ".mnemonic"))

	memoryRoot := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if err := os.MkdirAll(memoryRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if _, err := notes.Create(notes.CreateInput{
		RootDir: memoryRoot,
		Title:   "Searchable Note",
		Body:    []byte("alpha beta gamma\n"),
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := index.RebuildProjectIndex("550e8400-e29b-41d4-a716-446655440000", memoryRoot); err != nil {
		t.Fatalf("RebuildProjectIndex() error = %v", err)
	}

	session, _ := connectToMCPServerWithEnv(t, repoRoot, projectRoot, env)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"query": "alpha", "limit": 10},
	})
	if err != nil {
		t.Fatalf("CallTool(search_notes) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("search_notes returned tool error: text=%q content=%#v", resultText(t, result), result.Content)
	}

	out := decodeSearchNotesOutput(t, result)
	if len(out.Hits) != 1 {
		t.Fatalf("hits len = %d, want 1", len(out.Hits))
	}
	hit := out.Hits[0]
	if hit.NoteID == "" || hit.Slug == "" || hit.Title == "" || hit.Path == "" || hit.Snippet == "" || hit.ContentHash == "" {
		t.Fatalf("unexpected empty search hit: %#v", hit)
	}
	if hit.Slug != "searchable-note" {
		t.Fatalf("slug = %q, want %q", hit.Slug, "searchable-note")
	}
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
	if err != nil {
		t.Fatalf("CallTool(search_notes, missing index) error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error, got %#v", result.Content)
	}
	if got := resultText(t, result); !strings.Contains(got, "mnemonic project reindex") {
		t.Fatalf("tool error text = %q, want reindex suggestion", got)
	}
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
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	effectivePaths, err := paths.ResolveEffectivePaths(paths.EffectiveInput{})
	if err != nil {
		t.Fatalf("ResolveEffectivePaths() error = %v", err)
	}

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
	if err != nil {
		t.Fatalf("server Connect() error = %v", err)
	}
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatalf("client Connect() error = %v", err)
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

	if result == nil {
		t.Fatal("result is nil")
	}

	if out, ok := decodeListNotesPayload(result.StructuredContent); ok {
		return out
	}

	for _, content := range result.Content {
		if out, ok := decodeListNotesContent(t, content); ok {
			return out
		}
	}

	t.Fatalf("result has no decodable list_notes payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return listNotesOutput{}
}

func decodeCreateNoteOutput(t *testing.T, result *mcp.CallToolResult) createNoteOutput {
	t.Helper()

	if result == nil {
		t.Fatal("result is nil")
	}

	if structured, ok := decodeCreateNotePayload(t, result.StructuredContent); ok {
		return structured
	}
	for _, content := range result.Content {
		if structured, ok := decodeCreateNoteContent(t, content); ok {
			return structured
		}
	}

	t.Fatalf("result has no decodable create_note payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return createNoteOutput{}
}

func decodeEditNoteOutput(t *testing.T, result *mcp.CallToolResult) editNoteOutput {
	t.Helper()

	if result == nil {
		t.Fatal("result is nil")
	}

	if structured, ok := decodeEditNotePayload(t, result.StructuredContent); ok {
		return structured
	}
	for _, content := range result.Content {
		if structured, ok := decodeEditNoteContent(t, content); ok {
			return structured
		}
	}

	t.Fatalf("result has no decodable edit_note payload: structured=%T content=%#v", result.StructuredContent, result.Content)
	return editNoteOutput{}
}

func decodeDeleteNoteOutput(t *testing.T, result *mcp.CallToolResult) deleteNoteOutput {
	t.Helper()

	if result == nil {
		t.Fatal("result is nil")
	}

	if structured, ok := decodeDeleteNotePayload(t, result.StructuredContent); ok {
		return structured
	}
	for _, content := range result.Content {
		if structured, ok := decodeDeleteNoteContent(t, content); ok {
			return structured
		}
	}

	t.Fatalf("result has no decodable delete_note payload: structured=%T content=%#v", result.StructuredContent, result.Content)
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
		if err != nil {
			t.Fatalf("Marshal(createNote structured map) error = %v", err)
		}
		var out createNoteOutput
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("Unmarshal(createNote structured map) error = %v", err)
		}
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
			t.Fatalf("Unmarshal(create_note text) error = %v", err)
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
		if err != nil {
			t.Fatalf("Marshal(editNote structured map) error = %v", err)
		}
		var out editNoteOutput
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("Unmarshal(editNote structured map) error = %v", err)
		}
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
			t.Fatalf("Unmarshal(edit_note text) error = %v", err)
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
		if err != nil {
			t.Fatalf("Marshal(deleteNote structured map) error = %v", err)
		}
		var out deleteNoteOutput
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("Unmarshal(deleteNote structured map) error = %v", err)
		}
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
			t.Fatalf("Unmarshal(delete_note text) error = %v", err)
		}
		return out, true
	case *mcp.ToolResultContent:
		return decodeDeleteNotePayload(t, v.StructuredContent)
	}

	return deleteNoteOutput{}, false
}

func decodeListTagsOutput(t *testing.T, result *mcp.CallToolResult) listTagsOutput {
	t.Helper()

	if result == nil {
		t.Fatal("result is nil")
	}

	if out, ok := decodeListTagsPayload(result.StructuredContent); ok {
		return out
	}

	for _, content := range result.Content {
		if out, ok := decodeListTagsContent(t, content); ok {
			return out
		}
	}

	t.Fatalf("result has no decodable list_tags payload: structured=%T content=%#v", result.StructuredContent, result.Content)
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
	t.Fatalf("result has no decodable list_backlinks payload: structured=%T content=%#v", result.StructuredContent, result.Content)
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

	if result == nil {
		t.Fatal("result is nil")
	}

	if out, ok := decodeReadNotePayload(result.StructuredContent); ok {
		return out
	}

	for _, content := range result.Content {
		if out, ok := decodeReadNoteContent(t, content); ok {
			return out
		}
	}

	t.Fatalf("result has no decodable read_note payload: structured=%T content=%#v", result.StructuredContent, result.Content)
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

	if result == nil {
		t.Fatal("result is nil")
	}

	if out, ok := decodeSearchNotesPayload(result.StructuredContent); ok {
		return out
	}

	for _, content := range result.Content {
		if out, ok := decodeSearchNotesContent(t, content); ok {
			return out
		}
	}

	t.Fatalf("result has no decodable search_notes payload: structured=%T content=%#v", result.StructuredContent, result.Content)
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
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
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
	if err := project.WriteMnemonicFile(path, file); err != nil {
		t.Fatalf("WriteMnemonicFile() error = %v", err)
	}
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
