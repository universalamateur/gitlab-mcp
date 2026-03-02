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
	AddTo(srv ToolAdder)

	// ListUserTodos returns a tool for listing all todos for the current user.
	ListUserTodos() server.ServerTool
}

// TodosTools provides tools to list and manage todos.
type TodosTools struct{}

// AddTo registers all todo-related tools with the provided MCPServer.
func (t *TodosTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		t.ListUserTodos(),
		t.CompleteTodoItem(),
		t.CompleteAllTodoItems(),
	)
}

type listUserTodosArgs struct {
	Action    string `mcp_desc:"Filter by the action that caused the todo item" mcp_enum:"assigned,mentioned,build_failed,marked,approval_required,unmergeable,directly_addressed,merge_train_removed,member_access_requested"`
	AuthorID  int64  `mcp_desc:"Filter by the ID of the author who created the todo item"`
	ProjectID int64  `mcp_desc:"Filter by the ID of the project the todo item belongs to"`
	GroupID   int64  `mcp_desc:"Filter by the ID of the group the todo item belongs to"`
	State     string `mcp_desc:"Filter by the state of the todo item, defaults to 'pending'" mcp_enum:"pending,done"`
	Type      string `mcp_desc:"Filter by the type of resource the todo item is associated with" mcp_enum:"Issue,MergeRequest,Commit,Epic,DesignManagement::Design,AlertManagement::Alert,Project,Namespace,Vulnerability,WikiPage::Meta"`
	Limit     int    `mcp_desc:"Maximum number of todos to return. If not set or zero, defaults to 100."`
}

// ListUserTodos returns a ServerTool for listing all todos for the current user.
func (t *TodosTools) ListUserTodos() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(t.listUserTodos),
		Tool: mcpargs.NewTool("list_user_todos", listUserTodosArgs{},
			mcp.WithDescription("Get all todos for the current user, with optional filtering."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (t *TodosTools) listUserTodos(ctx context.Context, client *gitlab.Client, args listUserTodosArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if args.Limit <= 0 {
		args.Limit = defaultTodosLimit
	}

	opt := gitlab.ListTodosOptions{
		ListOptions: newListOptions(request),
	}

	// Parse optional parameters
	opt.Action = todoActions[args.Action]
	opt.State = todoStates[args.State]

	if args.AuthorID != 0 {
		opt.AuthorID = new(args.AuthorID)
	}

	if args.ProjectID != 0 {
		opt.ProjectID = new(args.ProjectID)
	}

	if args.GroupID != 0 {
		opt.GroupID = new(args.GroupID)
	}

	if args.Type != "" {
		opt.Type = new(args.Type)
	}

	pager := func(p gitlab.PaginationOptionFunc) ([]*gitlab.Todo, *gitlab.Response, error) {
		return client.Todos.ListTodos(&opt, gitlab.WithContext(ctx), p)
	}

	todos, err := gitlab.ScanAndCollectN(pager, args.Limit)
	if err != nil {
		return nil, fmt.Errorf("ListTodos(): %w", err)
	}

	return newToolResultJSON(todos)
}

var todoStates = map[string]*string{
	"pending": new("pending"),
	"done":    new("done"),
	"":        new("pending"), // Default value
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

type completeTodoItemArgs struct {
	ID int64 `mcp_desc:"The ID of the todo item to mark as done" mcp_required:"true"`
}

// CompleteTodoItem returns a ServerTool for marking a single todo item as done.
// The tool accepts the ID of the todo item to be marked as done.
func (t *TodosTools) CompleteTodoItem() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(t.completeTodoItem),
		Tool: mcpargs.NewTool("complete_todo_item", completeTodoItemArgs{},
			mcp.WithDescription("Marks a single pending todo item as done"),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (t *TodosTools) completeTodoItem(ctx context.Context, client *gitlab.Client, args completeTodoItemArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, err := client.Todos.MarkTodoAsDone(args.ID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("MarkTodoAsDone(%d): %w", args.ID, err)
	}

	return mcp.NewToolResultText("success"), nil
}

// CompleteAllTodoItems returns a ServerTool for marking all pending todo items as done.
// This tool takes no parameters as it affects all pending todo items.
func (t *TodosTools) CompleteAllTodoItems() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(t.completeAllTodoItems),
		Tool: mcp.NewTool("complete_all_todo_items",
			mcp.WithDescription("Marks all pending todo items for the current user as done. Only perform this action when explicitly requested by the user."),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (t *TodosTools) completeAllTodoItems(ctx context.Context, client *gitlab.Client, _ struct{}, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, err := client.Todos.MarkAllTodosAsDone(gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("MarkAllTodosAsDone: %w", err)
	}

	return mcp.NewToolResultText("success"), nil
}
