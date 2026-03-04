package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// WorkItemsToolsInterface defines the interface for issue-related GitLab operations.
// It provides methods for retrieving and listing issues and related merge requests.
type WorkItemsToolsInterface interface {
	// AddTo registers all issue-related tools with the provided MCPServer.
	AddTo(srv ToolAdder)

	// GetWorkItem returns a tool for fetching a specific issue by its ID.
	GetWorkItem() server.ServerTool
}

// WorkItemsTools implements the WorkItemToolsInterface and provides tools for interacting with GitLab work items.
type WorkItemsTools struct{}

// AddTo registers all issue-related tools with the provided MCPServer.
func (i *WorkItemsTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		i.GetWorkItem(),
	)
}

type getWorkItemArgs struct {
	FullPath string `mcp_desc:"Full path of the GitLab namespace (group or project) containing the work item, e.g. 'fforster/gitlab-mcp'" mcp_required:"true"`
	IID      int64  `mcp_desc:"The internal ID of the work item" mcp_required:"true"`
}

// GetWorkItem returns a ServerTool for fetching a specific issue by its ID, including its discussions.
// If fetching discussions fails, an error is returned.
func (i *WorkItemsTools) GetWorkItem() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(i.getWorkItem),
		Tool: mcpargs.NewTool("get_work_item", getWorkItemArgs{},
			mcp.WithDescription("Retrieve a GitLab work item by its namespace (group or project) and internal ID. "+
				"Work items are GitLab's unified model for plan-stage objects such as issues, incidents, test cases, requirements, tasks, objectives, key results, epics, and tickets. "+
				"Use this tool whenever data about any of these work item types is required."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (i *WorkItemsTools) getWorkItem(ctx context.Context, client *gitlab.Client, args getWorkItemArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	wi, _, err := client.WorkItems.GetWorkItem(args.FullPath, args.IID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetWorkItem(%q, %d): %w", args.FullPath, args.IID, err)
	}

	return newToolResultJSON(wi)
}
