package cli

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/project"
	"github.com/ilyachch/mnemonic/internal/web"
	"github.com/spf13/cobra"
)

var (
	webServePortFlag string
	webServeAddrFlag string
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

		manager, err := web.NewServerManager(resolvedProject, container.Paths.MemoriesHome)
		if err != nil {
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
