package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/buildinfo"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/mcp/tools"
	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/project"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server is a transport shell for the MCP stdio adapter.
type Server struct {
	Project  app.ProjectResolution
	Paths    paths.EffectivePaths
	ReadOnly bool

	indexDBMu    sync.Mutex
	indexConn    *sql.DB
	indexDBOwned bool

	// testCloseDBErr, when set, makes closeIndexDB return this error.
	// This is a test-only hook and must never be set in production.
	testCloseDBErr error
}

// NewServer creates a new MCP shell server wrapper.
func NewServer(resolved app.ProjectResolution, effectivePaths paths.EffectivePaths, readOnly bool) *Server {
	return &Server{Project: resolved, Paths: effectivePaths, ReadOnly: readOnly}
}

// NewServerWithIndexDB creates a new MCP shell server wrapper that reuses an
// already-open index DB connection.
func NewServerWithIndexDB(resolved app.ProjectResolution, effectivePaths paths.EffectivePaths, indexDB *sql.DB, readOnly bool) *Server {
	return &Server{
		Project:      resolved,
		Paths:        effectivePaths,
		ReadOnly:     readOnly,
		indexConn:    indexDB,
		indexDBOwned: false,
	}
}

// Run starts the MCP server with the provided transport.
func (s *Server) Run(ctx context.Context, transport sdkmcp.Transport) error {
	defer func() {
		_ = s.closeIndexDB()
	}()

	return s.BuildSDKServer().Run(ctx, transport)
}

// BuildSDKServer creates a configured SDK MCP server for this project.
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

	tools.RegisterAll(sdkServer, s, s.readManifestDescription(), s.ReadOnly)
	return sdkServer
}

// readManifestDescription reads the optional description field from the
// project's mnemonic.toml manifest. Returns an empty string if the manifest
// cannot be read or parsed, so the MCP server always starts gracefully.
func (s *Server) readManifestDescription() string {
	if s.Project.ManifestAbs == "" {
		return ""
	}
	data, err := os.ReadFile(s.Project.ManifestAbs)
	if err != nil {
		return ""
	}
	manifest, err := project.ParseMnemonicManifest(data)
	if err != nil {
		return ""
	}
	return manifest.Description
}

// GetMemoriesRoot returns the resolved memories root path.
func (s *Server) GetMemoriesRoot() (string, error) {
	repoRoot := s.Project.RepoRootAbs
	if repoRoot == "" {
		repoRoot = filepath.Dir(s.Project.MnemonicFilePath)
	}
	return project.ResolveMemoriesRoot(project.MemoriesRootInput{
		Kind:         s.Project.Project.Kind,
		Slug:         s.Project.Project.Slug,
		MemoriesPath: s.Project.Project.MemoriesPath,
		MemoriesHome: s.Paths.MemoriesHome,
		RepoRoot:     repoRoot,
	})
}

// GetIndexDB returns the index database connection (singleton).
func (s *Server) GetIndexDB() (*sql.DB, error) {
	if s == nil {
		return nil, fmt.Errorf("server is nil")
	}

	s.indexDBMu.Lock()
	defer s.indexDBMu.Unlock()

	if s.indexConn != nil {
		return s.indexConn, nil
	}

	indexPath, err := index.Path(s.Project.Project.ID)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("index missing; run `mnemonic project reindex`")
		}
		return nil, fmt.Errorf("stat index %q: %w", indexPath, err)
	}

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(indexPath)+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open index database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping index database: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply PRAGMA busy_timeout: %w", err)
	}

	s.indexConn = db
	s.indexDBOwned = true
	return s.indexConn, nil
}

// RebuildIndex closes the current index DB connection and rebuilds the index.
func (s *Server) RebuildIndex(root string) error {
	if s == nil {
		return fmt.Errorf("server is nil")
	}
	if err := s.closeIndexDB(); err != nil {
		return err
	}
	if _, err := index.RebuildProjectIndex(s.Project.Project.ID, root); err != nil {
		return err
	}
	return nil
}

func (s *Server) closeIndexDB() error {
	if s == nil {
		return nil
	}

	if s.testCloseDBErr != nil {
		return s.testCloseDBErr
	}

	s.indexDBMu.Lock()
	defer s.indexDBMu.Unlock()

	if s.indexConn == nil {
		return nil
	}

	db := s.indexConn
	s.indexConn = nil
	owned := s.indexDBOwned
	s.indexDBOwned = false
	if !owned {
		return nil
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("close index database: %w", err)
	}
	return nil
}
