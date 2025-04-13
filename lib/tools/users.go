package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

// GetUser returns a ServerTool for fetching information about a specific user or the current user.
// If no user ID is provided, it returns information about the authenticated user.
func (u *usersService) GetUser() server.ServerTool {
	return server.ServerTool{
		Handler: u.getUser,
		Tool: mcp.NewTool("get_user",
			mcp.WithDescription("Get information about a specific user or the current user. In particular, this tool can be used to resolve a username to an ID."),
			mcp.WithString("user_id",
				mcp.Description("The ID or username of the user. If not provided, returns information about the authenticated user")),
		),
	}
}

func (u *usersService) getUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get current user
	if userID, ok := request.Params.Arguments["user_id"]; !ok || userID == nil {
		user, _, err := u.client.Users.CurrentUser(gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("CurrentUser: %w", err)
		}

		return newToolResultJSON(user)
	}

	// Get specific user
	id, err := idParam(request.Params.Arguments["user_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	switch id := id.(type) {
	case int:
		user, _, err := u.client.Users.GetUser(id, gitlab.GetUsersOptions{}, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("GetUser(%d): %w", id, err)
		}

		return newToolResultJSON(user)

	case string:
		id = strings.TrimPrefix(id, "@")
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
		return nil, fmt.Errorf("unexpected user ID type: %T", id)
	}
}

// GetUserStatus returns a ServerTool for fetching a user's status.
func (u *usersService) GetUserStatus() server.ServerTool {
	return server.ServerTool{
		Handler: u.getUserStatus,
		Tool: mcp.NewTool("get_user_status",
			mcp.WithDescription("Get a user's status"),
			mcp.WithNumber("user_id",
				mcp.Description("ID of the user to get a status of"),
				mcp.Required()),
		),
	}
}

func (u *usersService) getUserStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, err := intParam(request.Params.Arguments["user_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	status, _, err := u.client.Users.GetUserStatus(userID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetUserStatus(%d): %w", userID, err)
	}

	return newToolResultJSON(status)
}

// SetUserStatus returns a ServerTool for setting a user's status.
func (u *usersService) SetUserStatus() server.ServerTool {
	return server.ServerTool{
		Handler: u.setUserStatus,
		Tool: mcp.NewTool("set_user_status",
			mcp.WithDescription("Set the current user's status"),
			mcp.WithString("emoji",
				mcp.Description("Name of the emoji to use as status. If omitted `speech_balloon` is used. Emoji name can be one of the specified names in the Gemojione index.")),
			mcp.WithString("message",
				mcp.Description("Message to set as a status. It can also contain emoji codes. Cannot exceed 100 characters.")),
			mcp.WithString("availability",
				mcp.Description("The availability of the user: either `busy` or `not_set` if the user is available."),
				mcp.Enum("busy", "not_set")),
		),
	}
}

func (u *usersService) setUserStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.UserStatusOptions{}

	// Parse optional parameters
	if v, ok := request.Params.Arguments["emoji"].(string); ok && v != "" {
		opt.Emoji = gitlab.Ptr(v)
	}

	if v, ok := request.Params.Arguments["message"].(string); ok && v != "" {
		opt.Message = gitlab.Ptr(v)
	}

	if v, ok := request.Params.Arguments["availability"].(string); ok && v != "" {
		opt.Availability = gitlab.Ptr(gitlab.AvailabilityValue(v))
	}

	status, _, err := u.client.Users.SetUserStatus(opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("SetUserStatus: %w", err)
	}

	return newToolResultJSON(status)
}
