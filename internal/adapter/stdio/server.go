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

const defaultGlobalInstructions = "You are connected to mnemonic, a local-first knowledge base and search engine. Use the available MCP tools to read, search, create, edit, and manage notes in the selected project. Be concise and keep answers grounded in the project's notes."

// Run starts the stdio adapter on the provided MCP transport.
func (s *Server) Run(ctx context.Context, transport sdkmcp.Transport) error {
	if s == nil {
		return errors.New("stdio server is required")
	}
	return s.BuildSDKServer().Run(ctx, transport)
}

// BuildSDKServer creates the configured SDK MCP server for the selected KB.
func (s *Server) BuildSDKServer() *sdkmcp.Server {
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "mnemonic", Version: buildinfo.Version()},
		&sdkmcp.ServerOptions{
			Instructions: buildGlobalInstructions(s.KB.CustomInstructions, s.KB.Description),
			Logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
			Capabilities: &sdkmcp.ServerCapabilities{
				Tools: &sdkmcp.ToolCapabilities{ListChanged: true},
			},
		},
	)

	RegisterAll(sdkServer, s.Services, s.KB.Description, s.ReadOnly)
	return sdkServer
}

func buildGlobalInstructions(customInstructions, description string) string {
	parts := make([]string, 0, 3)
	if trimmed := strings.TrimSpace(customInstructions); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if trimmed := strings.TrimSpace(description); trimmed != "" {
		parts = append(parts, trimmed)
	}
	parts = append(parts, defaultGlobalInstructions)
	return strings.Join(parts, "\n\n")
}
