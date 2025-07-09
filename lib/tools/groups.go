package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type GroupsTools struct{}

// AddTo registers all group-related tools with the provided MCPServer.
func (g *GroupsTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		g.GetGroup(),
	)
}

type getGroupArgs struct {
	GroupID mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
}

func (g *GroupsTools) GetGroup() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(g.getGroup),
		Tool: mcpargs.NewTool("get_group", getGroupArgs{},
			mcp.WithDescription("Get all details of a GitLab group."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (g *GroupsTools) getGroup(ctx context.Context, client *gitlab.Client, args getGroupArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := gitlab.GetGroupOptions{}

	group, _, err := client.Groups.GetGroup(args.GroupID.Value(), &opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetGroup(%q): %w", args.GroupID.Value(), err)
	}

	group.Projects = nil       //nolint:SA1019
	group.SharedProjects = nil //nolint:SA1019

	return newToolResultJSON(group)
}
