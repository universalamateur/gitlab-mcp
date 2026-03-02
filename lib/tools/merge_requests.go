package tools

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/discussions"
	"gitlab.com/fforster/gitlab-mcp/lib/gliter"
	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

const defaultMergeRequestsLimit = 1000

// MergeRequestsServiceInterface defines the interface for merge request-related GitLab operations.
// It provides methods for retrieving and listing merge requests and their associated data such as
// approvals, commits, changes, participants, pipelines, and dependencies.
type MergeRequestsServiceInterface interface { //nolint:interfacebloat
	// AddTo registers all merge request-related tools with the provided MCPServer.
	AddTo(srv ToolAdder)

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

	// EditMergeRequest returns a tool for updating an existing merge request.
	EditMergeRequest() server.ServerTool

	// ListDraftNotes returns a tool for fetching draft MR notes.
	ListDraftNotes() server.ServerTool

	// CreateMergeRequest returns a tool for creating a new merge request.
	CreateMergeRequest() server.ServerTool
}

type MergeRequestsTools struct{}

// AddTo registers all merge request-related tools with the provided MCPServer.
// It adds tools for listing, retrieving, and managing merge requests and their associated data.
func (m *MergeRequestsTools) AddTo(srv ToolAdder) {
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
		m.EditMergeRequest(),
		m.ListDraftNotes(),
		m.CreateMergeRequest(),
	)
}

// detailedMergeRequest combines gitlab.MergeRequest with its discussions and diffs.
type detailedMergeRequest struct {
	MergeRequest *gitlab.MergeRequest       `json:"merge_request"`
	Discussions  []*gitlab.Discussion       `json:"discussions,omitempty"`
	Diffs        []*gitlab.MergeRequestDiff `json:"diffs,omitempty"`
}

type listUserMergeRequestsArgs struct {
	Username string `mcp_desc:"Filter merge requests by username. If left blank, returns merge requests for the authenticated user"`
	State    string `mcp_desc:"Return all merge requests or just those that are opened, closed, or merged" mcp_enum:"all,opened,closed,merged"`
	Role     string `mcp_desc:"Specify whether to list merge requests where the user is the author or reviewer" mcp_enum:"author,reviewer"`
	Limit    int    `mcp_desc:"The maximum number of merge requests to return. Defaults to 1000."`
}

// ListUserMergeRequests returns a ServerTool for listing all merge requests authored by or assigned to a specific user for review.
// The tool accepts optional username and state filter parameters, as well as a role parameter to determine
// whether to list merge requests where the user is the author or a reviewer.
func (m *MergeRequestsTools) ListUserMergeRequests() server.ServerTool {
	return server.ServerTool{
		Handler: ClientUserHandler(m.listUserMergeRequests),
		Tool: mcpargs.NewTool("list_user_merge_requests", listUserMergeRequestsArgs{},
			mcp.WithDescription("Get all merge requests authored by or assigned to a user for review"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) listUserMergeRequests(ctx context.Context, client *gitlab.Client, args listUserMergeRequestsArgs, currentUser *gitlab.User, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opts = gitlab.ListMergeRequestsOptions{
			Scope: new("all"),
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		mergeRequests []*gitlab.BasicMergeRequest
	)

	// Parse optional parameters
	username := currentUser.Username
	if args.Username != "" {
		username = args.Username
	}

	switch args.Role {
	case "author", "":
		opts.AuthorUsername = new(username)
	case "reviewer":
		opts.ReviewerUsername = new(username)
	default:
		return nil, fmt.Errorf("%w: invalid role: %q, must be one of 'author' or 'reviewer'", ErrInvalidArgument, args.Role)
	}

	if args.State != "" {
		opts.State = new(args.State)
	}

	if args.Limit <= 0 {
		args.Limit = defaultMergeRequestsLimit
	}

	nextPage := func(opts *gitlab.ListMergeRequestsOptions, page int64) {
		opts.ListOptions.Page = page
	}

	iter := gliter.All(ctx, client.MergeRequests.ListMergeRequests, opts, nextPage)
	iter = gliter.Limited(iter, args.Limit)

	for mr, err := range iter {
		if err != nil {
			return nil, fmt.Errorf("ListMergeRequests(): %w", err)
		}

		mergeRequests = append(mergeRequests, mr)
	}

	return newToolResultJSON(mergeRequests)
}

type listProjectMergeRequestsArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	State     string     `mcp_desc:"Return all merge requests or just those that are opened, closed, or merged" mcp_enum:"all,opened,closed,merged"`
	Limit     int        `mcp_desc:"The maximum number of merge requests to return. Defaults to 1000."`
}

// ListProjectMergeRequests returns a ServerTool for listing all merge requests within a specific project.
// The tool accepts a project ID and optional state filter parameters.
func (m *MergeRequestsTools) ListProjectMergeRequests() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.listProjectMergeRequests),
		Tool: mcpargs.NewTool("list_project_merge_requests", listProjectMergeRequestsArgs{},
			mcp.WithDescription("Get all merge requests for this project"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) listProjectMergeRequests(ctx context.Context, client *gitlab.Client, args listProjectMergeRequestsArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opts = gitlab.ListProjectMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		mergeRequests []*gitlab.BasicMergeRequest
	)

	// Parse optional parameters
	if args.State != "" {
		opts.State = new(args.State)
	}

	if args.Limit <= 0 {
		args.Limit = defaultMergeRequestsLimit
	}

	nextPage := func(opts *gitlab.ListProjectMergeRequestsOptions, page int64) {
		opts.Page = page
	}

	iter := gliter.AllWithID(ctx, args.ProjectID.Value(), client.MergeRequests.ListProjectMergeRequests, opts, nextPage)
	iter = gliter.Limited(iter, args.Limit)

	for mr, err := range iter {
		if err != nil {
			return nil, fmt.Errorf("ListProjectMergeRequests(%v): %w", args.ProjectID.Value(), err)
		}

		mergeRequests = append(mergeRequests, mr)
	}

	return newToolResultJSON(mergeRequests)
}

type listGroupMergeRequestsArgs struct {
	GroupID mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	State   string     `mcp_desc:"Return all merge requests or just those that are opened, closed, or merged" mcp_enum:"all,opened,closed,merged"`
	Limit   int        `mcp_desc:"The maximum number of merge requests to return. Defaults to 1000."`
}

// ListGroupMergeRequests returns a ServerTool for listing all merge requests within a specific group.
// The tool accepts a group ID and optional state filter parameters.
func (m *MergeRequestsTools) ListGroupMergeRequests() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.listGroupMergeRequests),
		Tool: mcpargs.NewTool("list_group_merge_requests", listGroupMergeRequestsArgs{},
			mcp.WithDescription("Get all merge requests for this group"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) listGroupMergeRequests(ctx context.Context, client *gitlab.Client, args listGroupMergeRequestsArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opts = gitlab.ListGroupMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		mergeRequests []*gitlab.BasicMergeRequest
	)

	// Parse optional parameters
	if args.State != "" {
		opts.State = new(args.State)
	}

	if args.Limit <= 0 {
		args.Limit = defaultMergeRequestsLimit
	}

	nextPage := func(opts *gitlab.ListGroupMergeRequestsOptions, page int64) {
		opts.Page = page
	}

	iter := gliter.AllWithID(ctx, args.GroupID.Value(), client.MergeRequests.ListGroupMergeRequests, opts, nextPage)
	iter = gliter.Limited(iter, args.Limit)

	for mr, err := range iter {
		if err != nil {
			return nil, fmt.Errorf("ListGroupMergeRequests(%v): %w", args.GroupID.Value(), err)
		}

		mergeRequests = append(mergeRequests, mr)
	}

	return newToolResultJSON(mergeRequests)
}

type getMergeRequestArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequest returns a ServerTool for fetching a specific merge request by its ID, including discussions and diffs.
// If fetching any part fails, an error is returned. Data is fetched concurrently.
func (m *MergeRequestsTools) GetMergeRequest() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.getMergeRequest),
		Tool: mcpargs.NewTool("get_merge_request", getMergeRequestArgs{},
			mcp.WithDescription("Get a single merge request, including its discussions and code changes (diffs)."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) getMergeRequest(ctx context.Context, client *gitlab.Client, args getMergeRequestArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		ret   detailedMergeRequest
		wg    sync.WaitGroup
		errCh = make(chan error)
	)

	// For MR, discussions, diffs
	wg.Add(3)

	// Goroutine to fetch Merge Request details
	go func() {
		defer wg.Done()

		var (
			opts = gitlab.GetMergeRequestsOptions{}
			err  error
		)

		ret.MergeRequest, _, err = client.MergeRequests.GetMergeRequest(args.ProjectID.Value(), args.MergeRequestIID, &opts, gitlab.WithContext(ctx))
		if err != nil {
			errCh <- fmt.Errorf("getting merge request IID %d (project %v): %w", args.MergeRequestIID, args.ProjectID.Value(), err)
		}
	}()

	// Goroutine to fetch Discussions
	go func() {
		defer wg.Done()

		discMgr, err := discussions.NewMergeRequestDiscussion(client, args.ProjectID, args.MergeRequestIID)
		if err != nil {
			errCh <- fmt.Errorf("creating discussion manager for MR IID %d (project %v): %w", args.MergeRequestIID, args.ProjectID.Value(), err)
			return
		}

		// Assuming non-confidential discussions by default.
		// A 'confidential' argument could be added to 'getMergeRequestArgs' if control over this is needed.
		const confidential = false

		ret.Discussions, err = discMgr.List(ctx, confidential)
		if err != nil {
			errCh <- fmt.Errorf("listing discussions for MR IID %d (project %v): %w", args.MergeRequestIID, args.ProjectID.Value(), err)
		}
	}()

	// Goroutine to fetch Diffs
	go func() {
		defer wg.Done()

		var err error

		ret.Diffs, err = m.listAllMergeRequestDiffs(ctx, client, args.ProjectID.Value(), args.MergeRequestIID)
		if err != nil {
			errCh <- fmt.Errorf("listing diffs for MR IID %d (project %v): %w", args.MergeRequestIID, args.ProjectID.Value(), err)
		}
	}()

	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errs error
	for err := range errCh {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return nil, errs
	}

	return newToolResultJSON(ret)
}

type getMergeRequestApprovalsArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequestApprovals returns a ServerTool for fetching approval information for a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsTools) GetMergeRequestApprovals() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.getMergeRequestApprovals),
		Tool: mcpargs.NewTool("get_merge_request_approvals", getMergeRequestApprovalsArgs{},
			mcp.WithDescription("Get approvals for a merge request"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) getMergeRequestApprovals(ctx context.Context, client *gitlab.Client, args getMergeRequestApprovalsArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	approvals, _, err := client.MergeRequests.GetMergeRequestApprovals(args.ProjectID.Value(), args.MergeRequestIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestApprovals(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(approvals)
}

type getMergeRequestCommitsArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequestCommits returns a ServerTool for fetching all commits associated with a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsTools) GetMergeRequestCommits() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.getMergeRequestCommits),
		Tool: mcpargs.NewTool("get_merge_request_commits", getMergeRequestCommitsArgs{},
			mcp.WithDescription("Get all commits associated with a merge request"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) getMergeRequestCommits(ctx context.Context, client *gitlab.Client, args getMergeRequestCommitsArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = gitlab.GetMergeRequestCommitsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		commits []*gitlab.Commit
	)

	for {
		c, resp, err := client.MergeRequests.GetMergeRequestCommits(args.ProjectID.Value(), args.MergeRequestIID, &opt, gitlab.WithContext(ctx))
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
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// ListMergeRequestDiffs returns a ServerTool for listing all versions of diffs for a specific merge request.
func (m *MergeRequestsTools) ListMergeRequestDiffs() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.listMergeRequestDiffs),
		Tool: mcpargs.NewTool("list_merge_request_diffs", listMergeRequestDiffsArgs{},
			mcp.WithDescription("Returns the changes made to files by the merge request. Diffs are presented in the unified diff format."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

// listMergeRequestDiffs is the handler for the ListMergeRequestDiffs tool.
func (m *MergeRequestsTools) listMergeRequestDiffs(ctx context.Context, client *gitlab.Client, args listMergeRequestDiffsArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	diffs, err := m.listAllMergeRequestDiffs(ctx, client, args.ProjectID.Value(), args.MergeRequestIID)
	if err != nil {
		return nil, err
	}

	return newToolResultJSON(diffs)
}

// listAllMergeRequestDiffs fetches all diffs for a given merge request, handling pagination.
func (m *MergeRequestsTools) listAllMergeRequestDiffs(ctx context.Context, client *gitlab.Client, projectID any, mrIID int64) ([]*gitlab.MergeRequestDiff, error) {
	var allDiffs []*gitlab.MergeRequestDiff

	opts := gitlab.ListMergeRequestDiffsOptions{
		ListOptions: gitlab.ListOptions{PerPage: maxPerPage},
		Unidiff:     new(true),
	}

	for {
		diffs, resp, err := client.MergeRequests.ListMergeRequestDiffs(projectID, mrIID, &opts, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("listing merge request diffs for MR IID %d (project %v): %w", mrIID, projectID, err)
		}

		allDiffs = append(allDiffs, diffs...)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return allDiffs, nil
}

type getMergeRequestParticipantsArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequestParticipants returns a ServerTool for fetching all participants of a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsTools) GetMergeRequestParticipants() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.getMergeRequestParticipants),
		Tool: mcpargs.NewTool("get_merge_request_participants", getMergeRequestParticipantsArgs{},
			mcp.WithDescription("Get a list of merge request participants"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) getMergeRequestParticipants(ctx context.Context, client *gitlab.Client, args getMergeRequestParticipantsArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	participants, _, err := client.MergeRequests.GetMergeRequestParticipants(args.ProjectID.Value(), args.MergeRequestIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestParticipants(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(participants)
}

type getMergeRequestReviewersArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequestReviewers returns a ServerTool for fetching all reviewers of a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsTools) GetMergeRequestReviewers() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.getMergeRequestReviewers),
		Tool: mcpargs.NewTool("get_merge_request_reviewers", getMergeRequestReviewersArgs{},
			mcp.WithDescription("Get a list of merge request reviewers"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) getMergeRequestReviewers(ctx context.Context, client *gitlab.Client, args getMergeRequestReviewersArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	reviewers, _, err := client.MergeRequests.GetMergeRequestReviewers(args.ProjectID.Value(), args.MergeRequestIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestReviewers(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(reviewers)
}

type listMergeRequestPipelinesArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// ListMergeRequestPipelines returns a ServerTool for listing all CI/CD pipelines for a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsTools) ListMergeRequestPipelines() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.listMergeRequestPipelines),
		Tool: mcpargs.NewTool("list_merge_request_pipelines", listMergeRequestPipelinesArgs{},
			mcp.WithDescription("Get a list of merge request pipelines"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) listMergeRequestPipelines(ctx context.Context, client *gitlab.Client, args listMergeRequestPipelinesArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pipelines, _, err := client.MergeRequests.ListMergeRequestPipelines(args.ProjectID.Value(), args.MergeRequestIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("ListMergeRequestPipelines(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(pipelines)
}

type getIssuesClosedOnMergeArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetIssuesClosedOnMerge returns a ServerTool for fetching all issues that would be closed by merging a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsTools) GetIssuesClosedOnMerge() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.getIssuesClosedOnMerge),
		Tool: mcpargs.NewTool("get_issues_closed_on_merge", getIssuesClosedOnMergeArgs{},
			mcp.WithDescription("Get all the issues that would be closed by merging the provided merge request"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) getIssuesClosedOnMerge(ctx context.Context, client *gitlab.Client, args getIssuesClosedOnMergeArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = gitlab.GetIssuesClosedOnMergeOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		issues []*gitlab.Issue
	)

	for {
		is, resp, err := client.MergeRequests.GetIssuesClosedOnMerge(args.ProjectID.Value(), args.MergeRequestIID, &opt, gitlab.WithContext(ctx))
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
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

// GetMergeRequestDependencies returns a ServerTool for fetching all dependencies of a specific merge request.
// The tool accepts a project ID and a merge request internal ID as parameters.
func (m *MergeRequestsTools) GetMergeRequestDependencies() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.getMergeRequestDependencies),
		Tool: mcpargs.NewTool("get_merge_request_dependencies", getMergeRequestDependenciesArgs{},
			mcp.WithDescription("Get merge request dependencies"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) getMergeRequestDependencies(ctx context.Context, client *gitlab.Client, args getMergeRequestDependenciesArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dependencies, _, err := client.MergeRequests.GetMergeRequestDependencies(args.ProjectID.Value(), args.MergeRequestIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetMergeRequestDependencies(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(dependencies)
}

type editMergeRequestArgs struct {
	ProjectID          mcpargs.ID           `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID    int64                `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
	Title              string               `mcp_desc:"Changes the title of the merge request"`
	Description        string               `mcp_desc:"Changes the description of the merge request. The description uses GitLab Flavored Markdown"`
	TargetBranch       string               `mcp_desc:"Changes the target branch of the merge request"`
	AssigneeIDs        string               `mcp_desc:"Comma-separated list of user IDs to assign the merge request to. Pass '-' to clear all assignees. Omit the parameter to leave assignees unchanged"`
	ReviewerIDs        string               `mcp_desc:"Comma-separated list of user IDs to set as reviewers for the merge request. Pass '-' to clear all reviewers. Omit the parameter to leave reviewers unchanged"`
	MilestoneID        int64                `mcp_desc:"The ID of a milestone to assign the merge request to"`
	AddLabels          string               `mcp_desc:"Comma-separated label names to add to the merge request"`
	RemoveLabels       string               `mcp_desc:"Comma-separated label names to remove from the merge request"`
	StateEvent         string               `mcp_desc:"The state of the merge request. Use 'close' to close the merge request or 'reopen' to reopen a closed merge request. Omit to keep the merge request state unchanged" mcp_enum:"close,reopen"`
	RemoveSourceBranch mcpargs.OptionalBool `mcp_desc:"Flag indicating if the merge request should remove the source branch when merging"`
	Squash             mcpargs.OptionalBool `mcp_desc:"If true, squash all commits into a single commit on merge"`
	DiscussionLocked   mcpargs.OptionalBool `mcp_desc:"Flag indicating if the merge request's discussion is locked. If true, only project members can add or edit comments"`
	AllowCollaboration mcpargs.OptionalBool `mcp_desc:"Allow commits from members who can merge to the target branch"`
}

// EditMergeRequest returns a ServerTool for updating an existing merge request.
// The tool can modify various aspects of the merge request including title, description, assignees,
// reviewers, labels, milestone, target branch, and state.
func (m *MergeRequestsTools) EditMergeRequest() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.editMergeRequest),
		Tool: mcpargs.NewTool("edit_merge_request", editMergeRequestArgs{},
			mcp.WithDescription("Updates an existing GitLab merge request. You can modify the merge request's title and description, add or remove labels, assign or unassign users, change reviewers, change the milestone, close or reopen the merge request, and control discussion settings."),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (m *MergeRequestsTools) editMergeRequest(ctx context.Context, client *gitlab.Client, args editMergeRequestArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := gitlab.UpdateMergeRequestOptions{}

	// Set basic fields if provided
	if args.Title != "" {
		opts.Title = new(args.Title)
	}

	if args.Description != "" {
		opts.Description = new(args.Description)
	}

	if args.TargetBranch != "" {
		opts.TargetBranch = new(args.TargetBranch)
	}

	if args.MilestoneID != 0 {
		opts.MilestoneID = new(args.MilestoneID)
	}

	// Sets of user IDs
	opts.AssigneeIDs = parseUserIDs(args.AssigneeIDs)
	opts.ReviewerIDs = parseUserIDs(args.ReviewerIDs)

	// Sets of labels
	opts.AddLabels = newLabelOptions(args.AddLabels)
	opts.RemoveLabels = newLabelOptions(args.RemoveLabels)

	switch args.StateEvent {
	case "close", "reopen":
		opts.StateEvent = new(args.StateEvent)
	case "":
		// no-op
	default:
		return nil, fmt.Errorf("%w: invalid state_event: %q, must be one of 'close' or 'reopen'", ErrInvalidArgument, args.StateEvent)
	}

	// Handle boolean flags using the parseBoolString utility
	opts.RemoveSourceBranch = args.RemoveSourceBranch.Ptr()
	opts.Squash = args.Squash.Ptr()

	opts.DiscussionLocked = args.DiscussionLocked.Ptr()
	opts.AllowCollaboration = args.AllowCollaboration.Ptr()

	mergeRequest, _, err := client.MergeRequests.UpdateMergeRequest(args.ProjectID.Value(), args.MergeRequestIID, &opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("UpdateMergeRequest(%q, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
	}

	return newToolResultJSON(mergeRequest)
}

type listDraftNotesArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
}

type createMergeRequestArgs struct {
	ProjectID          mcpargs.ID           `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	SourceBranch       string               `mcp_desc:"Source branch name" mcp_required:"true"`
	Title              string               `mcp_desc:"Merge request title" mcp_required:"true"`
	Description        string               `mcp_desc:"Merge request description"`
	TargetBranch       string               `mcp_desc:"Target branch name (defaults to project's default branch)"`
	Labels             string               `mcp_desc:"Comma-separated list of labels"`
	AssigneeIDs        string               `mcp_desc:"Comma-separated list of user IDs to assign the merge request to"`
	ReviewerIDs        string               `mcp_desc:"Comma-separated list of user IDs to set as reviewers"`
	MilestoneID        int64                `mcp_desc:"Global milestone ID to assign the merge request to"`
	RemoveSourceBranch mcpargs.OptionalBool `mcp_desc:"Whether to remove source branch after merge (defaults to project setting)"`
	Squash             mcpargs.OptionalBool `mcp_desc:"Whether to squash commits when merging (defaults to project setting)"`
}

// ListDraftNotes returns a ServerTool for listing all draft notes for a specific merge request.
func (m *MergeRequestsTools) ListDraftNotes() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.listDraftNotes),
		Tool: mcpargs.NewTool("list_draft_notes", listDraftNotesArgs{},
			mcp.WithDescription("Returns a list of draft notes for the merge request. Draft notes are pending merge request review comments that have not yet been published."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

// listDraftNotes returns all draft notes for the merge request.
func (m *MergeRequestsTools) listDraftNotes(ctx context.Context, client *gitlab.Client, args listDraftNotesArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = &gitlab.ListDraftNotesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		allNotes []*gitlab.DraftNote
	)

	for {
		notes, resp, err := client.DraftNotes.ListDraftNotes(args.ProjectID.Value(), args.MergeRequestIID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListDraftNotes(%v, %d): %w", args.ProjectID.Value(), args.MergeRequestIID, err)
		}

		allNotes = append(allNotes, notes...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(allNotes)
}

// CreateMergeRequest returns a ServerTool for creating a new merge request.
// The tool accepts project ID, source branch, title, and various optional parameters.
func (m *MergeRequestsTools) CreateMergeRequest() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(m.createMergeRequest),
		Tool: mcpargs.NewTool("create_merge_request", createMergeRequestArgs{},
			mcp.WithDescription("Creates a new GitLab merge request (MR) using an existing source branch. You can use the `create_commit` tool to create a new source branch containing file changes."),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (m *MergeRequestsTools) createMergeRequest(ctx context.Context, client *gitlab.Client, args createMergeRequestArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := &gitlab.CreateMergeRequestOptions{
		Title:        new(args.Title),
		SourceBranch: new(args.SourceBranch),
	}

	if args.Description != "" {
		opts.Description = new(args.Description)
	}

	if args.TargetBranch != "" {
		opts.TargetBranch = new(args.TargetBranch)
	}

	if args.MilestoneID != 0 {
		opts.MilestoneID = new(args.MilestoneID)
	}

	opts.AssigneeIDs = parseUserIDs(args.AssigneeIDs)
	opts.ReviewerIDs = parseUserIDs(args.ReviewerIDs)
	opts.Labels = newLabelOptions(args.Labels)

	opts.RemoveSourceBranch = args.RemoveSourceBranch.Ptr()
	opts.Squash = args.Squash.Ptr()

	mergeRequest, _, err := client.MergeRequests.CreateMergeRequest(args.ProjectID.Value(), opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CreateMergeRequest(%q): %w", args.ProjectID.Value(), err)
	}

	return newToolResultJSON(mergeRequest)
}
