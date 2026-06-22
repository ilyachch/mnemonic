package stdio

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/buildinfo"
	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/project"
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
		return nil, fmt.Errorf("knowledge base is required")
	}
	if services.Notes == nil {
		return nil, fmt.Errorf("notes service is required")
	}
	if services.Search == nil {
		return nil, fmt.Errorf("search service is required")
	}
	if services.Index == nil {
		return nil, fmt.Errorf("index service is required")
	}

	return &Server{
		KB:       k,
		Services: services,
		ReadOnly: readOnly,
	}, nil
}

// Run starts the stdio adapter on the provided MCP transport.
func (s *Server) Run(ctx context.Context, transport sdkmcp.Transport) error {
	if s == nil {
		return fmt.Errorf("stdio server is required")
	}
	return s.BuildSDKServer().Run(ctx, transport)
}

// BuildSDKServer creates the configured SDK MCP server for the selected KB.
func (s *Server) BuildSDKServer() *sdkmcp.Server {
	sdkServer := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "mnemonic", Version: buildinfo.Version()},
		&sdkmcp.ServerOptions{
			Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
			Capabilities: &sdkmcp.ServerCapabilities{
				Tools: &sdkmcp.ToolCapabilities{ListChanged: true},
			},
		},
	)

	RegisterAll(sdkServer, s.Services, s.readManifestDescription(), s.ReadOnly)
	return sdkServer
}

func (s *Server) readManifestDescription() string {
	if s == nil || strings.TrimSpace(s.KB.ManifestPath) == "" {
		return ""
	}

	data, err := os.ReadFile(s.KB.ManifestPath)
	if err != nil {
		return ""
	}
	manifest, err := project.ParseMnemonicManifest(data)
	if err != nil {
		return ""
	}
	return manifest.Description
}
