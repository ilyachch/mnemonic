package cli

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ilyachch/mnemonic/internal/paths"
	"github.com/ilyachch/mnemonic/internal/web"
	"github.com/ilyachch/mnemonic/internal/webauth"
	"github.com/spf13/cobra"
)

var (
	webAddrFlag   string
	webAuthDBFlag string
)

var webCmd = &cobra.Command{
	Use:          "web",
	Short:        "Run the web MCP server",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var webServeCmd = &cobra.Command{
	Use:          "serve",
	Short:        "Serve MCP over HTTP/SSE",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		mnemonicPaths, err := paths.GetMnemonicPaths()
		if err != nil {
			return err
		}

		slugs := web.ParseServeProjects(os.Getenv("MNEMONIC_SERVE_PROJECTS"))
		validator, err := web.NewServerManager(nil, os.Getenv("MNEMONIC_SUPERUSER_TOKEN"), mnemonicPaths.MemoriesHome, slugs)
		if err != nil {
			return err
		}
		_ = validator.Close()

		authPath, err := webauth.ResolveDBPath(webAuthDBFlag)
		if err != nil {
			return err
		}
		authStore, err := webauth.NewStore(authPath)
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

		server := &http.Server{
			Addr:    web.ServeAddr(webAddrFlag),
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
	webServeCmd.Flags().StringVar(&webAddrFlag, "addr", "", "listen address for the web server")
	webServeCmd.Flags().StringVar(&webAuthDBFlag, "auth-db", "", "path to the web auth database")
	webCmd.AddCommand(webServeCmd)
	RootCmd.AddCommand(webCmd)
}
