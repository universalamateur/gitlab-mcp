//nolint:goconst // Test data strings like label names and colors are repeated intentionally.
package tools

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	gitlab "gitlab.com/gitlab-org/api/client-go/v3"
	glabtest "gitlab.com/gitlab-org/api/client-go/v3/testing"

	"gitlab.com/fforster/gitlab-mcp/lib/glctx"
)

func TestListProjectLabels(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockLabelsServiceInterface)
		wantError bool
	}{
		{
			name: "list with defaults",
			args: map[string]any{
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockLabelsServiceInterface) {
				m.EXPECT().
					ListLabels(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opt *gitlab.ListLabelsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
						wantOpt := &gitlab.ListLabelsOptions{
							ListOptions: gitlab.ListOptions{PerPage: maxPerPage},
						}
						if diff := cmp.Diff(wantOpt, opt); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Label{
							{ID: 1, Name: "bug", Color: "#FF0000"},
							{ID: 2, Name: "feature", Color: "#00FF00"},
						}, okResponse, nil
					})
			},
		},
		{
			name: "list with search",
			args: map[string]any{
				"project_id": "owner/project",
				"search":     "bug",
			},
			setupMock: func(m *glabtest.MockLabelsServiceInterface) {
				m.EXPECT().
					ListLabels(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opt *gitlab.ListLabelsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Label, *gitlab.Response, error) {
						if opt.Search == nil || *opt.Search != "bug" {
							return nil, nil, fmt.Errorf("expected search=bug, got %v", opt.Search)
						}

						return []*gitlab.Label{{ID: 1, Name: "bug", Color: "#FF0000"}}, okResponse, nil
					})
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockLabelsServiceInterface) {
				m.EXPECT().
					ListLabels(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, gitlab.ErrNotFound)
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockLabels)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "list_project_labels"
			req.Params.Arguments = tt.args

			_, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("error mismatch, got: %v, want error: %v", err, tt.wantError)
			}
		})
	}
}

func TestCreateProjectLabel(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	gitlabClient := glabtest.NewTestClient(t)
	ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

	gitlabClient.MockLabels.EXPECT().
		CreateLabel(gomock.Eq("owner/project"), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, opt *gitlab.CreateLabelOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Label, *gitlab.Response, error) {
			if opt.Name == nil || *opt.Name != "urgent" {
				return nil, nil, fmt.Errorf("expected name=urgent, got %v", opt.Name)
			}

			if opt.Color == nil || *opt.Color != "#FF0000" {
				return nil, nil, fmt.Errorf("expected color=#FF0000, got %v", opt.Color)
			}

			return &gitlab.Label{ID: 10, Name: "urgent", Color: "#FF0000"}, okResponse, nil
		})

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest

	req.Params.Name = "create_project_label"
	req.Params.Arguments = map[string]any{
		"project_id": "owner/project",
		"name":       "urgent",
		"color":      "#FF0000",
	}

	_, err := srv.Client().CallTool(t.Context(), req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateProjectLabel(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	gitlabClient := glabtest.NewTestClient(t)
	ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

	gitlabClient.MockLabels.EXPECT().
		UpdateLabel(gomock.Eq("owner/project"), gomock.Eq("bug"), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, _ any, opt *gitlab.UpdateLabelOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Label, *gitlab.Response, error) {
			if opt.NewName == nil || *opt.NewName != "defect" {
				return nil, nil, fmt.Errorf("expected new_name=defect, got %v", opt.NewName)
			}

			return &gitlab.Label{ID: 1, Name: "defect", Color: "#FF0000"}, okResponse, nil
		})

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest

	req.Params.Name = "update_project_label"
	req.Params.Arguments = map[string]any{
		"project_id": "owner/project",
		"name":       "bug",
		"new_name":   "defect",
	}

	_, err := srv.Client().CallTool(t.Context(), req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteProjectLabel(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	gitlabClient := glabtest.NewTestClient(t)
	ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

	gitlabClient.MockLabels.EXPECT().
		DeleteLabel(gomock.Eq("owner/project"), gomock.Eq("bug"), gomock.Any(), gomock.Any()).
		Return(okResponse, nil)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest

	req.Params.Name = "delete_project_label"
	req.Params.Arguments = map[string]any{
		"project_id": "owner/project",
		"name":       "bug",
	}

	_, err := srv.Client().CallTool(t.Context(), req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListGroupLabels(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	gitlabClient := glabtest.NewTestClient(t)
	ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

	gitlabClient.MockGroupLabels.EXPECT().
		ListGroupLabels(gomock.Eq("my-group"), gomock.Any(), gomock.Any()).
		Return([]*gitlab.GroupLabel{
			{ID: 1, Name: "priority::high", Color: "#FF0000"},
		}, okResponse, nil)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest

	req.Params.Name = "list_group_labels"
	req.Params.Arguments = map[string]any{
		"group_id": "my-group",
	}

	_, err := srv.Client().CallTool(t.Context(), req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateGroupLabel(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	gitlabClient := glabtest.NewTestClient(t)
	ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

	gitlabClient.MockGroupLabels.EXPECT().
		CreateGroupLabel(gomock.Eq("my-group"), gomock.Any(), gomock.Any()).
		Return(&gitlab.GroupLabel{ID: 5, Name: "team::backend", Color: "#0000FF"}, okResponse, nil)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest

	req.Params.Name = "create_group_label"
	req.Params.Arguments = map[string]any{
		"group_id": "my-group",
		"name":     "team::backend",
		"color":    "#0000FF",
	}

	_, err := srv.Client().CallTool(t.Context(), req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateGroupLabel(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	gitlabClient := glabtest.NewTestClient(t)
	ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

	gitlabClient.MockGroupLabels.EXPECT().
		UpdateGroupLabel(gomock.Eq("my-group"), gomock.Eq("old-name"), gomock.Any(), gomock.Any()).
		Return(&gitlab.GroupLabel{ID: 5, Name: "new-name", Color: "#00FF00"}, okResponse, nil)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest

	req.Params.Name = "update_group_label"
	req.Params.Arguments = map[string]any{
		"group_id": "my-group",
		"name":     "old-name",
		"new_name": "new-name",
		"color":    "#00FF00",
	}

	_, err := srv.Client().CallTool(t.Context(), req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteGroupLabel(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	gitlabClient := glabtest.NewTestClient(t)
	ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

	gitlabClient.MockGroupLabels.EXPECT().
		DeleteGroupLabel(gomock.Eq("my-group"), gomock.Eq("old-label"), gomock.Any(), gomock.Any()).
		Return(okResponse, nil)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	var req mcp.CallToolRequest

	req.Params.Name = "delete_group_label"
	req.Params.Arguments = map[string]any{
		"group_id": "my-group",
		"name":     "old-label",
	}

	_, err := srv.Client().CallTool(t.Context(), req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
