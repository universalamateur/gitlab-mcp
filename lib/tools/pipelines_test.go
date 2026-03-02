//nolint:goconst // Test data strings like "success" are repeated intentionally.
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

func TestListProjectPipelines(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockPipelinesServiceInterface)
		wantError bool
	}{
		{
			name: "list with defaults",
			args: map[string]any{
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockPipelinesServiceInterface) {
				m.EXPECT().
					ListProjectPipelines(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opt *gitlab.ListProjectPipelinesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.PipelineInfo, *gitlab.Response, error) {
						wantOpt := &gitlab.ListProjectPipelinesOptions{
							ListOptions: gitlab.ListOptions{PerPage: maxPerPage},
						}
						if diff := cmp.Diff(wantOpt, opt); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.PipelineInfo{
							{ID: 1, Status: "success", Ref: "main"},
							{ID: 2, Status: "failed", Ref: "feature"},
						}, okResponse, nil
					})
			},
		},
		{
			name: "list with status and ref filters",
			args: map[string]any{
				"project_id": "owner/project",
				"status":     "success",
				"ref":        "main",
			},
			setupMock: func(m *glabtest.MockPipelinesServiceInterface) {
				m.EXPECT().
					ListProjectPipelines(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opt *gitlab.ListProjectPipelinesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.PipelineInfo, *gitlab.Response, error) {
						if opt.Status == nil || *opt.Status != "success" {
							return nil, nil, fmt.Errorf("expected status=success, got %v", opt.Status)
						}

						if opt.Ref == nil || *opt.Ref != "main" {
							return nil, nil, fmt.Errorf("expected ref=main, got %v", opt.Ref)
						}

						return []*gitlab.PipelineInfo{{ID: 1, Status: "success", Ref: "main"}}, okResponse, nil
					})
			},
		},
		{
			name: "list with source filter",
			args: map[string]any{
				"project_id": "owner/project",
				"source":     "web",
			},
			setupMock: func(m *glabtest.MockPipelinesServiceInterface) {
				m.EXPECT().
					ListProjectPipelines(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opt *gitlab.ListProjectPipelinesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.PipelineInfo, *gitlab.Response, error) {
						if opt.Source == nil || *opt.Source != "web" {
							return nil, nil, fmt.Errorf("expected source=web, got %v", opt.Source)
						}

						return []*gitlab.PipelineInfo{{ID: 1, Status: "success", Ref: "main"}}, okResponse, nil
					})
			},
		},
		{
			name: "list with sha and username filters",
			args: map[string]any{
				"project_id": "owner/project",
				"sha":        "abc123def456",
				"username":   "john_doe",
			},
			setupMock: func(m *glabtest.MockPipelinesServiceInterface) {
				m.EXPECT().
					ListProjectPipelines(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opt *gitlab.ListProjectPipelinesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.PipelineInfo, *gitlab.Response, error) {
						if opt.SHA == nil || *opt.SHA != "abc123def456" {
							return nil, nil, fmt.Errorf("expected sha=abc123def456, got %v", opt.SHA)
						}

						if opt.Username == nil || *opt.Username != "john_doe" {
							return nil, nil, fmt.Errorf("expected username=john_doe, got %v", opt.Username)
						}

						return []*gitlab.PipelineInfo{{ID: 1, Status: "success", Ref: "main"}}, okResponse, nil
					})
			},
		},
		{
			name: "list with order_by and sort_order",
			args: map[string]any{
				"project_id": "owner/project",
				"order_by":   "updated_at",
				"sort_order": "asc",
			},
			setupMock: func(m *glabtest.MockPipelinesServiceInterface) {
				m.EXPECT().
					ListProjectPipelines(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opt *gitlab.ListProjectPipelinesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.PipelineInfo, *gitlab.Response, error) {
						if opt.OrderBy == nil || *opt.OrderBy != "updated_at" {
							return nil, nil, fmt.Errorf("expected order_by=updated_at, got %v", opt.OrderBy)
						}

						if opt.Sort == nil || *opt.Sort != "asc" {
							return nil, nil, fmt.Errorf("expected sort=asc, got %v", opt.Sort)
						}

						return []*gitlab.PipelineInfo{{ID: 1, Status: "success", Ref: "main"}}, okResponse, nil
					})
			},
		},
		{
			name: "list with limit",
			args: map[string]any{
				"project_id": "owner/project",
				"limit":      5,
			},
			setupMock: func(m *glabtest.MockPipelinesServiceInterface) {
				m.EXPECT().
					ListProjectPipelines(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.PipelineInfo{
						{ID: 1, Status: "success", Ref: "main"},
						{ID: 2, Status: "failed", Ref: "feature"},
						{ID: 3, Status: "running", Ref: "develop"},
						{ID: 4, Status: "success", Ref: "staging"},
						{ID: 5, Status: "canceled", Ref: "hotfix"},
					}, okResponse, nil)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockPipelinesServiceInterface) {
				m.EXPECT().
					ListProjectPipelines(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, gitlab.ErrNotFound)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockPipelines)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "list_project_pipelines"
			req.Params.Arguments = tt.args

			_, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("error mismatch, got: %v, want error: %v", err, tt.wantError)
			}
		})
	}
}

func TestGetPipeline(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	gitlabClient := glabtest.NewTestClient(t)
	ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

	gitlabClient.MockPipelines.EXPECT().
		GetPipeline(gomock.Eq("owner/project"), gomock.Eq(int64(42)), gomock.Any()).
		Return(&gitlab.Pipeline{ID: 42, Status: "success", Ref: "main"}, okResponse, nil)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest

	req.Params.Name = "get_pipeline"
	req.Params.Arguments = map[string]any{
		"project_id":  "owner/project",
		"pipeline_id": 42,
	}

	_, err := srv.Client().CallTool(t.Context(), req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreatePipeline(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockPipelinesServiceInterface)
		wantError bool
	}{
		{
			name: "create without variables",
			args: map[string]any{
				"project_id": "owner/project",
				"ref":        "main",
			},
			setupMock: func(m *glabtest.MockPipelinesServiceInterface) {
				m.EXPECT().
					CreatePipeline(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opt *gitlab.CreatePipelineOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Pipeline, *gitlab.Response, error) {
						if opt.Ref == nil || *opt.Ref != "main" {
							return nil, nil, fmt.Errorf("expected ref=main, got %v", opt.Ref)
						}

						if opt.Variables != nil {
							return nil, nil, fmt.Errorf("expected no variables, got %v", opt.Variables)
						}

						return &gitlab.Pipeline{ID: 99, Status: "created", Ref: "main"}, okResponse, nil
					})
			},
		},
		{
			name: "create with variables",
			args: map[string]any{
				"project_id": "owner/project",
				"ref":        "main",
				"variables":  "FOO=bar,BAZ=qux",
			},
			setupMock: func(m *glabtest.MockPipelinesServiceInterface) {
				m.EXPECT().
					CreatePipeline(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opt *gitlab.CreatePipelineOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Pipeline, *gitlab.Response, error) {
						if opt.Variables == nil || len(*opt.Variables) != 2 {
							return nil, nil, fmt.Errorf("expected 2 variables, got %v", opt.Variables)
						}

						return &gitlab.Pipeline{ID: 100, Status: "created", Ref: "main"}, okResponse, nil
					})
			},
		},
		{
			name: "invalid variable format",
			args: map[string]any{
				"project_id": "owner/project",
				"ref":        "main",
				"variables":  "NOEQUALS",
			},
			setupMock: func(_ *glabtest.MockPipelinesServiceInterface) {},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockPipelines)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "create_pipeline"
			req.Params.Arguments = tt.args

			_, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("error mismatch, got: %v, want error: %v", err, tt.wantError)
			}
		})
	}
}

func TestRetryPipeline(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	gitlabClient := glabtest.NewTestClient(t)
	ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

	gitlabClient.MockPipelines.EXPECT().
		RetryPipelineBuild(gomock.Eq("owner/project"), gomock.Eq(int64(42)), gomock.Any()).
		Return(&gitlab.Pipeline{ID: 42, Status: "running"}, okResponse, nil)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest

	req.Params.Name = "retry_pipeline"
	req.Params.Arguments = map[string]any{
		"project_id":  "owner/project",
		"pipeline_id": 42,
	}

	_, err := srv.Client().CallTool(t.Context(), req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCancelPipeline(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	gitlabClient := glabtest.NewTestClient(t)
	ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

	gitlabClient.MockPipelines.EXPECT().
		CancelPipelineBuild(gomock.Eq("owner/project"), gomock.Eq(int64(42)), gomock.Any()).
		Return(&gitlab.Pipeline{ID: 42, Status: "canceled"}, okResponse, nil)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest

	req.Params.Name = "cancel_pipeline"
	req.Params.Arguments = map[string]any{
		"project_id":  "owner/project",
		"pipeline_id": 42,
	}

	_, err := srv.Client().CallTool(t.Context(), req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
