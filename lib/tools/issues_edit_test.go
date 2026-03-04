package tools

import (
	"fmt"
	"net/http"
	"strings"
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

func TestEditIssue(t *testing.T) {
	tests := []struct {
		name              string
		args              map[string]any
		initialIssueState *gitlab.Issue // State of the issue before editing
		setupMock         func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue)
		wantError         bool
		wantErrorResponse bool // If the tool call itself should return an error message
		wantResult        *gitlab.Issue
	}{
		{
			name: "update title",
			args: map[string]any{
				"project_id": "test/project",
				"issue_iid":  1,
				"title":      "New Title",
			},
			initialIssueState: &gitlab.Issue{IID: 1, Title: "Old Title"},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().
					GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(1)), gomock.Any()).
					Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().
					UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(1)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							Title: new("New Title"),
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					}).
					Return(ret, newResponse(http.StatusOK), nil)
			},
			wantResult: &gitlab.Issue{IID: 1, Title: "New Title"},
		},
		{
			name: "update description",
			args: map[string]any{
				"project_id":  "test/project",
				"issue_iid":   2,
				"description": "New Description",
			},
			initialIssueState: &gitlab.Issue{IID: 2, Description: "Old Description"},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(2)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(2)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							Description: new("New Description"),
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch for description (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{IID: 2, Description: "New Description"},
		},
		{
			name: "clear assignees",
			args: map[string]any{
				"project_id":   "test/project",
				"issue_iid":    3,
				"assignee_ids": "-", // Signal to clear assignees
			},
			initialIssueState: &gitlab.Issue{IID: 3, Assignees: []*gitlab.IssueAssignee{{ID: 100}}},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(3)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(3)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							AssigneeIDs: &[]int64{},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch for clearing assignees (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{IID: 3, Assignees: []*gitlab.IssueAssignee{}},
		},
		{
			name: "set multiple assignees",
			args: map[string]any{
				"project_id":   "test/project",
				"issue_iid":    4,
				"assignee_ids": "101,102",
			},
			initialIssueState: &gitlab.Issue{IID: 4},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(4)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(4)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							AssigneeIDs: &[]int64{101, 102},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch for setting assignees (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{IID: 4, Assignees: []*gitlab.IssueAssignee{{ID: 101}, {ID: 102}}},
		},
		{
			name: "add labels",
			args: map[string]any{
				"project_id": "test/project",
				"issue_iid":  5,
				"add_labels": "bug,critical",
			},
			initialIssueState: &gitlab.Issue{IID: 5},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(5)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(5)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							AddLabels: &gitlab.LabelOptions{"bug", "critical"},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch for adding labels (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{IID: 5, Labels: []string{"bug", "critical"}},
		},
		{
			name: "remove labels",
			args: map[string]any{
				"project_id":    "test/project",
				"issue_iid":     6,
				"remove_labels": "todo",
			},
			initialIssueState: &gitlab.Issue{IID: 6, Labels: []string{"todo", "bug"}},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(6)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(6)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							RemoveLabels: &gitlab.LabelOptions{"todo"},
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch for removing labels (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{IID: 6, Labels: []string{"bug"}}, // Assuming 'todo' is removed
		},
		{
			name: "close issue",
			args: map[string]any{
				"project_id":  "test/project",
				"issue_iid":   7,
				"state_event": "close",
			},
			initialIssueState: &gitlab.Issue{IID: 7, State: "opened"},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(7)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(7)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							StateEvent: new("close"),
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch for closing issue (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{IID: 7, State: "closed"},
		},
		{
			name: "reopen issue",
			args: map[string]any{
				"project_id":  "test/project",
				"issue_iid":   8,
				"state_event": "reopen",
			},
			initialIssueState: &gitlab.Issue{IID: 8, State: "closed"},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(8)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(8)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							StateEvent: new("reopen"),
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch for reopening issue (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{IID: 8, State: "opened"},
		},
		{
			name: "make issue confidential",
			args: map[string]any{
				"project_id":   "test/project",
				"issue_iid":    9,
				"confidential": true,
			},
			initialIssueState: &gitlab.Issue{IID: 9, Confidential: false},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(9)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(9)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							Confidential: new(true),
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch for making confidential (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{IID: 9, Confidential: true},
		},
		{
			name: "edit confidential issue with confidential_flag=true",
			args: map[string]any{
				"project_id":   "test/project",
				"issue_iid":    10,
				"title":        "Secure New Title",
				"confidential": true, // Required to edit confidential issue
			},
			initialIssueState: &gitlab.Issue{IID: 10, Title: "Secure Old Title", Confidential: true},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(10)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(10)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							Title:        new("Secure New Title"),
							Confidential: new(true), // confidential flag is passed through
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch for editing confidential issue (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{IID: 10, Title: "Secure New Title", Confidential: true},
		},
		{
			name: "fail to edit confidential issue without confidential_flag",
			args: map[string]any{
				"project_id": "test/project",
				"issue_iid":  11,
				"title":      "Attempted Title Update",
			},
			initialIssueState: &gitlab.Issue{IID: 11, Title: "Confidential", Confidential: true},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, _ *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(11)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
			},
			wantErrorResponse: true,
		},
		{
			name: "lock discussion",
			args: map[string]any{
				"project_id":        "test/project",
				"issue_iid":         12,
				"discussion_locked": true,
			},
			initialIssueState: &gitlab.Issue{IID: 12, DiscussionLocked: false},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(12)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(12)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							DiscussionLocked: new(true),
						}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch for locking discussion (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{IID: 12, DiscussionLocked: true},
		},
		{
			name: "set due date",
			args: map[string]any{
				"project_id": "test/project",
				"issue_iid":  15,
				"due_date":   "2025-07-01",
			},
			initialIssueState: &gitlab.Issue{IID: 15},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(15)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(15)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							DueDate: mustISOTime("2025-07-01"),
						}

						if diff := cmp.Diff(wantOpts, opts, isoTimeComparer); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch (-want/+got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{
				IID:     15,
				DueDate: mustISOTime("2025-07-01"),
			},
		},
		{
			name: "set weight",
			args: map[string]any{
				"project_id": "test/project",
				"issue_iid":  16,
				"weight":     42,
			},
			initialIssueState: &gitlab.Issue{IID: 16},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, ret *gitlab.Issue) {
				mockIssues.EXPECT().GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(16)), gomock.Any()).Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(16)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						wantOpts := &gitlab.UpdateIssueOptions{
							Weight: new(int64(42)),
						}

						if diff := cmp.Diff(wantOpts, opts, isoTimeComparer); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch (-want/+got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.Issue{
				IID:    16,
				Weight: 42,
			},
		},
		{
			name: "GetIssue returns error",
			args: map[string]any{
				"project_id": "test/project",
				"issue_iid":  13,
				"title":      "New Title",
			},
			initialIssueState: &gitlab.Issue{IID: 13},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, _ *gitlab.Issue, _ *gitlab.Issue) {
				mockIssues.EXPECT().
					GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(13)), gomock.Any()).
					Return(nil, newResponse(http.StatusNotFound), fmt.Errorf("issue not found"))
			},
			wantError: true,
		},
		{
			name: "UpdateIssue returns error",
			args: map[string]any{
				"project_id": "test/project",
				"issue_iid":  14,
				"title":      "New Title",
			},
			initialIssueState: &gitlab.Issue{IID: 14},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, initialIssue *gitlab.Issue, _ *gitlab.Issue) {
				mockIssues.EXPECT().
					GetIssue(gomock.Eq("test/project"), gomock.Eq(int64(14)), gomock.Any()).
					Return(initialIssue, newResponse(http.StatusOK), nil)
				mockIssues.EXPECT().
					UpdateIssue(gomock.Eq("test/project"), gomock.Eq(int64(14)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.UpdateIssueOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Issue, *gitlab.Response, error) {
						// Check that opts are what we expect even when an error is returned
						wantOpts := &gitlab.UpdateIssueOptions{Title: new("New Title")}
						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("UpdateIssue() options mismatch before error (-want +got):\n%s", diff)
						}

						return nil, newResponse(http.StatusInternalServerError), fmt.Errorf("update failed")
					})
			},
			wantError:  true,
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			// tt.setupMock now doesn't need expectedUpdateOpts
			tt.setupMock(gitlabClient.MockIssues, tt.initialIssueState, tt.wantResult)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "edit_issue"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("editIssue error mismatch, got: %v (%v), want error: %v", err, result, tt.wantError)
			}

			if err != nil {
				return
			}

			if result.IsError != tt.wantErrorResponse {
				t.Errorf("unexpected inline error status, got: %v (%s), want: %v", result.IsError, result.Content, tt.wantErrorResponse)
			}

			if result.IsError || tt.wantError {
				return
			}

			var got gitlab.Issue
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v -- result was: %s", err, result.Content)
			}

			if diff := cmp.Diff(tt.wantResult, &got, isoTimeComparer); diff != "" {
				t.Errorf("issue mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestIssueTimeTracking(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		setupMock  func(mockIssues *glabtest.MockIssuesServiceInterface, ret *gitlab.TimeStats)
		wantErrors []string
		wantResult *gitlab.TimeStats
	}{
		{
			name: "set time estimate",
			args: map[string]any{
				"project_id":    "test/project",
				"issue_iid":     1,
				"time_estimate": 40 * 3600,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, ret *gitlab.TimeStats) {
				mockIssues.EXPECT().
					SetTimeEstimate(gomock.Eq("test/project"), gomock.Eq(int64(1)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.SetTimeEstimateOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.TimeStats, *gitlab.Response, error) {
						wantOpts := &gitlab.SetTimeEstimateOptions{
							Duration: new(fmt.Sprintf("%ds", 40*3600)),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("SetTimeEstimate() options mismatch (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.TimeStats{
				TimeEstimate: 40 * 3600,
			},
		},
		{
			name: "reset time estimate",
			args: map[string]any{
				"project_id":    "test/project",
				"issue_iid":     2,
				"time_estimate": 0,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, ret *gitlab.TimeStats) {
				mockIssues.EXPECT().
					ResetTimeEstimate(gomock.Eq("test/project"), gomock.Eq(int64(2)), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, _ ...gitlab.RequestOptionFunc) (*gitlab.TimeStats, *gitlab.Response, error) {
						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.TimeStats{},
		},
		{
			name: "add spent time",
			args: map[string]any{
				"project_id":             "test/project",
				"issue_iid":              1,
				"add_spent_time_seconds": 3600,
				"add_spent_time_summary": "Writing unit tests",
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, ret *gitlab.TimeStats) {
				mockIssues.EXPECT().
					AddSpentTime(gomock.Eq("test/project"), gomock.Eq(int64(1)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, opts *gitlab.AddSpentTimeOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.TimeStats, *gitlab.Response, error) {
						wantOpts := &gitlab.AddSpentTimeOptions{
							Duration: new("3600s"),
							Summary:  new("Writing unit tests"),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, newResponse(http.StatusBadRequest), fmt.Errorf("AddSpentTime() options mismatch (-want +got):\n%s", diff)
						}

						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.TimeStats{
				TimeEstimate: 40 * 3600,
			},
		},
		{
			name: "reset spent time",
			args: map[string]any{
				"project_id":       "test/project",
				"issue_iid":        4,
				"reset_spent_time": true,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, ret *gitlab.TimeStats) {
				mockIssues.EXPECT().
					ResetSpentTime(gomock.Eq("test/project"), gomock.Eq(int64(4)), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, _ ...gitlab.RequestOptionFunc) (*gitlab.TimeStats, *gitlab.Response, error) {
						return ret, newResponse(http.StatusOK), nil
					})
			},
			wantResult: &gitlab.TimeStats{},
		},
		{
			name: "returns all errors",
			args: map[string]any{
				"project_id":       "test/project",
				"issue_iid":        5,
				"time_estimate":    0,
				"reset_spent_time": true,
			},
			setupMock: func(mockIssues *glabtest.MockIssuesServiceInterface, ret *gitlab.TimeStats) {
				mockIssues.EXPECT().
					ResetTimeEstimate(gomock.Eq("test/project"), gomock.Eq(int64(5)), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, _ ...gitlab.RequestOptionFunc) (*gitlab.TimeStats, *gitlab.Response, error) {
						return nil, newResponse(http.StatusInternalServerError), fmt.Errorf("reset time estimate failure")
					})
				mockIssues.EXPECT().
					ResetSpentTime(gomock.Eq("test/project"), gomock.Eq(int64(5)), gomock.Any()).
					DoAndReturn(func(_ any, _ int64, _ ...gitlab.RequestOptionFunc) (*gitlab.TimeStats, *gitlab.Response, error) {
						return nil, newResponse(http.StatusInternalServerError), fmt.Errorf("reset time spent failure")
					})
			},
			wantErrors: []string{"reset time estimate failure", "reset time spent failure"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			// tt.setupMock now doesn't need expectedUpdateOpts
			tt.setupMock(gitlabClient.MockIssues, tt.wantResult)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "issue_time_tracking"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr, wantErr := err != nil, len(tt.wantErrors) != 0; gotErr != wantErr {
				t.Errorf("issue_time_tracking error mismatch, got: %v (%v), want error: %v", err, result, tt.wantErrors)
			}

			if gotErr, wantErr := err != nil, len(tt.wantErrors) != 0; gotErr && wantErr {
				for _, wantErr := range tt.wantErrors {
					if !strings.Contains(err.Error(), wantErr) {
						t.Errorf("issue_time_tracking error %q did not contain expected %q", err, wantErr)
					}
				}
			}

			if err != nil {
				return
			}

			var got gitlab.TimeStats
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v -- result was: %s", err, result.Content)
			}

			if diff := cmp.Diff(tt.wantResult, &got); diff != "" {
				t.Errorf("issue mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}

func mustISOTime(s string) *gitlab.ISOTime {
	t, err := gitlab.ParseISOTime(s)
	if err != nil {
		panic(err)
	}

	return &t
}

var isoTimeComparer = cmp.Comparer(func(a, b gitlab.ISOTime) bool {
	return time.Time(a).Equal(time.Time(b))
})

func newResponse(statusCode int) *gitlab.Response {
	return &gitlab.Response{
		Response: &http.Response{
			StatusCode: statusCode,
		},
	}
}
