package tools

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// File action constants.
const (
	ActionCreate = "create"
	ActionDelete = "delete"
	ActionMove   = "move"
	ActionUpdate = "update"
)

// Error variables for static wrapping.
var (
	ErrInvalidAction        = errors.New("invalid action")
	ErrPreviousPathRequired = errors.New("previous_path is required when action is move")
	ErrContentsRequired     = errors.New("contents is required for action")
)

// RepositoryServiceInterface defines the interface for repository file-related GitLab operations.
// It provides methods for accessing and manipulating repository files and contents.
type RepositoryServiceInterface interface {
	// AddTo registers all repository file-related tools with the provided MCPServer.
	AddTo(srv ToolAdder)

	// ListRepositoryDirectory returns a tool for listing repository files and directories.
	ListRepositoryDirectory() server.ServerTool

	// GetRepositoryFileContents returns a tool for retrieving contents of a file either by blob SHA or by file path.
	GetRepositoryFileContents() server.ServerTool

	// CreateCommit returns a tool for creating a new commit with file changes.
	CreateCommit() server.ServerTool
}

// RepositoryTools implements the RepositoryServiceInterface and provides tools
// for listing repository files and retrieving file contents from GitLab.
type RepositoryTools struct{}

// AddTo registers all repository file-related tools with the provided MCPServer.
// It adds tools for listing repository tree and retrieving file contents.
func (r *RepositoryTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		r.ListRepositoryDirectory(),
		r.GetRepositoryFileContents(),
		r.CreateCommit(),
	)
}

type listRepositoryDirectoryArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"Specifies the project to list files from, using either owner/project format or numeric project ID" mcp_required:"true"`
	Path      string     `mcp_desc:"The path inside the repository to list files from; defaults to the repository root if not provided"`
	Ref       string     `mcp_desc:"Specifies which branch or tag to list files from; defaults to the project's default branch if not provided"`
	Recursive bool       `mcp_desc:"When set to true, lists files in subdirectories recursively instead of just the 'path' level"`
}

// ListRepositoryDirectory returns a ServerTool for listing repository files and directories.
// The tool accepts project ID and various optional parameters for customizing the tree listing.
func (r *RepositoryTools) ListRepositoryDirectory() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(r.listRepositoryDirectory),
		Tool: mcpargs.NewTool("list_repository_directory", listRepositoryDirectoryArgs{},
			mcp.WithDescription("Get a list of repository files and directories in a project. The returned JSON uses Git terminology, e.g. calling files 'blob' and directories 'tree'."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (r *RepositoryTools) listRepositoryDirectory(ctx context.Context, client *gitlab.Client, args listRepositoryDirectoryArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Prepare options for the API call
	opt := &gitlab.ListTreeOptions{
		ListOptions: newListOptions(request),
	}

	// Set optional parameters if provided
	if args.Path != "" {
		opt.Path = new(args.Path)
	}

	if args.Ref != "" {
		opt.Ref = new(args.Ref)
	}

	opt.Recursive = new(args.Recursive)

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
		return client.Repositories.ListTree(args.ProjectID.Value(), opt, gitlab.WithContext(ctx), p)
	}

	allTree, err := gitlab.ScanAndCollect(pager)
	if err != nil {
		return nil, fmt.Errorf("ListTree(%q): %w", args.ProjectID.Value(), err)
	}

	return newToolResultJSON(allTree)
}

type getRepositoryFileContentsArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"Specifies the project to get file contents from, using either owner/project format or numeric project ID" mcp_required:"true"`
	SHA       string     `mcp_desc:"Specifies the blob SHA to get contents from. Provide either 'sha' or 'file_path', but not both"`
	FilePath  string     `mcp_desc:"Specifies the path to the file in the repository. Provide either 'sha' or 'file_path', but not both"`
	Ref       string     `mcp_desc:"Specifies which branch or tag to use when accessing a file by path; defaults to the project's default branch if not provided"`
}

type createCommitArgs struct {
	ProjectID     mcpargs.ID `mcp_desc:"GitLab project ID (numeric or path format)" mcp_required:"true"`
	Branch        string     `mcp_desc:"Branch name (must not be a protected branch)" mcp_required:"true"`
	CommitMessage string     `mcp_desc:"Commit message" mcp_required:"true"`
	Path          string     `mcp_desc:"File path relative to repository root" mcp_required:"true"`
	Contents      string     `mcp_desc:"File contents (required for create/update actions, optional for move to overwrite content, not used for delete)"`
	Action        string     `mcp_desc:"File action - one of: create, delete, move, update" mcp_required:"true" mcp_enum:"create,delete,move,update"`
	PreviousPath  string     `mcp_desc:"Required when action=move - the previous file path"`
	StartBranch   string     `mcp_desc:"If set, creates a new branch based on StartBranch and commits to it."`
}

// GetRepositoryFileContents returns a ServerTool for retrieving the contents of a file.
// The tool accepts either a blob SHA or a file path with optional reference (branch/tag).
func (r *RepositoryTools) GetRepositoryFileContents() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(r.getRepositoryFileContents),
		Tool: mcpargs.NewTool("get_repository_file_contents", getRepositoryFileContentsArgs{},
			mcp.WithDescription("Get the contents of a single file from the repository."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (r *RepositoryTools) getRepositoryFileContents(ctx context.Context, client *gitlab.Client, args getRepositoryFileContentsArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if haveSHA, havePath := args.SHA != "", args.FilePath != ""; haveSHA == havePath {
		return mcp.NewToolResultError("exactly one of 'sha' or 'file_path' must be provided"), nil
	}

	// Choose the appropriate API endpoint based on the provided parameters
	if args.SHA != "" {
		// Use RawBlobContent when SHA is provided
		contents, _, err := client.Repositories.RawBlobContent(args.ProjectID.Value(), args.SHA, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("RawBlobContent(%q, %q): %w", args.ProjectID.Value(), args.SHA, err)
		}

		return mcp.NewToolResultText(string(contents)), nil
	}

	// Use GetRawFile when FilePath is provided
	opt := gitlab.GetRawFileOptions{}
	if args.Ref != "" {
		opt.Ref = new(args.Ref)
	}

	contents, _, err := client.RepositoryFiles.GetRawFile(args.ProjectID.Value(), args.FilePath, &opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetRawFile(%q, %q, ref=%q): %w", args.ProjectID.Value(), args.FilePath, args.Ref, err)
	}

	return mcp.NewToolResultText(string(contents)), nil
}

// CreateCommit returns a ServerTool for creating a new commit with file changes.
// The tool accepts project ID, branch, commit message, and file actions.
func (r *RepositoryTools) CreateCommit() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(r.createCommit),
		Tool: mcpargs.NewTool("create_commit", createCommitArgs{},
			mcp.WithDescription("Creates a new commit using GitLab's Commits API with support for file actions: create, delete, move, update."),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (r *RepositoryTools) createCommit(ctx context.Context, client *gitlab.Client, args createCommitArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	commitOptions, err := r.newCreateCommitOptions(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	commit, _, err := client.Commits.CreateCommit(args.ProjectID.Value(), commitOptions, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CreateCommit(%q): %w", args.ProjectID.Value(), err)
	}

	return newToolResultJSON(commit)
}

func (r *RepositoryTools) newCreateCommitOptions(args createCommitArgs) (*gitlab.CreateCommitOptions, error) {
	// Validate action
	validActions := []gitlab.FileActionValue{
		gitlab.FileCreate,
		gitlab.FileDelete,
		gitlab.FileMove,
		gitlab.FileUpdate,
	}

	if !slices.Contains(validActions, gitlab.FileActionValue(args.Action)) {
		return nil, fmt.Errorf("%w: %s. Must be one of: %q", ErrInvalidArgument, args.Action, validActions)
	}

	// Build commit action
	commitAction := &gitlab.CommitActionOptions{
		Action:   new(gitlab.FileActionValue(args.Action)),
		FilePath: new(args.Path),
	}

	// Set action-specific fields
	switch args.Action {
	case ActionCreate, ActionUpdate:
		if args.Contents == "" {
			return nil, fmt.Errorf("%w: %s", ErrContentsRequired, args.Action)
		}

		commitAction.Content = new(args.Contents)
	case ActionMove:
		if args.PreviousPath == "" {
			return nil, ErrPreviousPathRequired
		}

		commitAction.PreviousPath = new(args.PreviousPath)

		if args.Contents != "" {
			commitAction.Content = new(args.Contents)
		}
	}

	// Return complete options
	opts := &gitlab.CreateCommitOptions{
		Branch:        new(args.Branch),
		CommitMessage: new(args.CommitMessage),
		Actions:       []*gitlab.CommitActionOptions{commitAction},
	}

	// Add StartBranch if provided
	if args.StartBranch != "" {
		opts.StartBranch = new(args.StartBranch)
	}

	return opts, nil
}
