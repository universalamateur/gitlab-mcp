package tools

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	glabtest "gitlab.com/gitlab-org/api/client-go/testing"

	"gitlab.com/fforster/gitlab-mcp/lib/glctx"
)

func TestListProjects(t *testing.T) {
	defaultOpts := gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPerPage,
		},
	}

	tests := []struct {
		name            string
		args            map[string]any
		setupMock       func(*glabtest.MockProjectsServiceInterface, []*gitlab.Project)
		want            []*gitlab.Project
		wantInlineError string
		wantError       bool
	}{
		{
			name: "without options",
			args: map[string]any{},
			setupMock: func(mockProjects *glabtest.MockProjectsServiceInterface, ret []*gitlab.Project) {
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListProjectsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
						wantOpts := defaultOpts

						if diff := cmp.Diff(&wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: []*gitlab.Project{
				{ID: 1, Name: "test/group/project1"},
				{ID: 2, Name: "test/group/project2"},
			},
		},
		{
			name: "with all options",
			args: map[string]any{
				"search":           "search_term",
				"owned":            true,
				"min_access_level": "guest",
				"membership":       true,
				"order_by":         "id",
				"sort_order":       "desc",
			},
			setupMock: func(mockProjects *glabtest.MockProjectsServiceInterface, ret []*gitlab.Project) {
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListProjectsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.Search = gitlab.Ptr("search_term")
						wantOpts.Owned = gitlab.Ptr(true)
						wantOpts.MinAccessLevel = gitlab.Ptr(gitlab.GuestPermissions)
						wantOpts.Membership = gitlab.Ptr(true)
						wantOpts.ListOptions.OrderBy = "id"
						wantOpts.ListOptions.Sort = "desc"

						if diff := cmp.Diff(&wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: []*gitlab.Project{
				{ID: 1, Name: "test/group/project1"},
				{ID: 2, Name: "test/group/project2"},
			},
		},
		{
			name: "internal server error",
			args: map[string]any{},
			setupMock: func(mockProjects *glabtest.MockProjectsServiceInterface, ret []*gitlab.Project) {
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, errors.New("internal server error"))
			},
			wantError: true,
		},
		{
			name: "limit truncates the result list",
			args: map[string]any{
				"limit": 2,
			},
			setupMock: func(mockProjects *glabtest.MockProjectsServiceInterface, ret []*gitlab.Project) {
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					Return([]*gitlab.Project{
						{ID: 1, Name: "test/group/project1"},
						{ID: 2, Name: "test/group/project2"},
						{ID: 3, Name: "test/group/project3"},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)
			},
			want: []*gitlab.Project{
				{ID: 1, Name: "test/group/project1"},
				{ID: 2, Name: "test/group/project2"},
			},
		},
		{
			name: "order_by similarity without search yields inline error",
			args: map[string]any{
				"order_by": "similarity",
			},
			setupMock:       func(mockProjects *glabtest.MockProjectsServiceInterface, ret []*gitlab.Project) {},
			wantInlineError: "search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockProjects, tt.want)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest
			req.Params.Name = "list_projects"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("list_projects error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			if tt.wantInlineError != "" {
				s, gotErr := resultToString(result)
				if gotErr == nil {
					t.Fatalf("got successful response %q, wanted inline error containing %q", s, tt.wantInlineError)
				}

				if !strings.Contains(gotErr.Error(), tt.wantInlineError) {
					t.Errorf("got inline error %q, want substring: %q", gotErr.Error(), tt.wantInlineError)
				}
			}

			if result.IsError {
				return
			}

			var got []*gitlab.Project
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("issue mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestListGroupProjects(t *testing.T) {
	defaultOpts := gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxPerPage,
		},
	}

	tests := []struct {
		name            string
		args            map[string]any
		setupMock       func(*glabtest.MockGroupsServiceInterface, any, []*gitlab.Project)
		want            []*gitlab.Project
		wantInlineError string
		wantError       bool
	}{
		{
			name: "without options",
			args: map[string]any{},
			setupMock: func(mockGroups *glabtest.MockGroupsServiceInterface, groupID any, ret []*gitlab.Project) {
				mockGroups.EXPECT().
					ListGroupProjects(gomock.Eq(groupID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListGroupProjectsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
						wantOpts := defaultOpts

						if diff := cmp.Diff(&wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: []*gitlab.Project{
				{ID: 1, Name: "test/group/project1"},
				{ID: 2, Name: "test/group/project2"},
			},
		},
		{
			name: "with all options",
			args: map[string]any{
				"search":           "search_term",
				"owned":            true,
				"min_access_level": "guest",
				"order_by":         "id",
				"sort_order":       "desc",
			},
			setupMock: func(mockGroups *glabtest.MockGroupsServiceInterface, groupID any, ret []*gitlab.Project) {
				mockGroups.EXPECT().
					ListGroupProjects(gomock.Eq(groupID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListGroupProjectsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.Search = gitlab.Ptr("search_term")
						wantOpts.Owned = gitlab.Ptr(true)
						wantOpts.MinAccessLevel = gitlab.Ptr(gitlab.GuestPermissions)
						wantOpts.ListOptions.OrderBy = "id"
						wantOpts.ListOptions.Sort = "desc"

						if diff := cmp.Diff(&wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: []*gitlab.Project{
				{ID: 1, Name: "test/group/project1"},
				{ID: 2, Name: "test/group/project2"},
			},
		},
		{
			name: "order_by similarity without search yields inline error",
			args: map[string]any{
				"order_by": "similarity",
			},
			setupMock:       func(mockGroups *glabtest.MockGroupsServiceInterface, groupID any, ret []*gitlab.Project) {},
			wantInlineError: "search",
		},
		{
			name: "membership yields inline error",
			args: map[string]any{
				"membership": true,
			},
			setupMock:       func(mockGroups *glabtest.MockGroupsServiceInterface, groupID any, ret []*gitlab.Project) {},
			wantInlineError: "group_id",
		},
		{
			name: "internal server error",
			args: map[string]any{},
			setupMock: func(mockGroups *glabtest.MockGroupsServiceInterface, groupID any, ret []*gitlab.Project) {
				mockGroups.EXPECT().
					ListGroupProjects(gomock.Eq(groupID), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, errors.New("internal server error"))
			},
			wantError: true,
		},
		{
			name: "limit truncates the result",
			args: map[string]any{
				"limit": 1,
			},
			setupMock: func(mockGroups *glabtest.MockGroupsServiceInterface, groupID any, ret []*gitlab.Project) {
				mockGroups.EXPECT().
					ListGroupProjects(gomock.Eq(groupID), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Project{
						{ID: 1, Name: "test/group/project1"},
						{ID: 2, Name: "test/group/project2"},
						{ID: 3, Name: "test/group/project3"},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)
			},
			want: []*gitlab.Project{
				{ID: 1, Name: "test/group/project1"},
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			groupID := 1000 + i
			tt.args["group_id"] = strconv.Itoa(groupID)

			tt.setupMock(gitlabClient.MockGroups, groupID, tt.want)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest
			req.Params.Name = "list_projects"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("list_projects error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			if tt.wantInlineError != "" {
				s, gotErr := resultToString(result)
				if gotErr == nil {
					t.Fatalf("got successful response %q, wanted inline error containing %q", s, tt.wantInlineError)
				}

				if !strings.Contains(gotErr.Error(), tt.wantInlineError) {
					t.Errorf("got inline error %q, want substring: %q", gotErr.Error(), tt.wantInlineError)
				}
			}

			if result.IsError {
				return
			}

			var got []*gitlab.Project
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("issue mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}
