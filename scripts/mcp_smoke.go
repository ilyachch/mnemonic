package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	manifestfmt "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/platform/idgen"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type createNoteOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
}

type resultEnvelope struct {
	StructuredContent any              `json:"structured_content,omitempty"`
	Content           []contentSummary `json:"content,omitempty"`
	IsError           bool             `json:"is_error"`
}

type contentSummary struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func main() {
	var (
		project   = flag.String("project", "", "optional mnemonic project selector")
		reindex   = flag.Bool("reindex", false, "run `mnemonic project reindex` before MCP checks")
		keepNotes = flag.Bool("keep-notes", false, "keep the created smoke-test notes instead of deleting them")
		timeout   = flag.Duration("timeout", 30*time.Second, "per-call timeout")
	)
	flag.Parse()

	repoRoot, err := os.Getwd()
	if err != nil {
		fatalf("getwd: %v", err)
	}

	env := os.Environ()
	if *project == "" {
		*project, env, err = prepareSmokeProject()
		if err != nil {
			fatalf("prepare smoke project: %v", err)
		}
		*reindex = true
	}

	command, args := mnemonicCommand(repoRoot)
	if *reindex {
		if err := runReindex(repoRoot, command, args, *project, env); err != nil {
			fatalf("reindex: %v", err)
		}
	}

	ctx := context.Background()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "mcp-smoke", Version: "v0.0.1"}, nil)

	cmd := exec.Command(command, append(args, mcpArgs(*project)...)...)
	cmd.Dir = repoRoot
	cmd.Stderr = os.Stderr
	cmd.Env = env

	session, err := client.Connect(ctx, &sdkmcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		fatalf("connect mcp: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	suffix := time.Now().UTC().Format("20060102-150405")
	alphaTitle := "MCP Smoke Alpha " + suffix
	alphaSlug := "mcp-smoke-alpha-" + suffix
	betaTitle := "MCP Smoke Beta " + suffix
	uniqueToken := "mcp-smoke-token-" + suffix

	var alpha createNoteOutput
	var beta createNoteOutput

	steps := []struct {
		name string
		run  func() error
	}{
		{
			name: "list_notes.before",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "list_notes", map[string]any{"limit": 20}, nil)
			},
		},
		{
			name: "list_tags.before",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "list_tags", map[string]any{"limit": 20}, nil)
			},
		},
		{
			name: "search_notes.before",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "search_notes", map[string]any{"query": uniqueToken, "limit": 10}, nil)
			},
		},
		{
			name: "create_note.alpha",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "create_note", map[string]any{
					"title": alphaTitle,
					"path":  alphaSlug + ".md",
					"body":  "Alpha smoke body.",
					"tags":  []string{"mcp", "smoke"},
				}, &alpha)
			},
		},
		{
			name: "create_note.beta",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "create_note", map[string]any{
					"title": betaTitle,
					"body":  fmt.Sprintf("Links to [[%s]].\nUnique token: %s.", alphaSlug, uniqueToken),
				}, &beta)
			},
		},
		{
			name: "list_notes.after_create",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "list_notes", map[string]any{"limit": 20}, nil)
			},
		},
		{
			name: "read_note.alpha",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "read_note", map[string]any{"identifier": alpha.NoteID}, nil)
			},
		},
		{
			name: "read_note.beta",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "read_note", map[string]any{"identifier": beta.NoteID}, nil)
			},
		},
		{
			name: "list_tags.after_create",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "list_tags", map[string]any{"limit": 20}, nil)
			},
		},
		{
			name: "search_notes.quoted",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "search_notes", map[string]any{"query": `"` + uniqueToken + `"`, "limit": 10}, nil)
			},
		},
		{
			name: "search_notes.raw",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "search_notes", map[string]any{"query": uniqueToken, "limit": 10}, nil)
			},
		},
		{
			name: "list_backlinks.alpha",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "list_backlinks", map[string]any{"identifier": alpha.NoteID, "limit": 10}, nil)
			},
		},
		{
			name: "edit_note.beta",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "edit_note", map[string]any{
					"identifier":    beta.NoteID,
					"if_match_hash": beta.ContentHash,
					"append":        "\nEdited by mcp smoke script.",
				}, &beta)
			},
		},
		{
			name: "read_note.beta.after_edit",
			run: func() error {
				return callAndPrint(ctx, session, *timeout, "read_note", map[string]any{"identifier": beta.NoteID}, nil)
			},
		},
	}

	hadFailure := false
	for _, step := range steps {
		if err := step.run(); err != nil {
			hadFailure = true
			fmt.Fprintf(os.Stderr, "STEP FAILED: %s: %v\n", step.name, err)
		}
	}

	if !*keepNotes {
		if beta.NoteID != "" && beta.ContentHash != "" {
			if err := callAndPrint(ctx, session, *timeout, "delete_note", map[string]any{
				"identifier":    beta.NoteID,
				"if_match_hash": beta.ContentHash,
			}, nil); err != nil {
				hadFailure = true
				fmt.Fprintf(os.Stderr, "CLEANUP FAILED: delete beta: %v\n", err)
			}
		}
		if alpha.NoteID != "" && alpha.ContentHash != "" {
			if err := callAndPrint(ctx, session, *timeout, "delete_note", map[string]any{
				"identifier":    alpha.NoteID,
				"if_match_hash": alpha.ContentHash,
			}, nil); err != nil {
				hadFailure = true
				fmt.Fprintf(os.Stderr, "CLEANUP FAILED: delete alpha: %v\n", err)
			}
		}
	}

	if hadFailure {
		os.Exit(1)
	}
}

func mnemonicCommand(repoRoot string) (string, []string) {
	return "go", []string{"run", "./cmd/mnemonic"}
}

func mcpArgs(project string) []string {
	args := []string{"mcp"}
	if project != "" {
		args = append(args, "--project", project)
	}
	return args
}

func runReindex(repoRoot, command string, args []string, project string, env []string) error {
	reindexArgs := append([]string{}, args...)
	reindexArgs = append(reindexArgs, "project", "reindex")
	if project != "" {
		reindexArgs = append(reindexArgs, project)
	}

	cmd := exec.Command(command, reindexArgs...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	return cmd.Run()
}

func prepareSmokeProject() (string, []string, error) {
	base, err := os.MkdirTemp("", "mnemonic-mcp-smoke-")
	if err != nil {
		return "", nil, err
	}

	projectRoot := filepath.Join(base, "project")
	memoriesHome := filepath.Join(base, "memories")
	memoriesDir := filepath.Join(projectRoot, ".mnemonic-memories", "personal")
	if err := os.MkdirAll(memoriesDir, 0o755); err != nil {
		return "", nil, err
	}
	if err := os.MkdirAll(memoriesHome, 0o755); err != nil {
		return "", nil, err
	}

	now := time.Now().UTC()
	manifest := manifestfmt.NewMnemonicManifest()
	manifest.ProjectID = idgen.NewUUID()
	manifest.Name = "personal"
	manifest.Slug = "personal"
	manifest.Type = manifestfmt.ManifestTypeLocal
	manifest.MarkdownFormatVersion = 1
	manifest.CreatedAt = now
	manifest.UpdatedAt = now
	manifest.Generator.App = "mnemonic"
	if err := manifestfmt.WriteMnemonicManifest(filepath.Join(memoriesDir, "mnemonic.toml"), manifest); err != nil {
		return "", nil, err
	}
	if err := manifestfmt.WritePointerFile(filepath.Join(memoriesHome, "personal.toml"), &manifestfmt.PointerFile{
		ManifestPath: filepath.Join(memoriesDir, "mnemonic.toml"),
	}); err != nil {
		return "", nil, err
	}

	env := append([]string{}, os.Environ()...)
	env = append(env,
		"XDG_CONFIG_HOME="+filepath.Join(base, "config"),
		"XDG_DATA_HOME="+filepath.Join(base, "data"),
		"XDG_STATE_HOME="+filepath.Join(base, "state"),
		"XDG_CACHE_HOME="+filepath.Join(base, "cache"),
		"GOMODCACHE="+filepath.Join(os.Getenv("HOME"), ".cache", "go", "pkg", "mod"),
		"MNEMONIC_MEMORIES_HOME="+memoriesHome,
	)

	return "personal", env, nil
}

func callAndPrint(
	ctx context.Context,
	session *sdkmcp.ClientSession,
	timeout time.Duration,
	name string,
	args map[string]any,
	decodeTarget any,
) error {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := session.CallTool(callCtx, &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	printResult(name, result)
	if err != nil {
		return err
	}
	if result == nil {
		return errors.New("nil result")
	}
	if result.IsError {
		return errors.New("tool returned error")
	}
	if decodeTarget != nil {
		if err := decodeStructured(result.StructuredContent, decodeTarget); err != nil {
			return err
		}
	}
	return nil
}

func printResult(name string, result *sdkmcp.CallToolResult) {
	envelope := resultEnvelope{IsError: result != nil && result.IsError}
	if result != nil {
		envelope.StructuredContent = result.StructuredContent
		for _, item := range result.Content {
			switch v := item.(type) {
			case *sdkmcp.TextContent:
				envelope.Content = append(envelope.Content, contentSummary{Type: "text", Text: v.Text})
			default:
				envelope.Content = append(envelope.Content, contentSummary{Type: fmt.Sprintf("%T", item)})
			}
		}
	}

	raw, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		fmt.Printf("=== %s ===\n<marshal error: %v>\n", name, err)
		return
	}
	fmt.Printf("=== %s ===\n%s\n", name, raw)
}

func decodeStructured(src any, dst any) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
