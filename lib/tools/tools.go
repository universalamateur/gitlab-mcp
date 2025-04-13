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
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const maxPerPage = 100

// Tools is the main entry point for GitLab API tools, providing access to
// GitLab epics, issues, and merge requests functionality.
type Tools struct {
	// Epics provides access to GitLab epic-related operations.
	Epics EpicServiceInterface

	// Issues provides access to GitLab issue-related operations.
	Issues IssuesServiceInterface

	// MergeRequests provides access to GitLab merge request-related operations.
	MergeRequests MergeRequestsServiceInterface

	// Users provides tools for looking up user IDs, a user's activity and status, etc.
	Users UsersServiceInterface
}

// New creates a new instance of Tools with the provided GitLab client and current user.
// It initializes all the service interfaces for epics, issues, and merge requests.
func New(client *gitlab.Client, currentUser string) *Tools {
	return &Tools{
		Epics:         NewEpicTools(client),
		Issues:        NewIssuesTools(client, currentUser),
		MergeRequests: NewMergeRequestsTools(client, currentUser),
		Users:         NewUsersTools(client, currentUser),
	}
}

// AddTo registers all GitLab tools with the provided MCPServer.
// It calls AddTo on all service interfaces to register their respective tools.
func (s *Tools) AddTo(srv *server.MCPServer) {
	s.Epics.AddTo(srv)
	s.Issues.AddTo(srv)
	s.MergeRequests.AddTo(srv)
	s.Users.AddTo(srv)
}

// newToolResultJSON encodes the provided value as JSON and returns it as a tool result.
// It handles the JSON encoding and error handling, providing a consistent way to return
// JSON responses from tool handlers.
func newToolResultJSON(v any) (*mcp.CallToolResult, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(v); err != nil {
		return nil, fmt.Errorf("encoding to JSON: %w", err)
	}
	return mcp.NewToolResultText(b.String()), nil
}

// idParam converts an MCP parameter (always a string) to either a string or an integer for the GitLab API.
// It attempts to convert the parameter to an integer first, and if that fails, it returns the original string.
// This is useful for GitLab API endpoints that accept either a numeric ID or a path-like string identifier.
func idParam(v any) (any, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("makeID: value %v is a %T, not a string", v, v)
	}

	id, err := strconv.Atoi(s)
	if err == nil {
		return id, nil
	}
	return s, nil
}

// intParam converts an MCP parameter to an integer.
// MCP parameters that represent numbers are usually provided as float64,
// so this function converts them to int, which is required by many GitLab API functions.
func intParam(v any) (int, error) {
	f, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("makeInt: value %v is a %T, not a number (float64)", v, v)
	}
	return int(f), nil
}
