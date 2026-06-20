package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/ilyachch/mnemonic/internal/webauth"
	"github.com/spf13/cobra"
)

var webUsersCmd = &cobra.Command{
	Use:          "users",
	Short:        "Manage web users",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var webUsersAddCmd = &cobra.Command{
	Use:   "add USERNAME",
	Short: "Add a web user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openWebAuthStore(webServeAuthDBFlag)
		if err != nil {
			return err
		}
		defer func() { _ = store.Close() }()

		token, err := generateWebAuthToken()
		if err != nil {
			return err
		}

		user, err := store.CreateUser(cmd.Context(), args[0], token)
		if err != nil {
			return err
		}

		output := webUsersAddOutput{
			Username: user.Username,
			Token:    token,
		}
		return PrintOutput(cmd.OutOrStdout(), token+"\n", output)
	},
}

var webUsersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List web users",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openWebAuthStore(webServeAuthDBFlag)
		if err != nil {
			return err
		}
		defer func() { _ = store.Close() }()

		users, err := store.ListUsers(cmd.Context())
		if err != nil {
			return err
		}

		names := make([]string, 0, len(users))
		for _, user := range users {
			names = append(names, user.Username)
		}

		output := webUsersListOutput{Users: names}
		return PrintOutput(cmd.OutOrStdout(), formatWebUsersHuman(users), output)
	},
}

var webUsersRevokeCmd = &cobra.Command{
	Use:   "revoke USERNAME",
	Short: "Revoke a web user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openWebAuthStore(webServeAuthDBFlag)
		if err != nil {
			return err
		}
		defer func() { _ = store.Close() }()

		if err := store.RevokeUser(cmd.Context(), args[0]); err != nil {
			return err
		}

		output := webUsersRevokeOutput{Username: args[0]}
		return PrintOutput(cmd.OutOrStdout(), fmt.Sprintf("%s revoked\n", args[0]), output)
	},
}

func init() {
	webUsersCmd.AddCommand(webUsersAddCmd)
	webUsersCmd.AddCommand(webUsersListCmd)
	webUsersCmd.AddCommand(webUsersRevokeCmd)
	webCmd.AddCommand(webUsersCmd)
}

type webUsersAddOutput struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

type webUsersListOutput struct {
	Users []string `json:"users"`
}

type webUsersRevokeOutput struct {
	Username string `json:"username"`
}

func generateWebAuthToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate web auth token: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

func formatWebUsersHuman(users []webauth.User) string {
	if len(users) == 0 {
		return "0 users\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d users\n", len(users))
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "USERNAME")
	for _, user := range users {
		_, _ = fmt.Fprintln(w, user.Username)
	}
	_ = w.Flush()
	return b.String()
}
