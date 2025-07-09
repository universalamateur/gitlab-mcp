package tools

import (
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

func TestGetGroup(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockGroupsServiceInterface, *gitlab.Group)
		want      *gitlab.Group
		wantError bool
	}{
		{
			name: "success",
			args: map[string]any{
				"group_id": "test/group",
			},
			setupMock: func(mockGroups *glabtest.MockGroupsServiceInterface, ret *gitlab.Group) {
				mockGroups.EXPECT().
					GetGroup(gomock.Eq("test/group"), gomock.Any(), gomock.Any()).
					Return(ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil)
			},
			want: &gitlab.Group{
				ID:   123,
				Name: "test/group",
			},
		},
		{
			name: "group not found",
			args: map[string]any{
				"group_id": "test/group",
			},
			setupMock: func(mockGroups *glabtest.MockGroupsServiceInterface, ret *gitlab.Group) {
				mockGroups.EXPECT().
					GetGroup(gomock.Eq("test/group"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, gitlab.ErrNotFound)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockGroups, tt.want)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest
			req.Params.Name = "get_group"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("getIssue error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			var got gitlab.Group
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, &got); diff != "" {
				t.Errorf("issue mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}
