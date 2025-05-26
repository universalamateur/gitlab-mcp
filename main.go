package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/build"
	"gitlab.com/fforster/gitlab-mcp/lib/tools"
)

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"GitLab",
		build.Version(),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	client, err := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	user, _, err := client.Users.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}

	tools := tools.New(client, user.Username)

	tools.AddTo(s)

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
