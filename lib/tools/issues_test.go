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

func TestGetIssue(t *testing.T) {
	tests := []struct {
		name              string
		args              map[string]any
		setupMock         func(*glabtest.MockIssuesServiceInterface, *glabtest.MockDiscussionsServiceInterface)
		wantError         bool
		wantErrorResponse bool
		want              *detailedIssue
	}{
		{
			name: "successful fetch of issue with discussions",
			args: map[string]any{
				"project_id": "test/project",
				"issue_iid":  123,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, mockDisc *glabtest.MockDiscussionsServiceInterface) {
				// Mock GetIssue
				mockIssues.EXPECT().
					GetIssue(gomock.Eq("test/project"), gomock.Eq(123), gomock.Any()).
					Return(&gitlab.Issue{
						ID:           12345,
						IID:          123,
						ProjectID:    54321,
						Title:        "Test issue",
						Description:  "Test description",
						State:        "opened",
						Confidential: false,
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)

				// Mock ListIssueDiscussions
				mockDisc.EXPECT().
					ListIssueDiscussions("test/project", gomock.Eq(123), gomock.Any(), gomock.Any()).
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
			},
			want: &detailedIssue{
				Issue: &gitlab.Issue{
					ID:           12345,
					IID:          123,
					ProjectID:    54321,
					Title:        "Test issue",
					Description:  "Test description",
					State:        "opened",
					Confidential: false,
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
			},
		},
		{
			name: "error in fetching issue",
			args: map[string]any{
				"project_id": "test/project",
				"issue_iid":  456,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, mockDisc *glabtest.MockDiscussionsServiceInterface) {
				// Mock GetIssue with error
				mockIssues.EXPECT().
					GetIssue(gomock.Any(), gomock.Eq(456), gomock.Any()).
					Return(nil, nil, fmt.Errorf("issue not found"))

				// This won't be called due to the error, but we need to set expectations
				// for all concurrent fetches to avoid test flakiness
				mockDisc.EXPECT().
					ListIssueDiscussions(gomock.Any(), gomock.Eq(456), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Discussion{}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil).AnyTimes()
			},
			wantError: true,
		},
		{
			name: "error in fetching discussions",
			args: map[string]any{
				"project_id": "test/project",
				"issue_iid":  789,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, mockDisc *glabtest.MockDiscussionsServiceInterface) {
				// Mock GetIssue
				mockIssues.EXPECT().
					GetIssue(gomock.Any(), gomock.Eq(789), gomock.Any()).
					Return(&gitlab.Issue{
						ID:           78901,
						IID:          789,
						Title:        "Test issue with discussion error",
						Confidential: false,
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)

				// Mock ListIssueDiscussions with error
				mockDisc.EXPECT().
					ListIssueDiscussions(gomock.Any(), gomock.Eq(789), gomock.Any(), gomock.Any()).
					Return(nil, nil, fmt.Errorf("discussions not available"))
			},
			wantError: true,
		},
		{
			name: "confidential issue without confidential flag",
			args: map[string]any{
				"project_id":   "test/project",
				"issue_iid":    101,
				"confidential": false,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, mockDisc *glabtest.MockDiscussionsServiceInterface) {
				// Mock GetIssue with confidential flag set to true
				mockIssues.EXPECT().
					GetIssue(gomock.Any(), gomock.Eq(101), gomock.Any()).
					Return(&gitlab.Issue{
						ID:           10101,
						IID:          101,
						Title:        "Confidential issue",
						Confidential: true,
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)

				// Mock ListIssueDiscussions
				mockDisc.EXPECT().
					ListIssueDiscussions(gomock.Any(), gomock.Eq(101), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Discussion{}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)
			},
			wantErrorResponse: true,
		},
		{
			name: "confidential issue with confidential flag",
			args: map[string]any{
				"project_id":   "test/project",
				"issue_iid":    102,
				"confidential": true,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, mockDisc *glabtest.MockDiscussionsServiceInterface) {
				// Mock GetIssue with confidential flag set to true
				mockIssues.EXPECT().
					GetIssue(gomock.Any(), gomock.Eq(102), gomock.Any()).
					Return(&gitlab.Issue{
						ID:           10202,
						IID:          102,
						Title:        "Confidential issue with access",
						Confidential: true,
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)

				// Mock ListIssueDiscussions
				mockDisc.EXPECT().
					ListIssueDiscussions(gomock.Any(), gomock.Eq(102), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Discussion{
						{
							ID: "confidential-discussion",
							Notes: []*gitlab.Note{
								{
									ID:     1,
									Body:   "Confidential comment",
									Author: gitlab.NoteAuthor{ID: 1, Username: "user1"},
								},
							},
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)
			},
			wantError: false,
			want: &detailedIssue{
				Issue: &gitlab.Issue{
					ID:           10202,
					IID:          102,
					Title:        "Confidential issue with access",
					Confidential: true,
				},
				Discussions: []*gitlab.Discussion{
					{
						ID: "confidential-discussion",
						Notes: []*gitlab.Note{
							{
								ID:     1,
								Body:   "Confidential comment",
								Author: gitlab.NoteAuthor{ID: 1, Username: "user1"},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			tt.setupMock(gitlabClient.MockIssues, gitlabClient.MockDiscussions)

			issuesService := NewIssuesTools(gitlabClient.Client, "test_user")

			srv, err := mcptest.NewServer(t, issuesService.GetIssue())
			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest
			req.Params.Name = "get_issue"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("getIssue error mismatch, got: %v, want error: %v", err, tt.wantError)
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

			var got detailedIssue
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, &got); diff != "" {
				t.Errorf("issue mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestListUserIssues(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockIssuesServiceInterface)
		wantError bool
		want      []*gitlab.Issue
	}{
		{
			name: "successful fetch with default values",
			args: map[string]any{},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				// Mock ListIssues
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						if opts.AssigneeUsername == nil || *opts.AssigneeUsername != "test_user" {
							t.Errorf("opts.AssigneeUsername = %v, want 'test_user'", opts.AssigneeUsername)
						}
						if opts.State == nil || *opts.State != defaultIssueState {
							t.Errorf("opts.State = %v, want %q", opts.State, defaultIssueState)
						}
						if opts.Confidential == nil || *opts.Confidential {
							t.Errorf("opts.Confidential = %v, want false", opts.Confidential)
						}

						return []*gitlab.Issue{
								{
									ID:          12345,
									IID:         123,
									ProjectID:   54321,
									Title:       "Test user issue 1",
									Description: "Test description 1",
									State:       "opened",
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0,
							}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:          12345,
					IID:         123,
					ProjectID:   54321,
					Title:       "Test user issue 1",
					Description: "Test description 1",
					State:       "opened",
				},
			},
		},
		{
			name: "successful fetch with custom assignee",
			args: map[string]any{
				"assignee": "other_user",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						if opts.AssigneeUsername == nil || *opts.AssigneeUsername != "other_user" {
							t.Errorf("opts.AssigneeUsername = %v, want 'other_user'", opts.AssigneeUsername)
						}

						return []*gitlab.Issue{
								{
									ID:    2001,
									IID:   201,
									Title: "Other user's issue",
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0,
							}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:    2001,
					IID:   201,
					Title: "Other user's issue",
				},
			},
		},
		{
			name: "successful fetch with pagination",
			args: map[string]any{},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				// First page
				firstCall := mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					Return([]*gitlab.Issue{
						{
							ID:    1001,
							IID:   101,
							Title: "First page issue",
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
						NextPage: 2,
					}, nil)

				// Second page (last)
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					After(firstCall.Call).
					Return([]*gitlab.Issue{
						{
							ID:    1002,
							IID:   102,
							Title: "Second page issue",
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
						NextPage: 0,
					}, nil)
			},
			want: []*gitlab.Issue{
				{
					ID:    1001,
					IID:   101,
					Title: "First page issue",
				},
				{
					ID:    1002,
					IID:   102,
					Title: "Second page issue",
				},
			},
		},
		{
			name: "error in fetching user issues",
			args: map[string]any{},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusInternalServerError},
					}, fmt.Errorf("internal server error"))
			},
			wantError: true,
		},
		{
			name: "fetch with confidential flag",
			args: map[string]any{
				"confidential": true,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						if opts.Confidential != nil {
							t.Errorf("opts.Confidential = %v, want nil", opts.Confidential)
						}

						return []*gitlab.Issue{
								{
									ID:           3001,
									IID:          301,
									Title:        "Confidential issue",
									Confidential: true,
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0,
							}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:           3001,
					IID:          301,
					Title:        "Confidential issue",
					Confidential: true,
				},
			},
		},
		{
			name: "fetch with state flag",
			args: map[string]any{
				"state": "closed",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						if opts.State == nil || *opts.State != "closed" {
							t.Errorf("opts.State = %v, want 'closed'", opts.State)
						}

						return []*gitlab.Issue{
								{
									ID:    4001,
									IID:   401,
									Title: "Closed issue",
									State: "closed",
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0,
							}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:    4001,
					IID:   401,
					Title: "Closed issue",
					State: "closed",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			tt.setupMock(gitlabClient.MockIssues)

			issuesService := NewIssuesTools(gitlabClient.Client, "test_user")

			srv, err := mcptest.NewServer(t, issuesService.ListUserIssues())
			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest
			req.Params.Name = "list_user_issues"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("listUserIssues error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			var got []*gitlab.Issue
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("issues mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestListGroupIssues(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockIssuesServiceInterface)
		wantError bool
		want      []*gitlab.Issue
	}{
		{
			name: "successful fetch with default values",
			args: map[string]any{
				"group_id": "test-group",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				// Mock ListGroupIssues
				mockIssues.EXPECT().
					ListGroupIssues(gomock.Eq("test-group"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListGroupIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						if opts.State == nil || *opts.State != defaultIssueState {
							t.Errorf("opts.State = %v, want %q", opts.State, defaultIssueState)
						}
						if opts.Confidential == nil || *opts.Confidential {
							t.Errorf("opts.Confidential = %v, want false", opts.Confidential)
						}

						return []*gitlab.Issue{
								{
									ID:          12345,
									IID:         123,
									ProjectID:   54321,
									Title:       "Test group issue 1",
									Description: "Test description 1",
									State:       "opened",
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0,
							}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:          12345,
					IID:         123,
					ProjectID:   54321,
					Title:       "Test group issue 1",
					Description: "Test description 1",
					State:       "opened",
				},
			},
		},
		{
			name: "successful fetch with pagination",
			args: map[string]any{
				"group_id": "test-group",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				// First page
				firstCall := mockIssues.EXPECT().
					ListGroupIssues(gomock.Eq("test-group"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Issue{
						{
							ID:    1001,
							IID:   101,
							Title: "First page issue",
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
						NextPage: 2,
					}, nil)

				// Second page (last)
				mockIssues.EXPECT().
					ListGroupIssues(gomock.Eq("test-group"), gomock.Any(), gomock.Any()).
					After(firstCall.Call).
					Return([]*gitlab.Issue{
						{
							ID:    1002,
							IID:   102,
							Title: "Second page issue",
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
						NextPage: 0,
					}, nil)
			},
			want: []*gitlab.Issue{
				{
					ID:    1001,
					IID:   101,
					Title: "First page issue",
				},
				{
					ID:    1002,
					IID:   102,
					Title: "Second page issue",
				},
			},
		},
		{
			name: "error in fetching group issues",
			args: map[string]any{
				"group_id": "nonexistent-group",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListGroupIssues(gomock.Eq("nonexistent-group"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusNotFound},
					}, fmt.Errorf("group not found"))
			},
			wantError: true,
		},
		{
			name: "fetch with confidential flag",
			args: map[string]any{
				"group_id":     "test-group",
				"confidential": true,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListGroupIssues(gomock.Eq("test-group"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListGroupIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						if opts.Confidential != nil {
							t.Errorf("opts.Confidential = %v, want nil", opts.Confidential)
						}

						return []*gitlab.Issue{
								{
									ID:           2001,
									IID:          201,
									Title:        "Confidential issue",
									Confidential: true,
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0,
							}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:           2001,
					IID:          201,
					Title:        "Confidential issue",
					Confidential: true,
				},
			},
		},
		{
			name: "fetch with state flag",
			args: map[string]any{
				"group_id": "test-group",
				"state":    "all",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListGroupIssues(gomock.Eq("test-group"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListGroupIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						if opts.State == nil || *opts.State != "all" {
							t.Errorf("opts.State = %v, want 'all'", opts.State)
						}

						return []*gitlab.Issue{
								{
									ID:    3001,
									IID:   301,
									Title: "Closed issue",
									State: "closed",
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0,
							}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:    3001,
					IID:   301,
					Title: "Closed issue",
					State: "closed",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			tt.setupMock(gitlabClient.MockIssues)

			issuesService := NewIssuesTools(gitlabClient.Client, "test_user")

			srv, err := mcptest.NewServer(t, issuesService.ListGroupIssues())
			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest
			req.Params.Name = "list_group_issues"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("listGroupIssues error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			var got []*gitlab.Issue
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("issues mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestListProjectIssues(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockIssuesServiceInterface)
		wantError bool
		want      []*gitlab.Issue
	}{
		{
			name: "successful fetch with default values",
			args: map[string]any{
				"project_id": "test-project",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				// Mock ListProjectIssues
				mockIssues.EXPECT().
					ListProjectIssues(gomock.Eq("test-project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListProjectIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						if opts.State == nil || *opts.State != defaultIssueState {
							t.Errorf("opts.State = %v, want %q", opts.State, defaultIssueState)
						}
						if opts.Confidential == nil || *opts.Confidential {
							t.Errorf("opts.Confidential = %v, want false", opts.Confidential)
						}

						return []*gitlab.Issue{
								{
									ID:          12345,
									IID:         123,
									ProjectID:   54321,
									Title:       "Test project issue 1",
									Description: "Test description 1",
									State:       "opened",
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0,
							}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:          12345,
					IID:         123,
					ProjectID:   54321,
					Title:       "Test project issue 1",
					Description: "Test description 1",
					State:       "opened",
				},
			},
		},
		{
			name: "successful fetch with pagination",
			args: map[string]any{
				"project_id": "test-project",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				// First page
				firstCall := mockIssues.EXPECT().
					ListProjectIssues(gomock.Eq("test-project"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Issue{
						{
							ID:    1001,
							IID:   101,
							Title: "First page issue",
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
						NextPage: 2,
					}, nil)

				// Second page (last)
				mockIssues.EXPECT().
					ListProjectIssues(gomock.Eq("test-project"), gomock.Any(), gomock.Any()).
					After(firstCall.Call).
					Return([]*gitlab.Issue{
						{
							ID:    1002,
							IID:   102,
							Title: "Second page issue",
						},
					}, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
						NextPage: 0,
					}, nil)
			},
			want: []*gitlab.Issue{
				{
					ID:    1001,
					IID:   101,
					Title: "First page issue",
				},
				{
					ID:    1002,
					IID:   102,
					Title: "Second page issue",
				},
			},
		},
		{
			name: "error in fetching project issues",
			args: map[string]any{
				"project_id": "nonexistent-project",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListProjectIssues(gomock.Eq("nonexistent-project"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusNotFound},
					}, fmt.Errorf("project not found"))
			},
			wantError: true,
		},
		{
			name: "fetch with confidential flag",
			args: map[string]any{
				"project_id":   "test-project",
				"confidential": true,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListProjectIssues(gomock.Eq("test-project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListProjectIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						if opts.Confidential != nil {
							t.Errorf("opts.Confidential = %v, want nil", opts.Confidential)
						}

						return []*gitlab.Issue{
								{
									ID:           2001,
									IID:          201,
									Title:        "Confidential issue",
									Confidential: true,
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0,
							}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:           2001,
					IID:          201,
					Title:        "Confidential issue",
					Confidential: true,
				},
			},
		},
		{
			name: "fetch with state flag",
			args: map[string]any{
				"project_id": "test-project",
				"state":      "all",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListProjectIssues(gomock.Eq("test-project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListProjectIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						if opts.State == nil || *opts.State != "all" {
							t.Errorf("opts.State = %v, want 'all'", opts.State)
						}

						return []*gitlab.Issue{
								{
									ID:    3001,
									IID:   301,
									Title: "Closed issue",
									State: "closed",
								},
								{
									ID:    3002,
									IID:   302,
									Title: "Open issue",
									State: "opened",
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
								NextPage: 0,
							}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:    3001,
					IID:   301,
					Title: "Closed issue",
					State: "closed",
				},
				{
					ID:    3002,
					IID:   302,
					Title: "Open issue",
					State: "opened",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			tt.setupMock(gitlabClient.MockIssues)

			issuesService := NewIssuesTools(gitlabClient.Client, "test_user")

			srv, err := mcptest.NewServer(t, issuesService.ListProjectIssues())
			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest
			req.Params.Name = "list_project_issues"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("listProjectIssues error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			var got []*gitlab.Issue
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("issues mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}
