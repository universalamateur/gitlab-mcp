package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"gitlab.com/fforster/gitlab-mcp/lib/gliter"
	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const defaultGroupsLimit = 1000

type GroupsTools struct{}

// AddTo registers all group-related tools with the provided MCPServer.
func (g *GroupsTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		g.GetGroup(),
		g.ListSubgroups(),
	)
}

type getGroupArgs struct {
	GroupID mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
}

func (g *GroupsTools) GetGroup() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(g.getGroup),
		Tool: mcpargs.NewTool("get_group", getGroupArgs{},
			mcp.WithDescription("Get all details of a GitLab group."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (g *GroupsTools) getGroup(ctx context.Context, client *gitlab.Client, args getGroupArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := gitlab.GetGroupOptions{}

	group, _, err := client.Groups.GetGroup(args.GroupID.Value(), &opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetGroup(%q): %w", args.GroupID.Value(), err)
	}

	group.Projects = nil       //nolint:SA1019
	group.SharedProjects = nil //nolint:SA1019

	return newToolResultJSON(group)
}

type listSubgroupsArgs struct {
	GroupID        mcpargs.ID `mcp_desc:"ID of the group either in owner/namespace format or the numeric group ID" mcp_required:"true"`
	Search         string     `mcp_desc:"Search subgroups against their short path (not the full path)"`
	Owned          bool       `mcp_desc:"If true, limits the search to groups owned by the current user. Default is false."`
	MinAccessLevel string     `mcp_desc:"Limit to groups where current user has at least this role" mcp_enum:"none,minimal,guest,planner,reporter,developer,maintainer,owner"`

	Limit     int    `mcp_desc:"The maximum number of subgroups to return. Defaults to 1000."`
	OrderBy   string `mcp_desc:"Sort subgroups by the selected field. Default is 'name'" mcp_enum:"id,name,path"`
	SortOrder string `mcp_desc:"Sort order to use. Default is 'asc'" mcp_enum:"asc,desc"`
}

func (g *GroupsTools) ListSubgroups() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(g.listSubgroups),
		Tool: mcpargs.NewTool("list_subgroups", listSubgroupsArgs{},
			mcp.WithDescription("Get a list of subgroups within a specific group."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (g *GroupsTools) listSubgroups(ctx context.Context, client *gitlab.Client, args listSubgroupsArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.ListSubGroupsOptions{
		ListOptions: newListOptions(request),
	}

	if args.Search != "" {
		opt.Search = gitlab.Ptr(args.Search)
	}

	if _, ok := request.GetArguments()["owned"]; ok {
		opt.Owned = gitlab.Ptr(args.Owned)
	}

	opt.MinAccessLevel = accessLevel(args.MinAccessLevel)

	nextPage := func(opts *gitlab.ListSubGroupsOptions, page int) {
		opts.Page = page
	}

	if args.Limit <= 0 {
		args.Limit = defaultGroupsLimit
	}

	iter := gliter.AllWithID(ctx, args.GroupID.Value(), client.Groups.ListSubGroups, *opt, nextPage)
	iter = gliter.Limited(iter, args.Limit)

	var groups []*gitlab.Group

	for group, err := range iter {
		if err != nil {
			return nil, fmt.Errorf("ListSubGroups(): %w", err)
		}

		groups = append(groups, group)
	}

	return newToolResultJSON(groups)
}

func accessLevel(s string) *gitlab.AccessLevelValue {
	v, ok := accessLevelMap[s]
	if !ok {
		return nil
	}

	return &v
}

var accessLevelMap = map[string]gitlab.AccessLevelValue{
	"none":       gitlab.NoPermissions,
	"minimal":    gitlab.MinimalAccessPermissions,
	"guest":      gitlab.GuestPermissions,
	"planner":    gitlab.PlannerPermissions,
	"reporter":   gitlab.ReporterPermissions,
	"developer":  gitlab.DeveloperPermissions,
	"maintainer": gitlab.MaintainerPermissions,
	"owner":      gitlab.OwnerPermissions,
}
