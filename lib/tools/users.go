package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// UsersServiceInterface defines the interface for user-related GitLab operations.
// It provides methods for retrieving user information, status, activities, and memberships.
type UsersServiceInterface interface {
	// AddTo registers all user-related tools with the provided MCPServer.
	AddTo(srv *server.MCPServer)

	// GetUser returns a tool for fetching information about a specific user or the current user.
	GetUser() server.ServerTool

	// GetUserStatus returns a tool for fetching a user's status.
	GetUserStatus() server.ServerTool

	// SetUserStatus returns a tool for setting a user's status.
	SetUserStatus() server.ServerTool
}

// NewUsersTools creates a new instance of UsersServiceInterface with the provided GitLab client
// and current user. It returns an implementation that can be used to interact with GitLab's user API.
func NewUsersTools(client *gitlab.Client, currentUser string) UsersServiceInterface {
	return &usersService{
		client:      client,
		currentUser: currentUser,
	}
}

type usersService struct {
	client      *gitlab.Client
	currentUser string
}

// AddTo registers all user-related tools with the provided MCPServer.
// It adds tools for retrieving user information, status, activities, and memberships.
func (u *usersService) AddTo(srv *server.MCPServer) {
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
func (u *usersService) GetUser() server.ServerTool {
	return server.ServerTool{
		Handler: u.getUser,
		Tool: mcpargs.NewTool("get_user", getUserArgs{},
			mcp.WithDescription("Get information about a specific user or the current user. In particular, this tool can be used to resolve a username to an ID."),
		),
	}
}

func (u *usersService) getUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args getUserArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	// Get current user
	if args.UserID.IsZero() {
		user, _, err := u.client.Users.CurrentUser(gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("CurrentUser: %w", err)
		}

		return newToolResultJSON(user)
	}

	switch {
	case args.UserID.Integer != 0:
		user, _, err := u.client.Users.GetUser(args.UserID.Integer, gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("GetUser(%d): %w", args.UserID.Integer, err)
		}

		return newToolResultJSON(user)

	case args.UserID.String != "":
		id := strings.TrimPrefix(args.UserID.String, "@")
		opts := gitlab.ListUsersOptions{
			Username: gitlab.Ptr(id),
		}

		users, _, err := u.client.Users.ListUsers(&opts, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListUsers(username=%q): %w", id, err)
		}

		if len(users) == 0 {
			return nil, fmt.Errorf("user not found: %q", id)
		}

		return newToolResultJSON(users[0])

	default:
		return nil, fmt.Errorf("unexpected user ID: %#v", args.UserID)
	}
}

type getUserStatusArgs struct {
	UserID int `mcp_desc:"ID of the user to get a status of" mcp_required:"true"`
}

// GetUserStatus returns a ServerTool for fetching a user's status.
func (u *usersService) GetUserStatus() server.ServerTool {
	return server.ServerTool{
		Handler: u.getUserStatus,
		Tool: mcpargs.NewTool("get_user_status", getUserStatusArgs{},
			mcp.WithDescription("Get a user's status"),
		),
	}
}

func (u *usersService) getUserStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args getUserStatusArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	status, _, err := u.client.Users.GetUserStatus(args.UserID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetUserStatus(%d): %w", args.UserID, err)
	}

	return newToolResultJSON(status)
}

type setUserStatusArgs struct {
	Emoji        string `mcp_desc:"Name of the emoji to use as status. If omitted 'speech_balloon' is used. Emoji name can be one of the specified names in the Gemojione index."`
	Message      string `mcp_desc:"Message to set as a status. It can also contain emoji codes. Cannot exceed 100 characters."`
	Availability string `mcp_desc:"The availability of the user: either 'busy' or 'not_set' if the user is available." mcp_enum:"busy,not_set"`
}

// SetUserStatus returns a ServerTool for setting a user's status.
func (u *usersService) SetUserStatus() server.ServerTool {
	return server.ServerTool{
		Handler: u.setUserStatus,
		Tool: mcpargs.NewTool("set_user_status", setUserStatusArgs{},
			mcp.WithDescription("Set the current user's status"),
		),
	}
}

func (u *usersService) setUserStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args setUserStatusArgs
	if err := mcpargs.Unmarshal(request.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	opt := &gitlab.UserStatusOptions{}

	// Set optional parameters
	if args.Emoji != "" {
		opt.Emoji = gitlab.Ptr(args.Emoji)
	}

	if args.Message != "" {
		opt.Message = gitlab.Ptr(args.Message)
	}

	if args.Availability != "" {
		opt.Availability = gitlab.Ptr(gitlab.AvailabilityValue(args.Availability))
	}

	status, _, err := u.client.Users.SetUserStatus(opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("SetUserStatus: %w", err)
	}

	return newToolResultJSON(status)
}
