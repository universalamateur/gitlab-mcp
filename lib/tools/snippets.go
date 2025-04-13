package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// SnippetsServiceInterface defines the interface for snippet-related GitLab operations.
// It provides methods for retrieving, listing, creating, updating, and deleting snippets.
type SnippetsServiceInterface interface {
	// AddTo registers all snippet-related tools with the provided MCPServer.
	AddTo(srv *server.MCPServer)

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

// NewSnippetsTools creates a new instance of SnippetsServiceInterface with the provided GitLab client
// and current user. It returns an implementation that can be used to interact with GitLab's snippet API.
func NewSnippetsTools(client *gitlab.Client, currentUser string) SnippetsServiceInterface {
	return &snippetsService{
		client:      client,
		currentUser: currentUser,
	}
}

type snippetsService struct {
	client      *gitlab.Client
	currentUser string
}

// AddTo registers all snippet-related tools with the provided MCPServer.
// It adds tools for listing, retrieving, creating, updating, and deleting snippets.
func (s *snippetsService) AddTo(srv *server.MCPServer) {
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
func (s *snippetsService) ListUserSnippets() server.ServerTool {
	return server.ServerTool{
		Handler: s.listUserSnippets,
		Tool: mcp.NewTool("list_user_snippets",
			mcp.WithDescription("List snippets owned by the current user"),
		),
	}
}

func (s *snippetsService) listUserSnippets(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		opt = &gitlab.ListSnippetsOptions{
			PerPage: maxPerPage,
		}
		snippets []*gitlab.Snippet
	)

	for {
		snips, resp, err := s.client.Snippets.ListSnippets(opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListSnippets: %w", err)
		}

		snippets = append(snippets, snips...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(snippets)
}

// ListAllSnippets returns a ServerTool for listing all snippets the user has access to.
// The include_private parameter determines whether to include private snippets or only public ones.
func (s *snippetsService) ListAllSnippets() server.ServerTool {
	return server.ServerTool{
		Handler: s.listAllSnippets,
		Tool: mcp.NewTool("list_all_snippets",
			mcp.WithDescription("List all snippets the user has access to"),
			mcp.WithBoolean("include_private",
				mcp.Description("Include private snippets in the results. If false (the default), only public snippets are returned")),
		),
	}
}

func (s *snippetsService) listAllSnippets(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		snippets []*gitlab.Snippet
	)

	// Check if we should include private snippets
	includePrivate, _ := request.Params.Arguments["include_private"].(bool)

	if includePrivate {
		// List all snippets (private and public)
		opt := &gitlab.ListAllSnippetsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}

		for {
			snips, resp, err := s.client.Snippets.ListAllSnippets(opt, gitlab.WithContext(ctx))
			if err != nil {
				return nil, fmt.Errorf("ListAllSnippets: %w", err)
			}

			snippets = append(snippets, snips...)

			if resp.NextPage == 0 {
				break
			}

			opt.Page = resp.NextPage
		}
	} else {
		// List only public snippets
		opt := &gitlab.ExploreSnippetsOptions{
			PerPage: maxPerPage,
		}

		for {
			snips, resp, err := s.client.Snippets.ExploreSnippets(opt, gitlab.WithContext(ctx))
			if err != nil {
				return nil, fmt.Errorf("ExploreSnippets: %w", err)
			}

			snippets = append(snippets, snips...)

			if resp.NextPage == 0 {
				break
			}

			opt.Page = resp.NextPage
		}
	}

	return newToolResultJSON(snippets)
}

// GetSnippet returns a ServerTool for fetching information about a specific snippet.
// The tool accepts a snippet ID as a parameter.
func (s *snippetsService) GetSnippet() server.ServerTool {
	return server.ServerTool{
		Handler: s.getSnippet,
		Tool: mcp.NewTool("get_snippet",
			mcp.WithDescription("Returns the metadata of a snippet, such as title and description. File content is not returned."),
			mcp.WithNumber("snippet_id",
				mcp.Description("The ID of the snippet"),
				mcp.Required()),
		),
	}
}

func (s *snippetsService) getSnippet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	snippetID, err := intParam(request.Params.Arguments["snippet_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid snippet ID: %w", err)
	}

	snippet, _, err := s.client.Snippets.GetSnippet(int(snippetID), gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetSnippet(%d): %w", int(snippetID), err)
	}

	return newToolResultJSON(snippet)
}

// GetSnippetContent returns a ServerTool for fetching the content of a specific snippet.
// The tool accepts a snippet ID as a parameter.
func (s *snippetsService) GetSnippetContent() server.ServerTool {
	return server.ServerTool{
		Handler: s.getSnippetContent,
		Tool: mcp.NewTool("get_snippet_content",
			mcp.WithDescription("Get the raw content of a snippet"),
			mcp.WithNumber("snippet_id",
				mcp.Description("The ID of the snippet"),
				mcp.Required()),
		),
	}
}

func (s *snippetsService) getSnippetContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	snippetID, err := intParam(request.Params.Arguments["snippet_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid snippet ID: %w", err)
	}

	content, _, err := s.client.Snippets.SnippetContent(snippetID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("SnippetContent(%d): %w", snippetID, err)
	}

	return newToolResultJSON(map[string]string{"content": string(content)})
}

// CreateSnippet returns a ServerTool for creating a new snippet.
// The tool accepts title, file_name, content, visibility, and description parameters.
func (s *snippetsService) CreateSnippet() server.ServerTool {
	return server.ServerTool{
		Handler: s.createSnippet,
		Tool: mcp.NewTool("create_snippet",
			mcp.WithDescription("Create a new snippet"),
			mcp.WithString("title",
				mcp.Description("The title of the snippet"),
				mcp.Required()),
			mcp.WithString("file_name",
				mcp.Description("The name of the snippet file"),
				mcp.Required()),
			mcp.WithString("content",
				mcp.Description("The content of the snippet"),
				mcp.Required()),
			mcp.WithString("visibility",
				mcp.Description("The visibility level of the snippet. Default to private."),
				mcp.Enum("private", "internal", "public")),
			mcp.WithString("description",
				mcp.Description("The description of the snippet")),
		),
	}
}

func (s *snippetsService) createSnippet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Required arguments
	title, ok := request.Params.Arguments["title"].(string)
	if !ok || title == "" {
		return nil, errors.New(`the "title" argument is required`)
	}

	fileName, ok := request.Params.Arguments["file_name"].(string)
	if !ok || fileName == "" {
		return nil, errors.New(`the "file_name" argument is required`)
	}

	content, ok := request.Params.Arguments["content"].(string)
	if !ok || content == "" {
		return nil, errors.New(`the "content" argument is required`)
	}

	// Optional arguments
	visibility, ok := request.Params.Arguments["visibility"].(string)
	if !ok || visibility == "" {
		visibility = "private"
	}

	var visibilityLevel gitlab.VisibilityValue
	switch visibility {
	case "private":
		visibilityLevel = gitlab.PrivateVisibility
	case "internal":
		visibilityLevel = gitlab.InternalVisibility
	case "public":
		visibilityLevel = gitlab.PublicVisibility
	default:
		return nil, fmt.Errorf("invalid visibility level: %s", visibility)
	}

	opt := &gitlab.CreateSnippetOptions{
		Title:      gitlab.Ptr(title),
		Visibility: &visibilityLevel,
		Files: gitlab.Ptr([]*gitlab.CreateSnippetFileOptions{
			{
				FilePath: gitlab.Ptr(fileName),
				Content:  gitlab.Ptr(content),
			},
		}),
	}

	// Add description if provided
	if description, ok := request.Params.Arguments["description"].(string); ok && description != "" {
		opt.Description = gitlab.Ptr(description)
	}

	snippet, _, err := s.client.Snippets.CreateSnippet(opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CreateSnippet: %w", err)
	}

	return newToolResultJSON(snippet)
}

// UpdateSnippet returns a ServerTool for updating an existing snippet.
// The tool accepts snippet_id and optional title, file_name, content, visibility, and description parameters.
func (s *snippetsService) UpdateSnippet() server.ServerTool {
	return server.ServerTool{
		Handler: s.updateSnippet,
		Tool: mcp.NewTool("update_snippet",
			mcp.WithDescription("Update an existing snippet"),
			mcp.WithNumber("snippet_id",
				mcp.Description("The ID of the snippet to update"),
				mcp.Required()),
			mcp.WithString("title",
				mcp.Description("The title of the snippet")),
			mcp.WithString("description",
				mcp.Description("The description of the snippet")),
			mcp.WithString("visibility",
				mcp.Description("The visibility level of the snippet"),
				mcp.Enum("private", "internal", "public")),
			mcp.WithString("file_action",
				mcp.Description("The action to perform on the file. Default is 'replace'"),
				mcp.Enum("create", "update", "delete", "move"),
				mcp.Required()),
			mcp.WithString("file_name",
				mcp.Description("The name of the snippet file"),
				mcp.Required()),
			mcp.WithString("content",
				mcp.Description("The content of the snippet"),
				mcp.Required()),
		),
	}
}

func (s *snippetsService) updateSnippet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	snippetID, err := intParam(request.Params.Arguments["snippet_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid snippet ID: %w", err)
	}

	opt := &gitlab.UpdateSnippetOptions{}

	// Required parameters
	fileAction, ok := request.Params.Arguments["file_action"].(string)
	if !ok || fileAction == "" {
		return nil, errors.New(`the "file_action" argument is required`)
	}

	var fileActionValue gitlab.FileActionValue
	switch fileAction {
	case "create":
		fileActionValue = gitlab.FileCreate
	case "update":
		fileActionValue = gitlab.FileUpdate
	case "delete":
		fileActionValue = gitlab.FileDelete
	case "move":
		fileActionValue = gitlab.FileMove
	default:
		return nil, fmt.Errorf("invalid file action: %s", fileAction)
	}

	fileName, ok := request.Params.Arguments["file_name"].(string)
	if !ok || fileName == "" {
		return nil, errors.New(`the "file_name" argument is required`)
	}

	content, ok := request.Params.Arguments["content"].(string)
	if !ok || content == "" {
		return nil, errors.New(`the "content" argument is required`)
	}

	opt.Files = gitlab.Ptr([]*gitlab.UpdateSnippetFileOptions{{
		Action:   gitlab.Ptr(string(fileActionValue)),
		FilePath: gitlab.Ptr(fileName),
		Content:  gitlab.Ptr(content),
	}})

	// Add optional parameters if provided
	if title, ok := request.Params.Arguments["title"].(string); ok && title != "" {
		opt.Title = gitlab.Ptr(title)
	}

	if description, ok := request.Params.Arguments["description"].(string); ok && description != "" {
		opt.Description = gitlab.Ptr(description)
	}

	if visibility, ok := request.Params.Arguments["visibility"].(string); ok && visibility != "" {
		var visibilityLevel gitlab.VisibilityValue
		switch visibility {
		case "private":
			visibilityLevel = gitlab.PrivateVisibility
		case "internal":
			visibilityLevel = gitlab.InternalVisibility
		case "public":
			visibilityLevel = gitlab.PublicVisibility
		default:
			return nil, fmt.Errorf("invalid visibility level: %s", visibility)
		}
		opt.Visibility = &visibilityLevel
	}

	snippet, _, err := s.client.Snippets.UpdateSnippet(int(snippetID), opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("UpdateSnippet(%d): %w", int(snippetID), err)
	}

	return newToolResultJSON(snippet)
}

// DeleteSnippet returns a ServerTool for deleting a snippet.
// The tool accepts a snippet ID as a parameter.
func (s *snippetsService) DeleteSnippet() server.ServerTool {
	return server.ServerTool{
		Handler: s.deleteSnippet,
		Tool: mcp.NewTool("delete_snippet",
			mcp.WithDescription("Delete a snippet"),
			mcp.WithNumber("snippet_id",
				mcp.Description("The ID of the snippet to delete"),
				mcp.Required()),
		),
	}
}

func (s *snippetsService) deleteSnippet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	snippetID, err := intParam(request.Params.Arguments["snippet_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid snippet ID: %w", err)
	}

	_, err = s.client.Snippets.DeleteSnippet(int(snippetID), gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("DeleteSnippet(%d): %w", int(snippetID), err)
	}

	return newToolResultJSON(map[string]string{"result": "success"})
}
