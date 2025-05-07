package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

const defaultTodosLimit = 100

// TodosServiceInterface defines the interface for todo-related GitLab operations.
type TodosServiceInterface interface {
	// AddTo registers all todo-related tools with the provided MCPServer.
	AddTo(srv *server.MCPServer)

	// ListUserTodos returns a tool for listing all todos for the current user.
	ListUserTodos() server.ServerTool
}

// NewTodosTools creates a new instance of TodosServiceInterface with the provided GitLab client.
func NewTodosTools(client *gitlab.Client) *TodosService {
	return &TodosService{client: client}
}

type TodosService struct {
	client *gitlab.Client
}

// AddTo registers all todo-related tools with the provided MCPServer.
func (t *TodosService) AddTo(srv *server.MCPServer) {
	srv.AddTools(
		t.ListUserTodos(),
	)
}

type listUserTodosArgs struct {
	State string `mcp_desc:"Return all todos that are pending or done" mcp_enum:"pending,done"`
	Limit int    `mcp_desc:"Maximum number of todos to return. If not set or zero, defaults to 100."`
}

// ListUserTodos returns a ServerTool for listing all todos for the current user.
func (t *TodosService) ListUserTodos() server.ServerTool {
	return server.ServerTool{
		Handler: t.listUserTodos,
		Tool: mcpargs.NewTool("list_user_todos", listUserTodosArgs{},
			mcp.WithDescription("Get all todos for the current user, with optional filtering by state (pending, done) and a limit on the number returned (default 100)."),
		),
	}
}

func (t *TodosService) listUserTodos(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args listUserTodosArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, err //nolint:wrapcheck
	}

	// Only allow 'pending' or 'done', default to 'pending' if not set or invalid
	state := args.State
	if state != "pending" && state != "done" {
		state = "pending"
	}

	limit := args.Limit
	if limit <= 0 {
		limit = defaultTodosLimit
	}

	opt := &gitlab.ListTodosOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPerPage,
		},
		State: gitlab.Ptr(state),
	}

	var todos []*gitlab.Todo

	for {
		ts, resp, err := t.client.Todos.ListTodos(opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListTodos: %w", err)
		}

		todos = append(todos, ts...)
		if limit > 0 && len(todos) >= limit {
			todos = todos[:limit] // Ensure we don't exceed the limit
			break
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(todos)
}
