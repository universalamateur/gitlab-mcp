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
	Action    string `mcp_desc:"Filter by the action that caused the todo item" mcp_enum:"assigned,mentioned,build_failed,marked,approval_required,unmergeable,directly_addressed,merge_train_removed,member_access_requested"`
	AuthorID  int    `mcp_desc:"Filter by the ID of the author who created the todo item"`
	ProjectID int    `mcp_desc:"Filter by the ID of the project the todo item belongs to"`
	State     string `mcp_desc:"Filter by the state of the todo item, defaults to 'pending'" mcp_enum:"pending,done"`
	Type      string `mcp_desc:"Filter by the type of resource the todo item is associated with" mcp_enum:"Issue,MergeRequest,Commit,Epic,DesignManagement::Design,AlertManagement::Alert,Project,Namespace,Vulnerability,WikiPage::Meta"`
	Limit     int    `mcp_desc:"Maximum number of todos to return. If not set or zero, defaults to 100."`
}

// ListUserTodos returns a ServerTool for listing all todos for the current user.
func (t *TodosService) ListUserTodos() server.ServerTool {
	return server.ServerTool{
		Handler: t.listUserTodos,
		Tool: mcpargs.NewTool("list_user_todos", listUserTodosArgs{},
			mcp.WithDescription("Get all todos for the current user, with optional filtering."),
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
			PerPage: min(limit, maxPerPage),
		},
	}

	// Parse optional parameters
	opt.Action = todoActions[args.Action]

	if args.AuthorID != 0 {
		opt.AuthorID = gitlab.Ptr(args.AuthorID)
	}

	if args.ProjectID != 0 {
		opt.ProjectID = gitlab.Ptr(args.ProjectID)
	}

	if args.State != "" {
		opt.State = gitlab.Ptr(state)
	}

	if args.Type != "" {
		opt.Type = gitlab.Ptr(args.Type)
	}

	var todos []*gitlab.Todo

	for {
		ts, resp, err := t.client.Todos.ListTodos(opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListTodos: %w", err)
		}

		todos = append(todos, ts...)
		if limit > 0 && len(todos) >= limit {
			todos = todos[:limit]
			break
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return newToolResultJSON(todos)
}

// todoActions is a map of all possible todo actions.
var todoActions = map[string]*gitlab.TodoAction{
	string(gitlab.TodoAssigned):          gitlab.Ptr(gitlab.TodoAssigned),
	string(gitlab.TodoMentioned):         gitlab.Ptr(gitlab.TodoMentioned),
	string(gitlab.TodoBuildFailed):       gitlab.Ptr(gitlab.TodoBuildFailed),
	string(gitlab.TodoMarked):            gitlab.Ptr(gitlab.TodoMarked),
	string(gitlab.TodoApprovalRequired):  gitlab.Ptr(gitlab.TodoApprovalRequired),
	string(gitlab.TodoDirectlyAddressed): gitlab.Ptr(gitlab.TodoDirectlyAddressed),
}
