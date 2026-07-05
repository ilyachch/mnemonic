package cli

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	webadapter "github.com/ilyachch/mnemonic/internal/adapter/web"
	"github.com/spf13/cobra"
)

var (
	resolveRuntimeApp      = runtimeAppForSelectedProject
	newWebServer           = webadapter.NewServer
	webServeListenAndServe = webServeListenAndServeReal
)

func newWebServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "serve",
		Short:        "Serve MCP over HTTP/SSE",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, err := resolveRuntimeApp(cmd)
			if err != nil {
				return err
			}

			manager, err := newWebServer(webadapter.ServerInput{
				KB:           runtime.KB,
				Services:     runtime.Services,
				ProjectToken: os.Getenv("MNEMONIC_PROJECT_TOKEN"),
				ReadOnly:     stdioReadOnlyEnabled(cmd),
				Logger:       loggerFromContext(commandContext(cmd)),
			})
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
			addr := resolveWebServeAddr(addrFlag, portFlag)
			server := &http.Server{
				Addr:    addr,
				Handler: manager,
			}

			err = webServeListenAndServe(server)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("serve web MCP: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().String("port", "", "listen port for the web server")
	cmd.Flags().String("addr", "", "listen address for the web server")
	_ = cmd.Flags().MarkHidden("addr")
	return cmd
}

func normalizeWebListenAddr(portFlag string) string {
	port := strings.TrimSpace(strings.TrimPrefix(portFlag, ":"))
	if port == "" {
		return ""
	}
	return ":" + port
}

func resolveWebServeAddr(addrFlag, portFlag string) string {
	if strings.TrimSpace(portFlag) != "" {
		return normalizeWebListenAddr(portFlag)
	}
	if addr := strings.TrimSpace(addrFlag); addr != "" {
		return addr
	}
	if envValue := strings.TrimSpace(os.Getenv("MNEMONIC_WEB_ADDR")); envValue != "" {
		return envValue
	}
	return ":8080"
}

func webServeListenAndServeReal(server *http.Server) error {
	return server.ListenAndServe()
}
