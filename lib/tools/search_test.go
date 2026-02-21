package tools

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	glabtest "gitlab.com/gitlab-org/api/client-go/testing"

	"gitlab.com/fforster/gitlab-mcp/lib/glctx"
)

func TestSearch(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockSearchServiceInterface)
		wantError bool
	}{
		{
			name: "global issues search",
			args: map[string]any{
				"search": "bug",
				"scope":  "issues",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Issues(gomock.Eq("bug"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(query string, opt *gitlab.SearchOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpt := &gitlab.SearchOptions{
							ListOptions: gitlab.ListOptions{PerPage: maxPerPage},
						}
						if diff := cmp.Diff(wantOpt, opt); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{{ID: 1, Title: "Bug report"}}, okResponse, nil
					})
			},
		},
		{
			name: "global blobs search",
			args: map[string]any{
				"search": "TODO",
				"scope":  "blobs",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Blobs(gomock.Eq("TODO"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Blob{{Filename: "main.go", Data: "// TODO"}}, okResponse, nil)
			},
		},
		{
			name: "group merge_requests search",
			args: map[string]any{
				"search":   "fix",
				"scope":    "merge_requests",
				"group_id": "my-group",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					MergeRequestsByGroup(gomock.Eq("my-group"), gomock.Eq("fix"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.MergeRequest{{BasicMergeRequest: gitlab.BasicMergeRequest{ID: 10, Title: "Fix typo"}}}, okResponse, nil)
			},
		},
		{
			name: "project notes search",
			args: map[string]any{
				"search":     "review",
				"scope":      "notes",
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					NotesByProject(gomock.Eq("owner/project"), gomock.Eq("review"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Note{{ID: 5, Body: "Please review"}}, okResponse, nil)
			},
		},
		{
			name: "notes scope without project_id fails",
			args: map[string]any{
				"search": "review",
				"scope":  "notes",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},
		{
			name: "projects scope at project level fails",
			args: map[string]any{
				"search":     "my-project",
				"scope":      "projects",
				"project_id": "owner/project",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},
		{
			name: "snippet_titles scope at group level fails",
			args: map[string]any{
				"search":   "snippet",
				"scope":    "snippet_titles",
				"group_id": "my-group",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},
		{
			name: "API error propagated",
			args: map[string]any{
				"search": "error",
				"scope":  "issues",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Issues(gomock.Eq("error"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, gitlab.ErrNotFound)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockSearch)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "search"
			req.Params.Arguments = tt.args

			_, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("search error mismatch, got: %v, want error: %v", err, tt.wantError)
			}
		})
	}
}
