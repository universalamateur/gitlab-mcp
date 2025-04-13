package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"
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
func NewIssuesTools(client *gitlab.Client, currentUser string) IssuesServiceInterface {
	return &issuesService{
		client:      client,
		currentUser: currentUser,
	}
}

type issuesService struct {
	client      *gitlab.Client
	currentUser string
}

// AddTo registers all issue-related tools with the provided MCPServer.
// It adds tools for listing, retrieving, and managing issues and their related merge requests.
func (i *issuesService) AddTo(srv *server.MCPServer) {
	srv.AddTools(
		i.ListUserIssues(),
		i.ListGroupIssues(),
		i.ListProjectIssues(),
		i.GetIssue(),
		i.ListMergeRequestsRelatedToIssue(),
	)
}

// ListUserIssues returns a ServerTool for listing all issues assigned to a user.
// The tool accepts optional assignee and state filter parameters.
func (i *issuesService) ListUserIssues() server.ServerTool {
	return server.ServerTool{
		Handler: i.listUserIssues,
		Tool: mcp.NewTool("list_user_issues",
			mcp.WithDescription("Lists all issues assigned to a user"),
			mcp.WithBoolean("assignee",
				mcp.Description("Filter issues by the assignee. If left blank, returns all issues assigned to the authenticated user")),
			mcp.WithString("state",
				mcp.Description("Return all issues or just those that are opened or closed"),
				mcp.Enum("all", "opened", "closed")),
		),
	}
}

func (i *issuesService) listUserIssues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if a, ok := request.Params.Arguments["assignee"].(string); ok && a != "" {
		assignee = a
	}
	opt.AssigneeUsername = gitlab.Ptr(assignee)

	if v, ok := request.Params.Arguments["state"].(string); ok && v != "" {
		opt.State = gitlab.Ptr(v)
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

// ListGroupIssues returns a ServerTool for listing all issues within a specific group.
// The tool accepts a group ID and optional state filter parameters.
func (i *issuesService) ListGroupIssues() server.ServerTool {
	return server.ServerTool{
		Handler: i.listGroupIssues,
		Tool: mcp.NewTool("list_group_issues",
			mcp.WithDescription("Get a list of a group's issues"),
			mcp.WithString("group_id",
				mcp.Description("ID of the group either in owner/namespace format or the numeric group ID"),
				mcp.Required()),
			mcp.WithString("state",
				mcp.Description("Return all issues or just those that are opened or closed"),
				mcp.Enum("all", "opened", "closed")),
		),
	}
}

func (i *issuesService) listGroupIssues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = gitlab.ListGroupIssuesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		issues []*gitlab.Issue
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
		is, resp, err := i.client.Issues.ListGroupIssues(groupID, &opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListGroupIssues(%q): %w", groupID, err)
		}

		issues = append(issues, is...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(issues)
}

// ListProjectIssues returns a ServerTool for listing all issues within a specific project.
// The tool accepts a project ID and optional state filter parameters.
func (i *issuesService) ListProjectIssues() server.ServerTool {
	return server.ServerTool{
		Handler: i.listProjectIssues,
		Tool: mcp.NewTool("list_project_issues",
			mcp.WithDescription("Get a list of a project's issues"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithString("state",
				mcp.Description("Return all issues or just those that are opened or closed"),
				mcp.Enum("all", "opened", "closed")),
		),
	}
}

func (i *issuesService) listProjectIssues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = &gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		issues []*gitlab.Issue
	)

	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Parse optional parameters
	if v, ok := request.Params.Arguments["state"].(string); ok && v != "" {
		opt.State = gitlab.Ptr(v)
	}

	for {
		is, resp, err := i.client.Issues.ListProjectIssues(projectID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListProjectIssues(%q): %w", projectID, err)
		}

		issues = append(issues, is...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(issues)
}

// GetIssue returns a ServerTool for fetching information about a specific issue.
// The tool accepts a project ID and an issue internal ID as parameters.
func (i *issuesService) GetIssue() server.ServerTool {
	return server.ServerTool{
		Handler: i.getIssue,
		Tool: mcp.NewTool("get_issue",
			mcp.WithDescription("Get a single project issue"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("issue_iid",
				mcp.Description("The internal ID of the project issue"),
				mcp.Required()),
		),
	}
}

func (i *issuesService) getIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	issueIID, err := intParam(request.Params.Arguments["issue_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid issue IID: %w", err)
	}

	issue, _, err := i.client.Issues.GetIssue(projectID, int(issueIID), gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetIssue(%q, %d): %w", projectID, int(issueIID), err)
	}

	return newToolResultJSON(issue)
}

// ListMergeRequestsRelatedToIssue returns a ServerTool for listing all merge requests related to a specific issue.
// The tool accepts a project ID and an issue internal ID as parameters.
func (i *issuesService) ListMergeRequestsRelatedToIssue() server.ServerTool {
	return server.ServerTool{
		Handler: i.listMergeRequestsRelatedToIssue,
		Tool: mcp.NewTool("list_merge_requests_related_to_issue",
			mcp.WithDescription("Get all merge requests that are related to the specified issue"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("issue_iid",
				mcp.Description("The internal ID of the project issue"),
				mcp.Required()),
		),
	}
}

func (i *issuesService) listMergeRequestsRelatedToIssue(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = gitlab.ListMergeRequestsRelatedToIssueOptions{
			PerPage: maxPerPage,
		}
		mergeRequests []*gitlab.BasicMergeRequest
	)

	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	issueIID, err := intParam(request.Params.Arguments["issue_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid issue IID: %w", err)
	}

	for {
		mrs, resp, err := i.client.Issues.ListMergeRequestsRelatedToIssue(projectID, int(issueIID), &opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListMergeRequestsRelatedToIssue(%q, %d): %w", projectID, int(issueIID), err)
		}

		mergeRequests = append(mergeRequests, mrs...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(mergeRequests)
}
