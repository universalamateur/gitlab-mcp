package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/fforster/gitlab-mcp/cmd"
	"gitlab.com/fforster/gitlab-mcp/internal/config"
	"gitlab.com/fforster/gitlab-mcp/internal/gitlabclient"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	client, err := gitlabclient.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create GitLab client: %v", err)
	}

	if err := cmd.New(client).ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}
