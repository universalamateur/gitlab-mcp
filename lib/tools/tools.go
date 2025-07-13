// Package tools provides a set of GitLab API tools for interacting with GitLab resources
// such as epics, issues, and merge requests. This package wraps the GitLab client-go library
// and exposes its functionality through a structured MCP (Machine Conversation Protocol) interface.
//
// The tools package enables applications to query and manipulate GitLab resources
// through a unified API. It handles pagination automatically and provides
// convenient access to GitLab's epics, issues, and merge requests endpoints.
// All results are returned as JSON for easy integration with other systems.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/glctx"
	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

const maxPerPage = 100

// ErrArgumentType is an error indicating that the argument provided is of an invalid type.
var ErrArgumentType = errors.New("invalid argument type")

// ToolAdder defines an interface for adding tools to an MCP server.
type ToolAdder interface {
	AddTools(tools ...server.ServerTool)
}

// AddTo registers all implemented GitLab tools with the provided MCPServer.
func AddTo(srv ToolAdder) {
	new(DiscussionTools).AddTo(srv)
	new(EpicTools).AddTo(srv)
	new(EventsTools).AddTo(srv)
	new(GroupsTools).AddTo(srv)
	new(IssuesTools).AddTo(srv)
	new(JobsTools).AddTo(srv)
	new(MergeRequestsTools).AddTo(srv)
	new(ProjectsTools).AddTo(srv)
	new(RepositoryTools).AddTo(srv)
	new(SnippetsTools).AddTo(srv)
	new(TodosTools).AddTo(srv)
	new(UsersTools).AddTo(srv)
}

// ClientHandler wraps a tool handler function with a GitLab client,
// initializing the client from the provided context before calling the handler.
func ClientHandler[A any](hndl func(ctx context.Context, client *gitlab.Client, args A, req mcp.CallToolRequest) (*mcp.CallToolResult, error)) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := glctx.Client(ctx)
		if err != nil {
			return nil, fmt.Errorf("glctx.Client: %w", err)
		}

		var args A
		if err := mcpargs.Unmarshal(req.GetArguments(), &args); err != nil {
			return nil, fmt.Errorf("mcpargs.Unmarshal: %w", err)
		}

		return hndl(ctx, client, args, req)
	}
}

// ClientUserHandler wraps a tool handler function with a GitLab client,
// initializing the client from the provided context before calling the handler.
func ClientUserHandler[A any](hndl func(ctx context.Context, client *gitlab.Client, args A, currentUser *gitlab.User, req mcp.CallToolRequest) (*mcp.CallToolResult, error)) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := glctx.Client(ctx)
		if err != nil {
			return nil, fmt.Errorf("glctx.Client: %w", err)
		}

		var args A
		if err := mcpargs.Unmarshal(req.GetArguments(), &args); err != nil {
			return nil, fmt.Errorf("mcpargs.Unmarshal: %w", err)
		}

		user, _, err := client.Users.CurrentUser(gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("CurrentUser: %w", err)
		}

		return hndl(ctx, client, args, user, req)
	}
}

// newToolResultJSON encodes the provided value as JSON and returns it as a tool result.
// It handles the JSON encoding and error handling, providing a consistent way to return
// JSON responses from tool handlers.
func newToolResultJSON(v any) (*mcp.CallToolResult, error) {
	// If v is a slice and v is nil, we return "[]" (indicating an empty slice) rather than "null".
	if value := reflect.ValueOf(v); value.Kind() == reflect.Slice && value.IsNil() {
		return mcp.NewToolResultText("[]"), nil
	}

	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(v); err != nil {
		return nil, fmt.Errorf("encoding to JSON: %w", err)
	}

	return mcp.NewToolResultText(b.String()), nil
}

// newListOptions initializes a ListOptions struct populated with default values
// and values from common arguments in the provided MCP CallToolRequest.
func newListOptions(req mcp.CallToolRequest) gitlab.ListOptions {
	opts := gitlab.ListOptions{
		PerPage: maxPerPage,
	}

	opts.OrderBy = req.GetString("order_by", "")
	opts.Sort = req.GetString("sort_order", "")

	return opts
}
