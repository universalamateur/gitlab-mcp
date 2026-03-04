package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// EpicServiceInterface defines the interface for epic-related GitLab operations.
// It provides methods for retrieving and listing epics and their associated issues.
type EpicServiceInterface interface {
	// AddTo registers all epic-related tools with the provided MCPServer.
	AddTo(srv ToolAdder)

	// ListGroupEpics returns a tool for listing all epics in a specific group.
	ListGroupEpics() server.ServerTool

	// GetEpicLinks returns a tool for fetching all child epics of a specific epic.
	GetEpicLinks() server.ServerTool

	// ListEpicIssues returns a tool for listing all issues assigned to a specific epic.
	ListEpicIssues() server.ServerTool

	// CreateEpic creates a new epic.
	CreateEpic() server.ServerTool

	// EditEpic edits an existing epic.
	EditEpic() server.ServerTool
}

type EpicTools struct{}

// AddTo registers all epic-related tools with the provided MCPServer.
// It adds tools for listing, retrieving, and managing epics and their associated issues.
func (e *EpicTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		e.ListGroupEpics(),
		e.GetEpicLinks(),
		e.ListEpicIssues(),
		e.CreateEpic(),
		e.EditEpic(),
	)
}

type listGroupEpicsArgs struct {
	GroupID mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	State   string     `mcp_desc:"Return all epics or just those that are opened or closed" mcp_enum:"all,opened,closed"`
}

// ListGroupEpics returns a ServerTool for listing all epics in a specific group.
// The tool accepts a group ID and optional state filter parameters.
func (e *EpicTools) ListGroupEpics() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(e.listGroupEpics),
		Tool: mcpargs.NewTool("list_group_epics", listGroupEpicsArgs{},
			mcp.WithDescription("Get all epics for a specific group"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (e *EpicTools) listGroupEpics(ctx context.Context, client *gitlab.Client, args listGroupEpicsArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.ListGroupEpicsOptions{
		ListOptions: newListOptions(request),
	}

	// Parse optional parameters
	if args.State != "" {
		opt.State = new(args.State)
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Epic, *gitlab.Response, error) {
		return client.Epics.ListGroupEpics(args.GroupID.Value(), opt, gitlab.WithContext(ctx), p)
	}

	epics, err := gitlab.ScanAndCollect(pager)
	if err != nil {
		return nil, fmt.Errorf("ListGroupEpics(%q): %w", args.GroupID.Value(), err)
	}

	return newToolResultJSON(epics)
}

type getEpicLinksArgs struct {
	GroupID mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	EpicIID int64      `mcp_desc:"Internal ID of the epic to fetch child epics for" mcp_required:"true"`
}

// GetEpicLinks returns a ServerTool for retrieving all child epics of a specific epic.
// The tool accepts a group ID and an epic internal ID as parameters.
func (e *EpicTools) GetEpicLinks() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(e.getEpicLinks),
		Tool: mcpargs.NewTool("get_epic_links", getEpicLinksArgs{},
			mcp.WithDescription("GetEpicLinks gets all child epics of an epic."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (e *EpicTools) getEpicLinks(ctx context.Context, client *gitlab.Client, args getEpicLinksArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	epics, _, err := client.Epics.GetEpicLinks(args.GroupID.Value(), args.EpicIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetEpicLinks(%q, %d): %w", args.GroupID.Value(), args.EpicIID, err)
	}

	return newToolResultJSON(epics)
}

type listEpicIssuesArgs struct {
	GroupID mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	EpicIID int64      `mcp_desc:"Internal ID of the epic to fetch issues for" mcp_required:"true"`
}

// ListEpicIssues returns a ServerTool for listing all issues assigned to a specific epic.
// The tool accepts a group ID and an epic internal ID as parameters.
func (e *EpicTools) ListEpicIssues() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(e.listEpicIssues),
		Tool: mcpargs.NewTool("list_epic_issues", listEpicIssuesArgs{},
			mcp.WithDescription("Returns a list of issues assigned to the provided epic"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (e *EpicTools) listEpicIssues(ctx context.Context, client *gitlab.Client, args listEpicIssuesArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := newListOptions(request)

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
		return client.EpicIssues.ListEpicIssues(args.GroupID.Value(), args.EpicIID, &opts, gitlab.WithContext(ctx), p)
	}

	issues, err := gitlab.ScanAndCollect(pager)
	if err != nil {
		return nil, fmt.Errorf("ListEpicIssues(%q, %d): %w", args.GroupID.Value(), args.EpicIID, err)
	}

	return newToolResultJSON(issues)
}

// createEpicArgs holds the arguments required to create a new epic.
type createEpicArgs struct {
	GroupID      mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	Title        string     `mcp_desc:"Title of the epic" mcp_required:"true"`
	Labels       string     `mcp_desc:"Comma-separated set of labels for the epic"`
	Description  string     `mcp_desc:"Description of the epic"`
	Confidential bool       `mcp_desc:"Whether the epic is confidential"`
	DueDate      string     `mcp_desc:"Due date of the epic (format: YYYY-MM-DD)"`
	ParentID     int64      `mcp_desc:"ID of the parent epic"`
}

// CreateEpic returns a ServerTool for creating a new epic.
func (e *EpicTools) CreateEpic() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(e.createEpic),
		Tool: mcpargs.NewTool("create_epic", createEpicArgs{},
			mcp.WithDescription("Creates a new GitLab epic in the specified group"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (e *EpicTools) createEpic(ctx context.Context, client *gitlab.Client, args createEpicArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := &gitlab.CreateEpicOptions{
		Title: new(args.Title),
	}

	opts.Labels = newLabelOptions(args.Labels)

	if args.Description != "" {
		opts.Description = new(args.Description)
	}

	if _, ok := request.GetArguments()["confidential"]; ok {
		opts.Confidential = new(args.Confidential)
	}

	if args.DueDate != "" {
		dueDate, err := gitlab.ParseISOTime(args.DueDate)
		if err != nil {
			return nil, fmt.Errorf("Invalid due_date format: %w", err)
		}

		opts.DueDateFixed = new(dueDate)
		opts.DueDateIsFixed = new(true)
	}

	if args.ParentID != 0 {
		opts.ParentID = new(args.ParentID)
	}

	epic, _, err := client.Epics.CreateEpic(args.GroupID.Value(), opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CreateEpic(%q): %w", args.GroupID.Value(), err)
	}

	return newToolResultJSON(epic)
}

type editEpicArgs struct {
	GroupID      mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	EpicIID      int64      `mcp_desc:"Internal ID of the epic to edit" mcp_required:"true"`
	Title        string     `mcp_desc:"Changes the title of the epic"`
	Description  string     `mcp_desc:"Changes the description of the epic. The description uses GitLab Flavored Markdown"`
	ParentID     int64      `mcp_desc:"Global ID of the parent epic"`
	AddLabels    string     `mcp_desc:"Comma-separated label names to add to the epic"`
	RemoveLabels string     `mcp_desc:"Comma-separated label names to remove from the epic"`
	StateEvent   string     `mcp_desc:"The state of the epic. Use 'close' to close the epic or 'reopen' to reopen a closed epic. Omit to keep the epic state unchanged" mcp_enum:"close,reopen"`
	Confidential bool       `mcp_desc:"If true, enables editing confidential epics or makes a public epic confidential. Default is false. Note: By design, specifying 'false' explicitly does not make confidential epics public - this is a security measure."`
	DueDate      string     `mcp_desc:"Due date of the epic (format: YYYY-MM-DD)"`
}

// EditEpic returns a ServerTool for updating an existing epic.
func (e *EpicTools) EditEpic() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(e.editEpic),
		Tool: mcpargs.NewTool("edit_epic", editEpicArgs{},
			mcp.WithDescription("Edit an existing GitLab epic"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (e *EpicTools) editEpic(ctx context.Context, client *gitlab.Client, args editEpicArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// First, fetch the epic to check if it's confidential
	epic, _, err := client.Epics.GetEpic(args.GroupID.Value(), args.EpicIID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetEpic(%q, %d): %w", args.GroupID.Value(), args.EpicIID, err)
	}

	if epic.Confidential && !args.Confidential {
		return mcp.NewToolResultError(fmt.Sprintf("cannot edit epic %d as it is confidential. Ensure it is safe to be shared with the model, then set confidential=true to edit", args.EpicIID)), nil
	}

	opts := &gitlab.UpdateEpicOptions{}

	if args.Title != "" {
		opts.Title = new(args.Title)
	}

	if args.Description != "" {
		opts.Description = new(args.Description)
	}

	if args.ParentID != 0 {
		opts.ParentID = new(args.ParentID)
	}

	opts.AddLabels = newLabelOptions(args.AddLabels)
	opts.RemoveLabels = newLabelOptions(args.RemoveLabels)

	if args.StateEvent != "" {
		opts.StateEvent = new(args.StateEvent)
	}

	if args.Confidential {
		opts.Confidential = new(args.Confidential)
	}

	if args.DueDate != "" {
		dueDate, err := gitlab.ParseISOTime(args.DueDate)
		if err != nil {
			return nil, fmt.Errorf("gitlab.ParseISOTime(%q): %w", args.DueDate, err)
		}

		opts.DueDateFixed = new(dueDate)
		opts.DueDateIsFixed = new(true)
	}

	epic, _, err = client.Epics.UpdateEpic(args.GroupID.Value(), args.EpicIID, opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("UpdateEpic(%q, %d): %w", args.GroupID.Value(), args.EpicIID, err)
	}

	return newToolResultJSON(epic)
}
