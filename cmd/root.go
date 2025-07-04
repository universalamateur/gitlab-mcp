// Package cmd implements the root command for Cobra.
package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"gitlab.com/fforster/gitlab-mcp/lib/build"
	"gitlab.com/fforster/gitlab-mcp/lib/glclient"
	"gitlab.com/fforster/gitlab-mcp/lib/glctx"
	"gitlab.com/fforster/gitlab-mcp/lib/tools"
)

// New creates a new command hierarchy for Cobra with the provided GitLab client.
func New() *cobra.Command {
	cmd := newRootCommand()

	cmd.AddCommand(newAuthCommand())
	cmd.AddCommand(newVersionCommand())

	return cmd
}

type rootCommand struct {
	baseURL  glclient.BaseURLFlag
	clientID string
}

// newRootCommand returns the root command for the CLI.
func newRootCommand() *cobra.Command {
	rc := new(rootCommand)

	cmd := &cobra.Command{
		Use:   "gitlab-mcp",
		Short: "GitLab MCP server",
		Long:  "A command-line tool that provides an MCP server for interacting with GitLab.",
		RunE:  rc.run,
		Args:  cobra.NoArgs,
	}

	fs := cmd.Flags()
	fs.Var(&rc.baseURL, "base-url", "Custom GitLab instance URL")
	fs.StringVar(&rc.clientID, "client-id", "", "OAuth client ID for custom GitLab instance")

	return cmd
}

func (c *rootCommand) run(cmd *cobra.Command, _ []string) error {
	cfg, err := glclient.NewConfig(cmd.Flags())
	if err != nil {
		return fmt.Errorf("glclient.NewConfig: %w", err)
	}

	client, err := glclient.New(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("creating GitLab client: %w", err)
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"GitLab",
		build.Version(),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	tools.AddTo(s)

	opts := []server.StdioOption{
		server.WithStdioContextFunc(func(ctx context.Context) context.Context {
			return glctx.WithClient(ctx, client)
		}),
	}

	// Start the server
	err = server.ServeStdio(s, opts...)

	switch {
	case errors.Is(err, context.Canceled):
		return nil

	case err != nil:
		return fmt.Errorf("ServeStdio: %w", err)
	}

	return nil
}
