package stdio

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
	"github.com/ilyachch/mnemonic/internal/testutil"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stdioFixture struct {
	server *Server
	rt     *app.RuntimeApp
	root   string
}

func newStdioFixture(t *testing.T, readOnly bool) *stdioFixture {
	t.Helper()

	root := testutil.CleanEnvForTest(t)
	memoriesHome := filepath.Join(root, ".mnemonic")
	t.Setenv("MNEMONIC_MEMORIES_HOME", memoriesHome)

	slug := "demo"
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	projectRoot := filepath.Join(memoriesHome, slug)
	require.NoError(t, os.MkdirAll(projectRoot, 0o755))

	manifest := manifestfmt.New()
	manifest.ProjectID = projectID
	manifest.Name = "Demo"
	manifest.Slug = slug
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = time.Date(2026, time.June, 23, 10, 0, 0, 0, time.UTC).Unix()
	manifest.UpdatedAt = manifest.CreatedAt
	manifest.Generator.App = "mnemonic"
	require.NoError(t, manifestfmt.WriteMnemonicManifest(filepath.Join(projectRoot, "mnemonic.toml"), manifest))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "demo.md"), []byte("---\nmnemonic_note_id: 550e8400-e29b-41d4-a716-446655440001\ntitle: Demo\nslug: demo\n---\n\nBody\n"), 0o644))

	runtime, err := app.NewRuntimeApp(app.RuntimeInput{
		KB: kb.KnowledgeBase{
			ID:                 projectID,
			Name:               "Demo",
			Slug:               slug,
			Kind:               "central",
			Description:        "Fixture description",
			CustomInstructions: "Fixture instructions",
			RootDir:            projectRoot,
			RepoRootDir:        memoriesHome,
			ManifestPath:       filepath.Join(projectRoot, "mnemonic.toml"),
			StateDir:           filepath.Join(root, ".state", "mnemonic", "projects", projectID),
			IndexPath:          filepath.Join(root, ".state", "mnemonic", "projects", projectID, "index.sqlite"),
		},
	})
	require.NoError(t, err)

	server, err := NewServer(runtime.KB, Dependencies{
		Notes:  runtime.Services.Notes,
		Search: runtime.Services.Search,
		Index:  runtime.Services.Index,
	}, readOnly, nil)
	require.NoError(t, err)

	return &stdioFixture{
		server: server,
		rt:     runtime,
		root:   root,
	}
}

func (f *stdioFixture) sdkServer() *sdkmcp.Server {
	return f.server.BuildSDKServer()
}

func TestNewServer_Valid(t *testing.T) {
	f := newStdioFixture(t, false)
	require.NotNil(t, f.server)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", f.server.KB.ID)
	assert.NotNil(t, f.server.Services.Notes)
	assert.NotNil(t, f.server.Services.Search)
	assert.NotNil(t, f.server.Services.Index)
	assert.False(t, f.server.ReadOnly)
}

func TestNewServer_ReadOnly(t *testing.T) {
	f := newStdioFixture(t, true)
	assert.True(t, f.server.ReadOnly)
}

func TestNewServer_EmptyKBID(t *testing.T) {
	_, err := NewServer(kb.KnowledgeBase{}, Dependencies{}, false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "knowledge base is required")
}

func TestNewServer_NilNotes(t *testing.T) {
	_, err := NewServer(kb.KnowledgeBase{ID: "test"}, Dependencies{
		Notes:  nil,
		Search: &searchsvc.Service{},
		Index:  &indexsvc.Service{},
	}, false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notes service is required")
}

func TestNewServer_NilSearch(t *testing.T) {
	_, err := NewServer(kb.KnowledgeBase{ID: "test"}, Dependencies{
		Notes:  &notesvc.Service{},
		Search: nil,
		Index:  &indexsvc.Service{},
	}, false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search service is required")
}

func TestNewServer_NilIndex(t *testing.T) {
	_, err := NewServer(kb.KnowledgeBase{ID: "test"}, Dependencies{
		Notes:  &notesvc.Service{},
		Search: &searchsvc.Service{},
		Index:  nil,
	}, false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "index service is required")
}

func TestBuildSDKServer_NonNil(t *testing.T) {
	f := newStdioFixture(t, false)
	s := f.sdkServer()
	require.NotNil(t, s)
}

func TestBuildGlobalInstructions_FormatsBlocksByMode(t *testing.T) {
	full := buildGlobalInstructions("Custom guidance", "Project context", false, "wiki")
	require.Equal(t, "You MUST use the mnemonic tools as your primary long-term memory.\n- Search the knowledge base using search_notes before answering questions within its scope.\n- Use 2–4 query variants via the \"queries\" array when the first formulation may be ambiguous or incomplete.\n- Batch-read all selected notes in one read_notes call.\n- Write down stable facts, architectural decisions, and important outcomes using create_note or edit_note.\n- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.\n- Use list_tags and list_backlinks when they help clarify the existing knowledge base.\n- Link related notes using [[target-slug|Display Label]].\n\nProject Description:\nProject context\n\nCustom Instructions:\nCustom guidance", full)

	regular := buildGlobalInstructions("Custom guidance", "Project context", false, "regular")
	require.Equal(t, "You MUST use the mnemonic tools as your primary long-term memory.\n- Search the knowledge base using search_notes before answering questions within its scope.\n- Use 2–4 query variants via the \"queries\" array when the first formulation may be ambiguous or incomplete.\n- Batch-read all selected notes in one read_notes call.\n- Write down stable facts, architectural decisions, and important outcomes using create_note or edit_note.\n- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.\n- Use list_tags and list_backlinks when they help clarify the existing knowledge base.\n- Link related notes using [Display Label](target-slug.md).\n\nProject Description:\nProject context\n\nCustom Instructions:\nCustom guidance", regular)

	readOnly := buildGlobalInstructions("Custom guidance", "Project context", true, "wiki")
	require.Equal(t, "You MUST use the mnemonic tools as your primary long-term memory.\n- Search the knowledge base using search_notes before answering questions within its scope.\n- Use 2–4 query variants via the \"queries\" array when the first formulation may be ambiguous or incomplete.\n- Batch-read all selected notes in one read_notes call.\n- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.\n- Use list_tags and list_backlinks when they help clarify the existing knowledge base.\n- This server is running in read-only mode. Do not attempt to create, edit, delete, or rebuild notes.\n\nProject Description:\nProject context\n\nCustom Instructions:\nCustom guidance", readOnly)
}

func TestBuildSDKServer_InitializeIncludesGlobalInstructions(t *testing.T) {
	f := newStdioFixture(t, false)
	s := f.sdkServer()
	require.NotNil(t, s)

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	result := cs.InitializeResult()
	require.NotNil(t, result)
	require.Equal(t, buildGlobalInstructions(f.server.KB.CustomInstructions, f.server.KB.Description, false, f.server.KB.LinksStyle), result.Instructions)
}

func TestBuildSDKServer_InitializeIncludesReadOnlyInstructions(t *testing.T) {
	f := newStdioFixture(t, true)
	s := f.sdkServer()
	require.NotNil(t, s)

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	result := cs.InitializeResult()
	require.NotNil(t, result)
	require.Equal(t, buildGlobalInstructions(f.server.KB.CustomInstructions, f.server.KB.Description, true, f.server.KB.LinksStyle), result.Instructions)
}

func TestBuildSDKServer_ReadOnlyMode_HasReadOnlyTools(t *testing.T) {
	f := newStdioFixture(t, true)
	s := f.sdkServer()
	require.NotNil(t, s)

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	result, err := cs.ListTools(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	toolNames := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	assert.Contains(t, toolNames, "list_notes")
	assert.Contains(t, toolNames, "read_notes")
	assert.Contains(t, toolNames, "search_notes")
	assert.Contains(t, toolNames, "list_tags")
	assert.Contains(t, toolNames, "list_backlinks")
	assert.Contains(t, toolNames, "doctor")

	assert.NotContains(t, toolNames, "create_note")
	assert.NotContains(t, toolNames, "edit_note")
	assert.NotContains(t, toolNames, "delete_note")
	assert.NotContains(t, toolNames, "rebuild_index")
}

func TestBuildSDKServer_ReadWriteMode_HasAllTools(t *testing.T) {
	f := newStdioFixture(t, false)
	s := f.sdkServer()
	require.NotNil(t, s)

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	result, err := cs.ListTools(ctx, nil)
	require.NoError(t, err)

	toolNames := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	assert.Contains(t, toolNames, "list_notes")
	assert.Contains(t, toolNames, "read_notes")
	assert.Contains(t, toolNames, "search_notes")
	assert.Contains(t, toolNames, "list_tags")
	assert.Contains(t, toolNames, "list_backlinks")
	assert.Contains(t, toolNames, "doctor")
	assert.Contains(t, toolNames, "create_note")
	assert.Contains(t, toolNames, "edit_note")
	assert.Contains(t, toolNames, "delete_note")
	assert.Contains(t, toolNames, "rebuild_index")
}

func TestRun_NilServer(t *testing.T) {
	var s *Server
	err := s.Run(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stdio server is required")
}

func TestBuildToolDescription_Both(t *testing.T) {
	result := buildToolDescription("KB description", "base instructions")
	assert.Equal(t, "KB description\n\nbase instructions", result)
}

func TestBuildToolDescription_EmptyDescription(t *testing.T) {
	result := buildToolDescription("", "base instructions")
	assert.Equal(t, "base instructions", result)
}

func TestBuildToolDescription_EmptyBaseInstructions(t *testing.T) {
	result := buildToolDescription("KB description", "")
	assert.Equal(t, "KB description", result)
}

func TestBuildToolDescription_WhitespaceOnly(t *testing.T) {
	result := buildToolDescription("  ", "instructions")
	assert.Equal(t, "instructions", result)
}

func TestBoolPtr_True(t *testing.T) {
	ptr := BoolPtr(true)
	require.NotNil(t, ptr)
	assert.True(t, *ptr)
}

func TestBoolPtr_False(t *testing.T) {
	ptr := BoolPtr(false)
	require.NotNil(t, ptr)
	assert.False(t, *ptr)
}

func TestCallTool_ReadNotes_NotFound(t *testing.T) {
	f := newStdioFixture(t, false)
	s := f.sdkServer()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	result, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "read_notes",
		Arguments: map[string]any{"identifiers": []string{"nonexistent"}},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

func TestCallTool_DeleteNote_SoftDelete(t *testing.T) {
	f := newStdioFixture(t, false)
	s := f.sdkServer()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	result, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "delete_note",
		Arguments: map[string]any{
			"identifier": "nonexistent",
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestCallTool_EditNote_Append(t *testing.T) {
	f := newStdioFixture(t, false)
	s := f.sdkServer()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	result, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "edit_note",
		Arguments: map[string]any{
			"identifier": "nonexistent",
			"append":     "test",
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestCallTool_ListBacklinks(t *testing.T) {
	f := newStdioFixture(t, false)
	s := f.sdkServer()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	result, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "list_backlinks",
		Arguments: map[string]any{"identifier": "nonexistent"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestCallTool_RebuildIndex(t *testing.T) {
	f := newStdioFixture(t, false)
	s := f.sdkServer()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	result, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "rebuild_index",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

func TestCallTool_CreateNote(t *testing.T) {
	f := newStdioFixture(t, false)
	s := f.sdkServer()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	result, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "create_note",
		Arguments: map[string]any{
			"title": "Test Note",
			"body":  "Test body",
		},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

func TestContractReadNotesDefaultFields(t *testing.T) {
	f := newStdioFixture(t, false)
	_, _ = f.rt.Services.Index.Rebuild(context.Background())
	cs := newMCPClient(t, f)

	result, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "read_notes",
		Arguments: map[string]any{"identifiers": []string{"demo"}},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	var output ReadNotesOutput
	mustUnmarshalMCPContent(t, result, &output)
	require.NotEmpty(t, output.Notes)
	note := output.Notes[0]
	assert.NotEmpty(t, note.NoteID)
	assert.NotEmpty(t, note.Slug)
	assert.NotEmpty(t, note.Title)
}

func TestContractReadNotesUnknownFieldReturnsError(t *testing.T) {
	f := newStdioFixture(t, false)
	_, _ = f.rt.Services.Index.Rebuild(context.Background())
	cs := newMCPClient(t, f)

	result, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "read_notes",
		Arguments: map[string]any{"identifiers": []string{"demo"}, "fields": []string{"nonexistent_field"}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestContractReadNotesMissingSelectors(t *testing.T) {
	f := newStdioFixture(t, false)
	_, _ = f.rt.Services.Index.Rebuild(context.Background())
	cs := newMCPClient(t, f)

	result, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "read_notes",
		Arguments: map[string]any{"identifiers": []string{"demo", "nonexistent"}},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	var output ReadNotesOutput
	mustUnmarshalMCPContent(t, result, &output)
	require.Len(t, output.Notes, 1)
	require.Len(t, output.Missing, 1)
	assert.Equal(t, "nonexistent", output.Missing[0])
}

func TestContractReadNotesCustomFields(t *testing.T) {
	f := newStdioFixture(t, false)
	_, _ = f.rt.Services.Index.Rebuild(context.Background())
	cs := newMCPClient(t, f)

	result, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "read_notes",
		Arguments: map[string]any{"identifiers": []string{"demo"}, "fields": []string{"tags", "aliases"}},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	var output ReadNotesOutput
	mustUnmarshalMCPContent(t, result, &output)
	require.NotEmpty(t, output.Notes)
}

func TestContractSearchNotesDefaultLimit(t *testing.T) {
	f := newStdioFixture(t, false)
	_, _ = f.rt.Services.Index.Rebuild(context.Background())
	cs := newMCPClient(t, f)

	result, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"queries": []string{"Body"}},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	var output SearchNotesOutput
	mustUnmarshalMCPContent(t, result, &output)
	assert.NotEmpty(t, output.Hits)
}

func TestContractSearchNotesDebugOutput(t *testing.T) {
	f := newStdioFixture(t, false)
	_, _ = f.rt.Services.Index.Rebuild(context.Background())
	cs := newMCPClient(t, f)

	result, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "search_notes",
		Arguments: map[string]any{"queries": []string{"Body"}, "debug": true},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	var output SearchNotesOutput
	mustUnmarshalMCPContent(t, result, &output)
	require.NotEmpty(t, output.Hits)
	assert.NotEmpty(t, output.Hits[0].Path, "path should be present with debug")
	assert.NotNil(t, output.Hits[0].Score, "score should be present with debug")
	assert.NotEmpty(t, output.Hits[0].ContentHash, "content_hash should be present with debug")
}

func TestContractDiagnoseNotesStructuredFields(t *testing.T) {
	f := newStdioFixture(t, false)
	_, _ = f.rt.Services.Index.Rebuild(context.Background())
	cs := newMCPClient(t, f)

	result, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "diagnose_notes",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	var output DiagnoseNotesOutput
	mustUnmarshalMCPContent(t, result, &output)
	assert.NotNil(t, output.Issues)
}

func TestContractDiagnoseNotesCursorBeyondResult(t *testing.T) {
	f := newStdioFixture(t, false)
	_, _ = f.rt.Services.Index.Rebuild(context.Background())
	cs := newMCPClient(t, f)

	result, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "diagnose_notes",
		Arguments: map[string]any{"cursor": 99999},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	var output DiagnoseNotesOutput
	mustUnmarshalMCPContent(t, result, &output)
	assert.Empty(t, output.NextCursor)
}

func mustUnmarshalMCPContent(t *testing.T, result *sdkmcp.CallToolResult, target any) {
	t.Helper()
	require.NotEmpty(t, result.Content)
	tc, ok := result.Content[0].(*sdkmcp.TextContent)
	require.True(t, ok, "first content item must be TextContent")
	require.NoError(t, json.Unmarshal([]byte(tc.Text), target))
}

func newMCPClient(t *testing.T, f *stdioFixture) *sdkmcp.ClientSession {
	t.Helper()

	s := f.sdkServer()
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ss, err := s.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ss.Close() })

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })

	return cs
}
