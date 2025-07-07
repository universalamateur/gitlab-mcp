// Package cmd implements the root command for Cobra.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"gitlab.com/fforster/gitlab-mcp/lib/auth"
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
	// MCP flags
	transport  string
	listenAddr string

	// TLS flags
	tlsKey  string
	tlsCert string

	// GitLab flags
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

	// MCP flags
	fs.StringVar(&rc.transport, "transport", "stdio", "Transport protocol to use for communication with the MCP server")
	fs.StringVar(&rc.listenAddr, "listen-addr", ":8080", "Address to listen on, in the form of 'host:port'.")

	// TLS flags
	fs.StringVar(&rc.tlsKey, "tls-key", "", "Path to TLS private key file for HTTPS server")
	fs.StringVar(&rc.tlsCert, "tls-cert", "", "Path to TLS certificate file for HTTPS server")

	// GitLab flags
	fs.Var(&rc.baseURL, "base-url", "Custom GitLab instance URL")
	fs.StringVar(&rc.clientID, "client-id", "", "OAuth client ID for custom GitLab instance")

	return cmd
}

var ErrInvalidTransport = errors.New("invalid transport protocol")

func (c *rootCommand) run(cmd *cobra.Command, _ []string) error {
	// Create a new MCP server
	s := server.NewMCPServer(
		"GitLab",
		build.Version(),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	tools.AddTo(s)

	switch strings.ToLower(c.transport) {
	case "stdio":
		return c.serveStdio(s, cmd)

	case "http":
		return c.serveHTTP(cmd.Context(), s)

	default:
		return fmt.Errorf("%w: %q", ErrInvalidTransport, c.transport)
	}
}

func (c *rootCommand) serveStdio(srv *server.MCPServer, cmd *cobra.Command) error {
	cfg, err := glclient.ConfigFromFlags(cmd.Context(), cmd.Flags())
	if err != nil {
		return fmt.Errorf("glclient.NewConfig: %w", err)
	}

	client, err := glclient.New(cfg)
	if err != nil {
		return fmt.Errorf("creating GitLab client: %w", err)
	}

	opts := []server.StdioOption{
		server.WithStdioContextFunc(func(ctx context.Context) context.Context {
			return glctx.WithClient(ctx, client)
		}),
	}

	// Start the server
	err = server.ServeStdio(srv, opts...)

	switch {
	case errors.Is(err, context.Canceled):
		return nil

	case err != nil:
		return fmt.Errorf("ServeStdio: %w", err)
	}

	return nil
}

//nolint:funlen
func (c *rootCommand) serveHTTP(ctx context.Context, mcpSrv *server.MCPServer) error {
	opts := []server.StreamableHTTPOption{
		server.WithHTTPContextFunc(httpContextFunc(glclient.ServerConfig{
			BaseURL: c.baseURL.String(),
		})),
	}

	httpMCPServer := server.NewStreamableHTTPServer(mcpSrv, opts...)

	mux := http.NewServeMux()

	mux.Handle("/mcp", auth.Wrap(httpMCPServer))

	httpServer := &http.Server{
		Addr:         c.listenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	errChan := make(chan error, 1)

	go func() {
		var err error

		switch {
		case c.tlsKey != "" && c.tlsCert != "":
			log.Printf("Listening on %s (with TLS)", c.listenAddr)

			err = httpServer.ListenAndServeTLS(c.tlsCert, c.tlsKey)

		default:
			log.Printf("Listening on %s (without TLS)", c.listenAddr)

			err = httpServer.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal or error
	select {
	case <-stop:
		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}

		return nil

	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)

	case <-ctx.Done():
		log.Println("Shutting down server...")

		if err := httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}

		return nil
	}
}

func httpContextFunc(serverConfig glclient.ServerConfig) server.HTTPContextFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		clientConfig, err := glclient.ConfigFromRequest(r, serverConfig)
		if err != nil {
			return glctx.WithClientError(ctx, fmt.Errorf("ConfigFromRequest: %w", err))
		}

		client, err := glclient.New(clientConfig)
		if err != nil {
			return glctx.WithClientError(ctx, fmt.Errorf("creating GitLab client: %w", err))
		}

		return glctx.WithClient(ctx, client)
	}
}
