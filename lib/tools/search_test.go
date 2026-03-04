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

func TestSearch(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockSearchServiceInterface)
		wantError bool
	}{
		// ------------------------------------------------------------------ global
		{
			name: "global issues search",
			args: map[string]any{
				"search": "bug",
				"scope":  "issues",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Issues(gomock.Eq("bug"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(query string, opt *gitlab.SearchOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
						wantOpt := &gitlab.SearchOptions{
							ListOptions: gitlab.ListOptions{PerPage: maxPerPage},
						}
						if diff := cmp.Diff(wantOpt, opt); diff != "" {
							return nil, nil, fmt.Errorf("options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Issue{{ID: 1, Title: "Bug report"}}, okResponse, nil
					})
			},
		},
		{
			name: "global projects search",
			args: map[string]any{
				"search": "myproject",
				"scope":  "projects",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Projects(gomock.Eq("myproject"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Project{{ID: 1, Name: "myproject"}}, okResponse, nil)
			},
		},
		{
			name: "global merge_requests search",
			args: map[string]any{
				"search": "fix",
				"scope":  "merge_requests",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					MergeRequests(gomock.Eq("fix"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.MergeRequest{{BasicMergeRequest: gitlab.BasicMergeRequest{ID: 1, Title: "Fix bug"}}}, okResponse, nil)
			},
		},
		{
			name: "global milestones search",
			args: map[string]any{
				"search": "v1",
				"scope":  "milestones",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Milestones(gomock.Eq("v1"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Milestone{{ID: 1, Title: "v1.0"}}, okResponse, nil)
			},
		},
		{
			name: "global snippet_titles search",
			args: map[string]any{
				"search": "gist",
				"scope":  "snippet_titles",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					SnippetTitles(gomock.Eq("gist"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Snippet{{ID: 1, Title: "My gist"}}, okResponse, nil)
			},
		},
		{
			name: "global wiki_blobs search",
			args: map[string]any{
				"search": "install",
				"scope":  "wiki_blobs",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					WikiBlobs(gomock.Eq("install"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Wiki{{Slug: "installation"}}, okResponse, nil)
			},
		},
		{
			name: "global commits search",
			args: map[string]any{
				"search": "chore",
				"scope":  "commits",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Commits(gomock.Eq("chore"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Commit{{ID: "abc123", Title: "chore: tidy"}}, okResponse, nil)
			},
		},
		{
			name: "global blobs search",
			args: map[string]any{
				"search": "TODO",
				"scope":  "blobs",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Blobs(gomock.Eq("TODO"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Blob{{Filename: "main.go", Data: "// TODO"}}, okResponse, nil)
			},
		},
		{
			name: "global users search",
			args: map[string]any{
				"search": "alice",
				"scope":  "users",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Users(gomock.Eq("alice"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.User{{ID: 1, Username: "alice"}}, okResponse, nil)
			},
		},
		{
			name: "global notes scope fails — requires project_id",
			args: map[string]any{
				"search": "review",
				"scope":  "notes",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},

		// ------------------------------------------------------------------ group
		{
			name: "group issues search",
			args: map[string]any{
				"search":   "bug",
				"scope":    "issues",
				"group_id": "my-group",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					IssuesByGroup(gomock.Eq("my-group"), gomock.Eq("bug"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Issue{{ID: 2, Title: "Group bug"}}, okResponse, nil)
			},
		},
		{
			name: "group projects search",
			args: map[string]any{
				"search":   "myproject",
				"scope":    "projects",
				"group_id": "my-group",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					ProjectsByGroup(gomock.Eq("my-group"), gomock.Eq("myproject"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Project{{ID: 1, Name: "myproject"}}, okResponse, nil)
			},
		},
		{
			name: "group merge_requests search",
			args: map[string]any{
				"search":   "fix",
				"scope":    "merge_requests",
				"group_id": "my-group",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					MergeRequestsByGroup(gomock.Eq("my-group"), gomock.Eq("fix"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.MergeRequest{{BasicMergeRequest: gitlab.BasicMergeRequest{ID: 10, Title: "Fix typo"}}}, okResponse, nil)
			},
		},
		{
			name: "group milestones search",
			args: map[string]any{
				"search":   "v2",
				"scope":    "milestones",
				"group_id": "my-group",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					MilestonesByGroup(gomock.Eq("my-group"), gomock.Eq("v2"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Milestone{{ID: 2, Title: "v2.0"}}, okResponse, nil)
			},
		},
		{
			name: "group wiki_blobs search",
			args: map[string]any{
				"search":   "setup",
				"scope":    "wiki_blobs",
				"group_id": "my-group",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					WikiBlobsByGroup(gomock.Eq("my-group"), gomock.Eq("setup"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Wiki{{Slug: "setup"}}, okResponse, nil)
			},
		},
		{
			name: "group commits search",
			args: map[string]any{
				"search":   "refactor",
				"scope":    "commits",
				"group_id": "my-group",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					CommitsByGroup(gomock.Eq("my-group"), gomock.Eq("refactor"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Commit{{ID: "def456", Title: "refactor: extract helper"}}, okResponse, nil)
			},
		},
		{
			name: "group blobs search",
			args: map[string]any{
				"search":   "FIXME",
				"scope":    "blobs",
				"group_id": "my-group",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					BlobsByGroup(gomock.Eq("my-group"), gomock.Eq("FIXME"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Blob{{Filename: "util.go", Data: "// FIXME"}}, okResponse, nil)
			},
		},
		{
			name: "group users search",
			args: map[string]any{
				"search":   "bob",
				"scope":    "users",
				"group_id": "my-group",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					UsersByGroup(gomock.Eq("my-group"), gomock.Eq("bob"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.User{{ID: 2, Username: "bob"}}, okResponse, nil)
			},
		},
		{
			name: "group snippet_titles scope fails — not available for group searches",
			args: map[string]any{
				"search":   "snippet",
				"scope":    "snippet_titles",
				"group_id": "my-group",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},
		{
			name: "group notes scope fails — requires project_id",
			args: map[string]any{
				"search":   "review",
				"scope":    "notes",
				"group_id": "my-group",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},

		// ------------------------------------------------------------------ project
		{
			name: "project issues search",
			args: map[string]any{
				"search":     "crash",
				"scope":      "issues",
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					IssuesByProject(gomock.Eq("owner/project"), gomock.Eq("crash"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Issue{{ID: 3, Title: "Crash on startup"}}, okResponse, nil)
			},
		},
		{
			name: "project merge_requests search",
			args: map[string]any{
				"search":     "hotfix",
				"scope":      "merge_requests",
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					MergeRequestsByProject(gomock.Eq("owner/project"), gomock.Eq("hotfix"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.MergeRequest{{BasicMergeRequest: gitlab.BasicMergeRequest{ID: 20, Title: "Hotfix"}}}, okResponse, nil)
			},
		},
		{
			name: "project milestones search",
			args: map[string]any{
				"search":     "v3",
				"scope":      "milestones",
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					MilestonesByProject(gomock.Eq("owner/project"), gomock.Eq("v3"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Milestone{{ID: 3, Title: "v3.0"}}, okResponse, nil)
			},
		},
		{
			name: "project notes search",
			args: map[string]any{
				"search":     "review",
				"scope":      "notes",
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					NotesByProject(gomock.Eq("owner/project"), gomock.Eq("review"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Note{{ID: 5, Body: "Please review"}}, okResponse, nil)
			},
		},
		{
			name: "project wiki_blobs search",
			args: map[string]any{
				"search":     "deploy",
				"scope":      "wiki_blobs",
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					WikiBlobsByProject(gomock.Eq("owner/project"), gomock.Eq("deploy"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Wiki{{Slug: "deployment"}}, okResponse, nil)
			},
		},
		{
			name: "project commits search",
			args: map[string]any{
				"search":     "init",
				"scope":      "commits",
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					CommitsByProject(gomock.Eq("owner/project"), gomock.Eq("init"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Commit{{ID: "ghi789", Title: "init: bootstrap"}}, okResponse, nil)
			},
		},
		{
			name: "project blobs search",
			args: map[string]any{
				"search":     "HACK",
				"scope":      "blobs",
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					BlobsByProject(gomock.Eq("owner/project"), gomock.Eq("HACK"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.Blob{{Filename: "hack.go", Data: "// HACK"}}, okResponse, nil)
			},
		},
		{
			name: "project users search",
			args: map[string]any{
				"search":     "carol",
				"scope":      "users",
				"project_id": "owner/project",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					UsersByProject(gomock.Eq("owner/project"), gomock.Eq("carol"), gomock.Any(), gomock.Any()).
					Return([]*gitlab.User{{ID: 3, Username: "carol"}}, okResponse, nil)
			},
		},
		{
			name: "project projects scope fails — not available for project searches",
			args: map[string]any{
				"search":     "my-project",
				"scope":      "projects",
				"project_id": "owner/project",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},
		{
			name: "project snippet_titles scope fails — not available for project searches",
			args: map[string]any{
				"search":     "snippet",
				"scope":      "snippet_titles",
				"project_id": "owner/project",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},

		// ------------------------------------------------------------------ invalid scope (default branch)
		{
			name: "global unknown scope fails",
			args: map[string]any{
				"search": "anything",
				"scope":  "invalid_scope",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},
		{
			name: "group unknown scope fails",
			args: map[string]any{
				"search":   "anything",
				"scope":    "invalid_scope",
				"group_id": "my-group",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},
		{
			name: "project unknown scope fails",
			args: map[string]any{
				"search":     "anything",
				"scope":      "invalid_scope",
				"project_id": "owner/project",
			},
			setupMock: func(_ *glabtest.MockSearchServiceInterface) {},
			wantError: true,
		},

		// ------------------------------------------------------------------ misc
		{
			name: "searchAllGlobal API error propagated",
			args: map[string]any{
				"search": "error",
				"scope":  "issues",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Issues(gomock.Eq("error"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, gitlab.ErrNotFound)
			},
			wantError: true,
		},
		{
			name: "searchAllNamespace API error propagated",
			args: map[string]any{
				"search":   "error",
				"scope":    "issues",
				"group_id": "my-group",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					IssuesByGroup(gomock.Eq("my-group"), gomock.Eq("error"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, gitlab.ErrNotFound)
			},
			wantError: true,
		},
		{
			name: "blobs search passes ref option",
			args: map[string]any{
				"search": "TODO",
				"scope":  "blobs",
				"ref":    "main",
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Blobs(gomock.Eq("TODO"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ string, opt *gitlab.SearchOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Blob, *gitlab.Response, error) {
						if opt.Ref == nil || *opt.Ref != "main" {
							return nil, nil, fmt.Errorf("expected ref %q, got %v", "main", opt.Ref)
						}

						return []*gitlab.Blob{{Filename: "main.go"}}, okResponse, nil
					})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockSearch)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "search"
			req.Params.Arguments = tt.args

			_, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("search error mismatch, got: %v, want error: %v", err, tt.wantError)
			}
		})
	}
}

func TestSearch_Limit(t *testing.T) {
	okResponse := &gitlab.Response{Response: &http.Response{StatusCode: http.StatusOK}}

	allIssues := []*gitlab.Issue{
		{ID: 1, Title: "First issue"},
		{ID: 2, Title: "Second issue"},
		{ID: 3, Title: "Third issue"},
	}

	wantIssues := []*gitlab.Issue{
		{ID: 1, Title: "First issue"},
	}

	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockSearchServiceInterface)
	}{
		{
			name: "global search respects limit",
			args: map[string]any{
				"search": "bug",
				"scope":  "issues",
				"limit":  1,
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					Issues(gomock.Eq("bug"), gomock.Any(), gomock.Any()).
					Return(allIssues, okResponse, nil)
			},
		},
		{
			name: "group search respects limit",
			args: map[string]any{
				"search":   "bug",
				"scope":    "issues",
				"group_id": "my-group",
				"limit":    1,
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					IssuesByGroup(gomock.Eq("my-group"), gomock.Eq("bug"), gomock.Any(), gomock.Any()).
					Return(allIssues, okResponse, nil)
			},
		},
		{
			name: "project search respects limit",
			args: map[string]any{
				"search":     "bug",
				"scope":      "issues",
				"project_id": "owner/project",
				"limit":      1,
			},
			setupMock: func(m *glabtest.MockSearchServiceInterface) {
				m.EXPECT().
					IssuesByProject(gomock.Eq("owner/project"), gomock.Eq("bug"), gomock.Any(), gomock.Any()).
					Return(allIssues, okResponse, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockSearch)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "search"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var got []*gitlab.Issue
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(wantIssues, got); diff != "" {
				t.Errorf("issues mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}
