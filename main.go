package main

import (
	"context"
	"log"
	"os"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/cmd"
	"gitlab.com/fforster/gitlab-mcp/lib/build"
)

func main() {
	ctx := context.Background()

	client, err := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"),
		gitlab.WithRequestOptions(
			gitlab.WithHeader("User-Agent", "gitlab-mcp/"+build.Version()),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.New(client).ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}
