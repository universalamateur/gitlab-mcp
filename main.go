package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/fforster/gitlab-mcp/cmd"
	"gitlab.com/fforster/gitlab-mcp/lib/glclient"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := glclient.NewConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	client, err := glclient.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create GitLab client: %v", err)
	}

	if err := cmd.New(client).ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}
