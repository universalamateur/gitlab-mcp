package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// IssuesServiceInterface defines the interface for issue-related GitLab operations.
// It provides methods for retrieving and listing issues and related merge requests.
type IssuesServiceInterface interface {
	// AddTo registers all issue-related tools with the provided MCPServer.
	AddTo(srv *server.MCPServer)

	// ListUserIssues returns a tool for listing all issues assigned to a user.
	ListUserIssues() server.ServerTool

	// ListGroupIssues returns a tool for listing all issues within a specific group.
	ListGroupIssues() server.ServerTool

	// ListProjectIssues returns a tool for listing all issues within a specific project.
	ListProjectIssues() server.ServerTool

	// GetIssue returns a tool for fetching a specific issue by its ID.
	GetIssue() server.ServerTool

	// ListMergeRequestsRelatedToIssue returns a tool for listing all merge requests related to a specific issue.
	ListMergeRequestsRelatedToIssue() server.ServerTool
}

// NewIssuesTools creates a new instance of IssuesServiceInterface with the provided GitLab client
// and current user. It returns an implementation that can be used to interact with GitLab's issue API.
func NewIssuesTools(client *gitlab.Client, currentUser string) *IssuesService {
	return &IssuesService{
		client:      client,
		currentUser: currentUser,
	}
}

type IssuesService struct {
	client      *gitlab.Client
	currentUser string
}

// AddTo registers all issue-related tools with the provided MCPServer.
// It adds tools for listing, retrieving, and managing issues and their related merge requests.
func (i *IssuesService) AddTo(srv *server.MCPServer) {
	srv.AddTools(
		i.ListUserIssues(),
		i.ListGroupIssues(),
		i.ListProjectIssues(),
		i.GetIssue(),
		i.ListMergeRequestsRelatedToIssue(),
	)
}

type listUserIssuesArgs struct {
	Assignee string `mcp_desc:"Filter issues by the assignee. If left blank, returns all issues assigned to the authenticated user"`
	State    string `mcp_desc:"Return all issues or just those that are opened or closed" mcp_enum:"all,opened,closed"`
}

// ListUserIssues returns a ServerTool for listing all issues assigned to a user.
// The tool accepts optional assignee and state filter parameters.
func (i *IssuesService) ListUserIssues() server.ServerTool {
	return server.ServerTool{
		Handler: i.listUserIssues,
		Tool: mcpargs.NewTool("list_user_issues", listUserIssuesArgs{},
			mcp.WithDescription("Lists all issues assigned to a user"),
		),
	}
}

func (i *IssuesService) listUserIssues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listUserIssuesArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var (
		opt = gitlab.ListIssuesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		issues []*gitlab.Issue
	)

	// Parse optional parameters
	assignee := i.currentUser
	if args.Assignee != "" {
		assignee = args.Assignee
	}

	opt.AssigneeUsername = gitlab.Ptr(assignee)

	if args.State != "" {
		opt.State = gitlab.Ptr(args.State)
	}

	for {
		is, resp, err := i.client.Issues.ListIssues(&opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListIssues: %w", err)
		}

		issues = append(issues, is...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(issues)
}

type listGroupIssuesArgs struct {
	GroupID mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	State   string     `mcp_desc:"Return all issues or just those that are opened or closed" mcp_enum:"all,opened,closed"`
}

// ListGroupIssues returns a ServerTool for listing all issues within a specific group.
// The tool accepts a group ID and optional state filter parameters.
func (i *IssuesService) ListGroupIssues() server.ServerTool {
	return server.ServerTool{
		Handler: i.listGroupIssues,
		Tool: mcpargs.NewTool("list_group_issues", listGroupIssuesArgs{},
			mcp.WithDescription("Get a list of a group's issues"),
		),
	}
}

func (i *IssuesService) listGroupIssues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listGroupIssuesArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var (
		opt = gitlab.ListGroupIssuesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		issues []*gitlab.Issue
	)

	// Parse optional parameters
	if args.State != "" {
		opt.State = gitlab.Ptr(args.State)
	}

	for {
		is, resp, err := i.client.Issues.ListGroupIssues(args.GroupID.Value(), &opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListGroupIssues(%q): %w", args.GroupID.Value(), err)
		}

		issues = append(issues, is...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(issues)
}

type listProjectIssuesArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	State     string     `mcp_desc:"Return all issues or just those that are opened or closed" mcp_enum:"all,opened,closed"`
}

// ListProjectIssues returns a ServerTool for listing all issues within a specific project.
// The tool accepts a project ID and optional state filter parameters.
func (i *IssuesService) ListProjectIssues() server.ServerTool {
	return server.ServerTool{
		Handler: i.listProjectIssues,
		Tool: mcpargs.NewTool("list_project_issues", listProjectIssuesArgs{},
			mcp.WithDescription("Get a list of a project's issues"),
		),
	}
}

func (i *IssuesService) listProjectIssues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listProjectIssuesArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var (
		opt = &gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		issues []*gitlab.Issue
	)

	// Parse optional parameters
	if args.State != "" {
		opt.State = gitlab.Ptr(args.State)
	}

	for {
		is, resp, err := i.client.Issues.ListProjectIssues(args.ProjectID.Value(), opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListProjectIssues(%q): %w", args.ProjectID.Value(), err)
		}

		issues = append(issues, is...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(issues)
}

type getIssueArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	IssueIID  int        `mcp_desc:"The internal ID of the project issue" mcp_required:"true"`
}

// GetIssue returns a ServerTool for fetching information about a specific issue.
// The tool accepts a project ID and an issue internal ID as parameters.
func (i *IssuesService) GetIssue() server.ServerTool {
	return server.ServerTool{
		Handler: i.getIssue,
		Tool: mcpargs.NewTool("get_issue", getIssueArgs{},
			mcp.WithDescription("Get a single project issue"),
		),
	}
}

func (i *IssuesService) getIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args getIssueArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	issue, _, err := i.client.Issues.GetIssue(args.ProjectID.Value(), args.IssueIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetIssue(%q, %d): %w", args.ProjectID.Value(), args.IssueIID, err)
	}

	return newToolResultJSON(issue)
}

type listMergeRequestsRelatedToIssueArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	IssueIID  int        `mcp_desc:"The internal ID of the project issue" mcp_required:"true"`
}

// ListMergeRequestsRelatedToIssue returns a ServerTool for listing all merge requests related to a specific issue.
// The tool accepts a project ID and an issue internal ID as parameters.
func (i *IssuesService) ListMergeRequestsRelatedToIssue() server.ServerTool {
	return server.ServerTool{
		Handler: i.listMergeRequestsRelatedToIssue,
		Tool: mcpargs.NewTool("list_merge_requests_related_to_issue", listMergeRequestsRelatedToIssueArgs{},
			mcp.WithDescription("Get all merge requests that are related to the specified issue"),
		),
	}
}

func (i *IssuesService) listMergeRequestsRelatedToIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listMergeRequestsRelatedToIssueArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var (
		opt = gitlab.ListMergeRequestsRelatedToIssueOptions{
			PerPage: maxPerPage,
		}
		mergeRequests []*gitlab.BasicMergeRequest
	)

	for {
		mrs, resp, err := i.client.Issues.ListMergeRequestsRelatedToIssue(args.ProjectID.Value(), args.IssueIID, &opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListMergeRequestsRelatedToIssue(%q, %d): %w", args.ProjectID.Value(), args.IssueIID, err)
		}

		mergeRequests = append(mergeRequests, mrs...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(mergeRequests)
}
