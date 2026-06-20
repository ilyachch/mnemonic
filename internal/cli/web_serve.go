package cli

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/web"
	"github.com/spf13/cobra"
)

var (
	webServePortFlag     string
	webServeProjectsFlag string
	webServeAddrFlag     string
	webServeAuthDBFlag   string
)

var webServeCmd = &cobra.Command{
	Use:          "serve",
	Short:        "Serve MCP over HTTP/SSE",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		mnemonicPaths, err := paths.GetMnemonicPaths()
		if err != nil {
			return err
		}

		slugs := resolveWebServeProjects(webServeProjectsFlag)
		validator, err := web.NewServerManager(nil, os.Getenv("MNEMONIC_SUPERUSER_TOKEN"), mnemonicPaths.MemoriesHome, slugs)
		if err != nil {
			return err
		}
		_ = validator.Close()

		authStore, err := openWebAuthStore(webServeAuthDBFlag)
		if err != nil {
			return err
		}

		manager, err := web.NewServerManager(authStore, os.Getenv("MNEMONIC_SUPERUSER_TOKEN"), mnemonicPaths.MemoriesHome, slugs)
		if err != nil {
			_ = authStore.Close()
			return err
		}
		defer func() {
			_ = manager.Close()
		}()

		addr := web.ServeAddr(webServeAddrFlag)
		if strings.TrimSpace(webServePortFlag) != "" {
			addr = normalizeWebListenAddr(webServePortFlag, "")
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
	webCmd.PersistentFlags().StringVar(&webServeAuthDBFlag, "auth-db", "", "path to the web auth database")

	webServeCmd.Flags().StringVar(&webServePortFlag, "port", "", "listen port for the web server")
	webServeCmd.Flags().StringVar(&webServeProjectsFlag, "projects", "", "comma-separated project slugs to expose")
	webServeCmd.Flags().StringVar(&webServeAddrFlag, "addr", "", "listen address for the web server")
	_ = webServeCmd.Flags().MarkHidden("addr")

	webCmd.AddCommand(webServeCmd)
}
