package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/fforster/gitlab-mcp/lib/auth"
	"gitlab.com/fforster/gitlab-mcp/lib/glclient"
)

type authCommand struct {
	baseURL  glclient.BaseURLFlag
	clientID string
}

func newAuthCommand() *cobra.Command {
	ac := new(authCommand)

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Create an authentication token",
		Long:  "Performs the OAuth2 Device Authorization Grant Flow to obtain an access token for the GitLab MCP server.",
		RunE:  ac.run,
		Args:  cobra.NoArgs,
	}

	fs := cmd.Flags()
	fs.Var(&ac.baseURL, "base-url", "Custom GitLab instance URL")
	fs.StringVar(&ac.clientID, "client-id", auth.DefaultClientID, "OAuth client ID for custom GitLab instance")

	return cmd
}

// runAuth performs the OAuth Device Authorization Grant Flow with GitLab to obtain an access token
// for the GitLab MCP server.
func (ac *authCommand) run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create a new OAuth2 config
	config := auth.Config(ac.baseURL.String(), ac.clientID)

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

	if err := auth.StoreToken(auth.TokenPath(), ac.baseURL.String(), token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	fmt.Println("Success.")

	return nil
}
