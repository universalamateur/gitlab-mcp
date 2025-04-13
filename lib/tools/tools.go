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

type Tools struct {
	Epics         EpicServiceInterface
	Issues        IssuesServiceInterface
	MergeRequests MergeRequestsServiceInterface
}

func New(client *gitlab.Client, currentUser string) *Tools {
	return &Tools{
		Epics:         NewEpicTools(client),
		Issues:        NewIssuesTools(client, currentUser),
		MergeRequests: NewMergeRequestsTools(client, currentUser),
	}
}

func (s *Tools) AddTo(srv *server.MCPServer) {
	s.Epics.AddTo(srv)
	s.Issues.AddTo(srv)
	s.MergeRequests.AddTo(srv)
}

// newToolResultJSON encodes the provided value as JSON and returns it as a tool result
func newToolResultJSON(v any) (*mcp.CallToolResult, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(v); err != nil {
		return nil, fmt.Errorf("encoding to JSON: %w", err)
	}
	return mcp.NewToolResultText(b.String()), nil
}

// idParam converts an MCP parameter (always a string) to either a string or an integer for the GitLab API.
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
func intParam(v any) (int, error) {
	f, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("makeInt: value %v is a %T, not a number (float64)", v, v)
	}
	return int(f), nil
}
