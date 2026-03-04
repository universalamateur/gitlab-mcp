package tools

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	glabtest "gitlab.com/gitlab-org/api/client-go/v2/testing"

	"gitlab.com/fforster/gitlab-mcp/lib/glctx"
)

func TestCreateMergeRequest(t *testing.T) {
	tests := []struct {
		name              string
		args              map[string]any
		setupMock         func(mockMergeRequests *glabtest.MockMergeRequestsServiceInterface, ret *gitlab.MergeRequest)
		wantError         bool
		wantErrorResponse bool // If the tool call itself should return an error message
		wantResult        *gitlab.MergeRequest
	}{
		{
			name: "minimal merge request creation",
			args: map[string]any{
				"project_id":    "test/project",
				"source_branch": "feature-branch",
				"title":         "Add new feature",
			},
			setupMock: func(mockMergeRequests *glabtest.MockMergeRequestsServiceInterface, ret *gitlab.MergeRequest) {
				mockMergeRequests.EXPECT().
					CreateMergeRequest(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateMergeRequestOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateMergeRequestOptions{
							Title:        new("Add new feature"),
							SourceBranch: new("feature-branch"),
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateMergeRequest() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:          1,
					Title:        "Add new feature",
					SourceBranch: "feature-branch",
					TargetBranch: "main",
				},
			},
		},
		{
			name: "complete merge request creation with all options",
			args: map[string]any{
				"project_id":           "test/project",
				"source_branch":        "feature-branch",
				"title":                "Add comprehensive feature",
				"description":          "This adds a comprehensive new feature with tests",
				"target_branch":        "develop",
				"labels":               "feature,enhancement",
				"assignee_ids":         "1,2",
				"reviewer_ids":         "3,4",
				"milestone_id":         5,
				"remove_source_branch": true,
				"squash":               true,
			},
			setupMock: func(mockMergeRequests *glabtest.MockMergeRequestsServiceInterface, ret *gitlab.MergeRequest) {
				mockMergeRequests.EXPECT().
					CreateMergeRequest(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateMergeRequestOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateMergeRequestOptions{
							Title:              new("Add comprehensive feature"),
							SourceBranch:       new("feature-branch"),
							Description:        new("This adds a comprehensive new feature with tests"),
							TargetBranch:       new("develop"),
							Labels:             &gitlab.LabelOptions{"feature", "enhancement"},
							AssigneeIDs:        &[]int64{1, 2},
							ReviewerIDs:        &[]int64{3, 4},
							MilestoneID:        new(int64(5)),
							RemoveSourceBranch: new(true),
							Squash:             new(true),
						}
						if diff := cmp.Diff(wantOpts, opts, assigneeIDComparer); diff != "" {
							return nil, nil, fmt.Errorf("CreateMergeRequest() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:                     2,
					Title:                   "Add comprehensive feature",
					Description:             "This adds a comprehensive new feature with tests",
					SourceBranch:            "feature-branch",
					TargetBranch:            "develop",
					Squash:                  true,
					ForceRemoveSourceBranch: true,
				},
			},
		},
		{
			name: "merge request with description only",
			args: map[string]any{
				"project_id":    "test/project",
				"source_branch": "bugfix-branch",
				"title":         "Fix critical bug",
				"description":   "This fixes a critical bug in the authentication system",
			},
			setupMock: func(mockMergeRequests *glabtest.MockMergeRequestsServiceInterface, ret *gitlab.MergeRequest) {
				mockMergeRequests.EXPECT().
					CreateMergeRequest(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateMergeRequestOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateMergeRequestOptions{
							Title:        new("Fix critical bug"),
							SourceBranch: new("bugfix-branch"),
							Description:  new("This fixes a critical bug in the authentication system"),
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateMergeRequest() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:          3,
					Title:        "Fix critical bug",
					Description:  "This fixes a critical bug in the authentication system",
					SourceBranch: "bugfix-branch",
					TargetBranch: "main",
				},
			},
		},
		{
			name: "merge request with target branch",
			args: map[string]any{
				"project_id":    "test/project",
				"source_branch": "feature-x",
				"title":         "Implement feature X",
				"target_branch": "staging",
			},
			setupMock: func(mockMergeRequests *glabtest.MockMergeRequestsServiceInterface, ret *gitlab.MergeRequest) {
				mockMergeRequests.EXPECT().
					CreateMergeRequest(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateMergeRequestOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateMergeRequestOptions{
							Title:        new("Implement feature X"),
							SourceBranch: new("feature-x"),
							TargetBranch: new("staging"),
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateMergeRequest() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:          4,
					Title:        "Implement feature X",
					SourceBranch: "feature-x",
					TargetBranch: "staging",
				},
			},
		},
		{
			name: "merge request with single assignee and reviewer",
			args: map[string]any{
				"project_id":    "test/project",
				"source_branch": "docs-update",
				"title":         "Update documentation",
				"assignee_ids":  "42",
				"reviewer_ids":  "99",
			},
			setupMock: func(mockMergeRequests *glabtest.MockMergeRequestsServiceInterface, ret *gitlab.MergeRequest) {
				mockMergeRequests.EXPECT().
					CreateMergeRequest(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateMergeRequestOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateMergeRequestOptions{
							Title:        new("Update documentation"),
							SourceBranch: new("docs-update"),
							AssigneeIDs:  &[]int64{42},
							ReviewerIDs:  &[]int64{99},
						}
						if diff := cmp.Diff(wantOpts, opts, assigneeIDComparer); diff != "" {
							return nil, nil, fmt.Errorf("CreateMergeRequest() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:          5,
					Title:        "Update documentation",
					SourceBranch: "docs-update",
					TargetBranch: "main",
				},
			},
		},
		{
			name: "merge request with labels",
			args: map[string]any{
				"project_id":    "test/project",
				"source_branch": "refactor-code",
				"title":         "Refactor authentication module",
				"labels":        "refactor,breaking-change,high-priority",
			},
			setupMock: func(mockMergeRequests *glabtest.MockMergeRequestsServiceInterface, ret *gitlab.MergeRequest) {
				mockMergeRequests.EXPECT().
					CreateMergeRequest(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateMergeRequestOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateMergeRequestOptions{
							Title:        new("Refactor authentication module"),
							SourceBranch: new("refactor-code"),
							Labels:       &gitlab.LabelOptions{"refactor", "breaking-change", "high-priority"},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateMergeRequest() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:          6,
					Title:        "Refactor authentication module",
					SourceBranch: "refactor-code",
					TargetBranch: "main",
				},
			},
		},
		{
			name: "merge request with milestone",
			args: map[string]any{
				"project_id":    "test/project",
				"source_branch": "milestone-feature",
				"title":         "Add feature for milestone",
				"milestone_id":  10,
			},
			setupMock: func(mockMergeRequests *glabtest.MockMergeRequestsServiceInterface, ret *gitlab.MergeRequest) {
				mockMergeRequests.EXPECT().
					CreateMergeRequest(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateMergeRequestOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateMergeRequestOptions{
							Title:        new("Add feature for milestone"),
							SourceBranch: new("milestone-feature"),
							MilestoneID:  new(int64(10)),
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateMergeRequest() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:          7,
					Title:        "Add feature for milestone",
					SourceBranch: "milestone-feature",
					TargetBranch: "main",
				},
			},
		},
		{
			name: "API error - branch conflict",
			args: map[string]any{
				"project_id":    "test/project",
				"source_branch": "conflicting-branch",
				"title":         "This will conflict",
			},
			setupMock: func(mockMergeRequests *glabtest.MockMergeRequestsServiceInterface, ret *gitlab.MergeRequest) {
				mockMergeRequests.EXPECT().
					CreateMergeRequest(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil, fmt.Errorf("source branch does not exist"))
			},
			wantError: true,
		},
		{
			name: "API error - insufficient permissions",
			args: map[string]any{
				"project_id":    "restricted/project",
				"source_branch": "feature-branch",
				"title":         "Unauthorized access",
			},
			setupMock: func(mockMergeRequests *glabtest.MockMergeRequestsServiceInterface, ret *gitlab.MergeRequest) {
				mockMergeRequests.EXPECT().
					CreateMergeRequest(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil, fmt.Errorf("403 Forbidden"))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			testClient := glabtest.NewTestClient(t)
			mockMergeRequests := testClient.MockMergeRequests
			ctx := glctx.WithClient(t.Context(), testClient.Client)

			var expectedResult *gitlab.MergeRequest
			if tt.wantResult != nil {
				expectedResult = tt.wantResult
			}

			if tt.setupMock != nil {
				tt.setupMock(mockMergeRequests, expectedResult)
			}

			// Create MCP server and test the tool
			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			// Test direct function call
			var req mcp.CallToolRequest

			req.Params.Name = "create_merge_request"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("CreateMergeRequest() error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			if tt.wantResult != nil {
				var got *gitlab.MergeRequest

				if err := unmarshalResult(result, &got); err != nil {
					t.Fatalf("decode error: %v", err)
				}

				if diff := cmp.Diff(tt.wantResult, got); diff != "" {
					t.Errorf("CreateMergeRequest() result mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
