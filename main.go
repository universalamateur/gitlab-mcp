package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/fforster/gitlab-mcp/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := cmd.New().ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}
