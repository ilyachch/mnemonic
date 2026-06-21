package cli

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/index"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/web"
	"github.com/spf13/cobra"
)

var (
	webServePortFlag string
	webServeAddrFlag string
	openWebIndexDB   = openWebIndexDBReal
)

var webServeCmd = &cobra.Command{
	Use:          "serve",
	Short:        "Serve MCP over HTTP/SSE",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		container, err := mustAppContainer()
		if err != nil {
			return err
		}

		resolvedProject, err := container.Services.ProjectResolver.Resolve(app.ProjectResolveInput{
			ProjectSelector:  projectSelectorValue(),
			EnvironmentValue: os.Getenv(project.EnvironmentProjectSelector),
		})
		if err != nil {
			return err
		}

		indexDB, err := openWebIndexDB(resolvedProject)
		if err != nil {
			return err
		}

		manager, err := web.NewServer(resolvedProject, container.Paths, indexDB, os.Getenv("MNEMONIC_WEB_TOKEN"), mcpReadOnlyEnabled())
		if err != nil {
			_ = indexDB.Close()
			return err
		}
		defer func() {
			_ = manager.Close()
		}()

		addr := web.ServeAddr(webServeAddrFlag)
		if strings.TrimSpace(webServePortFlag) != "" {
			addr = normalizeWebListenAddr(webServePortFlag)
		}
		server := &http.Server{
			Addr:    addr,
			Handler: manager,
		}

		err = server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("serve web MCP: %w", err)
		}
		return nil
	},
}

func init() {
	webServeCmd.Flags().StringVar(&webServePortFlag, "port", "", "listen port for the web server")
	webServeCmd.Flags().StringVar(&webServeAddrFlag, "addr", "", "listen address for the web server")
	_ = webServeCmd.Flags().MarkHidden("addr")

	webCmd.AddCommand(webServeCmd)
}

func normalizeWebListenAddr(portFlag string) string {
	port := strings.TrimSpace(strings.TrimPrefix(portFlag, ":"))
	if port == "" {
		return ""
	}
	return fmt.Sprintf(":%s", port)
}

func openWebIndexDBReal(resolution app.ProjectResolution) (*sql.DB, error) {
	indexPath, err := index.Path(resolution.Project.ID)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			return nil, app.NewNotFoundError(fmt.Sprintf("index for %q is missing; run `mnemonic project reindex`", resolution.Project.Slug), nil)
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
		return nil, fmt.Errorf("apply busy timeout: %w", err)
	}
	return db, nil
}
