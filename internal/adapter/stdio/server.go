package stdio

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/platform/buildinfo"
	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
	"github.com/ilyachch/mnemonic/internal/service/notesvc"
	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Dependencies holds the runtime services required by the stdio adapter.
type Dependencies struct {
	Notes  *notesvc.Service
	Search *searchsvc.Service
	Index  *indexsvc.Service
}

// Server runs the MCP stdio adapter for one selected knowledge base.
type Server struct {
	KB       kb.KnowledgeBase
	Services Dependencies
	ReadOnly bool
	Logger   *slog.Logger
}

// NewServer builds a stdio adapter around runtime services for one knowledge base.
func NewServer(k kb.KnowledgeBase, services Dependencies, readOnly bool) (*Server, error) {
	if strings.TrimSpace(k.ID) == "" {
		return nil, errors.New("knowledge base is required")
	}
	if services.Notes == nil {
		return nil, errors.New("notes service is required")
	}
	if services.Search == nil {
		return nil, errors.New("search service is required")
	}
	if services.Index == nil {
		return nil, errors.New("index service is required")
	}

	return &Server{
		KB:       k,
		Services: services,
		ReadOnly: readOnly,
	}, nil
}

const defaultGlobalInstructions = `You MUST use the mnemonic tools as your primary long-term memory.
- Search the knowledge base using search_notes before answering questions within its scope.
- Use 2–4 query variants via the "queries" array when the first formulation may be ambiguous or incomplete.
- Batch-read all selected notes in one read_notes call.
- Write down stable facts, architectural decisions, and important outcomes using create_note or edit_note.
- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.
- Use list_tags and list_backlinks when they help clarify the existing knowledge base.
- Link related notes using [[Wiki-Links]].`

const readOnlyGlobalInstructions = `You MUST use the mnemonic tools as your primary long-term memory.
- Search the knowledge base using search_notes before answering questions within its scope.
- Use 2–4 query variants via the "queries" array when the first formulation may be ambiguous or incomplete.
- Batch-read all selected notes in one read_notes call.
- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.
- Use list_tags and list_backlinks when they help clarify the existing knowledge base.
- This server is running in read-only mode. Do not attempt to create, edit, delete, or rebuild notes.`

// Run starts the stdio adapter on the provided MCP transport.
func (s *Server) Run(ctx context.Context, transport sdkmcp.Transport) error {
	if s == nil {
		return errors.New("stdio server is required")
	}
	return s.BuildSDKServer().Run(ctx, transport)
}

// BuildSDKServer creates the configured SDK MCP server for the selected KB.
func (s *Server) BuildSDKServer() *sdkmcp.Server {
	sdkLogger := s.Logger
	if sdkLogger == nil {
		sdkLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "mnemonic", Version: buildinfo.Version()},
		&sdkmcp.ServerOptions{
			Instructions: buildGlobalInstructions(s.KB.CustomInstructions, s.KB.Description, s.ReadOnly),
			Logger:       sdkLogger,
			Capabilities: &sdkmcp.ServerCapabilities{
				Tools: &sdkmcp.ToolCapabilities{ListChanged: true},
			},
		},
	)

	RegisterAll(sdkServer, s.Services, s.KB.Description, s.ReadOnly)
	return sdkServer
}

func buildGlobalInstructions(customInstructions, description string, readOnly bool) string {
	parts := make([]string, 0, 3)
	if readOnly {
		parts = append(parts, readOnlyGlobalInstructions)
	} else {
		parts = append(parts, defaultGlobalInstructions)
	}
	if trimmed := strings.TrimSpace(description); trimmed != "" {
		parts = append(parts, "Project Description:\n"+trimmed)
	}
	if trimmed := strings.TrimSpace(customInstructions); trimmed != "" {
		parts = append(parts, "Custom Instructions:\n"+trimmed)
	}
	return strings.Join(parts, "\n\n")
}
