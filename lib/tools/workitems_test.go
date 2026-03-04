package tools

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	glabtest "gitlab.com/gitlab-org/api/client-go/v2/testing"

	"gitlab.com/fforster/gitlab-mcp/lib/glctx"
)

func TestGetWorkItem(t *testing.T) {
	tests := []struct {
		name              string
		args              map[string]any
		setupMock         func(*glabtest.MockWorkItemsServiceInterface)
		wantError         bool
		wantErrorResponse bool
		want              *gitlab.WorkItem
	}{
		{
			name: "successful fetch of work item",
			args: map[string]any{
				"full_path": "test/project",
				"iid":       123,
			},
			setupMock: func(mock *glabtest.MockWorkItemsServiceInterface) {
				mock.EXPECT().
					GetWorkItem(gomock.Eq("test/project"), gomock.Eq(int64(123)), gomock.Any()).
					Return(
						&gitlab.WorkItem{
							ID:        175125690,
							IID:       123,
							State:     "CLOSED",
							Status:    new("Complete"),
							Title:     "Validate the generated metrics",
							CreatedAt: new(time.Date(2025, time.October, 13, 13, 19, 49, 0, time.UTC)), // "2025-10-13T13:19:49Z",
						},
						&gitlab.Response{},
						nil,
					)
			},
			want: &gitlab.WorkItem{
				ID:        175125690,
				IID:       123,
				State:     "CLOSED",
				Status:    new("Complete"),
				Title:     "Validate the generated metrics",
				CreatedAt: new(time.Date(2025, time.October, 13, 13, 19, 49, 0, time.UTC)), // "2025-10-13T13:19:49Z",
			},
		},
		{
			name: "error fetching work item",
			args: map[string]any{
				"full_path": "test/project",
				"iid":       123,
			},
			setupMock: func(mock *glabtest.MockWorkItemsServiceInterface) {
				mock.EXPECT().
					GetWorkItem(gomock.Eq("test/project"), gomock.Eq(int64(123)), gomock.Any()).
					Return(
						nil,
						&gitlab.Response{},
						errors.New("work item not found"),
					)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockWorkItems)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "get_work_item"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("get_work_item error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			if result.IsError != tt.wantErrorResponse {
				t.Errorf("unexpected inline error status, got: %v, want: %v", result.IsError, tt.wantErrorResponse)
			}

			if result.IsError {
				return
			}

			var got gitlab.WorkItem
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, &got); diff != "" {
				t.Errorf("issue mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}
