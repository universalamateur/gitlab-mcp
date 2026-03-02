package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// SnippetsServiceInterface defines the interface for snippet-related GitLab operations.
// It provides methods for retrieving, listing, creating, updating, and deleting snippets.
type SnippetsServiceInterface interface {
	// AddTo registers all snippet-related tools with the provided MCPServer.
	AddTo(srv ToolAdder)

	// ListUserSnippets returns a tool for listing snippets owned by the current user.
	ListUserSnippets() server.ServerTool

	// ListAllSnippets returns a tool for listing all snippets the user has access to.
	ListAllSnippets() server.ServerTool

	// GetSnippet returns a tool for fetching a specific snippet by its ID.
	GetSnippet() server.ServerTool

	// GetSnippetContent returns a tool for fetching the content of a specific snippet.
	GetSnippetContent() server.ServerTool

	// CreateSnippet returns a tool for creating a new snippet.
	CreateSnippet() server.ServerTool

	// UpdateSnippet returns a tool for updating an existing snippet.
	UpdateSnippet() server.ServerTool

	// DeleteSnippet returns a tool for deleting a snippet.
	DeleteSnippet() server.ServerTool
}

const (
	snippetVisibilityPrivate  = "private"
	snippetVisibilityInternal = "internal"
	snippetVisibilityPublic   = "public"
)

// SnippetsTools implements tools for interacting with GitLab snippets.
// It provides methods for listing, creating, updating, deleting, and retrieving snippets.
type SnippetsTools struct{}

// AddTo registers all snippet-related tools with the provided MCPServer.
// It adds tools for listing, retrieving, creating, updating, and deleting snippets.
func (s *SnippetsTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		s.ListUserSnippets(),
		s.ListAllSnippets(),
		s.GetSnippet(),
		s.GetSnippetContent(),
		s.CreateSnippet(),
		s.UpdateSnippet(),
		s.DeleteSnippet(),
	)
}

// ListUserSnippets returns a ServerTool for listing snippets owned by the current user.
func (s *SnippetsTools) ListUserSnippets() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(s.listUserSnippets),
		Tool: mcp.NewTool("list_user_snippets",
			mcp.WithDescription("List snippets owned by the current user"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (s *SnippetsTools) listUserSnippets(ctx context.Context, client *gitlab.Client, _ struct{}, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.ListSnippetsOptions{
		ListOptions: newListOptions(request),
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Snippet, *gitlab.Response, error) {
		return client.Snippets.ListSnippets(opt, gitlab.WithContext(ctx), p)
	}

	snippets, err := gitlab.ScanAndCollect(pager)
	if err != nil {
		return nil, fmt.Errorf("ListSnippets: %w", err)
	}

	return newToolResultJSON(snippets)
}

type listAllSnippetsArgs struct {
	IncludePrivate bool `mcp_desc:"Include private snippets in the results. If false (the default), only public snippets are returned"`
}

// ListAllSnippets returns a ServerTool for listing all snippets the user has access to.
// The include_private parameter determines whether to include private snippets or only public ones.
func (s *SnippetsTools) ListAllSnippets() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(s.listAllSnippets),
		Tool: mcpargs.NewTool("list_all_snippets", listAllSnippetsArgs{},
			mcp.WithDescription("List all snippets the user has access to"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (s *SnippetsTools) listAllSnippets(ctx context.Context, client *gitlab.Client, args listAllSnippetsArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if args.IncludePrivate {
		return s.listAllSnippetsIncludingPrivate(ctx, client, request)
	}

	return s.listAllPublicSnippets(ctx, client, request)
}

// listAllSnippetsIncludingPrivate lists all snippets, private and public.
func (s *SnippetsTools) listAllSnippetsIncludingPrivate(ctx context.Context, client *gitlab.Client, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.ListAllSnippetsOptions{
		ListOptions: newListOptions(request),
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Snippet, *gitlab.Response, error) {
		return client.Snippets.ListAllSnippets(opt, gitlab.WithContext(ctx), p)
	}

	snippets, err := gitlab.ScanAndCollect(pager)
	if err != nil {
		return nil, fmt.Errorf("ListAllSnippets: %w", err)
	}

	return newToolResultJSON(snippets)
}

// listAllPublicSnippets lists all public snippets.
func (s *SnippetsTools) listAllPublicSnippets(ctx context.Context, client *gitlab.Client, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// List only public snippets
	opt := &gitlab.ExploreSnippetsOptions{
		ListOptions: newListOptions(request),
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Snippet, *gitlab.Response, error) {
		return client.Snippets.ExploreSnippets(opt, gitlab.WithContext(ctx), p)
	}

	snippets, err := gitlab.ScanAndCollect(pager)
	if err != nil {
		return nil, fmt.Errorf("ExploreSnippets: %w", err)
	}

	return newToolResultJSON(snippets)
}

type getSnippetArgs struct {
	SnippetID int64 `mcp_desc:"The ID of the snippet" mcp_required:"true"`
}

// GetSnippet returns a ServerTool for fetching information about a specific snippet.
// The tool accepts a snippet ID as a parameter.
func (s *SnippetsTools) GetSnippet() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(s.getSnippet),
		Tool: mcpargs.NewTool("get_snippet", getSnippetArgs{},
			mcp.WithDescription("Returns the metadata of a snippet, such as title and description. File content is not returned."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (s *SnippetsTools) getSnippet(ctx context.Context, client *gitlab.Client, args getSnippetArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	snippet, _, err := client.Snippets.GetSnippet(args.SnippetID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetSnippet(%d): %w", args.SnippetID, err)
	}

	return newToolResultJSON(snippet)
}

type getSnippetContentArgs struct {
	SnippetID int64 `mcp_desc:"The ID of the snippet" mcp_required:"true"`
}

// GetSnippetContent returns a ServerTool for fetching the content of a specific snippet.
// The tool accepts a snippet ID as a parameter.
func (s *SnippetsTools) GetSnippetContent() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(s.getSnippetContent),
		Tool: mcpargs.NewTool("get_snippet_content", getSnippetContentArgs{},
			mcp.WithDescription("Get the raw content of a snippet"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (s *SnippetsTools) getSnippetContent(ctx context.Context, client *gitlab.Client, args getSnippetContentArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, _, err := client.Snippets.SnippetContent(args.SnippetID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("SnippetContent(%d): %w", args.SnippetID, err)
	}

	return newToolResultJSON(map[string]string{"content": string(content)})
}

type createSnippetArgs struct {
	Title       string `mcp_desc:"The title of the snippet" mcp_required:"true"`
	FileName    string `mcp_desc:"The name of the snippet file" mcp_required:"true"`
	Content     string `mcp_desc:"The content of the snippet" mcp_required:"true"`
	Visibility  string `mcp_desc:"The visibility level of the snippet. Default to private." mcp_enum:"private,internal,public"`
	Description string `mcp_desc:"The description of the snippet"`
}

// CreateSnippet returns a ServerTool for creating a new snippet.
// The tool accepts title, file_name, content, visibility, and description parameters.
func (s *SnippetsTools) CreateSnippet() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(s.createSnippet),
		Tool: mcpargs.NewTool("create_snippet", createSnippetArgs{},
			mcp.WithDescription("Creates a new snippet with a single file. "+
				"Creating snippets with multiple files is a multi-step process: "+
				"create a snippet with one file using 'create_snippet', "+
				"then add additional files using 'update_snippet' "+
				"with a different file name and 'file_action' set to 'create'. "+
				"When creating a snippet, include its 'web_url' in your response."),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (s *SnippetsTools) createSnippet(ctx context.Context, client *gitlab.Client, args createSnippetArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Set default visibility if not provided
	visibility := args.Visibility
	if visibility == "" {
		visibility = snippetVisibilityPrivate
	}

	var visibilityLevel gitlab.VisibilityValue

	switch visibility {
	case snippetVisibilityPrivate:
		visibilityLevel = gitlab.PrivateVisibility
	case snippetVisibilityInternal:
		visibilityLevel = gitlab.InternalVisibility
	case snippetVisibilityPublic:
		visibilityLevel = gitlab.PublicVisibility
	default:
		return nil, fmt.Errorf("%w: invalid visibility level: %s", ErrInvalidArgument, visibility)
	}

	opt := &gitlab.CreateSnippetOptions{
		Title:      new(args.Title),
		Visibility: &visibilityLevel,
		Files: new([]*gitlab.CreateSnippetFileOptions{
			{
				FilePath: new(args.FileName),
				Content:  new(args.Content),
			},
		}),
	}

	// Add description if provided
	if args.Description != "" {
		opt.Description = new(args.Description)
	}

	snippet, _, err := client.Snippets.CreateSnippet(opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CreateSnippet: %w", err)
	}

	return newToolResultJSON(snippet)
}

type updateSnippetArgs struct {
	SnippetID   int64  `mcp_desc:"The ID of the snippet to update" mcp_required:"true"`
	Title       string `mcp_desc:"The title of the snippet"`
	Description string `mcp_desc:"The description of the snippet"`
	Visibility  string `mcp_desc:"The visibility level of the snippet" mcp_enum:"private,internal,public"`
	FileAction  string `mcp_desc:"The action to perform on the file. Default is 'update'" mcp_enum:"create,update,delete"`
	FileName    string `mcp_desc:"The name of the snippet file" mcp_required:"true"`
	Content     string `mcp_desc:"The content of the snippet" mcp_required:"true"`
}

// UpdateSnippet returns a ServerTool for updating an existing snippet.
// The tool accepts snippet_id and optional title, file_name, content, visibility, and description parameters.
func (s *SnippetsTools) UpdateSnippet() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(s.updateSnippet),
		Tool: mcpargs.NewTool("update_snippet", updateSnippetArgs{},
			mcp.WithDescription("Update an existing snippet. This tool can create, update, or delete a file in the snippet, as well as update snippet metadata."),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (s *SnippetsTools) updateSnippet(ctx context.Context, client *gitlab.Client, args updateSnippetArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.UpdateSnippetOptions{}

	var fileActionValue gitlab.FileActionValue

	switch args.FileAction {
	case "create":
		fileActionValue = gitlab.FileCreate
	case "update", "":
		fileActionValue = gitlab.FileUpdate
	case "delete":
		fileActionValue = gitlab.FileDelete
	default:
		return nil, fmt.Errorf("%w: invalid file action: %q", ErrInvalidArgument, args.FileAction)
	}

	opt.Files = new([]*gitlab.UpdateSnippetFileOptions{{
		Action:   new(string(fileActionValue)),
		FilePath: new(args.FileName),
		Content:  new(args.Content),
	}})

	// Add optional parameters if provided
	if args.Title != "" {
		opt.Title = new(args.Title)
	}

	if args.Description != "" {
		opt.Description = new(args.Description)
	}

	if args.Visibility != "" {
		var visibilityLevel gitlab.VisibilityValue

		switch args.Visibility {
		case snippetVisibilityPrivate:
			visibilityLevel = gitlab.PrivateVisibility
		case snippetVisibilityInternal:
			visibilityLevel = gitlab.InternalVisibility
		case snippetVisibilityPublic:
			visibilityLevel = gitlab.PublicVisibility
		default:
			return nil, fmt.Errorf("%w: invalid visibility level: %q", ErrInvalidArgument, args.Visibility)
		}

		opt.Visibility = new(visibilityLevel)
	}

	snippet, _, err := client.Snippets.UpdateSnippet(args.SnippetID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("UpdateSnippet(%d): %w", args.SnippetID, err)
	}

	return newToolResultJSON(snippet)
}

type deleteSnippetArgs struct {
	SnippetID int64 `mcp_desc:"The ID of the snippet to delete" mcp_required:"true"`
}

// DeleteSnippet returns a ServerTool for deleting a snippet.
// The tool accepts a snippet ID as a parameter.
func (s *SnippetsTools) DeleteSnippet() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(s.deleteSnippet),
		Tool: mcpargs.NewTool("delete_snippet", deleteSnippetArgs{},
			mcp.WithDescription("Delete a snippet"),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (s *SnippetsTools) deleteSnippet(ctx context.Context, client *gitlab.Client, args deleteSnippetArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, err := client.Snippets.DeleteSnippet(args.SnippetID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("DeleteSnippet(%d): %w", args.SnippetID, err)
	}

	return mcp.NewToolResultText("success"), nil
}
