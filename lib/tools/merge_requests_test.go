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
)

func TestGetMergeRequest(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockMergeRequestsServiceInterface, *glabtest.MockDiscussionsServiceInterface)
		want      *detailedMergeRequest
		wantErr   bool
	}{
		{
			name: "successful fetch of merge request with discussions and diffs",
			args: map[string]any{
				"project_id":        "test/project",
				"merge_request_iid": 123,
			},
			setupMock: func(mockMR *glabtest.MockMergeRequestsServiceInterface, mockDisc *glabtest.MockDiscussionsServiceInterface) {
				// Mock GetMergeRequest
				mockMR.EXPECT().
					GetMergeRequest(gomock.Eq("test/project"), gomock.Eq(123), gomock.Any(), gomock.Any()).
					Return(&gitlab.MergeRequest{
						BasicMergeRequest: gitlab.BasicMergeRequest{
							ID:          12345,
							IID:         123,
							ProjectID:   54321,
							Title:       "Test merge request",
							Description: "Test description",
							State:       "opened",
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)

				// Mock ListMergeRequestDiscussions
				mockDisc.EXPECT().
					ListMergeRequestDiscussions(gomock.Any(), gomock.Eq(123), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Discussion{
						{
							ID: "discussion1",
							Notes: []*gitlab.Note{
								{
									ID:     1,
									Body:   "Test comment",
									Author: gitlab.NoteAuthor{ID: 1, Username: "user1"},
								},
							},
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
						NextPage: 0,
					}, nil)

				// Mock ListMergeRequestDiffs
				mockMR.EXPECT().
					ListMergeRequestDiffs(gomock.Any(), gomock.Eq(123), gomock.Any(), gomock.Any()).
					Return([]*gitlab.MergeRequestDiff{
						{
							OldPath:     "file.go",
							NewPath:     "file.go",
							AMode:       "100644",
							BMode:       "100644",
							NewFile:     false,
							RenamedFile: false,
							DeletedFile: false,
							Diff:        "@@ -1,3 +1,4 @@\n Line 1\n+Line 2\n Line 3\n",
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
						NextPage: 0,
					}, nil)
			},
			want: &detailedMergeRequest{
				MergeRequest: &gitlab.MergeRequest{
					BasicMergeRequest: gitlab.BasicMergeRequest{
						ID:          12345,
						IID:         123,
						ProjectID:   54321,
						Title:       "Test merge request",
						Description: "Test description",
						State:       "opened",
					},
				},
				Discussions: []*gitlab.Discussion{
					{
						ID: "discussion1",
						Notes: []*gitlab.Note{
							{
								ID:     1,
								Body:   "Test comment",
								Author: gitlab.NoteAuthor{ID: 1, Username: "user1"},
							},
						},
					},
				},
				Diffs: []*gitlab.MergeRequestDiff{
					{
						OldPath:     "file.go",
						NewPath:     "file.go",
						AMode:       "100644",
						BMode:       "100644",
						NewFile:     false,
						RenamedFile: false,
						DeletedFile: false,
						Diff:        "@@ -1,3 +1,4 @@\n Line 1\n+Line 2\n Line 3\n",
					},
				},
			},
		},
		{
			name: "error in fetching merge request",
			args: map[string]any{
				"project_id":        "test/project",
				"merge_request_iid": 456,
			},
			setupMock: func(mockMR *glabtest.MockMergeRequestsServiceInterface, mockDisc *glabtest.MockDiscussionsServiceInterface) {
				// Mock GetMergeRequest with error
				mockMR.EXPECT().
					GetMergeRequest(gomock.Any(), gomock.Eq(456), gomock.Any(), gomock.Any()).
					Return(nil, nil, fmt.Errorf("merge request not found"))

				// These won't be called due to the error, but we need to set expectations
				// for all concurrent fetches to avoid test flakiness
				mockDisc.EXPECT().
					ListMergeRequestDiscussions(gomock.Any(), gomock.Eq(456), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Discussion{}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil).AnyTimes()

				mockMR.EXPECT().
					ListMergeRequestDiffs(gomock.Any(), gomock.Eq(456), gomock.Any(), gomock.Any()).
					Return([]*gitlab.MergeRequestDiff{}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil).AnyTimes()
			},
			wantErr: true,
		},
		{
			name: "error in fetching discussions",
			args: map[string]any{
				"project_id":        "test/project",
				"merge_request_iid": 789,
			},
			setupMock: func(mockMR *glabtest.MockMergeRequestsServiceInterface, mockDisc *glabtest.MockDiscussionsServiceInterface) {
				// Mock GetMergeRequest
				mockMR.EXPECT().
					GetMergeRequest(gomock.Any(), gomock.Eq(789), gomock.Any(), gomock.Any()).
					Return(&gitlab.MergeRequest{
						BasicMergeRequest: gitlab.BasicMergeRequest{
							ID:    78901,
							IID:   789,
							Title: "Test merge request with discussion error",
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)

				// Mock ListMergeRequestDiscussions with error
				mockDisc.EXPECT().
					ListMergeRequestDiscussions(gomock.Any(), gomock.Eq(789), gomock.Any(), gomock.Any()).
					Return(nil, nil, fmt.Errorf("discussions not available"))

				// Mock ListMergeRequestDiffs
				mockMR.EXPECT().
					ListMergeRequestDiffs(gomock.Any(), gomock.Eq(789), gomock.Any(), gomock.Any()).
					Return([]*gitlab.MergeRequestDiff{}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil).AnyTimes()
			},
			wantErr: true,
		},
		{
			name: "pagination handling for diffs",
			args: map[string]any{
				"project_id":        "test/project",
				"merge_request_iid": 101,
			},
			setupMock: func(mockMR *glabtest.MockMergeRequestsServiceInterface, mockDisc *glabtest.MockDiscussionsServiceInterface) {
				// Mock GetMergeRequest
				mockMR.EXPECT().
					GetMergeRequest(gomock.Eq("test/project"), gomock.Eq(101), gomock.Any(), gomock.Any()).
					Return(&gitlab.MergeRequest{
						BasicMergeRequest: gitlab.BasicMergeRequest{
							ID:    10101,
							IID:   101,
							Title: "Test merge request with paginated diffs",
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)

				// Mock ListMergeRequestDiscussions
				mockDisc.EXPECT().
					ListMergeRequestDiscussions(gomock.Any(), gomock.Eq(101), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Discussion{}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)

				// First page of diffs
				firstPage := mockMR.EXPECT().
					ListMergeRequestDiffs(gomock.Any(), gomock.Eq(101), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gitlab.ListMergeRequestDiffsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.MergeRequestDiff, *gitlab.Response, error) {
						if opts.Page != 0 {
							t.Errorf("expected first page request, got page %d", opts.Page)
						}

						return []*gitlab.MergeRequestDiff{
								{
									OldPath: "file1.go",
									NewPath: "file1.go",
									Diff:    "@@ -1,3 +1,3 @@\n Line 1\n-Line 2\n+Updated Line 2\n",
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 2,
							}, nil
					})

				// Second page of diffs
				mockMR.EXPECT().
					ListMergeRequestDiffs(gomock.Any(), gomock.Eq(101), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gitlab.ListMergeRequestDiffsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.MergeRequestDiff, *gitlab.Response, error) {
						if opts.Page != 2 {
							t.Errorf("expected second page request, got page %d", opts.Page)
						}

						return []*gitlab.MergeRequestDiff{
								{
									OldPath: "file2.go",
									NewPath: "file2.go",
									Diff:    "@@ -10,5 +10,6 @@\n func Example() {\n+    // Add comment\n     fmt.Println(\"test\")\n }\n",
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0, // No more pages
							}, nil
					}).
					After(firstPage.Call)
			},
			want: &detailedMergeRequest{
				MergeRequest: &gitlab.MergeRequest{
					BasicMergeRequest: gitlab.BasicMergeRequest{
						ID:    10101,
						IID:   101,
						Title: "Test merge request with paginated diffs",
					},
				},
				Diffs: []*gitlab.MergeRequestDiff{
					{
						OldPath: "file1.go",
						NewPath: "file1.go",
						Diff:    "@@ -1,3 +1,3 @@\n Line 1\n-Line 2\n+Updated Line 2\n",
					},
					{
						OldPath: "file2.go",
						NewPath: "file2.go",
						Diff:    "@@ -10,5 +10,6 @@\n func Example() {\n+    // Add comment\n     fmt.Println(\"test\")\n }\n",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			tt.setupMock(gitlabClient.MockMergeRequests, gitlabClient.MockDiscussions)

			mergeRequestsService := NewMergeRequestsTools(gitlabClient.Client, "test_user")

			srv, err := mcptest.NewServer(t, mergeRequestsService.GetMergeRequest())
			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}
			defer srv.Close()

			// Test direct function call
			var req mcp.CallToolRequest
			req.Params.Name = "get_merge_request"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Errorf("getIssue error mismatch, got: %v, want error: %v", err, tt.wantErr)
			}

			if err != nil {
				return
			}

			var got detailedMergeRequest
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, &got); diff != "" {
				t.Errorf("getIssue() mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}
