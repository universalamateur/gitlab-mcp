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
	scopeProjects      = "projects"
	scopeIssues        = "issues"
	scopeMergeRequests = "merge_requests"
	scopeMilestones    = "milestones"
	scopeSnippetTitles = "snippet_titles"
	scopeWikiBlobs     = "wiki_blobs"
	scopeCommits       = "commits"
	scopeBlobs         = "blobs"
	scopeUsers         = "users"
	scopeNotes         = "notes"
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

func (s *SearchTools) search(ctx context.Context, client *gitlab.Client, args searchArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.SearchOptions{
		ListOptions: gitlab.ListOptions{PerPage: maxPerPage},
	}

	if args.Ref != "" {
		opt.Ref = new(args.Ref)
	}

	reqOpt := gitlab.WithContext(ctx)

	var (
		results any
		err     error
	)

	switch {
	case !args.ProjectID.IsZero():
		results, err = s.searchProject(client, args.ProjectID.Value(), args.Search, args.Scope, opt, reqOpt)
	case !args.GroupID.IsZero():
		results, err = s.searchGroup(client, args.GroupID.Value(), args.Search, args.Scope, opt, reqOpt)
	default:
		results, err = s.searchGlobal(client, args.Search, args.Scope, opt, reqOpt)
	}

	if err != nil {
		return nil, err
	}

	return newToolResultJSON(results)
}

//nolint:cyclop // Dispatch by scope — each case is one line.
func (s *SearchTools) searchGlobal(client *gitlab.Client, query, scope string, opt *gitlab.SearchOptions, reqOpt gitlab.RequestOptionFunc) (any, error) {
	switch scope {
	case scopeProjects:
		return unwrap(client.Search.Projects(query, opt, reqOpt))
	case scopeIssues:
		return unwrap(client.Search.Issues(query, opt, reqOpt))
	case scopeMergeRequests:
		return unwrap(client.Search.MergeRequests(query, opt, reqOpt))
	case scopeMilestones:
		return unwrap(client.Search.Milestones(query, opt, reqOpt))
	case scopeSnippetTitles:
		return unwrap(client.Search.SnippetTitles(query, opt, reqOpt))
	case scopeWikiBlobs:
		return unwrap(client.Search.WikiBlobs(query, opt, reqOpt))
	case scopeCommits:
		return unwrap(client.Search.Commits(query, opt, reqOpt))
	case scopeBlobs:
		return unwrap(client.Search.Blobs(query, opt, reqOpt))
	case scopeUsers:
		return unwrap(client.Search.Users(query, opt, reqOpt))
	case scopeNotes:
		return nil, fmt.Errorf("%w: scope %q requires project_id", ErrInvalidArgument, scope)
	default:
		return nil, fmt.Errorf("%w: unsupported scope %q", ErrInvalidArgument, scope)
	}
}

//nolint:cyclop // Dispatch by scope — each case is one line.
func (s *SearchTools) searchGroup(client *gitlab.Client, gid any, query, scope string, opt *gitlab.SearchOptions, reqOpt gitlab.RequestOptionFunc) (any, error) {
	switch scope {
	case scopeProjects:
		return unwrap(client.Search.ProjectsByGroup(gid, query, opt, reqOpt))
	case scopeIssues:
		return unwrap(client.Search.IssuesByGroup(gid, query, opt, reqOpt))
	case scopeMergeRequests:
		return unwrap(client.Search.MergeRequestsByGroup(gid, query, opt, reqOpt))
	case scopeMilestones:
		return unwrap(client.Search.MilestonesByGroup(gid, query, opt, reqOpt))
	case scopeWikiBlobs:
		return unwrap(client.Search.WikiBlobsByGroup(gid, query, opt, reqOpt))
	case scopeCommits:
		return unwrap(client.Search.CommitsByGroup(gid, query, opt, reqOpt))
	case scopeBlobs:
		return unwrap(client.Search.BlobsByGroup(gid, query, opt, reqOpt))
	case scopeUsers:
		return unwrap(client.Search.UsersByGroup(gid, query, opt, reqOpt))
	case scopeSnippetTitles:
		return nil, fmt.Errorf("%w: scope %q is not available for group searches", ErrInvalidArgument, scope)
	case scopeNotes:
		return nil, fmt.Errorf("%w: scope %q requires project_id", ErrInvalidArgument, scope)
	default:
		return nil, fmt.Errorf("%w: unsupported scope %q", ErrInvalidArgument, scope)
	}
}

//nolint:cyclop // Dispatch by scope — each case is one line.
func (s *SearchTools) searchProject(client *gitlab.Client, pid any, query, scope string, opt *gitlab.SearchOptions, reqOpt gitlab.RequestOptionFunc) (any, error) {
	switch scope {
	case scopeIssues:
		return unwrap(client.Search.IssuesByProject(pid, query, opt, reqOpt))
	case scopeMergeRequests:
		return unwrap(client.Search.MergeRequestsByProject(pid, query, opt, reqOpt))
	case scopeMilestones:
		return unwrap(client.Search.MilestonesByProject(pid, query, opt, reqOpt))
	case scopeNotes:
		return unwrap(client.Search.NotesByProject(pid, query, opt, reqOpt))
	case scopeWikiBlobs:
		return unwrap(client.Search.WikiBlobsByProject(pid, query, opt, reqOpt))
	case scopeCommits:
		return unwrap(client.Search.CommitsByProject(pid, query, opt, reqOpt))
	case scopeBlobs:
		return unwrap(client.Search.BlobsByProject(pid, query, opt, reqOpt))
	case scopeUsers:
		return unwrap(client.Search.UsersByProject(pid, query, opt, reqOpt))
	case scopeProjects:
		return nil, fmt.Errorf("%w: scope %q is not available for project searches", ErrInvalidArgument, scope)
	case scopeSnippetTitles:
		return nil, fmt.Errorf("%w: scope %q is not available for project searches", ErrInvalidArgument, scope)
	default:
		return nil, fmt.Errorf("%w: unsupported scope %q", ErrInvalidArgument, scope)
	}
}

// unwrap discards the *gitlab.Response from a search API call.
//
//nolint:ireturn // Generic return is intentional — each search scope returns a different type.
func unwrap[T any](v T, _ *gitlab.Response, err error) (T, error) {
	return v, err
}
