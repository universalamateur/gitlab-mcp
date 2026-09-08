package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go/v3"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// UsersServiceInterface defines the interface for user-related GitLab operations.
// It provides methods for retrieving user information, status, activities, and memberships.
type UsersServiceInterface interface {
	// AddTo registers all user-related tools with the provided MCPServer.
	AddTo(srv ToolAdder)

	// GetUser returns a tool for fetching information about a specific user or the current user.
	GetUser() server.ServerTool

	// GetUserStatus returns a tool for fetching a user's status.
	GetUserStatus() server.ServerTool

	// SetUserStatus returns a tool for setting a user's status.
	SetUserStatus() server.ServerTool
}

// UsersTools implements tools for user-related GitLab operations.
type UsersTools struct{}

// AddTo registers all user-related tools with the provided MCPServer.
// It adds tools for retrieving user information, status, activities, and memberships.
func (u *UsersTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		u.GetUser(),
		u.GetUserStatus(),
		u.SetUserStatus(),
	)
}

type getUserArgs struct {
	UserID mcpargs.ID `mcp_desc:"The ID or username of the user. If not provided, returns information about the authenticated user"`
}

// GetUser returns a ServerTool for fetching information about a specific user or the current user.
// If no user ID is provided, it returns information about the authenticated user.
func (u *UsersTools) GetUser() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(u.getUser),
		Tool: mcpargs.NewTool("get_user", getUserArgs{},
			mcp.WithDescription("Get information about a specific user or the current user. In particular, this tool can be used to resolve a username to an ID."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (u *UsersTools) getUser(ctx context.Context, client *gitlab.Client, args getUserArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get current user
	if args.UserID.IsZero() {
		user, _, err := client.Users.CurrentUser(gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("CurrentUser: %w", err)
		}

		return newToolResultJSON(user)
	}

	switch {
	case args.UserID.Integer != 0:
		user, _, err := client.Users.GetUser(args.UserID.Integer, nil, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("GetUser(%d): %w", args.UserID.Integer, err)
		}

		return newToolResultJSON(user)

	case args.UserID.String != "":
		id := strings.TrimPrefix(args.UserID.String, "@")
		opts := gitlab.ListUsersOptions{
			Username: new(id),
		}

		users, _, err := client.Users.ListUsers(&opts, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListUsers(username=%q): %w", id, err)
		}

		if len(users) == 0 {
			return mcp.NewToolResultError("user not found"), nil
		}

		return newToolResultJSON(users[0])

	default:
		return nil, fmt.Errorf("%w: user ID: %#v", ErrInvalidArgument, args.UserID)
	}
}

type getUserStatusArgs struct {
	UserID mcpargs.ID `mcp_desc:"ID or username of the user to get status for" mcp_required:"true"`
}

// GetUserStatus returns a ServerTool for fetching a user's status.
func (u *UsersTools) GetUserStatus() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(u.getUserStatus),
		Tool: mcpargs.NewTool("get_user_status", getUserStatusArgs{},
			mcp.WithDescription("Get a user's status"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (u *UsersTools) getUserStatus(ctx context.Context, client *gitlab.Client, args getUserStatusArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID := args.UserID.Value()
	if s, ok := userID.(string); ok {
		userID = strings.TrimPrefix(s, "@")
	}

	status, _, err := client.Users.GetUserStatus(userID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetUserStatus(%v): %w", userID, err)
	}

	return newToolResultJSON(status)
}

type setUserStatusArgs struct {
	Emoji        string `mcp_desc:"Name of the emoji to use as status. If omitted 'speech_balloon' is used. Emoji name can be one of the specified names in the Gemojione index."`
	Message      string `mcp_desc:"Message to set as a status. It can also contain emoji codes. Cannot exceed 100 characters."`
	Availability string `mcp_desc:"The availability of the user: either 'busy' or 'not_set' if the user is available." mcp_enum:"busy,not_set"`
}

// SetUserStatus returns a ServerTool for setting a user's status.
func (u *UsersTools) SetUserStatus() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(u.setUserStatus),
		Tool: mcpargs.NewTool("set_user_status", setUserStatusArgs{},
			mcp.WithDescription("Set the current user's status"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (u *UsersTools) setUserStatus(ctx context.Context, client *gitlab.Client, args setUserStatusArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.UserStatusOptions{}

	// Set optional parameters
	if args.Emoji != "" {
		opt.Emoji = new(args.Emoji)
	}

	if args.Message != "" {
		opt.Message = new(args.Message)
	}

	if args.Availability != "" {
		opt.Availability = new(gitlab.AvailabilityValue(args.Availability))
	}

	status, _, err := client.Users.SetUserStatus(opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("SetUserStatus: %w", err)
	}

	return newToolResultJSON(status)
}
