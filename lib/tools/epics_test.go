package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	glabtest "gitlab.com/gitlab-org/api/client-go/testing"

	"gitlab.com/fforster/gitlab-mcp/lib/glctx"
	"gitlab.com/fforster/gitlab-mcp/lib/testing/gitlabtest"
)

func fakeGitlabClient() *gitlab.Client {
	epics := []*gitlab.Epic{
		{
			ID:      1001,
			IID:     1,
			GroupID: 10,
			State:   "opened",
		},
		{
			ID:       1002,
			IID:      2,
			GroupID:  10,
			ParentID: 1001, // parent in same group
			State:    "opened",
		},
		{
			ID:       1003,
			IID:      3,
			GroupID:  10,
			ParentID: 1004, // parent in other group
			State:    "closed",
		},
		{
			ID:      1004,
			IID:     4,
			GroupID: 20,
			State:   "opened",
		},
		{
			ID:       1005,
			IID:      5,
			GroupID:  20,
			ParentID: 1001, // parent in same group
			State:    "opened",
		},
		{
			ID:       1006,
			IID:      6,
			GroupID:  20,
			ParentID: 1002, // parent in other group
			State:    "opened",
		},
	}

	issues := []*gitlab.Issue{
		{
			ID: 101,
			Epic: &gitlab.Epic{
				ID: 1001,
			},
		},
		{
			ID: 102,
			Epic: &gitlab.Epic{
				ID: 1001,
			},
		},
	}

	return &gitlab.Client{
		Epics: &gitlabtest.FakeEpicsService{
			Epics: epics,
		},
		EpicIssues: &gitlabtest.FakeEpicIssuesService{
			Epics:  epics,
			Issues: issues,
		},
	}
}

func TestListGroupEpics(t *testing.T) {
	t.Parallel()

	var (
		gitlabClient = fakeGitlabClient()
		ctx          = glctx.WithClient(t.Context(), gitlabClient)
	)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	client := srv.Client()

	var req mcp.CallToolRequest

	req.Params.Name = "list_group_epics"
	req.Params.Arguments = map[string]any{
		"group_id": "10",
		"state":    "opened",
	}

	result, err := client.CallTool(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}

	var epics []*gitlab.Epic
	if err := unmarshalResult(result, &epics); err != nil {
		t.Fatal(err)
	}

	var (
		wantIDs = []int{1001, 1002}
		gotIDs  = epicIDs(epics)
	)

	if !cmp.Equal(gotIDs, wantIDs) {
		t.Errorf("got epics with ids %v, wanted %v", gotIDs, wantIDs)
	}
}

func TestGetEpic(t *testing.T) {
	t.Parallel()

	var (
		gitlabClient = fakeGitlabClient()
		ctx          = glctx.WithClient(t.Context(), gitlabClient)
	)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	client := srv.Client()

	var req mcp.CallToolRequest

	req.Params.Name = "get_epic"
	req.Params.Arguments = map[string]any{
		"group_id": "10",
		"epic_iid": 1,
	}

	result, err := client.CallTool(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}

	var epic gitlab.Epic
	if err := unmarshalResult(result, &epic); err != nil {
		t.Fatal(err)
	}

	if wantID := 1001; epic.ID != wantID {
		t.Errorf("got epic id %v, wanted %v", epic.ID, wantID)
	}
}

func TestGetEpicLinks(t *testing.T) {
	t.Parallel()

	var (
		gitlabClient = fakeGitlabClient()
		ctx          = glctx.WithClient(t.Context(), gitlabClient)
	)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	client := srv.Client()

	var req mcp.CallToolRequest

	req.Params.Name = "get_epic_links"
	req.Params.Arguments = map[string]any{
		"group_id": "10",
		"epic_iid": 1,
	}

	result, err := client.CallTool(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}

	var epics []*gitlab.Epic
	if err := unmarshalResult(result, &epics); err != nil {
		t.Fatal(err)
	}

	var (
		wantIDs = []int{1002, 1005}
		gotIDs  = epicIDs(epics)
	)

	if !cmp.Equal(gotIDs, wantIDs) {
		t.Errorf("got epics with ids %v, wanted %v", gotIDs, wantIDs)
	}
}

func TestListEpicIssues(t *testing.T) {
	t.Parallel()

	var (
		gitlabClient = fakeGitlabClient()
		ctx          = glctx.WithClient(t.Context(), gitlabClient)
	)

	srv := mcptest.NewUnstartedServer(t)
	AddTo(srv)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP test server: %v", err)
	}
	defer srv.Close()

	client := srv.Client()

	var req mcp.CallToolRequest

	req.Params.Name = "list_epic_issues"
	req.Params.Arguments = map[string]any{
		"group_id": "10",
		"epic_iid": 1,
	}

	result, err := client.CallTool(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}

	var issues []*gitlab.Issue
	if err := unmarshalResult(result, &issues); err != nil {
		t.Fatal(err)
	}

	var gotIDs []int
	for _, issue := range issues {
		gotIDs = append(gotIDs, issue.ID)
	}

	sort.Ints(gotIDs)

	wantIDs := []int{101, 102}

	if !cmp.Equal(gotIDs, wantIDs) {
		t.Errorf("got issues with ids %v, wanted %v", gotIDs, wantIDs)
	}
}

func epicIDs(epics []*gitlab.Epic) []int {
	var ids []int
	for _, epic := range epics {
		ids = append(ids, epic.ID)
	}

	sort.Ints(ids)

	return ids
}

func unmarshalResult(res *mcp.CallToolResult, v any) error {
	s, err := resultToString(res)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(s), v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

func resultToString(res *mcp.CallToolResult) (string, error) {
	var b strings.Builder

	for _, c := range res.Content {
		tc, ok := mcp.AsTextContent(c)
		if !ok {
			return "", fmt.Errorf("content is not text: %T", c)
		}

		b.WriteString(tc.Text)
	}

	if res.IsError {
		return "", fmt.Errorf("tool returned error: %s", b.String())
	}

	return b.String(), nil
}

func TestCreateEpic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(epicsService *glabtest.MockEpicsServiceInterface, groupID string, ret *gitlab.Epic)
		want      *gitlab.Epic
		wantError bool
	}{
		{
			name: "minimal successful call",
			args: map[string]any{
				"title": "Test title",
			},
			setupMock: func(epicsService *glabtest.MockEpicsServiceInterface, groupID string, ret *gitlab.Epic) {
				epicsService.EXPECT().CreateEpic(gomock.Eq(groupID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateEpicOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Epic, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateEpicOptions{
							Title: gitlab.Ptr("Test title"),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateEpic() options mismatch (-want/+got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: &gitlab.Epic{
				Title: "Test title",
			},
		},
		{
			name: "using all the options",
			args: map[string]any{
				"title":        "Test title",
				"labels":       "type::testing",
				"description":  "Test description",
				"confidential": false,
				"due_date":     "2024-09-17",
				"parent_id":    1001,
			},
			setupMock: func(epicsService *glabtest.MockEpicsServiceInterface, groupID string, ret *gitlab.Epic) {
				epicsService.EXPECT().CreateEpic(gomock.Eq(groupID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateEpicOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Epic, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateEpicOptions{
							Title:          gitlab.Ptr("Test title"),
							Labels:         &gitlab.LabelOptions{"type::testing"},
							Description:    gitlab.Ptr("Test description"),
							Confidential:   gitlab.Ptr(false),
							DueDateFixed:   mustISOTime("2024-09-17"),
							DueDateIsFixed: gitlab.Ptr(true),
							ParentID:       gitlab.Ptr(1001),
						}

						if diff := cmp.Diff(wantOpts, opts, isoTimeComparer); diff != "" {
							return nil, nil, fmt.Errorf("CreateEpic() options mismatch (-want/+got):\n%s", diff)
						}

						return ret, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: &gitlab.Epic{
				Title:          "Test title",
				Labels:         []string{"type::testing"},
				Description:    "Test description",
				Confidential:   false,
				DueDateFixed:   mustISOTime("2024-09-17"),
				DueDateIsFixed: true,
				ParentID:       1001,
			},
		},
		{
			name:      "missing required argument",
			args:      map[string]any{},
			setupMock: func(*glabtest.MockEpicsServiceInterface, string, *gitlab.Epic) { /* no-op */ },
			wantError: true,
		},
		{
			name: "invalid due date",
			args: map[string]any{
				"title":    "Test title",
				"due_date": "09/17/2024",
			},
			setupMock: func(*glabtest.MockEpicsServiceInterface, string, *gitlab.Epic) { /* no-op */ },
			wantError: true,
		},
		{
			name: "internal server error",
			args: map[string]any{
				"title": "Test title",
			},
			setupMock: func(epicsService *glabtest.MockEpicsServiceInterface, groupID string, ret *gitlab.Epic) {
				epicsService.EXPECT().CreateEpic(gomock.Eq(groupID), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusInternalServerError},
					}, fmt.Errorf("internal server error"))
			},
			wantError: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			groupID := fmt.Sprintf("test/group%d", i)
			tt.args["group_id"] = groupID

			tt.setupMock(gitlabClient.MockEpics, groupID, tt.want)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest
			req.Params.Name = "create_epic"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("create_epic error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			var got gitlab.Epic
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, &got, isoTimeComparer); diff != "" {
				t.Errorf("epic mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}
