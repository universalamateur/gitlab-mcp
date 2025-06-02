package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"gitlab.com/fforster/gitlab-mcp/lib/auth"
)

func newAuthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "auth",
		Short: "Create an authentication token",
		Long:  "Performs the OAuth2 Device Authorization Grant Flow to obtain an access token for the GitLab MCP server.",
		RunE:  runAuth,
		Args:  cobra.NoArgs,
	}
}

// runAuth performs the OAuth Device Authorization Grant Flow with GitLab to obtain an access token
// for the GitLab MCP server.
func runAuth(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create a new OAuth2 config
	config := auth.Config()

	da, err := config.DeviceAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device auth request: %w", err)
	}

	fmt.Println("Please visit the following URL to authorize the application:")
	switch {
	case da.VerificationURIComplete != "":
		fmt.Println(da.VerificationURIComplete)
	case da.VerificationURI != "" && da.UserCode != "":
		fmt.Println(da.VerificationURI)
		fmt.Println("and enter code", da.UserCode)
	default:
		fmt.Println(da.VerificationURI)
	}
	fmt.Println()

	fmt.Println("Waiting to retrieve an access token...")

	token, err := config.DeviceAccessToken(ctx, da)
	if err != nil {
		return fmt.Errorf("failed to obtain device access token: %w", err)
	}

	if err := storeToken(token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	fmt.Println("Success.")

	return nil
}

func storeToken(token *oauth2.Token) error {
	filename := auth.TokenPath()

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("MkdirAll(%q): %w", filepath.Dir(filename), err)
	}

	if err := auth.StoreToken(filename, token); err != nil {
		return fmt.Errorf("StoreToken(): %w", err)
	}

	return nil
}
