//nolint:goconst // "all" is used for scope and state. They should not use the same constant.
package tools

import (
	"fmt"
	"net/http"
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

// timeComparer allows cmp to compare time.Time values.
var timeComparer = cmp.Comparer(func(a, b time.Time) bool {
	return a.Equal(b)
})

var assigneeIDComparer = cmp.Comparer(func(a, b gitlab.AssigneeIDValue) bool {
	aData, err := a.MarshalJSON()
	if err != nil {
		panic(err)
	}

	bData, err := b.MarshalJSON()
	if err != nil {
		panic(err)
	}

	return string(aData) == string(bData)
})

func TestListIssues(t *testing.T) {
	// Default options template
	defaultOpts := gitlab.ListIssuesOptions{
		State:        new(defaultIssueState),
		Confidential: new(false),
		Scope:        new("all"),
		ListOptions: gitlab.ListOptions{
			PerPage: maxPerPage,
		},
	}

	cmpOpts := []cmp.Option{timeComparer, assigneeIDComparer}

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
						wantOpts := defaultOpts

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
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
			name: "issues assigned to current user",
			args: map[string]any{
				"assignee": "me",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.Scope = new("assigned_to_me")

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{
							{
								ID:    2002,
								IID:   202,
								Title: "My issue",
							},
						}, &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							NextPage: 0,
						}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:    2002,
					IID:   202,
					Title: "My issue",
				},
			},
		},
		{
			name: "issues created by current user",
			args: map[string]any{
				"author": "me",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.Scope = new("created_by_me")

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{
							{
								ID:    2002,
								IID:   202,
								Title: "My issue",
							},
						}, &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							NextPage: 0,
						}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:    2002,
					IID:   202,
					Title: "My issue",
				},
			},
		},
		{
			name: "arguments author, assignee, labels, milestone",
			args: map[string]any{
				"author":    "author_user",
				"assignee":  "other_user",
				"labels":    "bug,feature",
				"milestone": "v1.0",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.AuthorUsername = new("author_user")
						wantOpts.AssigneeUsername = new("other_user")
						wantOpts.Labels = &gitlab.LabelOptions{"bug", "feature"}
						wantOpts.Milestone = new("v1.0")

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
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
			name: "numeric user IDs",
			args: map[string]any{
				"author":   "5001",
				"assignee": "5002",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.AuthorID = new(int64(5001))
						wantOpts.AssigneeID = gitlab.AssigneeID(5002)

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{{ID: 5003}}, &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							NextPage: 0,
						}, nil
					})
			},
			want: []*gitlab.Issue{{ID: 5003}},
		},
		{
			name: "arguments created_before, created_after, updated_before, updated_after, due_date",
			args: map[string]any{
				"created_after":  "2025-01-01T00:00:00.000Z", // RFC 3339
				"created_before": "2025-02-01T00:00:00.000Z", // RFC 3339
				"updated_after":  "2025-03-01 00:00:00",      // custom format
				"updated_before": "2025-04-01 00:00:00",      // custom format
				"due_date":       "overdue",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.CreatedAfter = new(time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC))
						wantOpts.CreatedBefore = new(time.Date(2025, time.February, 1, 0, 0, 0, 0, time.UTC))
						wantOpts.UpdatedAfter = new(time.Date(2025, time.March, 1, 0, 0, 0, 0, time.Local))  //nolint:gosmopolitan
						wantOpts.UpdatedBefore = new(time.Date(2025, time.April, 1, 0, 0, 0, 0, time.Local)) //nolint:gosmopolitan
						wantOpts.DueDate = new("overdue")

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{
							{
								ID:    2002,
								IID:   202,
								Title: "My issue",
							},
						}, &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							NextPage: 0,
						}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:    2002,
					IID:   202,
					Title: "My issue",
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
						wantOpts := defaultOpts
						wantOpts.Confidential = nil // Confidential should be nil when true is passed (not filtered)

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
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
						wantOpts := defaultOpts
						wantOpts.State = new("closed")

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
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
		{
			name: "fetch with limit flag",
			args: map[string]any{
				// 349 is prime, so it is not a multiple of maxPerPage.
				"limit": 349,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, reqOpts ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						page := pageFromRequestOptions(t, reqOpts)

						if opts.ListOptions.PerPage <= 0 {
							return nil, nil, fmt.Errorf("per_page must be greater than 0")
						}

						index := (page - 1) * opts.ListOptions.PerPage
						if index >= 349 {
							return nil, nil, fmt.Errorf("page index %d exceeds limit %d (page = %d, per_page = %d)", index, 349, page, opts.ListOptions.PerPage)
						}

						return makeIssues(index, index+opts.ListOptions.PerPage), &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							NextPage: page + 1,
						}, nil
					}).AnyTimes()
			},
			want: makeIssues(0, 349),
		},
		{
			name: "order_by and sort_order",
			args: map[string]any{
				"order_by":   "priority",
				"sort_order": "desc",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gitlab.ListIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.ListOptions.OrderBy = "priority"
						wantOpts.ListOptions.Sort = "desc"

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{
							{
								ID:    5001,
								IID:   501,
								Title: "High priority issue",
							},
						}, &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							NextPage: 0,
						}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:    5001,
					IID:   501,
					Title: "High priority issue",
				},
			},
		},
		{
			name: "empty response",
			args: map[string]any{},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListIssues(gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
						NextPage: 0,
					}, nil)
			},
			want: []*gitlab.Issue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			gitlabClient.MockUsers.EXPECT().
				CurrentUser(gomock.Any()).
				Return(&gitlab.User{Username: "test_user"}, &gitlab.Response{
					Response: &http.Response{StatusCode: http.StatusOK},
				}, nil).AnyTimes()

			tt.setupMock(gitlabClient.MockIssues)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "list_issues"
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
	// Default options template
	defaultOpts := gitlab.ListGroupIssuesOptions{
		State:        new(defaultIssueState),
		Confidential: new(false),
		ListOptions: gitlab.ListOptions{
			PerPage: maxPerPage,
		},
	}

	cmpOpts := []cmp.Option{
		timeComparer,
		cmp.Comparer(func(id0, id1 gitlab.AssigneeIDValue) bool {
			json0, err := id0.MarshalJSON()
			if err != nil {
				panic(err)
			}

			json1, err := id1.MarshalJSON()
			if err != nil {
				panic(err)
			}

			return string(json0) == string(json1)
		}),
	}

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
						wantOpts := defaultOpts

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
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
						wantOpts := defaultOpts
						wantOpts.Confidential = nil // Confidential should be nil when true is passed (not filtered)

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
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
						wantOpts := defaultOpts
						wantOpts.State = new("all")

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
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
		{
			name: "fetch with various filters",
			args: map[string]any{
				"group_id":      "test-group",
				"labels":        "bug,feature",
				"milestone":     "v1.0",
				"author":        "test_author",
				"assignee":      "123",
				"search":        "critical issue",
				"created_after": "2023-01-01T00:00:00Z",
				"due_date":      "none",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListGroupIssues(gomock.Eq("test-group"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListGroupIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.Labels = &gitlab.LabelOptions{"bug", "feature"}
						wantOpts.Milestone = new("v1.0")
						wantOpts.AuthorUsername = new("test_author")
						wantOpts.AssigneeID = gitlab.AssigneeID(123)
						wantOpts.Search = new("critical issue")
						createdAfter, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
						wantOpts.CreatedAfter = &createdAfter
						wantOpts.DueDate = new("0")

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{
							{ID: 4001, Title: "Filtered group issue"},
						}, &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
						}, nil
					})
			},
			want: []*gitlab.Issue{
				{ID: 4001, Title: "Filtered group issue"},
			},
		},
		{
			name: "fetch with limit flag",
			args: map[string]any{
				"group_id": "test-group",
				// 349 is prime, so it is not a multiple of maxPerPage.
				"limit": 349,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListGroupIssues(gomock.Eq("test-group"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListGroupIssuesOptions, reqOpts ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						page := pageFromRequestOptions(t, reqOpts)

						if opts.ListOptions.PerPage <= 0 {
							return nil, nil, fmt.Errorf("per_page must be greater than 0")
						}

						index := (page - 1) * opts.ListOptions.PerPage
						if index >= 349 {
							return nil, nil, fmt.Errorf("page index %d exceeds limit %d (page = %d, per_page = %d)", index, 349, page, opts.ListOptions.PerPage)
						}

						return makeIssues(index, index+opts.ListOptions.PerPage), &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							NextPage: page + 1,
						}, nil
					}).AnyTimes()
			},
			want: makeIssues(0, 349),
		},
		{
			name: "order_by and sort_order",
			args: map[string]any{
				"group_id":   "test-group",
				"order_by":   "priority",
				"sort_order": "desc",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListGroupIssues(gomock.Eq("test-group"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListGroupIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.ListOptions.OrderBy = "priority"
						wantOpts.ListOptions.Sort = "desc"

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{
							{
								ID:    5001,
								IID:   501,
								Title: "High priority issue",
							},
						}, &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							NextPage: 0,
						}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:    5001,
					IID:   501,
					Title: "High priority issue",
				},
			},
		},
		{
			name: "empty response",
			args: map[string]any{
				"group_id": "test-group",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				// Mock ListGroupIssues
				mockIssues.EXPECT().
					ListGroupIssues(gomock.Eq("test-group"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)
			},
			want: []*gitlab.Issue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockIssues)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
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
	defaultOpts := gitlab.ListProjectIssuesOptions{
		State:        new(defaultIssueState),
		Confidential: new(false),
		ListOptions: gitlab.ListOptions{
			PerPage: maxPerPage,
		},
	}

	cmpOpts := []cmp.Option{timeComparer}

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
						wantOpts := defaultOpts

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
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
						wantOpts := defaultOpts
						wantOpts.Confidential = nil // Confidential should be nil when true is passed (not filtered)

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
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
						wantOpts := defaultOpts
						wantOpts.State = new("all")

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
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
		{
			name: "fetch with various filters",
			args: map[string]any{
				"project_id":     "test-project",
				"labels":         "enhancement",
				"milestone":      "v2.0",
				"iteration_id":   456,
				"author":         "789",
				"assignee":       "test_assignee",
				"search":         "urgent fix",
				"updated_before": "2024-12-31T23:59:59Z",
				"due_date":       "week",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListProjectIssues(gomock.Eq("test-project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListProjectIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						updatedBefore, _ := time.Parse(time.RFC3339, "2024-12-31T23:59:59Z")

						wantOpts := gitlab.ListProjectIssuesOptions{
							State:            new(defaultIssueState),
							Confidential:     new(false),
							Labels:           &gitlab.LabelOptions{"enhancement"},
							Milestone:        new("v2.0"),
							IterationID:      new(int64(456)),
							AuthorID:         new(int64(789)),
							AssigneeUsername: new("test_assignee"),
							Search:           new("urgent fix"),
							UpdatedBefore:    &updatedBefore,
							DueDate:          new("week"),
							ListOptions: gitlab.ListOptions{
								PerPage: maxPerPage,
							},
						}

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{
							{ID: 5001, Title: "Filtered project issue"},
						}, &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
						}, nil
					})
			},
			want: []*gitlab.Issue{
				{ID: 5001, Title: "Filtered project issue"},
			},
		},
		{
			name: "fetch with limit flag",
			args: map[string]any{
				"project_id": "test-project",
				// 349 is prime, so it is not a multiple of maxPerPage.
				"limit": 349,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListProjectIssues(gomock.Eq("test-project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListProjectIssuesOptions, reqOpts ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						page := pageFromRequestOptions(t, reqOpts)

						if opts.ListOptions.PerPage <= 0 {
							return nil, nil, fmt.Errorf("per_page must be greater than 0")
						}

						index := (page - 1) * opts.ListOptions.PerPage
						if index >= 349 {
							return nil, nil, fmt.Errorf("page index %d exceeds limit %d (page = %d, per_page = %d)", index, 349, page, opts.ListOptions.PerPage)
						}

						return makeIssues(index, index+opts.ListOptions.PerPage), &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							NextPage: page + 1,
						}, nil
					}).AnyTimes()
			},
			want: makeIssues(0, 349),
		},
		{
			name: "order_by and sort_order",
			args: map[string]any{
				"project_id": "test-project",
				"order_by":   "priority",
				"sort_order": "desc",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				mockIssues.EXPECT().
					ListProjectIssues(gomock.Eq("test-project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListProjectIssuesOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := defaultOpts
						wantOpts.ListOptions.OrderBy = "priority"
						wantOpts.ListOptions.Sort = "desc"

						if diff := cmp.Diff(&wantOpts, opts, cmpOpts...); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{
							{
								ID:    5001,
								IID:   501,
								Title: "High priority issue",
							},
						}, &gitlab.Response{
							Response: &http.Response{StatusCode: http.StatusOK},
							NextPage: 0,
						}, nil
					})
			},
			want: []*gitlab.Issue{
				{
					ID:    5001,
					IID:   501,
					Title: "High priority issue",
				},
			},
		},
		{
			name: "empty response",
			args: map[string]any{
				"project_id": "test-project",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface) {
				// Mock ListProjectIssues
				mockIssues.EXPECT().
					ListProjectIssues(gomock.Eq("test-project"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)
			},
			want: []*gitlab.Issue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockIssues)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
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

func makeIssues(from, to int64) []*gitlab.Issue {
	issues := make([]*gitlab.Issue, 0, to-from)

	for i := from; i < to; i++ {
		issues = append(issues, &gitlab.Issue{
			ID:    1000000 + i,
			IID:   i,
			Title: fmt.Sprintf("Issue %d", i),
			State: defaultIssueState,
		})
	}

	return issues
}
