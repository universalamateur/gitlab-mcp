package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/gliter"
	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

const defaultProjectsLimit = 1000

type ProjectsTools struct{}

// AddTo registers all project-related tools with the provided MCPServer.
func (g *ProjectsTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		g.ListProjects(),
	)
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

	nextPage := func(opts *gitlab.ListProjectsOptions, page int) {
		opts.Page = page
	}

	if args.Limit <= 0 {
		args.Limit = defaultProjectsLimit
	}

	iter := gliter.All(ctx, client.Projects.ListProjects, *opt, nextPage)
	iter = gliter.Limited(iter, args.Limit)

	var projects []*gitlab.Project

	for project, err := range iter {
		if err != nil {
			return nil, fmt.Errorf("ListProjects: %w", err)
		}

		projects = append(projects, project)
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

	nextPage := func(opts *gitlab.ListGroupProjectsOptions, page int) {
		opts.Page = page
	}

	if args.Limit <= 0 {
		args.Limit = defaultProjectsLimit
	}

	iter := gliter.AllWithID(ctx, args.GroupID.Value(), client.Groups.ListGroupProjects, *opt, nextPage)
	iter = gliter.Limited(iter, args.Limit)

	var projects []*gitlab.Project

	for project, err := range iter {
		if err != nil {
			return nil, fmt.Errorf("ListProjects: %w", err)
		}

		projects = append(projects, project)
	}

	return newToolResultJSON(projects)
}
