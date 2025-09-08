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

func TestListSubgroups(t *testing.T) {
	defaultOpts := gitlab.ListSubGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPerPage,
		},
	}

	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockGroupsServiceInterface, []*gitlab.Group)
		want      []*gitlab.Group
		wantError bool
	}{
		{
			name: "without options",
			args: map[string]any{
				"group_id": "test/group",
			},
			setupMock: func(mockGroups *glabtest.MockGroupsServiceInterface, ret []*gitlab.Group) {
				mockGroups.EXPECT().
					ListSubGroups(gomock.Eq("test/group"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListSubGroupsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
						wantOpts := defaultOpts

						if diff := cmp.Diff(&wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: []*gitlab.Group{
				{ID: 1, Name: "test/group/sub1"},
				{ID: 2, Name: "test/group/sub2"},
			},
		},
		{
			name: "with all options",
			args: map[string]any{
				"group_id":         "test/group",
				"search":           "search_term",
				"owned":            true,
				"min_access_level": "guest",
				"order_by":         "id",
				"sort_order":       "desc",
			},
			setupMock: func(mockGroups *glabtest.MockGroupsServiceInterface, ret []*gitlab.Group) {
				mockGroups.EXPECT().
					ListSubGroups(gomock.Eq("test/group"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListSubGroupsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.Search = gitlab.Ptr("search_term")
						wantOpts.Owned = gitlab.Ptr(true)
						wantOpts.MinAccessLevel = gitlab.Ptr(gitlab.GuestPermissions)
						wantOpts.ListOptions.OrderBy = "id"
						wantOpts.ListOptions.Sort = sortOrderDescending

						if diff := cmp.Diff(&wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: []*gitlab.Group{
				{ID: 1, Name: "test/group/sub1"},
				{ID: 2, Name: "test/group/sub2"},
			},
		},
		{
			name: "group not found",
			args: map[string]any{
				"group_id": "test/group",
			},
			setupMock: func(mockGroups *glabtest.MockGroupsServiceInterface, ret []*gitlab.Group) {
				mockGroups.EXPECT().
					ListSubGroups(gomock.Eq("test/group"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, gitlab.ErrNotFound)
			},
			wantError: true,
		},
		{
			name: "limit truncates the result list",
			args: map[string]any{
				"group_id": "test/group",
				"limit":    2,
			},
			setupMock: func(mockGroups *glabtest.MockGroupsServiceInterface, _ []*gitlab.Group) {
				mockGroups.EXPECT().
					ListSubGroups(gomock.Eq("test/group"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Group{
						{ID: 1, Name: "test/group/sub1"},
						{ID: 2, Name: "test/group/sub2"},
						{ID: 3, Name: "test/group/sub3"},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)
			},
			want: []*gitlab.Group{
				{ID: 1, Name: "test/group/sub1"},
				{ID: 2, Name: "test/group/sub2"},
			},
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

			req.Params.Name = "list_subgroups"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("getIssue error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			var got []*gitlab.Group
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("issue mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}
