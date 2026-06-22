package cli

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/web"
	"github.com/spf13/cobra"
)

var (
	newWebServer           = web.NewServer
	webServeListenAndServe = webServeListenAndServeReal
)

var webServeCmd = &cobra.Command{
	Use:          "serve",
	Short:        "Serve MCP over HTTP/SSE",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := runtimeAppForSelectedProject(cmd)
		if err != nil {
			return err
		}

		manager, err := newWebServer(runtime, os.Getenv("MNEMONIC_PROJECT_TOKEN"), stdioReadOnlyEnabled(cmd))
		if err != nil {
			return err
		}
		defer func() { _ = manager.Close() }()

		addrFlag, err := cmd.Flags().GetString("addr")
		if err != nil {
			return err
		}
		portFlag, err := cmd.Flags().GetString("port")
		if err != nil {
			return err
		}
		addr := web.ServeAddr(addrFlag)
		if strings.TrimSpace(portFlag) != "" {
			addr = normalizeWebListenAddr(portFlag)
		}
		server := &http.Server{
			Addr:    addr,
			Handler: manager,
		}

		err = webServeListenAndServe(server)
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("serve web MCP: %w", err)
		}
		return nil
	},
}

func normalizeWebListenAddr(portFlag string) string {
	port := strings.TrimSpace(strings.TrimPrefix(portFlag, ":"))
	if port == "" {
		return ""
	}
	return fmt.Sprintf(":%s", port)
}

func webServeListenAndServeReal(server *http.Server) error {
	return server.ListenAndServe()
}
