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

func TestCreateCommit(t *testing.T) {
	tests := []struct {
		name              string
		args              map[string]any
		setupMock         func(mockCommits *glabtest.MockCommitsServiceInterface, ret *gitlab.Commit)
		wantError         bool
		wantErrorResponse bool // If the tool call itself should return an error message
		wantResult        *gitlab.Commit
	}{
		{
			name: "create file",
			args: map[string]any{
				"project_id":     "test/project",
				"branch":         "feature-branch",
				"commit_message": "Add new file",
				"path":           "new-file.txt",
				"contents":       "Hello, World!",
				"action":         "create",
			},
			setupMock: func(mockCommits *glabtest.MockCommitsServiceInterface, ret *gitlab.Commit) {
				mockCommits.EXPECT().
					CreateCommit(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateCommitOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Commit, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateCommitOptions{
							Branch:        new("feature-branch"),
							CommitMessage: new("Add new file"),
							Actions: []*gitlab.CommitActionOptions{
								{
									Action:   new(gitlab.FileCreate),
									FilePath: new("new-file.txt"),
									Content:  new("Hello, World!"),
								},
							},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateCommit() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.Commit{
				ID:      "abc123",
				Message: "Add new file",
				Title:   "Add new file",
			},
		},
		{
			name: "update file",
			args: map[string]any{
				"project_id":     "test/project",
				"branch":         "main",
				"commit_message": "Update existing file",
				"path":           "existing-file.txt",
				"contents":       "Updated content",
				"action":         "update",
			},
			setupMock: func(mockCommits *glabtest.MockCommitsServiceInterface, ret *gitlab.Commit) {
				mockCommits.EXPECT().
					CreateCommit(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateCommitOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Commit, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateCommitOptions{
							Branch:        new("main"),
							CommitMessage: new("Update existing file"),
							Actions: []*gitlab.CommitActionOptions{
								{
									Action:   new(gitlab.FileUpdate),
									FilePath: new("existing-file.txt"),
									Content:  new("Updated content"),
								},
							},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateCommit() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.Commit{
				ID:      "def456",
				Message: "Update existing file",
				Title:   "Update existing file",
			},
		},
		{
			name: "delete file",
			args: map[string]any{
				"project_id":     "test/project",
				"branch":         "cleanup",
				"commit_message": "Remove old file",
				"path":           "old-file.txt",
				"action":         "delete",
			},
			setupMock: func(mockCommits *glabtest.MockCommitsServiceInterface, ret *gitlab.Commit) {
				mockCommits.EXPECT().
					CreateCommit(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateCommitOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Commit, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateCommitOptions{
							Branch:        new("cleanup"),
							CommitMessage: new("Remove old file"),
							Actions: []*gitlab.CommitActionOptions{
								{
									Action:   new(gitlab.FileDelete),
									FilePath: new("old-file.txt"),
								},
							},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateCommit() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.Commit{
				ID:      "ghi789",
				Message: "Remove old file",
				Title:   "Remove old file",
			},
		},
		{
			name: "move file with content",
			args: map[string]any{
				"project_id":     "test/project",
				"branch":         "refactor",
				"commit_message": "Move and update file",
				"path":           "new/location/file.txt",
				"previous_path":  "old/location/file.txt",
				"contents":       "Updated content in new location",
				"action":         "move",
			},
			setupMock: func(mockCommits *glabtest.MockCommitsServiceInterface, ret *gitlab.Commit) {
				mockCommits.EXPECT().
					CreateCommit(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateCommitOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Commit, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateCommitOptions{
							Branch:        new("refactor"),
							CommitMessage: new("Move and update file"),
							Actions: []*gitlab.CommitActionOptions{
								{
									Action:       new(gitlab.FileMove),
									FilePath:     new("new/location/file.txt"),
									PreviousPath: new("old/location/file.txt"),
									Content:      new("Updated content in new location"),
								},
							},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateCommit() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.Commit{
				ID:      "jkl012",
				Message: "Move and update file",
				Title:   "Move and update file",
			},
		},
		{
			name: "move file without content change",
			args: map[string]any{
				"project_id":     "test/project",
				"branch":         "reorganize",
				"commit_message": "Move file to new location",
				"path":           "docs/README.md",
				"previous_path":  "README.md",
				"action":         "move",
			},
			setupMock: func(mockCommits *glabtest.MockCommitsServiceInterface, ret *gitlab.Commit) {
				mockCommits.EXPECT().
					CreateCommit(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateCommitOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Commit, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateCommitOptions{
							Branch:        new("reorganize"),
							CommitMessage: new("Move file to new location"),
							Actions: []*gitlab.CommitActionOptions{
								{
									Action:       new(gitlab.FileMove),
									FilePath:     new("docs/README.md"),
									PreviousPath: new("README.md"),
								},
							},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateCommit() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.Commit{
				ID:      "mno345",
				Message: "Move file to new location",
				Title:   "Move file to new location",
			},
		},
		{
			name: "create file with start branch",
			args: map[string]any{
				"project_id":     "test/project",
				"branch":         "feature-branch",
				"start_branch":   "main",
				"commit_message": "Add new file from main",
				"path":           "new-file.txt",
				"contents":       "Hello from main!",
				"action":         "create",
			},
			setupMock: func(mockCommits *glabtest.MockCommitsServiceInterface, ret *gitlab.Commit) {
				mockCommits.EXPECT().
					CreateCommit(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateCommitOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Commit, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateCommitOptions{
							Branch:        new("feature-branch"),
							StartBranch:   new("main"),
							CommitMessage: new("Add new file from main"),
							Actions: []*gitlab.CommitActionOptions{
								{
									Action:   new(gitlab.FileCreate),
									FilePath: new("new-file.txt"),
									Content:  new("Hello from main!"),
								},
							},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateCommit() options mismatch (-want +got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusCreated}}, nil
					})
			},
			wantResult: &gitlab.Commit{
				ID:      "start123",
				Message: "Add new file from main",
				Title:   "Add new file from main",
			},
		},
		{
			name: "invalid action",
			args: map[string]any{
				"project_id":     "test/project",
				"branch":         "test",
				"commit_message": "Test",
				"path":           "test.txt",
				"action":         "invalid",
			},
			wantErrorResponse: true,
		},
		{
			name: "create action missing content",
			args: map[string]any{
				"project_id":     "test/project",
				"branch":         "test",
				"commit_message": "Test",
				"path":           "test.txt",
				"action":         "create",
			},
			wantErrorResponse: true,
		},
		{
			name: "move action missing previous_path",
			args: map[string]any{
				"project_id":     "test/project",
				"branch":         "test",
				"commit_message": "Test",
				"path":           "test.txt",
				"action":         "move",
			},
			wantErrorResponse: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"project_id":     "test/project",
				"branch":         "protected-branch",
				"commit_message": "Try to commit",
				"path":           "test.txt",
				"contents":       "content",
				"action":         "create",
			},
			setupMock: func(mockCommits *glabtest.MockCommitsServiceInterface, ret *gitlab.Commit) {
				mockCommits.EXPECT().
					CreateCommit(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil, fmt.Errorf("branch is protected"))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			testClient := glabtest.NewTestClient(t)
			mockCommits := testClient.MockCommits
			ctx := glctx.WithClient(t.Context(), testClient.Client)

			var expectedResult *gitlab.Commit
			if tt.wantResult != nil {
				expectedResult = tt.wantResult
			}

			if tt.setupMock != nil {
				tt.setupMock(mockCommits, expectedResult)
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

			req.Params.Name = "create_commit"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("CreateCommit() error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			if tt.wantResult != nil {
				var got *gitlab.Commit

				if err := unmarshalResult(result, &got); err != nil {
					t.Fatalf("decode error: %v", err)
				}

				if diff := cmp.Diff(tt.wantResult, got); diff != "" {
					t.Errorf("CreateCommit() result mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
