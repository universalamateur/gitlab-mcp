package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// MergeRequestsServiceInterface defines the interface for merge request-related GitLab operations.
// It provides methods for retrieving and listing merge requests and their associated data such as
// approvals, commits, changes, participants, pipelines, and dependencies.
type MergeRequestsServiceInterface interface { //nolint:interfacebloat
	// AddTo registers all merge request-related tools with the provided MCPServer.
	AddTo(srv *server.MCPServer)

	// ListUserMergeRequests returns a tool for listing all merge requests authored by or assigned to a specific user for review.
	ListUserMergeRequests() server.ServerTool

	// ListProjectMergeRequests returns a tool for listing all merge requests within a specific project.
	ListProjectMergeRequests() server.ServerTool

	// ListGroupMergeRequests returns a tool for listing all merge requests within a specific group.
	ListGroupMergeRequests() server.ServerTool

	// GetMergeRequest returns a tool for fetching a specific merge request by its ID.
	GetMergeRequest() server.ServerTool

	// GetMergeRequestApprovals returns a tool for fetching approval information for a specific merge request.
	GetMergeRequestApprovals() server.ServerTool

	// GetMergeRequestCommits returns a tool for fetching all commits associated with a specific merge request.
	GetMergeRequestCommits() server.ServerTool

	// ListMergeRequestDiffs returns a tool for listing diffs of the files changed in a merge request.
	ListMergeRequestDiffs() server.ServerTool

	// GetMergeRequestParticipants returns a tool for fetching all participants of a specific merge request.
	GetMergeRequestParticipants() server.ServerTool

	// GetMergeRequestReviewers returns a tool for fetching all reviewers of a specific merge request.
	GetMergeRequestReviewers() server.ServerTool

	// ListMergeRequestPipelines returns a tool for listing all CI/CD pipelines for a specific merge request.
	ListMergeRequestPipelines() server.ServerTool

	// GetIssuesClosedOnMerge returns a tool for fetching all issues that would be closed by merging a specific merge request.
	GetIssuesClosedOnMerge() server.ServerTool

	// GetMergeRequestDependencies returns a tool for fetching all dependencies of a specific merge request.
	GetMergeRequestDependencies() server.ServerTool
}

// NewMergeRequestsTools creates a new instance of MergeRequestsServiceInterface with the provided GitLab client
// and current user. It returns an implementation that can be used to interact with GitLab's merge request API.
func NewMergeRequestsTools(client *gitlab.Client, currentUser string) *MergeRequestsService {
	return &MergeRequestsService{
		client:      client,
		currentUser: currentUser,
	}
}

type MergeRequestsService struct {
	client      *gitlab.Client
	currentUser string
}

// AddTo registers all merge request-related tools with the provided MCPServer.
// It adds tools for listing, retrieving, and managing merge requests and their associated data.
func (m *MergeRequestsService) AddTo(srv *server.MCPServer) {
	srv.AddTools(
		m.ListUserMergeRequests(),
		m.ListProjectMergeRequests(),
		m.ListGroupMergeRequests(),
		m.GetMergeRequest(),
		m.GetMergeRequestApprovals(),
		m.GetMergeRequestCommits(),
		m.ListMergeRequestDiffs(),
		m.GetMergeRequestParticipants(),
		m.GetMergeRequestReviewers(),
		m.ListMergeRequestPipelines(),
		m.GetIssuesClosedOnMerge(),
		m.GetMergeRequestDependencies(),
	)
}

type listUserMergeRequestsArgs struct {
	Username string `mcp_desc:"Filter merge requests by username. If left blank, returns merge requests for the authenticated user"`
	State    string `mcp_desc:"Return all merge requests or just those that are opened, closed, or merged" mcp_enum:"all,opened,closed,merged"`
	Role     string `mcp_desc:"Specify whether to list merge requests where the user is the author or reviewer" mcp_enum:"author,reviewer"`
}

// ListUserMergeRequests returns a ServerTool for listing all merge requests authored by or assigned to a specific user for review.
// The tool accepts optional username and state filter parameters, as well as a role parameter to determine
// whether to list merge requests where the user is the author or a reviewer.
func (m *MergeRequestsService) ListUserMergeRequests() server.ServerTool {
	return server.ServerTool{
		Handler: m.listUserMergeRequests,
		Tool: mcpargs.NewTool("list_user_merge_requests", listUserMergeRequestsArgs{},
			mcp.WithDescription("Get all merge requests authored by or assigned to a user for review"),
		),
	}
}

func (m *MergeRequestsService) listUserMergeRequests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listUserMergeRequestsArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var (
		opt = gitlab.ListMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		mergeRequests []*gitlab.BasicMergeRequest
	)

	// Parse optional parameters
	username := m.currentUser
	if args.Username != "" {
		username = args.Username
	}

	switch args.Role {
	case "author", "":
		opt.AuthorUsername = gitlab.Ptr(username)
	case "reviewer":
		opt.ReviewerUsername = gitlab.Ptr(username)
	default:
		return nil, fmt.Errorf("%w: invalid role: %q, must be one of 'author' or 'reviewer'", ErrArgumentType, args.Role)
	}

	if args.State != "" {
		opt.State = gitlab.Ptr(args.State)
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

type listProjectMergeRequestsArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	State     string     `mcp_desc:"Return all merge requests or just those that are opened, closed, or merged" mcp_enum:"all,opened,closed,merged"`
}

// ListProjectMergeRequests returns a ServerTool for listing all merge requests within a specific project.
// The tool accepts a project ID and optional state filter parameters.
func (m *MergeRequestsService) ListProjectMergeRequests() server.ServerTool {
	return server.ServerTool{
		Handler: m.listProjectMergeRequests,
		Tool: mcpargs.NewTool("list_project_merge_requests", listProjectMergeRequestsArgs{},
			mcp.WithDescription("Get all merge requests for this project"),
		),
	}
}

func (m *MergeRequestsService) listProjectMergeRequests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listProjectMergeRequestsArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var (
		opt = &gitlab.ListProjectMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		mergeRequests []*gitlab.BasicMergeRequest
	)

	// Parse optional parameters
	if args.State != "" {
		opt.State = gitlab.Ptr(args.State)
	}

	for {
		mrs, resp, err := m.client.MergeRequests.ListProjectMergeRequests(args.ProjectID.Value(), opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListProjectMergeRequests(%q): %w", args.ProjectID.Value(), err)
		}

		mergeRequests = append(mergeRequests, mrs...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(mergeRequests)
}

type listGroupMergeRequestsArgs struct {
	GroupID mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	State   string     `mcp_desc:"Return all merge requests or just those that are opened, closed, or merged" mcp_enum:"all,opened,closed,merged"`
}

// ListGroupMergeRequests returns a ServerTool for listing all merge requests within a specific group.
// The tool accepts a group ID and optional state filter parameters.
func (m *MergeRequestsService) ListGroupMergeRequests() server.ServerTool {
	return server.ServerTool{
		Handler: m.listGroupMergeRequests,
		Tool: mcpargs.NewTool("list_group_merge_requests", listGroupMergeRequestsArgs{},
			mcp.WithDescription("Get all merge requests for this group"),
		),
	}
}

func (m *MergeRequestsService) listGroupMergeRequests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listGroupMergeRequestsArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var (
		opt = &gitlab.ListGroupMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		mergeRequests []*gitlab.BasicMergeRequest
	)

	// Parse optional parameters
	if args.State != "" {
		opt.State = gitlab.Ptr(args.State)
	}

	for {
		mrs, resp, err := m.client.MergeRequests.ListGroupMergeRequests(args.GroupID.Value(), opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListGroupMergeRequests(%q): %w", args.GroupID.Value(), err)
		}

		mergeRequests = append(mergeRequests, mrs...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(mergeRequests)
}

type getMergeRequestArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int        `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequest returns a ServerTool for fetching information about a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsService) GetMergeRequest() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequest,
		Tool: mcpargs.NewTool("get_merge_request", getMergeRequestArgs{},
			mcp.WithDescription("Get a single merge request"),
		),
	}
}

func (m *MergeRequestsService) getMergeRequest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args getMergeRequestArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var opt gitlab.GetMergeRequestsOptions

	mr, _, err := m.client.MergeRequests.GetMergeRequest(args.ProjectID.Value(), args.MergeRequestIID, &opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequest(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(mr)
}

type getMergeRequestApprovalsArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int        `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequestApprovals returns a ServerTool for fetching approval information for a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsService) GetMergeRequestApprovals() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestApprovals,
		Tool: mcpargs.NewTool("get_merge_request_approvals", getMergeRequestApprovalsArgs{},
			mcp.WithDescription("Get approvals for a merge request"),
		),
	}
}

func (m *MergeRequestsService) getMergeRequestApprovals(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args getMergeRequestApprovalsArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	approvals, _, err := m.client.MergeRequests.GetMergeRequestApprovals(args.ProjectID.Value(), args.MergeRequestIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestApprovals(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(approvals)
}

type getMergeRequestCommitsArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int        `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequestCommits returns a ServerTool for fetching all commits associated with a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsService) GetMergeRequestCommits() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestCommits,
		Tool: mcpargs.NewTool("get_merge_request_commits", getMergeRequestCommitsArgs{},
			mcp.WithDescription("Get all commits associated with a merge request"),
		),
	}
}

func (m *MergeRequestsService) getMergeRequestCommits(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args getMergeRequestCommitsArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var (
		opt = gitlab.GetMergeRequestCommitsOptions{
			PerPage: maxPerPage,
		}
		commits []*gitlab.Commit
	)

	for {
		c, resp, err := m.client.MergeRequests.GetMergeRequestCommits(args.ProjectID.Value(), args.MergeRequestIID, &opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("GetMergeRequestCommits(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
		}

		commits = append(commits, c...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(commits)
}

type listMergeRequestDiffsArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int        `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// ListMergeRequestDiffs returns a ServerTool for listing all versions of diffs for a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsService) ListMergeRequestDiffs() server.ServerTool {
	return server.ServerTool{
		Handler: m.listMergeRequestDiffs,
		Tool: mcpargs.NewTool("list_merge_request_diffs", listMergeRequestDiffsArgs{},
			mcp.WithDescription("Returns the changes made to files by the merge request. Diffs are presented in the unified diff format."),
		),
	}
}

func (m *MergeRequestsService) listMergeRequestDiffs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listMergeRequestDiffsArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var (
		opt = gitlab.ListMergeRequestDiffsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
			Unidiff: gitlab.Ptr(true),
		}
		diffs []*gitlab.MergeRequestDiff
	)

	for {
		d, resp, err := m.client.MergeRequests.ListMergeRequestDiffs(args.ProjectID.Value(), args.MergeRequestIID, &opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListMergeRequestDiffs(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
		}

		diffs = append(diffs, d...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(diffs)
}

type getMergeRequestParticipantsArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int        `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequestParticipants returns a ServerTool for fetching all participants of a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsService) GetMergeRequestParticipants() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestParticipants,
		Tool: mcpargs.NewTool("get_merge_request_participants", getMergeRequestParticipantsArgs{},
			mcp.WithDescription("Get a list of merge request participants"),
		),
	}
}

func (m *MergeRequestsService) getMergeRequestParticipants(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args getMergeRequestParticipantsArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	participants, _, err := m.client.MergeRequests.GetMergeRequestParticipants(args.ProjectID.Value(), args.MergeRequestIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestParticipants(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(participants)
}

type getMergeRequestReviewersArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int        `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequestReviewers returns a ServerTool for fetching all reviewers of a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsService) GetMergeRequestReviewers() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestReviewers,
		Tool: mcpargs.NewTool("get_merge_request_reviewers", getMergeRequestReviewersArgs{},
			mcp.WithDescription("Get a list of merge request reviewers"),
		),
	}
}

func (m *MergeRequestsService) getMergeRequestReviewers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args getMergeRequestReviewersArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	reviewers, _, err := m.client.MergeRequests.GetMergeRequestReviewers(args.ProjectID.Value(), args.MergeRequestIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestReviewers(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(reviewers)
}

type listMergeRequestPipelinesArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int        `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// ListMergeRequestPipelines returns a ServerTool for listing all CI/CD pipelines for a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsService) ListMergeRequestPipelines() server.ServerTool {
	return server.ServerTool{
		Handler: m.listMergeRequestPipelines,
		Tool: mcpargs.NewTool("list_merge_request_pipelines", listMergeRequestPipelinesArgs{},
			mcp.WithDescription("Get a list of merge request pipelines"),
		),
	}
}

func (m *MergeRequestsService) listMergeRequestPipelines(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listMergeRequestPipelinesArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	pipelines, _, err := m.client.MergeRequests.ListMergeRequestPipelines(args.ProjectID.Value(), args.MergeRequestIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("ListMergeRequestPipelines(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(pipelines)
}

type getIssuesClosedOnMergeArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int        `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetIssuesClosedOnMerge returns a ServerTool for fetching all issues that would be closed by merging a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsService) GetIssuesClosedOnMerge() server.ServerTool {
	return server.ServerTool{
		Handler: m.getIssuesClosedOnMerge,
		Tool: mcpargs.NewTool("get_issues_closed_on_merge", getIssuesClosedOnMergeArgs{},
			mcp.WithDescription("Get all the issues that would be closed by merging the provided merge request"),
		),
	}
}

func (m *MergeRequestsService) getIssuesClosedOnMerge(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args getIssuesClosedOnMergeArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	var (
		opt = gitlab.GetIssuesClosedOnMergeOptions{
			PerPage: maxPerPage,
		}
		issues []*gitlab.Issue
	)

	for {
		is, resp, err := m.client.MergeRequests.GetIssuesClosedOnMerge(args.ProjectID.Value(), args.MergeRequestIID, &opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("GetIssuesClosedOnMerge(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
		}

		issues = append(issues, is...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(issues)
}

type getMergeRequestDependenciesArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int        `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequestDependencies returns a ServerTool for fetching all dependencies of a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsService) GetMergeRequestDependencies() server.ServerTool {
	return server.ServerTool{
		Handler: m.getMergeRequestDependencies,
		Tool: mcpargs.NewTool("get_merge_request_dependencies", getMergeRequestDependenciesArgs{},
			mcp.WithDescription("Get merge request dependencies"),
		),
	}
}

func (m *MergeRequestsService) getMergeRequestDependencies(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args getMergeRequestDependenciesArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	dependencies, _, err := m.client.MergeRequests.GetMergeRequestDependencies(args.ProjectID.Value(), args.MergeRequestIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestDependencies(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(dependencies)
}
