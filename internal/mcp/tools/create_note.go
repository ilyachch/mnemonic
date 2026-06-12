package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
	mnemonicfs "github.com/ilyachch/mnemonic/internal/fs"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/notes"
	"github.com/ilyachch/mnemonic/internal/project"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const createNoteDescription = `Save new durable knowledge (workflows, preferences, facts).
YOU MUST persist learned context for future use. Search first to avoid duplicates (use 'edit_note' if related context exists). Do NOT save secrets or chat logs.`

type CreateNoteInput struct {
	Title string   `json:"title" jsonschema:"Clear title for durable project knowledge."`
	Body  string   `json:"body,omitempty" jsonschema:"Markdown body with verified durable knowledge, context, and useful links. Do not include secrets, guesses, or temporary chat details."`
	Path  string   `json:"path,omitempty" jsonschema:"Optional relative path under the project memory root."`
	Tags  []string `json:"tags,omitempty" jsonschema:"Optional project memory tags. Prefer existing tag conventions."`
	Type  string   `json:"type,omitempty" jsonschema:"Optional note type, such as decision, convention, architecture, bug, setup, api, integration, or task-outcome."`
}

type CreateNoteOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
}

func RegisterCreateNote(s *sdkmcp.Server, deps Dependencies) {
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "create_note",
		Description: createNoteDescription,
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: BoolPtr(false),
			OpenWorldHint:   BoolPtr(false),
		},
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input CreateNoteInput) (*sdkmcp.CallToolResult, CreateNoteOutput, error) {
		root, err := deps.GetMemoriesRoot()
		if err != nil {
			return nil, CreateNoteOutput{}, err
		}

		created, err := createNote(root, input)
		if err != nil {
			return nil, CreateNoteOutput{}, err
		}
		if err := deps.RebuildIndex(root); err != nil {
			return nil, CreateNoteOutput{}, err
		}

		_ = ctx
		return nil, created, nil
	})
}

func createNote(root string, input CreateNoteInput) (CreateNoteOutput, error) {
	if input.Title == "" {
		return CreateNoteOutput{}, app.NewCLIUsageError("note title is required", nil)
	}

	slug, err := project.Slugify(input.Title)
	if err != nil {
		return CreateNoteOutput{}, err
	}

	relPath, err := createNotePath(slug, input.Path)
	if err != nil {
		return CreateNoteOutput{}, err
	}

	absPath := filepath.Join(root, filepath.FromSlash(relPath))
	if err := ensurePathInsideRoot(root, absPath); err != nil {
		return CreateNoteOutput{}, err
	}
	if _, err := os.Stat(absPath); err == nil {
		return CreateNoteOutput{}, app.NewAmbiguousError(fmt.Sprintf("note path %q already exists", relPath), nil)
	} else if !os.IsNotExist(err) {
		return CreateNoteOutput{}, fmt.Errorf("check note path %q: %w", absPath, err)
	}

	timestamp := project.NowUTC().UTC()
	note := markdown.Note{
		MnemonicNoteID: project.NewUUID(),
		Title:          input.Title,
		Slug:           slug,
		Tags:           dedupeTags(input.Tags),
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
		Type:           input.Type,
		Body:           []byte(input.Body),
	}

	rendered, err := markdown.RenderNote(note)
	if err != nil {
		return CreateNoteOutput{}, err
	}
	if err := mnemonicfs.AtomicWriteFile(absPath, rendered, 0o644); err != nil {
		return CreateNoteOutput{}, err
	}

	return CreateNoteOutput{
		NoteID:      note.MnemonicNoteID,
		Slug:        slug,
		Path:        relPath,
		ContentHash: notes.HashBytes(rendered),
	}, nil
}

func createNotePath(slug, requestedPath string) (string, error) {
	if requestedPath == "" {
		return slug + ".md", nil
	}

	cleaned := filepath.ToSlash(filepath.Clean(requestedPath))
	switch {
	case cleaned == ".", cleaned == "":
		return "", app.NewCLIUsageError("note path is required", nil)
	case filepath.IsAbs(requestedPath), strings.HasPrefix(cleaned, "../"), cleaned == "..":
		return "", app.NewCLIUsageError("note path must be relative to the project memories root", nil)
	}
	if filepath.Ext(cleaned) == "" {
		cleaned += ".md"
	}
	return cleaned, nil
}

func ensurePathInsideRoot(root, target string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve memories root path: %w", err)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve note path: %w", err)
	}

	if absTarget != absRoot && !strings.HasPrefix(absTarget, absRoot+string(os.PathSeparator)) {
		return app.NewCLIUsageError("note path must stay inside the project memories root", nil)
	}

	return nil
}

func dedupeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}

	return out
}