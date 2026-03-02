package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

const defaultProjectsLimit = 1000

type ProjectsTools struct{}

// AddTo registers all project-related tools with the provided MCPServer.
func (g *ProjectsTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		g.GetProject(),
		g.ListProjects(),
	)
}

type getProjectArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/namespace format or the numeric project ID" mcp_required:"true"`
}

// GetProject returns an MCP ServerTool for getting a single project.
func (p *ProjectsTools) GetProject() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(p.getProject),
		Tool: mcpargs.NewTool("get_project", getProjectArgs{},
			mcp.WithDescription("Get a single project"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (p *ProjectsTools) getProject(ctx context.Context, client *gitlab.Client, args getProjectArgs, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project, _, err := client.Projects.GetProject(args.ProjectID.Value(), nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetProject(%q): %w", args.ProjectID.Value(), err)
	}

	return newToolResultJSON(project)
}

type listProjectsArgs struct {
	GroupID mcpargs.ID `mcp_desc:"Filter projects by group ID, either in owner/namespace or numeric ID format."`

	Search         string               `mcp_desc:"Search subgroups against their short path (not the full path)"`
	Owned          bool                 `mcp_desc:"If true, limits the search to groups owned by the current user. Default is false."`
	MinAccessLevel string               `mcp_desc:"Limit to groups where current user has at least this role" mcp_enum:"none,minimal,guest,planner,reporter,developer,maintainer,owner"`
	Membership     mcpargs.OptionalBool `mcp_desc:"If true, limits the search to projects that the current user is a member of. Default is false."`

	Limit     int    `mcp_desc:"The maximum number of subgroups to return. Defaults to 1000."`
	OrderBy   string `mcp_desc:"Sort subgroups by the selected field. Default is 'created_at'" mcp_enum:"id,name,path,created_at,updated_at,star_count,last_activity_at,similarity"`
	SortOrder string `mcp_desc:"Sort order to use. Default is 'asc'" mcp_enum:"asc,desc"`
}

// ListProjects returns an MCP ServerTool for listing projects.
func (p *ProjectsTools) ListProjects() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(p.listProjects),
		Tool: mcpargs.NewTool("list_projects", listProjectsArgs{},
			mcp.WithDescription("List GitLab projects."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (p *ProjectsTools) listProjects(ctx context.Context, client *gitlab.Client, args listProjectsArgs, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !args.GroupID.IsZero() {
		return p.listGroupProjects(ctx, client, args, req)
	}

	opt := &gitlab.ListProjectsOptions{
		ListOptions: newListOptions(req),
	}

	if args.Search != "" {
		opt.Search = &args.Search
	}

	if args.Owned {
		opt.Owned = &args.Owned
	}

	if args.MinAccessLevel != "" {
		opt.MinAccessLevel = accessLevel(args.MinAccessLevel)
	}

	opt.Membership = args.Membership.Ptr()

	if args.OrderBy == "similarity" && (args.Search == "" || opt.Membership == nil || !*opt.Membership) {
		return mcp.NewToolResultError(`The search order by "similarity" is only available when both, the "search" argument is non-empty and the "membership" argument is true.`), nil
	}

	if args.Limit <= 0 {
		args.Limit = defaultProjectsLimit
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
		return client.Projects.ListProjects(opt, gitlab.WithContext(ctx), p)
	}

	projects, err := gitlab.ScanAndCollectN(pager, args.Limit)
	if err != nil {
		return nil, fmt.Errorf("ListProjects: %w", err)
	}

	return newToolResultJSON(projects)
}

func (p *ProjectsTools) listGroupProjects(ctx context.Context, client *gitlab.Client, args listProjectsArgs, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.ListGroupProjectsOptions{
		ListOptions: newListOptions(req),
	}

	if args.Search != "" {
		opt.Search = &args.Search
	}

	if args.Owned {
		opt.Owned = &args.Owned
	}

	if args.MinAccessLevel != "" {
		opt.MinAccessLevel = accessLevel(args.MinAccessLevel)
	}

	if _, ok := req.GetArguments()["membership"]; ok {
		return mcp.NewToolResultError(`The "group_id" and "membership" arguments are mutually exclusive.`), nil
	}

	if args.OrderBy == "similarity" && args.Search == "" {
		return mcp.NewToolResultError(`The search order by "similarity" is only available when the "search" argument is non-empty.`), nil
	}

	if args.Limit <= 0 {
		args.Limit = defaultProjectsLimit
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
		return client.Groups.ListGroupProjects(args.GroupID.Value(), opt, gitlab.WithContext(ctx), p)
	}

	projects, err := gitlab.ScanAndCollectN(pager, args.Limit)
	if err != nil {
		return nil, fmt.Errorf("ListProjects: %w", err)
	}

	return newToolResultJSON(projects)
}
