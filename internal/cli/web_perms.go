package cli

import (
	"fmt"
	"strings"

	"github.com/ilyachch/mnemonic/internal/app"
	"github.com/ilyachch/mnemonic/internal/registry"
	"github.com/spf13/cobra"
)

var webPermsLevelFlag string

var webPermsCmd = &cobra.Command{
	Use:          "perms",
	Short:        "Manage web permissions",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var webPermsGrantCmd = &cobra.Command{
	Use:   "grant USERNAME PROJECT",
	Short: "Grant project access to a user",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openWebAuthStore(webServeAuthDBFlag)
		if err != nil {
			return err
		}
		defer func() { _ = store.Close() }()

		user, err := store.GetUserByUsername(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		projectEntry, err := resolveWebProjectSlug(args[1])
		if err != nil {
			return err
		}

		level := strings.TrimSpace(webPermsLevelFlag)
		if level == "" {
			level = "ro"
		}
		if err := store.GrantPermission(cmd.Context(), user.UserID, projectEntry.Slug, level); err != nil {
			return err
		}

		output := webPermsGrantOutput{
			Username:    user.Username,
			ProjectSlug: projectEntry.Slug,
			Level:       level,
		}
		return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s %s %s granted\n", user.Username, projectEntry.Slug, level), output)
	},
}

var webPermsRevokeCmd = &cobra.Command{
	Use:   "revoke USERNAME PROJECT",
	Short: "Revoke project access from a user",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openWebAuthStore(webServeAuthDBFlag)
		if err != nil {
			return err
		}
		defer func() { _ = store.Close() }()

		user, err := store.GetUserByUsername(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		projectEntry, err := resolveWebProjectSlug(args[1])
		if err != nil {
			return err
		}

		if err := store.RevokePermission(cmd.Context(), user.UserID, projectEntry.Slug); err != nil {
			return err
		}

		output := webPermsRevokeOutput{
			Username:    user.Username,
			ProjectSlug: projectEntry.Slug,
		}
		return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s %s revoked\n", user.Username, projectEntry.Slug), output)
	},
}

func init() {
	webPermsGrantCmd.Flags().StringVar(&webPermsLevelFlag, "level", "ro", "permission level to grant: ro or rw")

	webPermsCmd.AddCommand(webPermsGrantCmd)
	webPermsCmd.AddCommand(webPermsRevokeCmd)
	webCmd.AddCommand(webPermsCmd)
}

type webPermsGrantOutput struct {
	Username    string `json:"username"`
	ProjectSlug string `json:"project_slug"`
	Level       string `json:"level"`
}

type webPermsRevokeOutput struct {
	Username    string `json:"username"`
	ProjectSlug string `json:"project_slug"`
}

func resolveWebProjectSlug(slug string) (registry.Entry, error) {
	container, err := mustAppContainer()
	if err != nil {
		return registry.Entry{}, err
	}

	entry, err := registry.Resolve(container.Paths.MemoriesHome, slug)
	if err != nil {
		if _, ok := err.(registry.ErrNotFound); ok {
			return registry.Entry{}, app.NewNotFoundError(err.Error(), nil)
		}
		return registry.Entry{}, err
	}
	return entry, nil
}
