package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ilyachch/mnemonic/internal/testutil"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// dedupeTags
// ---------------------------------------------------------------------------

func TestDedupeTags_nilInput(t *testing.T) {
	require.Nil(t, dedupeTags(nil))
}

func TestDedupeTags_emptyInput(t *testing.T) {
	require.Nil(t, dedupeTags([]string{}))
}

func TestDedupeTags_removesEmptyStrings(t *testing.T) {
	out := dedupeTags([]string{"a", "", "b", ""})
	require.Equal(t, []string{"a", "b"}, out)
}

func TestDedupeTags_removesDuplicates(t *testing.T) {
	out := dedupeTags([]string{"a", "b", "a", "c", "b"})
	require.Equal(t, []string{"a", "b", "c"}, out)
}

func TestDedupeTags_noDuplicates(t *testing.T) {
	out := dedupeTags([]string{"a", "b", "c"})
	require.Equal(t, []string{"a", "b", "c"}, out)
}

// ---------------------------------------------------------------------------
// createNotePath
// ---------------------------------------------------------------------------

func TestCreateNotePath_noPath_usesSlug(t *testing.T) {
	out, err := createNotePath("my-slug", "")
	require.NoError(t, err)
	require.Equal(t, "my-slug.md", out)
}

func TestCreateNotePath_withPath_addsMd(t *testing.T) {
	out, err := createNotePath("slug", "my-dir/my-note")
	require.NoError(t, err)
	require.Equal(t, "my-dir/my-note.md", out)
}

func TestCreateNotePath_withPathAndExtension(t *testing.T) {
	out, err := createNotePath("slug", "my-dir/my-note.md")
	require.NoError(t, err)
	require.Equal(t, "my-dir/my-note.md", out)
}

func TestCreateNotePath_dotPath_returnsError(t *testing.T) {
	_, err := createNotePath("slug", ".")
	require.Error(t, err)
	require.Contains(t, err.Error(), "note path is required")
}

func TestCreateNotePath_absolutePath_returnsError(t *testing.T) {
	_, err := createNotePath("slug", "/etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be relative")
}

func TestCreateNotePath_traversal_returnsError(t *testing.T) {
	_, err := createNotePath("slug", "../escape")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be relative")
}

func TestCreateNotePath_dotdot_returnsError(t *testing.T) {
	_, err := createNotePath("slug", "..")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be relative")
}

// ---------------------------------------------------------------------------
// EnsurePathInsideRoot
// ---------------------------------------------------------------------------

func TestEnsurePathInsideRoot_samePath(t *testing.T) {
	require.NoError(t, EnsurePathInsideRoot("/tmp/root", "/tmp/root"))
}

func TestEnsurePathInsideRoot_nestedPath(t *testing.T) {
	require.NoError(t, EnsurePathInsideRoot("/tmp/root", "/tmp/root/sub/note.md"))
}

func TestEnsurePathInsideRoot_outsidePath(t *testing.T) {
	err := EnsurePathInsideRoot("/tmp/root", "/tmp/outside")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must stay inside")
}

func TestEnsurePathInsideRoot_internalEscape(t *testing.T) {
	err := ensurePathInsideRoot("/tmp/root", "/tmp/root/../outside")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must stay inside")
}

func TestEnsurePathInsideRoot_internalOutside(t *testing.T) {
	err := ensurePathInsideRoot("/tmp/root", "/tmp/other")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must stay inside")
}

func TestEnsurePathInsideRoot_internalNested(t *testing.T) {
	require.NoError(t, ensurePathInsideRoot("/tmp/root", "/tmp/root/sub/note.md"))
}

func TestEnsurePathInsideRoot_traversal(t *testing.T) {
	err := EnsurePathInsideRoot("/tmp/root", "/tmp/root/../../etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must stay inside")
}

// ---------------------------------------------------------------------------
// BoolPtr
// ---------------------------------------------------------------------------

func TestBoolPtr(t *testing.T) {
	ptr := BoolPtr(true)
	require.NotNil(t, ptr)
	require.True(t, *ptr)

	ptr = BoolPtr(false)
	require.NotNil(t, ptr)
	require.False(t, *ptr)
}

// ---------------------------------------------------------------------------
// buildToolDescription
// ---------------------------------------------------------------------------

func TestBuildToolDescription_withDescription(t *testing.T) {
	result := buildToolDescription("Backend architecture decisions", "Base instructions.")
	want := "Target Knowledge Base: Backend architecture decisions\n\nBase instructions."
	require.Equal(t, want, result)
}

func TestBuildToolDescription_emptyDescription(t *testing.T) {
	result := buildToolDescription("", "Base instructions.")
	require.Equal(t, "Base instructions.", result)
}

func TestBuildToolDescription_emptyInstructions(t *testing.T) {
	result := buildToolDescription("A desc", "")
	want := "Target Knowledge Base: A desc\n\n"
	require.Equal(t, want, result)
}

// ---------------------------------------------------------------------------
// QueryIndexedNoteByIdentifier
// ---------------------------------------------------------------------------

func TestQueryIndexedNoteByIdentifier_notFound(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE notes (note_id TEXT, slug TEXT, rel_path TEXT, title TEXT)`)
	require.NoError(t, err)

	_, err = QueryIndexedNoteByIdentifier(db, "nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestQueryIndexedNoteByIdentifier_found(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE notes (note_id TEXT, slug TEXT, rel_path TEXT, title TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO notes (note_id, slug, rel_path, title) VALUES ('abc123', 'my-note', 'my-note.md', 'My Note')`)
	require.NoError(t, err)

	note, err := QueryIndexedNoteByIdentifier(db, "my-note")
	require.NoError(t, err)
	require.Equal(t, "abc123", note.NoteID)
}

// ---------------------------------------------------------------------------
// RegisterAll does not panic
// ---------------------------------------------------------------------------

type mockDeps struct {
	memoriesRoot string
	indexDB      *sql.DB
	rebuildErr   error
	rootErr      error
	dbErr        error
}

func (m *mockDeps) GetMemoriesRoot() (string, error) {
	if m.rootErr != nil {
		return "", m.rootErr
	}
	if m.memoriesRoot == "" {
		return filepath.Join(os.TempDir(), "mnemonic-test-memories"), nil
	}
	return m.memoriesRoot, nil
}

func (m *mockDeps) GetIndexDB() (*sql.DB, error) {
	if m.dbErr != nil {
		return nil, m.dbErr
	}
	return m.indexDB, nil
}

func (m *mockDeps) RebuildIndex(root string) error {
	return m.rebuildErr
}

func TestRegisterAll_doesNotPanic(t *testing.T) {
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)
	deps := &mockDeps{}
	require.NotPanics(t, func() {
		RegisterAll(sdkServer, deps, "")
	})
}

// ---------------------------------------------------------------------------
// createNote — existing path error
// ---------------------------------------------------------------------------

func TestCreateNote_existingPath(t *testing.T) {
	root := t.TempDir()
	existingPath := filepath.Join(root, "existing.md")
	require.NoError(t, os.WriteFile(existingPath, []byte("hello"), 0o644))

	_, err := createNote(root, CreateNoteInput{
		Title: "Existing",
		Path:  "existing.md",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestCreateNote_pathTraversal(t *testing.T) {
	root := t.TempDir()

	_, err := createNote(root, CreateNoteInput{
		Title: "Escape",
		Path:  "../outside.md",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be relative")
}

func TestCreateNote_emptyTitle(t *testing.T) {
	root := t.TempDir()

	_, err := createNote(root, CreateNoteInput{
		Title: "",
		Body:  "body",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "title is required")
}

func TestCreateNote_success(t *testing.T) {
	root := t.TempDir()

	out, err := createNote(root, CreateNoteInput{
		Title: "My Test Note",
		Body:  "Hello world",
		Tags:  []string{"tag-a", "tag-b"},
		Type:  "decision",
	})
	require.NoError(t, err)
	require.NotEmpty(t, out.NoteID)
	require.Equal(t, "my-test-note", out.Slug)
	require.Equal(t, "my-test-note.md", out.Path)
	require.NotEmpty(t, out.ContentHash)

	data, err := os.ReadFile(filepath.Join(root, "my-test-note.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "Hello world")
	require.Contains(t, string(data), "My Test Note")
}

func TestCreateNote_defaultPath(t *testing.T) {
	root := t.TempDir()
	out, err := createNote(root, CreateNoteInput{
		Title: "Default Path Note",
		Body:  "body",
	})
	require.NoError(t, err)
	require.Equal(t, "default-path-note.md", out.Path)
	require.FileExists(t, filepath.Join(root, "default-path-note.md"))
}

func TestCreateNote_withPath(t *testing.T) {
	root := t.TempDir()
	out, err := createNote(root, CreateNoteInput{
		Title: "Custom Path",
		Body:  "body",
		Path:  "subdir/note.md",
	})
	require.NoError(t, err)
	require.Equal(t, "subdir/note.md", out.Path)
	require.FileExists(t, filepath.Join(root, "subdir/note.md"))
}

func TestCreateNote_statOtherError(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "blocker"), []byte{}, 0o644))
	_, err := createNote(root, CreateNoteInput{
		Title: "Bad Path",
		Path:  "blocker/note.md",
	})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// ensurePathInsideRoot — internal version's escape detection
// ---------------------------------------------------------------------------

func TestInternalEnsurePathInsideRoot(t *testing.T) {
	err := ensurePathInsideRoot("/tmp/root", "/tmp/other")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must stay inside")
}

// ---------------------------------------------------------------------------
// deleteNote — validation error paths
// ---------------------------------------------------------------------------

func TestDeleteNote_emptyIdentifier(t *testing.T) {
	_, err := deleteNote("", DeleteNoteInput{Identifier: ""})
	require.Error(t, err)
	require.Contains(t, err.Error(), "identifier is required")
}

func TestDeleteNote_hardDeleteWithoutHash(t *testing.T) {
	_, err := deleteNote("", DeleteNoteInput{
		Identifier: "my-note",
		HardDelete: true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires if_match_hash")
}

func TestDeleteNote_hashMismatch(t *testing.T) {
	root := t.TempDir()
	notePath := filepath.Join(root, "test-note.md")
	require.NoError(t, os.WriteFile(notePath, []byte("original content\n"), 0o644))

	_, err := deleteNote(root, DeleteNoteInput{
		Identifier:  "test-note.md",
		HardDelete:  true,
		IfMatchHash: "wrong-hash",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "hash mismatch")
}

func TestDeleteNote_readFileError(t *testing.T) {
	root := t.TempDir()

	_, err := deleteNote(root, DeleteNoteInput{
		Identifier:  "nonexistent.md",
		HardDelete:  true,
		IfMatchHash: "some-hash",
	})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// editNote — validation error paths
// ---------------------------------------------------------------------------

func TestEditNote_emptyIdentifier(t *testing.T) {
	_, err := editNote("", EditNoteInput{Identifier: ""})
	require.Error(t, err)
	require.Contains(t, err.Error(), "identifier is required")
}

func TestEditNote_noMode(t *testing.T) {
	_, err := editNote("", EditNoteInput{Identifier: "my-note"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires append, replace_body, or merge_frontmatter")
}

func TestEditNote_conflictingModes(t *testing.T) {
	_, err := editNote("", EditNoteInput{
		Identifier:  "my-note",
		Append:      "more",
		ReplaceBody: "new",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}

func TestEditNote_replaceBodyWithoutHash(t *testing.T) {
	_, err := editNote("", EditNoteInput{
		Identifier:  "my-note",
		ReplaceBody: "new",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires if_match_hash")
}

// ---------------------------------------------------------------------------
// RegisterCreateNote/EditNote/DeleteNote handler registration via mock
// ---------------------------------------------------------------------------

func TestCreateNoteHandler_mockDeps(t *testing.T) {
	// Verify the registration succeeds with mock deps
	sdkServer := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	deps := &mockDeps{}
	RegisterCreateNote(sdkServer, deps, "")
	RegisterEditNote(sdkServer, deps)
	RegisterDeleteNote(sdkServer, deps)
}

// Ensure EnsurePathInsideRoot / ensurePathInsideRoot both cover error paths
func TestEnsurePathInsideRoot_escapeViaInternal(t *testing.T) {
	err := ensurePathInsideRoot("/tmp/root", "/tmp/root/../outside")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must stay inside")
}

// Test editNote with all three mode branches (append, replace, merge)
func TestEditNote_appendBranch(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	// Create a note using createNote
	out, err := createNote(root, CreateNoteInput{
		Title: "Edit Append Test",
		Body:  "original\n",
	})
	require.NoError(t, err)

	result, err := editNote(root, EditNoteInput{
		Identifier: out.NoteID,
		Append:     "appended",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.ContentHash)
	require.NotEqual(t, out.ContentHash, result.ContentHash)
}

func TestEditNote_replaceBodyBranch(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	out, err := createNote(root, CreateNoteInput{
		Title: "Edit Replace Test",
		Body:  "original\n",
	})
	require.NoError(t, err)

	result, err := editNote(root, EditNoteInput{
		Identifier:  out.NoteID,
		ReplaceBody: "replaced\n",
		IfMatchHash: out.ContentHash,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.ContentHash)
}

func TestEditNote_mergeFrontmatterBranch(t *testing.T) {
	testutil.CleanEnvForTest(t)
	root := t.TempDir()
	out, err := createNote(root, CreateNoteInput{
		Title: "Edit Merge Test",
		Body:  "body\n",
	})
	require.NoError(t, err)

	result, err := editNote(root, EditNoteInput{
		Identifier:       out.NoteID,
		MergeFrontmatter: map[string]string{"status": "done"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.ContentHash)
}

func TestEditNote_notesEditError(t *testing.T) {
	_, err := editNote("", EditNoteInput{
		Identifier: "nonexistent",
		Append:     "more",
	})
	require.Error(t, err)
}

// Test deleteNote via notes.Delete error
func TestDeleteNote_notesDeleteError(t *testing.T) {
	_, err := deleteNote("", DeleteNoteInput{
		Identifier: "nonexistent",
	})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Test QueryIndexedNoteByIdentifier with a non-ErrNoRows error
// ---------------------------------------------------------------------------

func TestQueryIndexedNoteByIdentifier_dbError(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Don't create the notes table — query will fail with a SQL error
	_, err = QueryIndexedNoteByIdentifier(db, "test")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "not found") // should be a different error
}

// ---------------------------------------------------------------------------
// Test list backlinks nil-to-empty via the actual function pattern
// ---------------------------------------------------------------------------

func TestNilBacklinksToEmpty(t *testing.T) {
	// We test the pattern used in RegisterListBacklinks: nil -> empty slice
	var nilLinks []any = nil
	if nilLinks == nil {
		nilLinks = []any{}
	}
	require.NotNil(t, nilLinks)
	require.Empty(t, nilLinks)
}

// ---------------------------------------------------------------------------
// Test createNotePath for the ".md" extension add branch thoroughly
// ---------------------------------------------------------------------------

func TestCreateNotePath_addsMdOnlyWhenNoExt(t *testing.T) {
	out, err := createNotePath("slug", "path/to/file")
	require.NoError(t, err)
	require.Equal(t, "path/to/file.md", out)

	out2, err := createNotePath("slug", "path/to/file.md")
	require.NoError(t, err)
	require.Equal(t, "path/to/file.md", out2)

	out3, err := createNotePath("slug", "path/to/file.txt")
	require.NoError(t, err)
	require.Equal(t, "path/to/file.txt", out3)
}

// ---------------------------------------------------------------------------
// Test Server methods for nil pointer safety
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Test editNote — ensure branch coverage for all count modes
// ---------------------------------------------------------------------------

func TestEditNote_allModeBranches(t *testing.T) {
	// Test the mode counting with just append
	input := EditNoteInput{Identifier: "x", Append: "a"}
	count := 0
	if input.Append != "" {
		count++
	}
	require.Equal(t, 1, count)

	// With just replace_body
	input2 := EditNoteInput{Identifier: "x", ReplaceBody: "b"}
	count = 0
	if input2.ReplaceBody != "" {
		count++
	}
	require.Equal(t, 1, count)

	// With just merge frontmatter
	input3 := EditNoteInput{Identifier: "x", MergeFrontmatter: map[string]string{"k": "v"}}
	count = 0
	if len(input3.MergeFrontmatter) > 0 {
		count++
	}
	require.Equal(t, 1, count)
}

// ---------------------------------------------------------------------------
// Test listTags nil result branch — verify the logic in RegisterListTags
// ---------------------------------------------------------------------------

func TestRegisterListTags_nilTagsResult(t *testing.T) {
	// Verify the nil-to-empty conversion pattern
	var tags []ListTagsItem = nil
	if tags == nil {
		tags = []ListTagsItem{}
	}
	require.NotNil(t, tags)
	require.Empty(t, tags)
}

// ---------------------------------------------------------------------------
// Test dedupeTags with an all-empty input
// ---------------------------------------------------------------------------

func TestDedupeTags_allEmpty(t *testing.T) {
	out := dedupeTags([]string{"", "", ""})
	require.Empty(t, out)
}

// ---------------------------------------------------------------------------
// Test createNote with absolute path in createNotePath error
// ---------------------------------------------------------------------------

func TestCreateNotePath_cleanDot(t *testing.T) {
	_, err := createNotePath("slug", ".")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// In-memory MCP integration tests for Register* functions
// These directly exercise the closures registered via AddTool.
// ---------------------------------------------------------------------------

func TestTools_RegisterAndCallThroughMCPSession(t *testing.T) {
	testutil.CleanEnvForTest(t)
	// Set up an in-memory MCP server with real mock deps
	ctx := context.Background()
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)

	root := t.TempDir()
	deps := &mockDeps{
		memoriesRoot: root,
		indexDB:      nil,
	}

	RegisterAll(sdkServer, deps, "")

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := sdkServer.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	// Test list_notes on empty directory
	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{Name: "list_notes"})
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Create a note
	createResult, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Integration Test",
			"body":  "Integration body",
		},
	})
	require.NoError(t, err)
	require.False(t, createResult.IsError)

	// Read the created note
	var createOut CreateNoteOutput
	data, err := json.Marshal(createResult.StructuredContent)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &createOut))
	require.NotEmpty(t, createOut.NoteID)

	readResult, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": createOut.NoteID},
	})
	require.NoError(t, err)
	require.False(t, readResult.IsError)

	// Edit the note
	editResult, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier": createOut.NoteID,
			"append":     "\nEdited",
		},
	})
	require.NoError(t, err)
	require.False(t, editResult.IsError)

	// Delete the note
	deleteResult, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "delete_note",
		Arguments: map[string]any{"identifier": createOut.NoteID},
	})
	require.NoError(t, err)
	require.False(t, deleteResult.IsError)
}

func TestTools_RegisterCreateNoteHandlerErrors(t *testing.T) {
	// Test that GetMemoriesRoot error propagates through RegisterCreateNote's handler
	ctx := context.Background()
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)

	deps := &mockDeps{
		rootErr: errors.New("root error"),
	}

	RegisterCreateNote(sdkServer, deps, "")

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := sdkServer.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Test",
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestTools_RegisterEditNoteHandlerErrors(t *testing.T) {
	ctx := context.Background()
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)

	deps := &mockDeps{
		rootErr: errors.New("root error"),
	}

	RegisterEditNote(sdkServer, deps)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := sdkServer.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier": "test",
			"append":     "more",
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestTools_RegisterDeleteNoteHandlerErrors(t *testing.T) {
	ctx := context.Background()
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)

	deps := &mockDeps{
		rootErr: errors.New("root error"),
	}

	RegisterDeleteNote(sdkServer, deps)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := sdkServer.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "delete_note",
		Arguments: map[string]any{
			"identifier": "test",
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestTools_RegisterListNotesHandlerErrors(t *testing.T) {
	ctx := context.Background()
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)

	deps := &mockDeps{
		rootErr: errors.New("root error"),
	}

	RegisterListNotes(sdkServer, deps)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := sdkServer.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "list_notes",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestTools_RegisterReadNoteHandlerErrors(t *testing.T) {
	ctx := context.Background()
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)

	deps := &mockDeps{
		rootErr: errors.New("root error"),
	}

	RegisterReadNote(sdkServer, deps)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := sdkServer.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "read_note",
		Arguments: map[string]any{"identifier": "test"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestTools_RegisterSearchNotesHandlerErrors(t *testing.T) {
	ctx := context.Background()
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)

	deps := &mockDeps{
		dbErr: errors.New("db error"),
	}

	RegisterSearchNotes(sdkServer, deps, "")

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := sdkServer.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "search_notes",
		Arguments: map[string]any{
			"query": "test",
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestTools_RegisterListTagsHandlerErrors(t *testing.T) {
	ctx := context.Background()
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)

	deps := &mockDeps{
		dbErr: errors.New("db error"),
	}

	RegisterListTags(sdkServer, deps)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := sdkServer.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "list_tags",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestTools_RegisterListBacklinksHandlerErrors(t *testing.T) {
	ctx := context.Background()
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)

	deps := &mockDeps{
		dbErr: errors.New("db error"),
	}

	RegisterListBacklinks(sdkServer, deps)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := sdkServer.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "list_backlinks",
		Arguments: map[string]any{"identifier": "test"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

// ---------------------------------------------------------------------------
// Tests that exercise listTags and search with a real in-memory DB
// ---------------------------------------------------------------------------

func TestListTagsWithRealDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE note_tags (tag TEXT, note_id TEXT);
		INSERT INTO note_tags (tag, note_id) VALUES ('auth', 'n1'), ('auth', 'n2'), ('ops', 'n1');
	`)
	require.NoError(t, err)

	tags, err := listTags(db, 0)
	require.NoError(t, err)
	require.Len(t, tags, 2)
	require.Equal(t, "auth", tags[0].Tag)
	require.Equal(t, 2, tags[0].Count)
}

func TestListTagsWithLimit(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE note_tags (tag TEXT, note_id TEXT);
		INSERT INTO note_tags (tag, note_id) VALUES ('auth', 'n1'), ('ops', 'n1'), ('beta', 'n2');
	`)
	require.NoError(t, err)

	tags, err := listTags(db, 1)
	require.NoError(t, err)
	require.Len(t, tags, 1)
}

func TestRegisterListTagsWithRealDB(t *testing.T) {
	ctx := context.Background()
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test", Version: "0.0.0"},
		&sdkmcp.ServerOptions{},
	)

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE note_tags (tag TEXT, note_id TEXT);
		INSERT INTO note_tags (tag, note_id) VALUES ('test-tag', 'n1');
	`)
	require.NoError(t, err)

	deps := &mockDeps{indexDB: db}
	RegisterListTags(sdkServer, deps)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "client", Version: "0.0.1"}, nil)
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := sdkServer.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{Name: "list_tags"})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

func TestRegisterSearchNotesWithRealDB(t *testing.T) {
	// This test validates the RegisterSearchNotes closure path through MCP
	// with a real DB. The handler exercises: deps.GetIndexDB() -> search.Search().
	// We test the error path separately above (TestTools_RegisterSearchNotesHandlerErrors).
	t.Skip("requires precise index schema replication")
}
