package tools

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// IssuesServiceInterface defines the interface for issue-related GitLab operations.
// It provides methods for retrieving and listing issues and related merge requests.
type IssuesServiceInterface interface {
	// AddTo registers all issue-related tools with the provided MCPServer.
	AddTo(srv ToolAdder)

	// ListUserIssues returns a tool for listing all issues assigned to a user.
	ListUserIssues() server.ServerTool

	// ListGroupIssues returns a tool for listing all issues within a specific group.
	ListGroupIssues() server.ServerTool

	// ListProjectIssues returns a tool for listing all issues within a specific project.
	ListProjectIssues() server.ServerTool

	// ListMergeRequestsRelatedToIssue returns a tool for listing all merge requests related to a specific issue.
	ListMergeRequestsRelatedToIssue() server.ServerTool

	// CreateIssue returns a tool for creating a new issue in a project.
	CreateIssue() server.ServerTool

	// EditIssue returns a tool for updating an existing issue.
	EditIssue() server.ServerTool

	// IssueTimeTracking returns a tool for retrieving time tracking information related to issues.
	IssueTimeTracking() server.ServerTool
}

type IssuesTools struct{}

const (
	defaultIssueState  = "opened"
	defaultIssuesLimit = 1000
)

// AddTo registers all issue-related tools with the provided MCPServer.
// It adds tools for listing, retrieving, and managing issues and their related merge requests.
func (i *IssuesTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		i.ListIssues(),
		i.ListGroupIssues(),
		i.ListProjectIssues(),
		i.ListMergeRequestsRelatedToIssue(),
		i.CreateIssue(),
		i.EditIssue(),
		i.IssueTimeTracking(),
	)
}

type listIssuesArgs struct {
	Author        mcpargs.ID `mcp_desc:"Filter issues by the author. The special value 'me' returns all issues created by the authenticated user"`
	Assignee      mcpargs.ID `mcp_desc:"Filter issues by the assignee. The special value 'me' returns all issues assigned to the authenticated user"`
	State         string     `mcp_desc:"Filter issues by state, with 'all' returning closed and opened issues. Defaults to 'opened'." mcp_enum:"all,opened,closed"`
	Confidential  bool       `mcp_desc:"If true, includes confidential issues. Default is false, which excludes confidential issues"`
	Milestone     string     `mcp_desc:"The milestone title to filter by"`
	IterationID   int64      `mcp_desc:"The iteration ID to filter by"`
	Labels        string     `mcp_desc:"Comma-separated list of label names to filter by"`
	Search        string     `mcp_desc:"Search issues against issue title and description"`
	CreatedAfter  string     `mcp_desc:"Return issues created on or after the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	CreatedBefore string     `mcp_desc:"Return issues created on or before the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	UpdatedAfter  string     `mcp_desc:"Return issues updated on or after the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	UpdatedBefore string     `mcp_desc:"Return issues updated on or before the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	DueDate       string     `mcp_desc:"Return issues that have no due date, are overdue, or whose due date is this week, this month, or between two weeks ago and next month" mcp_enum:"none,any,today,tomorrow,overdue,week,month,recent"`
	OrderBy       string     `mcp_desc:"Sort issues by the selected field. Default is 'created_at'" mcp_enum:"created_at,due_date,label_priority,milestone_due,popularity,priority,relative_position,title,updated_at,weight"`
	SortOrder     string     `mcp_desc:"Sort order to use. Default is 'desc'" mcp_enum:"asc,desc"`
	Limit         int        `mcp_desc:"The maximum number of issues to return. Defaults to 1000."`
}

// ListIssues returns a ServerTool for listing all issues a user has access to.
func (i *IssuesTools) ListIssues() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(i.listIssues),
		Tool: mcpargs.NewTool("list_issues", listIssuesArgs{},
			mcp.WithDescription("Lists GitLab issues with various filter options"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (i *IssuesTools) listIssues(ctx context.Context, client *gitlab.Client, args listIssuesArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt, err := listUserIssuesOptions(args)
	if err != nil {
		return nil, fmt.Errorf("invalid filter options: %w", err)
	}

	opt.ListOptions = newListOptions(request)

	if args.Limit <= 0 {
		args.Limit = defaultIssuesLimit
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
		return client.Issues.ListIssues(opt, gitlab.WithContext(ctx), p)
	}

	issues, err := gitlab.ScanAndCollectN(pager, args.Limit)
	if err != nil {
		return nil, fmt.Errorf("ListIssues(): %w", err)
	}

	return newToolResultJSON(issues)
}

// listUserIssuesOptions builds GitLab API options from the project issues arguments.
func listUserIssuesOptions(args listIssuesArgs) (*gitlab.ListIssuesOptions, error) {
	opt := &gitlab.ListIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPerPage,
		},
		Scope: new("all"),
	}

	state := args.State
	if state == "" {
		state = defaultIssueState
	}

	opt.State = new(state)

	// The handling of confidential issues is a bit unintuitive: if
	// args.Confidential is set to true, we DO NOT set opt.Confidential at all,
	// so the response includes both, confidential and public issues. Otherwise we
	// set the filter to false to only return public issues.
	if !args.Confidential {
		opt.Confidential = new(false)
	}

	opt.Labels = newLabelOptions(args.Labels)

	if args.Milestone != "" {
		opt.Milestone = new(args.Milestone)
	}

	if args.IterationID != 0 {
		opt.IterationID = new(args.IterationID)
	}

	switch {
	case args.Author.Integer != 0:
		opt.AuthorID = new(args.Author.Integer)
	case args.Author.String == "me":
		opt.Scope = new("created_by_me")
	case args.Author.String != "":
		opt.AuthorUsername = new(args.Author.String)
	}

	switch {
	case args.Assignee.Integer != 0:
		opt.AssigneeID = gitlab.AssigneeID(args.Assignee.Integer)
	case args.Assignee.String == "me":
		opt.Scope = new("assigned_to_me")
	case args.Assignee.String != "":
		opt.AssigneeUsername = new(args.Assignee.String)
	}

	if args.Search != "" {
		opt.Search = new(args.Search)
	}

	// Date filters
	var err error
	if opt.CreatedAfter, err = parseTime(args.CreatedAfter); err != nil {
		return nil, fmt.Errorf("invalid created_after date: %w", err)
	}

	if opt.CreatedBefore, err = parseTime(args.CreatedBefore); err != nil {
		return nil, fmt.Errorf("invalid created_before date: %w", err)
	}

	if opt.UpdatedAfter, err = parseTime(args.UpdatedAfter); err != nil {
		return nil, fmt.Errorf("invalid updated_after date: %w", err)
	}

	if opt.UpdatedBefore, err = parseTime(args.UpdatedBefore); err != nil {
		return nil, fmt.Errorf("invalid updated_before date: %w", err)
	}

	opt.DueDate = dueDateMap[args.DueDate]

	return opt, nil
}

type listGroupIssuesArgs struct {
	GroupID       mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	State         string     `mcp_desc:"Filter issues by state, with 'all' returning closed and opened issues. Defaults to 'opened'." mcp_enum:"all,opened,closed"`
	Confidential  bool       `mcp_desc:"If true, includes confidential issues. Default is false, which excludes confidential issues"`
	Labels        string     `mcp_desc:"Comma-separated list of label names to filter by"`
	Milestone     string     `mcp_desc:"The milestone title to filter by"`
	Author        mcpargs.ID `mcp_desc:"Filter by author ID or username"`
	Assignee      mcpargs.ID `mcp_desc:"Filter by assignee ID or username"`
	Search        string     `mcp_desc:"Search issues against their title and description"`
	CreatedAfter  string     `mcp_desc:"Return issues created on or after the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	CreatedBefore string     `mcp_desc:"Return issues created on or before the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	UpdatedAfter  string     `mcp_desc:"Return issues updated on or after the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	UpdatedBefore string     `mcp_desc:"Return issues updated on or before the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	DueDate       string     `mcp_desc:"Return issues that have no due date, are overdue, or whose due date is this week, this month, or between two weeks ago and next month" mcp_enum:"none,any,today,tomorrow,overdue,week,month,recent"`
	OrderBy       string     `mcp_desc:"Sort issues by the selected field. Default is 'created_at'" mcp_enum:"created_at,due_date,label_priority,milestone_due,popularity,priority,relative_position,title,updated_at,weight"`
	SortOrder     string     `mcp_desc:"Sort order to use. Default is 'desc'" mcp_enum:"asc,desc"`
	Limit         int        `mcp_desc:"The maximum number of issues to return. Defaults to 1000."`
}

// ListGroupIssues returns a ServerTool for listing all issues within a specific group.
// The tool accepts a group ID and optional filter parameters.
func (i *IssuesTools) ListGroupIssues() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(i.listGroupIssues),
		Tool: mcpargs.NewTool("list_group_issues", listGroupIssuesArgs{},
			mcp.WithDescription("Get a list of a group's issues"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (i *IssuesTools) listGroupIssues(ctx context.Context, client *gitlab.Client, args listGroupIssuesArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt, err := listGroupIssuesOptions(args)
	if err != nil {
		return nil, fmt.Errorf("invalid filter options: %w", err)
	}

	opt.ListOptions = newListOptions(request)

	if args.Limit <= 0 {
		args.Limit = defaultIssuesLimit
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
		return client.Issues.ListGroupIssues(args.GroupID.Value(), opt, gitlab.WithContext(ctx), p)
	}

	issues, err := gitlab.ScanAndCollectN(pager, args.Limit)
	if err != nil {
		return nil, fmt.Errorf("ListGroupIssues(%q): %w", args.GroupID.Value(), err)
	}

	return newToolResultJSON(issues)
}

type listProjectIssuesArgs struct {
	ProjectID     mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	State         string     `mcp_desc:"Filter issues by state, with 'all' returning closed and opened issues. Defaults to 'opened'." mcp_enum:"all,opened,closed"`
	Confidential  bool       `mcp_desc:"If true, includes confidential issues. Default is false, which excludes confidential issues"`
	Labels        string     `mcp_desc:"Comma-separated list of label names to filter by"`
	Milestone     string     `mcp_desc:"The milestone title to filter by"`
	IterationID   int64      `mcp_desc:"The iteration ID to filter by"`
	Author        mcpargs.ID `mcp_desc:"Filter by author ID or username"`
	Assignee      mcpargs.ID `mcp_desc:"Filter by assignee ID or username"`
	Search        string     `mcp_desc:"Search issues against their title and description"`
	CreatedAfter  string     `mcp_desc:"Return issues created on or after the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	CreatedBefore string     `mcp_desc:"Return issues created on or before the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	UpdatedAfter  string     `mcp_desc:"Return issues updated on or after the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	UpdatedBefore string     `mcp_desc:"Return issues updated on or before the given time (format: RFC3339 or '2006-01-02 15:04:05')"`
	DueDate       string     `mcp_desc:"Return issues that have no due date, are overdue, or whose due date is this week, this month, or between two weeks ago and next month" mcp_enum:"none,any,today,tomorrow,overdue,week,month,recent"`
	OrderBy       string     `mcp_desc:"Sort issues by the selected field. Default is 'created_at'" mcp_enum:"created_at,due_date,label_priority,milestone_due,popularity,priority,relative_position,title,updated_at,weight"`
	SortOrder     string     `mcp_desc:"Sort order to use. Default is 'desc'" mcp_enum:"asc,desc"`
	Limit         int        `mcp_desc:"The maximum number of issues to return. Defaults to 1000."`
}

// ListProjectIssues returns a ServerTool for listing all issues within a specific project.
// The tool accepts a project ID and optional filter parameters.
func (i *IssuesTools) ListProjectIssues() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(i.listProjectIssues),
		Tool: mcpargs.NewTool("list_project_issues", listProjectIssuesArgs{},
			mcp.WithDescription("Get a list of a project's issues"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (i *IssuesTools) listProjectIssues(ctx context.Context, client *gitlab.Client, args listProjectIssuesArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt, err := listProjectIssuesOptions(args)
	if err != nil {
		return nil, fmt.Errorf("invalid filter options: %w", err)
	}

	opt.ListOptions = newListOptions(request)

	if args.Limit <= 0 {
		args.Limit = defaultIssuesLimit
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
		return client.Issues.ListProjectIssues(args.ProjectID.Value(), opt, gitlab.WithContext(ctx), p)
	}

	issues, err := gitlab.ScanAndCollectN(pager, args.Limit)
	if err != nil {
		return nil, fmt.Errorf("ListProjectIssues(%q): %w", args.ProjectID.Value(), err)
	}

	return newToolResultJSON(issues)
}

// listProjectIssuesOptions builds GitLab API options from the project issues arguments.
func listProjectIssuesOptions(args listProjectIssuesArgs) (*gitlab.ListProjectIssuesOptions, error) {
	opt := &gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPerPage,
		},
	}

	state := args.State
	if state == "" {
		state = defaultIssueState
	}

	opt.State = new(state)

	// The handling of confidential issues is a bit unintuitive: if
	// args.Confidential is set to true, we DO NOT set opt.Confidential at all,
	// so the response includes both, confidential and public issues. Otherwise we
	// set the filter to false to only return public issues.
	if !args.Confidential {
		opt.Confidential = new(false)
	}

	opt.Labels = newLabelOptions(args.Labels)

	if args.Milestone != "" {
		opt.Milestone = new(args.Milestone)
	}

	if args.IterationID != 0 {
		opt.IterationID = new(args.IterationID)
	}

	switch {
	case args.Author.Integer != 0:
		opt.AuthorID = new(args.Author.Integer)
	case args.Author.String != "":
		opt.AuthorUsername = new(args.Author.String)
	}

	switch {
	case args.Assignee.Integer != 0:
		opt.AssigneeID = gitlab.AssigneeID(args.Assignee.Integer)
	case args.Assignee.String != "":
		opt.AssigneeUsername = new(args.Assignee.String)
	}

	if args.Search != "" {
		opt.Search = new(args.Search)
	}

	// Date filters
	var err error
	if opt.CreatedAfter, err = parseTime(args.CreatedAfter); err != nil {
		return nil, fmt.Errorf("invalid created_after date: %w", err)
	}

	if opt.CreatedBefore, err = parseTime(args.CreatedBefore); err != nil {
		return nil, fmt.Errorf("invalid created_before date: %w", err)
	}

	if opt.UpdatedAfter, err = parseTime(args.UpdatedAfter); err != nil {
		return nil, fmt.Errorf("invalid updated_after date: %w", err)
	}

	if opt.UpdatedBefore, err = parseTime(args.UpdatedBefore); err != nil {
		return nil, fmt.Errorf("invalid updated_before date: %w", err)
	}

	opt.DueDate = dueDateMap[args.DueDate]

	return opt, nil
}

// listGroupIssuesOptions builds GitLab API options from the group issues arguments.
func listGroupIssuesOptions(args listGroupIssuesArgs) (*gitlab.ListGroupIssuesOptions, error) {
	opt := &gitlab.ListGroupIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPerPage,
		},
	}

	state := args.State
	if state == "" {
		state = defaultIssueState
	}

	opt.State = new(state)

	// The handling of confidential issues is a bit unintuitive: if
	// args.Confidential is set to true, we DO NOT set opt.Confidential at all,
	// so the response includes both, confidential and public issues. Otherwise we
	// set the filter to false to only return public issues.
	if !args.Confidential {
		opt.Confidential = new(args.Confidential)
	}

	opt.Labels = newLabelOptions(args.Labels)

	if args.Milestone != "" {
		opt.Milestone = new(args.Milestone)
	}

	switch {
	case args.Author.Integer != 0:
		opt.AuthorID = new(args.Author.Integer)
	case args.Author.String != "":
		opt.AuthorUsername = new(args.Author.String)
	}

	switch {
	case args.Assignee.Integer != 0:
		opt.AssigneeID = gitlab.AssigneeID(args.Assignee.Integer)
	case args.Assignee.String != "":
		opt.AssigneeUsername = new(args.Assignee.String)
	}

	if args.Search != "" {
		opt.Search = new(args.Search)
	}

	// Date filters
	var err error
	if opt.CreatedAfter, err = parseTime(args.CreatedAfter); err != nil {
		return nil, fmt.Errorf("invalid created_after date: %w", err)
	}

	if opt.CreatedBefore, err = parseTime(args.CreatedBefore); err != nil {
		return nil, fmt.Errorf("invalid created_before date: %w", err)
	}

	if opt.UpdatedAfter, err = parseTime(args.UpdatedAfter); err != nil {
		return nil, fmt.Errorf("invalid updated_after date: %w", err)
	}

	if opt.UpdatedBefore, err = parseTime(args.UpdatedBefore); err != nil {
		return nil, fmt.Errorf("invalid updated_before date: %w", err)
	}

	opt.DueDate = dueDateMap[args.DueDate]

	return opt, nil
}

var dueDateMap = map[string]*string{
	"today":    new("today"),
	"tomorrow": new("tomorrow"),
	"overdue":  new("overdue"),
	"week":     new("week"),
	"month":    new("month"),
	"recent":   new("next_month_and_previous_two_weeks"),
	"any":      new("any"),
	"none":     new("0"),
}

var errParseTime = errors.New("parsing string as time failed")

// parseTime parses a time string and returns a time.Time pointer.
// The string can have one out of three time formats: RFC3339, DateTime, or DateOnly.
// If the time string is empty, it returns a nil pointer.
func parseTime(timeStr string) (*time.Time, error) {
	if timeStr == "" {
		return nil, nil
	}

	for _, format := range []string{time.RFC3339, time.DateTime, time.DateOnly} {
		t, err := time.ParseInLocation(format, timeStr, time.Local) //nolint:gosmopolitan
		if err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("timeStr = %q: %w", timeStr, errParseTime)
}

type listMergeRequestsRelatedToIssueArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	IssueIID  int64      `mcp_desc:"The internal ID of the project issue" mcp_required:"true"`
}

// ListMergeRequestsRelatedToIssue returns a ServerTool for listing all merge requests related to a specific issue.
// The tool accepts a project ID and an issue internal ID as parameters.
func (i *IssuesTools) ListMergeRequestsRelatedToIssue() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(i.listMergeRequestsRelatedToIssue),
		Tool: mcpargs.NewTool("list_merge_requests_related_to_issue", listMergeRequestsRelatedToIssueArgs{},
			mcp.WithDescription("Get all merge requests that are related to the specified issue"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (i *IssuesTools) listMergeRequestsRelatedToIssue(ctx context.Context, client *gitlab.Client, args listMergeRequestsRelatedToIssueArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := gitlab.ListMergeRequestsRelatedToIssueOptions{
		ListOptions: newListOptions(request),
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.BasicMergeRequest, *gitlab.Response, error) {
		return client.Issues.ListMergeRequestsRelatedToIssue(args.ProjectID.Value(), args.IssueIID, &opts, gitlab.WithContext(ctx), p)
	}

	mergeRequests, err := gitlab.ScanAndCollect(pager)
	if err != nil {
		return nil, fmt.Errorf("ListMergeRequestsRelatedToIssue(%q, %d): %w", args.ProjectID.Value(), args.IssueIID, err)
	}

	return newToolResultJSON(mergeRequests)
}

type createIssueArgs struct {
	ProjectID    mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	Title        string     `mcp_desc:"The title of the issue to create" mcp_required:"true"`
	Description  string     `mcp_desc:"The description of the issue in GitLab Flavored Markdown"`
	AssigneeIDs  string     `mcp_desc:"Comma-separated list of user IDs to assign the issue to"`
	MilestoneID  int64      `mcp_desc:"The ID of a milestone to assign the issue to"`
	EpicID       int64      `mcp_desc:"The global ID of an epic to assign the issue to"`
	Labels       string     `mcp_desc:"Comma-separated label names to assign to the new issue"`
	Confidential bool       `mcp_desc:"Set to true to make the issue confidential"`
}

// CreateIssue returns a ServerTool for creating a new issue.
// The tool accepts project ID, title, and other optional issue parameters.
func (i *IssuesTools) CreateIssue() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(i.createIssue),
		Tool: mcpargs.NewTool("create_issue", createIssueArgs{},
			mcp.WithDescription("Creates a new GitLab issue"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (i *IssuesTools) createIssue(ctx context.Context, client *gitlab.Client, args createIssueArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.CreateIssueOptions{
		Title: new(args.Title),
	}

	// Set optional parameters
	if args.Description != "" {
		opt.Description = new(args.Description)
	}

	if args.MilestoneID != 0 {
		opt.MilestoneID = new(args.MilestoneID)
	}

	if args.EpicID != 0 {
		opt.EpicID = new(args.EpicID)
	}

	opt.AssigneeIDs = parseUserIDs(args.AssigneeIDs)
	opt.Labels = newLabelOptions(args.Labels)

	if _, ok := request.GetArguments()["confidential"]; ok {
		opt.Confidential = new(args.Confidential)
	}

	issue, _, err := client.Issues.CreateIssue(args.ProjectID.Value(), opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CreateIssue(%q): %w", args.ProjectID.Value(), err)
	}

	return newToolResultJSON(issue)
}

type editIssueArgs struct {
	ProjectID        mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	IssueIID         int64      `mcp_desc:"The internal ID of the project issue" mcp_required:"true"`
	Title            string     `mcp_desc:"Changes the title of the issue"`
	Description      string     `mcp_desc:"Changes the description of the issue. The description uses GitLab Flavored Markdown"`
	AssigneeIDs      string     `mcp_desc:"Comma-separated list of user IDs to assign the issue to. Pass a single hyphen ('-') to clear all assignees. Omit the parameter to leave assignees unchanged"`
	MilestoneID      int64      `mcp_desc:"The ID of a milestone to assign the issue to. Set to 0 to remove the milestone"`
	EpicID           int64      `mcp_desc:"The global ID of an epic to assign the issue to"`
	AddLabels        string     `mcp_desc:"Comma-separated label names to add to the issue"`
	RemoveLabels     string     `mcp_desc:"Comma-separated label names to remove from the issue"`
	StateEvent       string     `mcp_desc:"The state of the issue. Use 'close' to close the issue or 'reopen' to reopen a closed issue. Omit to keep the issue state unchanged" mcp_enum:"close,reopen"`
	Confidential     bool       `mcp_desc:"If true, enables editing confidential issues or makes a public issue confidential. Default is false. Note: By design, specifying 'false' explicitly does not make confidential issues public - this is a security measure."`
	DiscussionLocked bool       `mcp_desc:"Flag indicating if the issue's discussion is locked. If true, only project members can add or edit comments"`
	DueDate          string     `mcp_desc:"Due date of the issue in the format 'YYYY-MM-DD'"`
	Weight           int64      `mcp_desc:"Sets the weight of the issue, which must be a non-negative integer value."`
}

// EditIssue returns a ServerTool for updating an existing issue.
// The tool accepts project ID, issue IID, and other parameters to update the issue.
func (i *IssuesTools) EditIssue() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(i.editIssue),
		Tool: mcpargs.NewTool("edit_issue", editIssueArgs{},
			mcp.WithDescription("Updates an existing GitLab issue. You can modify the issue's title and description, add or remove labels, assign or unassign users, change the milestone, close or reopen the issue, and control confidentiality and discussion settings."),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (i *IssuesTools) editIssue(ctx context.Context, client *gitlab.Client, args editIssueArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// First, fetch the issue to check if it's confidential
	issue, _, err := client.Issues.GetIssue(args.ProjectID.Value(), args.IssueIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetIssue(%q, %d): %w", args.ProjectID.Value(), args.IssueIID, err)
	}

	if issue.Confidential && !args.Confidential {
		return mcp.NewToolResultError(fmt.Sprintf("cannot edit issue %d as it is confidential. Ensure it is safe to be shared with the model, then set confidential=true to edit", args.IssueIID)), nil
	}

	opt := &gitlab.UpdateIssueOptions{}

	// Build update options based on provided arguments
	if args.Title != "" {
		opt.Title = new(args.Title)
	}

	if args.Description != "" {
		opt.Description = new(args.Description)
	}

	if args.StateEvent != "" {
		opt.StateEvent = new(args.StateEvent)
	}

	if args.MilestoneID != 0 {
		opt.MilestoneID = new(args.MilestoneID)
	}

	if args.EpicID != 0 {
		opt.EpicID = new(args.EpicID)
	}

	opt.AssigneeIDs = parseUserIDs(args.AssigneeIDs)
	opt.AddLabels = newLabelOptions(args.AddLabels)
	opt.RemoveLabels = newLabelOptions(args.RemoveLabels)

	if args.Confidential {
		opt.Confidential = new(args.Confidential)
	}

	// Only set these if explicitly provided
	if _, ok := request.GetArguments()["discussion_locked"]; ok {
		opt.DiscussionLocked = new(args.DiscussionLocked)
	}

	if args.DueDate != "" {
		date, err := gitlab.ParseISOTime(args.DueDate)
		if err != nil {
			return nil, fmt.Errorf("gitlab.ParseISOTime(%q): %w", args.DueDate, err)
		}

		opt.DueDate = new(date)
	}

	if _, ok := request.GetArguments()["weight"]; ok {
		opt.Weight = new(args.Weight)
	}

	issue, _, err = client.Issues.UpdateIssue(args.ProjectID.Value(), args.IssueIID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("UpdateIssue(%q, %d): %w", args.ProjectID.Value(), args.IssueIID, err)
	}

	return newToolResultJSON(issue)
}

type issueTimeTrackingArgs struct {
	ProjectID           mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	IssueIID            int64      `mcp_desc:"The internal ID of the project issue" mcp_required:"true"`
	TimeEstimate        int        `mcp_desc:"Sets the estimated time of work for this issue in seconds. Set to zero to clear the estimation."`
	AddSpentTimeSeconds int        `mcp_desc:"Adds time spent working on this issue in seconds"`
	AddSpentTimeSummary string     `mcp_desc:"The summary of the time spent working on this issue"`
	ResetSpentTime      bool       `mcp_desc:"Resets the time spent to 0 seconds"`
}

// IssueTimeTracking returns a ServerTool for updating time tracking details on an issue.
func (i *IssuesTools) IssueTimeTracking() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(i.issueTimeTracking),
		Tool: mcpargs.NewTool("issue_time_tracking", issueTimeTrackingArgs{},
			mcp.WithDescription("Updates time tracking information for an issue, such as the time estimate and spent time."),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (i *IssuesTools) issueTimeTracking(ctx context.Context, client *gitlab.Client, args issueTimeTrackingArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		lastResult *gitlab.TimeStats
		errs       error
	)

	if _, ok := request.GetArguments()["time_estimate"]; ok {
		switch args.TimeEstimate {
		case 0:
			timeStats, _, err := client.Issues.ResetTimeEstimate(args.ProjectID.Value(), args.IssueIID, gitlab.WithContext(ctx))
			switch err {
			case nil:
				lastResult = timeStats
			default:
				errs = errors.Join(errs, fmt.Errorf("ResetTimeEstimate(%v, %d): %w", args.ProjectID.Value(), args.IssueIID, err))
			}

		default:
			opts := gitlab.SetTimeEstimateOptions{
				Duration: new(fmt.Sprintf("%ds", args.TimeEstimate)),
			}

			timeStats, _, err := client.Issues.SetTimeEstimate(args.ProjectID.Value(), args.IssueIID, &opts, gitlab.WithContext(ctx))
			switch err {
			case nil:
				lastResult = timeStats
			default:
				errs = errors.Join(errs, fmt.Errorf("SetTimeEstimate(%v, %d): %w", args.ProjectID.Value(), args.IssueIID, err))
			}
		}
	}

	if args.ResetSpentTime {
		timeStats, _, err := client.Issues.ResetSpentTime(args.ProjectID.Value(), args.IssueIID, gitlab.WithContext(ctx))
		switch err {
		case nil:
			lastResult = timeStats
		default:
			errs = errors.Join(errs, fmt.Errorf("ResetSpentTime(%v, %d): %w", args.ProjectID.Value(), args.IssueIID, err))
		}
	}

	if args.AddSpentTimeSeconds > 0 {
		opts := gitlab.AddSpentTimeOptions{
			Duration: new(fmt.Sprintf("%ds", args.AddSpentTimeSeconds)),
			Summary:  new(args.AddSpentTimeSummary),
		}

		timeStats, _, err := client.Issues.AddSpentTime(args.ProjectID.Value(), args.IssueIID, &opts, gitlab.WithContext(ctx))
		switch err {
		case nil:
			lastResult = timeStats
		default:
			errs = errors.Join(errs, fmt.Errorf("AddSpentTime(%v, %d): %w", args.ProjectID.Value(), args.IssueIID, err))
		}
	}

	if errs != nil {
		return nil, errs
	}

	return newToolResultJSON(lastResult)
}

// parseUserIDs parses a comma-separated string of user IDs.
// Returns nil for "no change to assignees" and an empty slice for "clear assignees field".
func parseUserIDs(s string) *[]int64 {
	if s == "" {
		return nil
	}

	if s == "-" {
		clearAll := []int64{}
		return &clearAll
	}

	var userIDs []int64

	for idStr := range strings.SplitSeq(s, ",") {
		idStr = strings.TrimSpace(idStr)

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("Skipping invalid user ID: %q\n", idStr)
			continue
		}

		userIDs = append(userIDs, id)
	}

	if len(userIDs) == 0 {
		return nil
	}

	return &userIDs
}

// newLabelOptions converts a comma-separated string of label names into an array of LabelOptions.
// Returns nil if the labels string is empty.
func newLabelOptions(labels string) *gitlab.LabelOptions {
	if labels == "" {
		return nil
	}

	var labelOpts gitlab.LabelOptions

	for label := range strings.SplitSeq(labels, ",") {
		label = strings.TrimSpace(label)

		if label != "" {
			labelOpts = append(labelOpts, label)
		}
	}

	if len(labelOpts) == 0 {
		return nil
	}

	return &labelOpts
}
