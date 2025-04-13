package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type EpicServiceInterface interface {
	AddTo(srv *server.MCPServer)
	ListGroupEpics() server.ServerTool
	GetEpic() server.ServerTool
	GetEpicLinks() server.ServerTool
	ListEpicIssues() server.ServerTool
}

func NewEpicTools(client *gitlab.Client) EpicServiceInterface {
	return &epicService{client: client}
}

type epicService struct {
	client *gitlab.Client
}

func (e *epicService) AddTo(srv *server.MCPServer) {
	srv.AddTools(
		e.ListGroupEpics(),
		e.GetEpic(),
		e.GetEpicLinks(),
		e.ListEpicIssues(),
	)
}

func (e *epicService) ListGroupEpics() server.ServerTool {
	return server.ServerTool{
		Handler: e.listGroupEpics,
		Tool: mcp.NewTool("list_group_epics",
			mcp.WithDescription("Get all epics for a specific group"),
			mcp.WithString("group_id",
				mcp.Description("ID of the group either in owner/namespace format or the numeric group ID"),
				mcp.Required()),
			mcp.WithString("state",
				mcp.Description("Return all epics or just those that are opened or closed"),
				mcp.Enum("all", "opened", "closed")),
		),
	}
}

func (e *epicService) listGroupEpics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = &gitlab.ListGroupEpicsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		epics []*gitlab.Epic
	)

	groupID, err := idParam(request.Params.Arguments["group_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Parse optional parameters
	if v, ok := request.Params.Arguments["state"].(string); ok && v != "" {
		opt.State = gitlab.Ptr(v)
	}

	for {
		e, resp, err := e.client.Epics.ListGroupEpics(groupID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListGroupEpics(%q): %w", groupID, err)
		}

		epics = append(epics, e...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(epics)
}

func (e *epicService) GetEpic() server.ServerTool {
	return server.ServerTool{
		Handler: e.getEpic,
		Tool: mcp.NewTool("get_epic",
			mcp.WithDescription("Fetches information about an epic by ID"),
			mcp.WithString("group_id",
				mcp.Description("ID of the group either in owner/namespace format or the numeric group ID"),
				mcp.Required()),
			mcp.WithNumber("epic_iid",
				mcp.Description("Internal ID of the epic to fetch"),
				mcp.Required()),
		),
	}
}

func (e *epicService) getEpic(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupID, err := idParam(request.Params.Arguments["group_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	epicIID, err := intParam(request.Params.Arguments["epic_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid issue IID: %w", err)
	}

	epic, _, err := e.client.Epics.GetEpic(groupID, int(epicIID), gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetEpic(%q, %d): %w", groupID, int(epicIID), err)
	}

	return newToolResultJSON(epic)
}

func (e *epicService) GetEpicLinks() server.ServerTool {
	return server.ServerTool{
		Handler: e.getEpicLinks,
		Tool: mcp.NewTool("get_epic_links",
			mcp.WithDescription("GetEpicLinks gets all child epics of an epic."),
			mcp.WithString("group_id",
				mcp.Description("ID of the group either in owner/namespace format or the numeric group ID"),
				mcp.Required()),
			mcp.WithNumber("epic_iid",
				mcp.Description("Internal ID of the epic to fetch child epics for"),
				mcp.Required()),
		),
	}
}

func (e *epicService) getEpicLinks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupID, err := idParam(request.Params.Arguments["group_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	epicIID, err := intParam(request.Params.Arguments["epic_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid issue IID: %w", err)
	}

	epics, _, err := e.client.Epics.GetEpicLinks(groupID, int(epicIID), gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetEpicLinks(%q, %d): %w", groupID, int(epicIID), err)
	}

	return newToolResultJSON(epics)
}

func (e *epicService) ListEpicIssues() server.ServerTool {
	return server.ServerTool{
		Handler: e.listEpicIssues,
		Tool: mcp.NewTool("list_epic_issues",
			mcp.WithDescription("Returns a list of issues assigned to the provided epic"),
			mcp.WithString("group_id",
				mcp.Description("ID of the group either in owner/namespace format or the numeric group ID"),
				mcp.Required()),
			mcp.WithNumber("epic_iid",
				mcp.Description("Internal ID of the epic to fetch issues for"),
				mcp.Required()),
		),
	}
}

func (e *epicService) listEpicIssues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opts = gitlab.ListOptions{
			PerPage: maxPerPage,
		}
		issues []*gitlab.Issue
	)

	groupID, err := idParam(request.Params.Arguments["group_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	epicIID, err := intParam(request.Params.Arguments["epic_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid issue IID: %w", err)
	}

	// Fetch issues from GitLab
	for {
		is, resp, err := e.client.EpicIssues.ListEpicIssues(groupID, int(epicIID), &opts, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListEpicIssues(%q, %d): %w", groupID, int(epicIID), err)
		}

		issues = append(issues, is...)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return newToolResultJSON(issues)
}
