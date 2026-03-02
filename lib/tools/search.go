package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// Search scope constants used in the switch dispatchers.
const (
	searchScopeProjects      = "projects"
	searchScopeIssues        = "issues"
	searchScopeMergeRequests = "merge_requests"
	searchScopeMilestones    = "milestones"
	searchScopeSnippetTitles = "snippet_titles"
	searchScopeWikiBlobs     = "wiki_blobs"
	searchScopeCommits       = "commits"
	searchScopeBlobs         = "blobs"
	searchScopeUsers         = "users"
	searchScopeNotes         = "notes"
)

// SearchTools provides search capabilities across GitLab resources.
type SearchTools struct{}

// AddTo registers search tools with the MCP server.
func (s *SearchTools) AddTo(srv ToolAdder) {
	srv.AddTools(s.Search())
}

type searchArgs struct {
	Search    string     `mcp_desc:"Search query string" mcp_required:"true"`
	Scope     string     `mcp_desc:"What to search for" mcp_required:"true" mcp_enum:"projects,issues,merge_requests,milestones,snippet_titles,wiki_blobs,commits,blobs,users,notes"`
	GroupID   mcpargs.ID `mcp_desc:"Scope search to a group. Omit for global search"`
	ProjectID mcpargs.ID `mcp_desc:"Scope search to a project. Omit for global search. Required for 'notes' scope"`
	Ref       string     `mcp_desc:"Git ref to search in (branch or tag). Only applies to 'blobs' and 'commits' scopes"`
}

// Search returns a tool for searching GitLab resources across multiple scopes.
func (s *SearchTools) Search() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(s.search),
		Tool: mcpargs.NewTool("search", searchArgs{},
			mcp.WithDescription("Search GitLab resources. Set group_id or project_id to narrow the search scope, or omit both for a global search."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (s *SearchTools) search(ctx context.Context, client *gitlab.Client, args searchArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.SearchOptions{
		ListOptions: newListOptions(request),
	}

	if args.Ref != "" {
		opt.Ref = new(args.Ref)
	}

	switch {
	case !args.ProjectID.IsZero():
		return s.searchProject(ctx, client, args.ProjectID.Value(), args.Search, args.Scope, opt)
	case !args.GroupID.IsZero():
		return s.searchGroup(ctx, client, args.GroupID.Value(), args.Search, args.Scope, opt)
	default:
		return s.searchGlobal(ctx, client, args.Search, args.Scope, opt)
	}
}

//nolint:cyclop // Dispatch by scope — each case is one line.
func (s *SearchTools) searchGlobal(ctx context.Context, client *gitlab.Client, query, scope string, opt *gitlab.SearchOptions) (*mcp.CallToolResult, error) {
	switch scope {
	case searchScopeProjects:
		return searchAllGlobal(ctx, client.Search.Projects, query, opt)
	case searchScopeIssues:
		return searchAllGlobal(ctx, client.Search.Issues, query, opt)
	case searchScopeMergeRequests:
		return searchAllGlobal(ctx, client.Search.MergeRequests, query, opt)
	case searchScopeMilestones:
		return searchAllGlobal(ctx, client.Search.Milestones, query, opt)
	case searchScopeSnippetTitles:
		return searchAllGlobal(ctx, client.Search.SnippetTitles, query, opt)
	case searchScopeWikiBlobs:
		return searchAllGlobal(ctx, client.Search.WikiBlobs, query, opt)
	case searchScopeCommits:
		return searchAllGlobal(ctx, client.Search.Commits, query, opt)
	case searchScopeBlobs:
		return searchAllGlobal(ctx, client.Search.Blobs, query, opt)
	case searchScopeUsers:
		return searchAllGlobal(ctx, client.Search.Users, query, opt)
	case searchScopeNotes:
		return nil, fmt.Errorf("%w: scope %q requires project_id", ErrInvalidArgument, scope)
	default:
		return nil, fmt.Errorf("%w: unsupported scope %q", ErrInvalidArgument, scope)
	}
}

//nolint:cyclop // Dispatch by scope — each case is one line.
func (s *SearchTools) searchGroup(ctx context.Context, client *gitlab.Client, gid any, query, scope string, opt *gitlab.SearchOptions) (*mcp.CallToolResult, error) {
	switch scope {
	case searchScopeProjects:
		return searchAllNamespace(ctx, client.Search.ProjectsByGroup, gid, query, opt)
	case searchScopeIssues:
		return searchAllNamespace(ctx, client.Search.IssuesByGroup, gid, query, opt)
	case searchScopeMergeRequests:
		return searchAllNamespace(ctx, client.Search.MergeRequestsByGroup, gid, query, opt)
	case searchScopeMilestones:
		return searchAllNamespace(ctx, client.Search.MilestonesByGroup, gid, query, opt)
	case searchScopeWikiBlobs:
		return searchAllNamespace(ctx, client.Search.WikiBlobsByGroup, gid, query, opt)
	case searchScopeCommits:
		return searchAllNamespace(ctx, client.Search.CommitsByGroup, gid, query, opt)
	case searchScopeBlobs:
		return searchAllNamespace(ctx, client.Search.BlobsByGroup, gid, query, opt)
	case searchScopeUsers:
		return searchAllNamespace(ctx, client.Search.UsersByGroup, gid, query, opt)
	case searchScopeSnippetTitles:
		return nil, fmt.Errorf("%w: scope %q is not available for group searches", ErrInvalidArgument, scope)
	case searchScopeNotes:
		return nil, fmt.Errorf("%w: scope %q requires project_id", ErrInvalidArgument, scope)
	default:
		return nil, fmt.Errorf("%w: unsupported scope %q", ErrInvalidArgument, scope)
	}
}

//nolint:cyclop // Dispatch by scope — each case is one line.
func (s *SearchTools) searchProject(ctx context.Context, client *gitlab.Client, pid any, query, scope string, opt *gitlab.SearchOptions) (*mcp.CallToolResult, error) {
	switch scope {
	case searchScopeIssues:
		return searchAllNamespace(ctx, client.Search.IssuesByProject, pid, query, opt)
	case searchScopeMergeRequests:
		return searchAllNamespace(ctx, client.Search.MergeRequestsByProject, pid, query, opt)
	case searchScopeMilestones:
		return searchAllNamespace(ctx, client.Search.MilestonesByProject, pid, query, opt)
	case searchScopeNotes:
		return searchAllNamespace(ctx, client.Search.NotesByProject, pid, query, opt)
	case searchScopeWikiBlobs:
		return searchAllNamespace(ctx, client.Search.WikiBlobsByProject, pid, query, opt)
	case searchScopeCommits:
		return searchAllNamespace(ctx, client.Search.CommitsByProject, pid, query, opt)
	case searchScopeBlobs:
		return searchAllNamespace(ctx, client.Search.BlobsByProject, pid, query, opt)
	case searchScopeUsers:
		return searchAllNamespace(ctx, client.Search.UsersByProject, pid, query, opt)
	case searchScopeProjects:
		return nil, fmt.Errorf("%w: scope %q is not available for project searches", ErrInvalidArgument, scope)
	case searchScopeSnippetTitles:
		return nil, fmt.Errorf("%w: scope %q is not available for project searches", ErrInvalidArgument, scope)
	default:
		return nil, fmt.Errorf("%w: unsupported scope %q", ErrInvalidArgument, scope)
	}
}

func searchAllGlobal[T any](ctx context.Context, searchFunc func(string, *gitlab.SearchOptions, ...gitlab.RequestOptionFunc) ([]*T, *gitlab.Response, error), query string, opt *gitlab.SearchOptions) (*mcp.CallToolResult, error) {
	items, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*T, *gitlab.Response, error) {
		return searchFunc(query, opt, gitlab.WithContext(ctx), p)
	})
	if err != nil {
		return nil, fmt.Errorf("searchAllGlobal: %w", err)
	}

	return newToolResultJSON(items)
}

func searchAllNamespace[T any](ctx context.Context, searchFunc func(any, string, *gitlab.SearchOptions, ...gitlab.RequestOptionFunc) ([]*T, *gitlab.Response, error), namespaceID any, query string, opt *gitlab.SearchOptions) (*mcp.CallToolResult, error) {
	items, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*T, *gitlab.Response, error) {
		return searchFunc(namespaceID, query, opt, gitlab.WithContext(ctx), p)
	})
	if err != nil {
		return nil, fmt.Errorf("searchAllNamespace: %w", err)
	}

	return newToolResultJSON(items)
}
