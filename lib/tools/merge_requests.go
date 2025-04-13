package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type MergeRequestsServiceInterface interface {
	AddTo(srv *server.MCPServer)
	ListUserMergeRequests() server.ServerTool
	ListProjectMergeRequests() server.ServerTool
	ListGroupMergeRequests() server.ServerTool
	GetMergeRequest() server.ServerTool
	GetMergeRequestApprovals() server.ServerTool
	GetMergeRequestCommits() server.ServerTool
	GetMergeRequestChanges() server.ServerTool
	ListMergeRequestDiffs() server.ServerTool
	GetMergeRequestParticipants() server.ServerTool
	GetMergeRequestReviewers() server.ServerTool
	ListMergeRequestPipelines() server.ServerTool
	GetIssuesClosedOnMerge() server.ServerTool
	GetMergeRequestDependencies() server.ServerTool
}

func NewMergeRequestsTools(client *gitlab.Client, currentUser string) MergeRequestsServiceInterface {
	return &mergeRequestsService{
		client:      client,
		currentUser: currentUser,
	}
}

type mergeRequestsService struct {
	client      *gitlab.Client
	currentUser string
}

func (m *mergeRequestsService) AddTo(srv *server.MCPServer) {
	srv.AddTools(
		m.ListUserMergeRequests(),
		m.ListProjectMergeRequests(),
		m.ListGroupMergeRequests(),
		m.GetMergeRequest(),
		m.GetMergeRequestApprovals(),
		m.GetMergeRequestCommits(),
		m.GetMergeRequestChanges(),
		m.ListMergeRequestDiffs(),
		m.GetMergeRequestParticipants(),
		m.GetMergeRequestReviewers(),
		m.ListMergeRequestPipelines(),
		m.GetIssuesClosedOnMerge(),
		m.GetMergeRequestDependencies(),
	)
}

func (m *mergeRequestsService) ListUserMergeRequests() server.ServerTool {
	return server.ServerTool{
		Handler: m.listUserMergeRequests,
		Tool: mcp.NewTool("list_user_merge_requests",
			mcp.WithDescription("Get all merge requests authors by a user"),
			mcp.WithString("author",
				mcp.Description("Filter merge requests by author. If left blank, returns all merge requests authored by the authenticated user")),
			mcp.WithString("state",
				mcp.Description("Return all merge requests or just those that are opened, closed, or merged"),
				mcp.Enum("all", "opened", "closed", "merged")),
		),
	}
}

func (m *mergeRequestsService) listUserMergeRequests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = gitlab.ListMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		mergeRequests []*gitlab.BasicMergeRequest
	)

	// Parse optional parameters
	author := m.currentUser
	if a, ok := request.Params.Arguments["author"].(string); ok && a != "" {
		author = a
	}
	opt.AuthorUsername = gitlab.Ptr(author)

	if v, ok := request.Params.Arguments["state"].(string); ok && v != "" {
		opt.State = gitlab.Ptr(v)
	}

	for {
		mrs, resp, err := m.client.MergeRequests.ListMergeRequests(&opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListMergeRequests: %w", err)
		}

		mergeRequests = append(mergeRequests, mrs...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(mergeRequests)
}

func (m *mergeRequestsService) ListProjectMergeRequests() server.ServerTool {
	return server.ServerTool{
		Handler: m.listProjectMergeRequests,
		Tool: mcp.NewTool("list_project_merge_requests",
			mcp.WithDescription("Get all merge requests for this project"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithString("state",
				mcp.Description("Return all merge requests or just those that are opened, closed, or merged"),
				mcp.Enum("all", "opened", "closed", "merged")),
		),
	}
}

func (m *mergeRequestsService) listProjectMergeRequests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = &gitlab.ListProjectMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		mergeRequests []*gitlab.BasicMergeRequest
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
		mrs, resp, err := m.client.MergeRequests.ListProjectMergeRequests(projectID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListProjectMergeRequests(%q): %w", projectID, err)
		}

		mergeRequests = append(mergeRequests, mrs...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(mergeRequests)
}

func (m *mergeRequestsService) ListGroupMergeRequests() server.ServerTool {
	return server.ServerTool{
		Handler: m.listGroupMergeRequests,
		Tool: mcp.NewTool("list_group_merge_requests",
			mcp.WithDescription("Get all merge requests for this group"),
			mcp.WithString("group_id",
				mcp.Description("ID of the group either in owner/namespace format or the numeric group ID"),
				mcp.Required()),
			mcp.WithString("state",
				mcp.Description("Return all merge requests or just those that are opened, closed, or merged"),
				mcp.Enum("all", "opened", "closed", "merged")),
		),
	}
}

func (m *mergeRequestsService) listGroupMergeRequests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = &gitlab.ListGroupMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		mergeRequests []*gitlab.BasicMergeRequest
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
		mrs, resp, err := m.client.MergeRequests.ListGroupMergeRequests(groupID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListGroupMergeRequests(%q): %w", groupID, err)
		}

		mergeRequests = append(mergeRequests, mrs...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(mergeRequests)
}

func (m *mergeRequestsService) GetMergeRequest() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequest,
		Tool: mcp.NewTool("get_merge_request",
			mcp.WithDescription("Get a single merge request"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("merge_request_iid",
				mcp.Description("The internal ID of the merge request"),
				mcp.Required()),
		),
	}
}

func (m *mergeRequestsService) getMergeRequest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var opt gitlab.GetMergeRequestsOptions

	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	mrIID, err := intParam(request.Params.Arguments["merge_request_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid merge request IID: %w", err)
	}

	mr, _, err := m.client.MergeRequests.GetMergeRequest(projectID, int(mrIID), &opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequest(%q, %d): %w", projectID, int(mrIID), err)
	}

	return newToolResultJSON(mr)
}

func (m *mergeRequestsService) GetMergeRequestApprovals() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestApprovals,
		Tool: mcp.NewTool("get_merge_request_approvals",
			mcp.WithDescription("Get approvals for a merge request"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("merge_request_iid",
				mcp.Description("The internal ID of the merge request"),
				mcp.Required()),
		),
	}
}

func (m *mergeRequestsService) getMergeRequestApprovals(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	mrIID, err := intParam(request.Params.Arguments["merge_request_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid merge request IID: %w", err)
	}

	approvals, _, err := m.client.MergeRequests.GetMergeRequestApprovals(projectID, int(mrIID), gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestApprovals(%q, %d): %w", projectID, int(mrIID), err)
	}

	return newToolResultJSON(approvals)
}

func (m *mergeRequestsService) GetMergeRequestCommits() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestCommits,
		Tool: mcp.NewTool("get_merge_request_commits",
			mcp.WithDescription("Get all commits associated with a merge request"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("merge_request_iid",
				mcp.Description("The internal ID of the merge request"),
				mcp.Required()),
		),
	}
}

func (m *mergeRequestsService) getMergeRequestCommits(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = gitlab.GetMergeRequestCommitsOptions{
			PerPage: maxPerPage,
		}
		commits []*gitlab.Commit
	)

	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	mrIID, err := intParam(request.Params.Arguments["merge_request_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid merge request IID: %w", err)
	}

	for {
		c, resp, err := m.client.MergeRequests.GetMergeRequestCommits(projectID, int(mrIID), &opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("GetMergeRequestCommits(%q, %d): %w", projectID, int(mrIID), err)
		}

		commits = append(commits, c...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(commits)
}

func (m *mergeRequestsService) GetMergeRequestChanges() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestChanges,
		Tool: mcp.NewTool("get_merge_request_changes",
			mcp.WithDescription("Get information about the merge request including its files and changes"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("merge_request_iid",
				mcp.Description("The internal ID of the merge request"),
				mcp.Required()),
		),
	}
}

func (m *mergeRequestsService) getMergeRequestChanges(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var opt gitlab.GetMergeRequestChangesOptions

	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	mrIID, err := intParam(request.Params.Arguments["merge_request_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid merge request IID: %w", err)
	}

	mr, _, err := m.client.MergeRequests.GetMergeRequestChanges(projectID, int(mrIID), &opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestChanges(%q, %d): %w", projectID, int(mrIID), err)
	}

	return newToolResultJSON(mr)
}

func (m *mergeRequestsService) ListMergeRequestDiffs() server.ServerTool {
	return server.ServerTool{
		Handler: m.listMergeRequestDiffs,
		Tool: mcp.NewTool("list_merge_request_diffs",
			mcp.WithDescription("Get a list of merge request diff versions"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("merge_request_iid",
				mcp.Description("The internal ID of the merge request"),
				mcp.Required()),
		),
	}
}

func (m *mergeRequestsService) listMergeRequestDiffs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = gitlab.ListMergeRequestDiffsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		diffs []*gitlab.MergeRequestDiff
	)

	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	mrIID, err := intParam(request.Params.Arguments["merge_request_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid merge request IID: %w", err)
	}

	for {
		d, resp, err := m.client.MergeRequests.ListMergeRequestDiffs(projectID, int(mrIID), &opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListMergeRequestDiffs(%q, %d): %w", projectID, int(mrIID), err)
		}

		diffs = append(diffs, d...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(diffs)
}

func (m *mergeRequestsService) GetMergeRequestParticipants() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestParticipants,
		Tool: mcp.NewTool("get_merge_request_participants",
			mcp.WithDescription("Get a list of merge request participants"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("merge_request_iid",
				mcp.Description("The internal ID of the merge request"),
				mcp.Required()),
		),
	}
}

func (m *mergeRequestsService) getMergeRequestParticipants(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	mrIID, err := intParam(request.Params.Arguments["merge_request_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid merge request IID: %w", err)
	}

	participants, _, err := m.client.MergeRequests.GetMergeRequestParticipants(projectID, int(mrIID), gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestParticipants(%q, %d): %w", projectID, int(mrIID), err)
	}

	return newToolResultJSON(participants)
}

func (m *mergeRequestsService) GetMergeRequestReviewers() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestReviewers,
		Tool: mcp.NewTool("get_merge_request_reviewers",
			mcp.WithDescription("Get a list of merge request reviewers"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("merge_request_iid",
				mcp.Description("The internal ID of the merge request"),
				mcp.Required()),
		),
	}
}

func (m *mergeRequestsService) getMergeRequestReviewers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	mrIID, err := intParam(request.Params.Arguments["merge_request_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid merge request IID: %w", err)
	}

	reviewers, _, err := m.client.MergeRequests.GetMergeRequestReviewers(projectID, int(mrIID), gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestReviewers(%q, %d): %w", projectID, int(mrIID), err)
	}

	return newToolResultJSON(reviewers)
}

func (m *mergeRequestsService) ListMergeRequestPipelines() server.ServerTool {
	return server.ServerTool{
		Handler: m.listMergeRequestPipelines,
		Tool: mcp.NewTool("list_merge_request_pipelines",
			mcp.WithDescription("Get a list of merge request pipelines"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("merge_request_iid",
				mcp.Description("The internal ID of the merge request"),
				mcp.Required()),
		),
	}
}

func (m *mergeRequestsService) listMergeRequestPipelines(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	mrIID, err := intParam(request.Params.Arguments["merge_request_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid merge request IID: %w", err)
	}

	pipelines, _, err := m.client.MergeRequests.ListMergeRequestPipelines(projectID, int(mrIID), gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("ListMergeRequestPipelines(%q, %d): %w", projectID, int(mrIID), err)
	}

	return newToolResultJSON(pipelines)
}

func (m *mergeRequestsService) GetIssuesClosedOnMerge() server.ServerTool {
	return server.ServerTool{
		Handler: m.getIssuesClosedOnMerge,
		Tool: mcp.NewTool("get_issues_closed_on_merge",
			mcp.WithDescription("Get all the issues that would be closed by merging the provided merge request"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("merge_request_iid",
				mcp.Description("The internal ID of the merge request"),
				mcp.Required()),
		),
	}
}

func (m *mergeRequestsService) getIssuesClosedOnMerge(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = gitlab.GetIssuesClosedOnMergeOptions{
			PerPage: maxPerPage,
		}
		issues []*gitlab.Issue
	)

	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	mrIID, err := intParam(request.Params.Arguments["merge_request_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid merge request IID: %w", err)
	}

	for {
		is, resp, err := m.client.MergeRequests.GetIssuesClosedOnMerge(projectID, int(mrIID), &opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("GetIssuesClosedOnMerge(%q, %d): %w", projectID, int(mrIID), err)
		}

		issues = append(issues, is...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(issues)
}

func (m *mergeRequestsService) GetMergeRequestDependencies() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestDependencies,
		Tool: mcp.NewTool("get_merge_request_dependencies",
			mcp.WithDescription("Get merge request dependencies"),
			mcp.WithString("project_id",
				mcp.Description("ID of the project either in owner/project format or the numeric project ID"),
				mcp.Required()),
			mcp.WithNumber("merge_request_iid",
				mcp.Description("The internal ID of the merge request"),
				mcp.Required()),
		),
	}
}

func (m *mergeRequestsService) getMergeRequestDependencies(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := idParam(request.Params.Arguments["project_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	mrIID, err := intParam(request.Params.Arguments["merge_request_iid"])
	if err != nil {
		return nil, fmt.Errorf("invalid merge request IID: %w", err)
	}

	dependencies, _, err := m.client.MergeRequests.GetMergeRequestDependencies(projectID, int(mrIID), gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestDependencies(%q, %d): %w", projectID, int(mrIID), err)
	}

	return newToolResultJSON(dependencies)
}
