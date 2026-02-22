package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

const defaultLabelsLimit = 1000

// LabelsTools provides label management capabilities for projects and groups.
type LabelsTools struct{}

// AddTo registers label tools with the MCP server.
func (l *LabelsTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		l.ListProjectLabels(),
		l.CreateProjectLabel(),
		l.UpdateProjectLabel(),
		l.DeleteProjectLabel(),
		l.ListGroupLabels(),
		l.CreateGroupLabel(),
		l.UpdateGroupLabel(),
		l.DeleteGroupLabel(),
	)
}

// --- list_project_labels ---

type listProjectLabelsArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	Search    string     `mcp_desc:"Search labels by name"`
	Limit     int        `mcp_desc:"Maximum number of labels to return. Defaults to 1000."`
}

// ListProjectLabels returns a tool for listing labels in a project.
func (l *LabelsTools) ListProjectLabels() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(l.listProjectLabels),
		Tool: mcpargs.NewTool("list_project_labels", listProjectLabelsArgs{},
			mcp.WithDescription("List labels for a project"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (l *LabelsTools) listProjectLabels(ctx context.Context, client *gitlab.Client, args listProjectLabelsArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.ListLabelsOptions{
		ListOptions: gitlab.ListOptions{PerPage: maxPerPage},
	}

	if args.Search != "" {
		opt.Search = gitlab.Ptr(args.Search)
	}

	if args.Limit <= 0 {
		args.Limit = defaultLabelsLimit
	}

	var labels []*gitlab.Label

	for {
		page, resp, err := client.Labels.ListLabels(args.ProjectID.Value(), opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListLabels(%q): %w", args.ProjectID.Value(), err)
		}

		labels = append(labels, page...)

		if resp.NextPage == 0 || len(labels) >= args.Limit {
			break
		}

		opt.Page = resp.NextPage
	}

	if len(labels) > args.Limit {
		labels = labels[:args.Limit]
	}

	return newToolResultJSON(labels)
}

// --- create_project_label ---

type createProjectLabelArgs struct {
	ProjectID   mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	Name        string     `mcp_desc:"Name of the label" mcp_required:"true"`
	Color       string     `mcp_desc:"Color of the label in hex format (e.g. '#FF0000')" mcp_required:"true"`
	Description string     `mcp_desc:"Description of the label"`
	Priority    int64      `mcp_desc:"Priority of the label. Must be zero or positive"`
}

// CreateProjectLabel returns a tool for creating a label in a project.
func (l *LabelsTools) CreateProjectLabel() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(l.createProjectLabel),
		Tool: mcpargs.NewTool("create_project_label", createProjectLabelArgs{},
			mcp.WithDescription("Create a label in a project"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (l *LabelsTools) createProjectLabel(ctx context.Context, client *gitlab.Client, args createProjectLabelArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.CreateLabelOptions{
		Name:  gitlab.Ptr(args.Name),
		Color: gitlab.Ptr(args.Color),
	}

	if args.Description != "" {
		opt.Description = gitlab.Ptr(args.Description)
	}

	if _, ok := request.GetArguments()["priority"]; ok {
		opt.Priority = gitlab.Ptr(args.Priority)
	}

	label, _, err := client.Labels.CreateLabel(args.ProjectID.Value(), opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CreateLabel(%q): %w", args.ProjectID.Value(), err)
	}

	return newToolResultJSON(label)
}

// --- update_project_label ---

type updateProjectLabelArgs struct {
	ProjectID   mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	Name        string     `mcp_desc:"Current name of the label to update" mcp_required:"true"`
	NewName     string     `mcp_desc:"New name for the label"`
	Color       string     `mcp_desc:"New color in hex format (e.g. '#00FF00')"`
	Description string     `mcp_desc:"New description for the label"`
	Priority    int64      `mcp_desc:"New priority for the label"`
}

// UpdateProjectLabel returns a tool for updating a label in a project.
func (l *LabelsTools) UpdateProjectLabel() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(l.updateProjectLabel),
		Tool: mcpargs.NewTool("update_project_label", updateProjectLabelArgs{},
			mcp.WithDescription("Update a label in a project"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (l *LabelsTools) updateProjectLabel(ctx context.Context, client *gitlab.Client, args updateProjectLabelArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.UpdateLabelOptions{}

	if args.NewName != "" {
		opt.NewName = gitlab.Ptr(args.NewName)
	}

	if args.Color != "" {
		opt.Color = gitlab.Ptr(args.Color)
	}

	if args.Description != "" {
		opt.Description = gitlab.Ptr(args.Description)
	}

	if _, ok := request.GetArguments()["priority"]; ok {
		opt.Priority = gitlab.Ptr(args.Priority)
	}

	label, _, err := client.Labels.UpdateLabel(args.ProjectID.Value(), args.Name, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("UpdateLabel(%q, %q): %w", args.ProjectID.Value(), args.Name, err)
	}

	return newToolResultJSON(label)
}

// --- delete_project_label ---

type deleteProjectLabelArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	Name      string     `mcp_desc:"Name of the label to delete" mcp_required:"true"`
}

// DeleteProjectLabel returns a tool for deleting a label from a project.
func (l *LabelsTools) DeleteProjectLabel() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(l.deleteProjectLabel),
		Tool: mcpargs.NewTool("delete_project_label", deleteProjectLabelArgs{},
			mcp.WithDescription("Delete a label from a project"),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (l *LabelsTools) deleteProjectLabel(ctx context.Context, client *gitlab.Client, args deleteProjectLabelArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, err := client.Labels.DeleteLabel(args.ProjectID.Value(), args.Name, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("DeleteLabel(%q, %q): %w", args.ProjectID.Value(), args.Name, err)
	}

	return mcp.NewToolResultText("Label deleted"), nil
}

// --- list_group_labels ---

type listGroupLabelsArgs struct {
	GroupID mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	Search  string     `mcp_desc:"Search labels by name"`
	Limit   int        `mcp_desc:"Maximum number of labels to return. Defaults to 1000."`
}

// ListGroupLabels returns a tool for listing labels in a group.
func (l *LabelsTools) ListGroupLabels() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(l.listGroupLabels),
		Tool: mcpargs.NewTool("list_group_labels", listGroupLabelsArgs{},
			mcp.WithDescription("List labels for a group"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (l *LabelsTools) listGroupLabels(ctx context.Context, client *gitlab.Client, args listGroupLabelsArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.ListGroupLabelsOptions{
		ListOptions: gitlab.ListOptions{PerPage: maxPerPage},
	}

	if args.Search != "" {
		opt.Search = gitlab.Ptr(args.Search)
	}

	if args.Limit <= 0 {
		args.Limit = defaultLabelsLimit
	}

	var labels []*gitlab.GroupLabel

	for {
		page, resp, err := client.GroupLabels.ListGroupLabels(args.GroupID.Value(), opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListGroupLabels(%q): %w", args.GroupID.Value(), err)
		}

		labels = append(labels, page...)

		if resp.NextPage == 0 || len(labels) >= args.Limit {
			break
		}

		opt.Page = resp.NextPage
	}

	if len(labels) > args.Limit {
		labels = labels[:args.Limit]
	}

	return newToolResultJSON(labels)
}

// --- create_group_label ---

type createGroupLabelArgs struct {
	GroupID     mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	Name        string     `mcp_desc:"Name of the label" mcp_required:"true"`
	Color       string     `mcp_desc:"Color of the label in hex format (e.g. '#FF0000')" mcp_required:"true"`
	Description string     `mcp_desc:"Description of the label"`
	Priority    int64      `mcp_desc:"Priority of the label. Must be zero or positive"`
}

// CreateGroupLabel returns a tool for creating a label in a group.
func (l *LabelsTools) CreateGroupLabel() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(l.createGroupLabel),
		Tool: mcpargs.NewTool("create_group_label", createGroupLabelArgs{},
			mcp.WithDescription("Create a label in a group"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (l *LabelsTools) createGroupLabel(ctx context.Context, client *gitlab.Client, args createGroupLabelArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.CreateGroupLabelOptions{
		Name:  gitlab.Ptr(args.Name),
		Color: gitlab.Ptr(args.Color),
	}

	if args.Description != "" {
		opt.Description = gitlab.Ptr(args.Description)
	}

	if _, ok := request.GetArguments()["priority"]; ok {
		opt.Priority = gitlab.Ptr(args.Priority)
	}

	label, _, err := client.GroupLabels.CreateGroupLabel(args.GroupID.Value(), opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CreateGroupLabel(%q): %w", args.GroupID.Value(), err)
	}

	return newToolResultJSON(label)
}

// --- update_group_label ---

type updateGroupLabelArgs struct {
	GroupID     mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	Name        string     `mcp_desc:"Current name of the label to update" mcp_required:"true"`
	NewName     string     `mcp_desc:"New name for the label"`
	Color       string     `mcp_desc:"New color in hex format (e.g. '#00FF00')"`
	Description string     `mcp_desc:"New description for the label"`
	Priority    int64      `mcp_desc:"New priority for the label"`
}

// UpdateGroupLabel returns a tool for updating a label in a group.
func (l *LabelsTools) UpdateGroupLabel() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(l.updateGroupLabel),
		Tool: mcpargs.NewTool("update_group_label", updateGroupLabelArgs{},
			mcp.WithDescription("Update a label in a group"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (l *LabelsTools) updateGroupLabel(ctx context.Context, client *gitlab.Client, args updateGroupLabelArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.UpdateGroupLabelOptions{}

	if args.NewName != "" {
		opt.NewName = gitlab.Ptr(args.NewName)
	}

	if args.Color != "" {
		opt.Color = gitlab.Ptr(args.Color)
	}

	if args.Description != "" {
		opt.Description = gitlab.Ptr(args.Description)
	}

	if _, ok := request.GetArguments()["priority"]; ok {
		opt.Priority = gitlab.Ptr(args.Priority)
	}

	label, _, err := client.GroupLabels.UpdateGroupLabel(args.GroupID.Value(), args.Name, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("UpdateGroupLabel(%q, %q): %w", args.GroupID.Value(), args.Name, err)
	}

	return newToolResultJSON(label)
}

// --- delete_group_label ---

type deleteGroupLabelArgs struct {
	GroupID mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	Name    string     `mcp_desc:"Name of the label to delete" mcp_required:"true"`
}

// DeleteGroupLabel returns a tool for deleting a label from a group.
func (l *LabelsTools) DeleteGroupLabel() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(l.deleteGroupLabel),
		Tool: mcpargs.NewTool("delete_group_label", deleteGroupLabelArgs{},
			mcp.WithDescription("Delete a label from a group"),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (l *LabelsTools) deleteGroupLabel(ctx context.Context, client *gitlab.Client, args deleteGroupLabelArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, err := client.GroupLabels.DeleteGroupLabel(args.GroupID.Value(), args.Name, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("DeleteGroupLabel(%q, %q): %w", args.GroupID.Value(), args.Name, err)
	}

	return mcp.NewToolResultText("Label deleted"), nil
}
