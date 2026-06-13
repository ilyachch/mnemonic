package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

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

	command, args := mnemonicCommand(repoRoot)
	if *reindex {
		if err := runReindex(repoRoot, command, args, *project); err != nil {
			fatalf("reindex: %v", err)
		}
	}

	ctx := context.Background()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "mcp-smoke", Version: "v0.0.1"}, nil)

	cmd := exec.Command(command, append(args, mcpArgs(*project)...)...)
	cmd.Dir = repoRoot
	cmd.Stderr = os.Stderr

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

func runReindex(repoRoot, command string, args []string, project string) error {
	reindexArgs := append([]string{}, args...)
	reindexArgs = append(reindexArgs, "project", "reindex")
	if project != "" {
		reindexArgs = append(reindexArgs, project)
	}

	cmd := exec.Command(command, reindexArgs...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
		return fmt.Errorf("tool returned error")
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
