package tools

import (
	"errors"
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

func TestWikiFormatValue(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		want      gitlab.WikiFormatValue
		wantError bool
	}{
		{
			name:   "markdown",
			format: "markdown",
			want:   gitlab.WikiFormatMarkdown,
		},
		{
			name:   "rdoc",
			format: "rdoc",
			want:   gitlab.WikiFormatRDoc,
		},
		{
			name:   "asciidoc",
			format: "asciidoc",
			want:   gitlab.WikiFormatASCIIDoc,
		},
		{
			name:   "org",
			format: "org",
			want:   gitlab.WikiFormatOrg,
		},
		{
			name:      "unknown format",
			format:    "textile",
			wantError: true,
		},
		{
			name:      "empty format",
			format:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := wikiFormatValue(tt.format)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Fatalf("wikiFormatValue(%q) error mismatch, got: %v, want error: %v", tt.format, err, tt.wantError)
			}

			if tt.wantError {
				if !errors.Is(err, ErrInvalidArgument) {
					t.Errorf("wikiFormatValue(%q) error = %v, want it to wrap ErrInvalidArgument", tt.format, err)
				}

				return
			}

			if got == nil {
				t.Fatalf("wikiFormatValue(%q) = nil, want %q", tt.format, tt.want)
			}

			if *got != tt.want {
				t.Errorf("wikiFormatValue(%q) = %q, want %q", tt.format, *got, tt.want)
			}
		})
	}
}

func TestListWikiPages(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockWikisServiceInterface)
		wantError bool
		want      []*gitlab.Wiki
	}{
		{
			name: "list wiki pages by project path",
			args: map[string]any{
				"project_id": "test/project",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					ListWikis(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListWikisOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Wiki, *gitlab.Response, error) {
						wantOpts := &gitlab.ListWikisOptions{
							WithContent: new(false),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("ListWikis() options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Wiki{
								{
									Slug:   "home",
									Title:  "Home",
									Format: gitlab.WikiFormatMarkdown,
								},
								{
									Slug:   "Setup-Dokploy-Host",
									Title:  "Setup Dokploy Host",
									Format: gitlab.WikiFormatMarkdown,
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
							}, nil
					})
			},
			want: []*gitlab.Wiki{
				{
					Slug:   "home",
					Title:  "Home",
					Format: gitlab.WikiFormatMarkdown,
				},
				{
					Slug:   "Setup-Dokploy-Host",
					Title:  "Setup Dokploy Host",
					Format: gitlab.WikiFormatMarkdown,
				},
			},
		},
		{
			name: "list wiki pages by numeric project ID with content",
			args: map[string]any{
				"project_id":   "42",
				"with_content": true,
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					ListWikis(gomock.Eq(int64(42)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.ListWikisOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Wiki, *gitlab.Response, error) {
						wantOpts := &gitlab.ListWikisOptions{
							WithContent: new(true),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("ListWikis() options mismatch (-want/+got):\n%s", diff)
						}

						return []*gitlab.Wiki{
								{
									Slug:     "home",
									Title:    "Home",
									Content:  "# Home\n\nWelcome!",
									Encoding: "UTF-8",
									Format:   gitlab.WikiFormatMarkdown,
								},
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
							}, nil
					})
			},
			want: []*gitlab.Wiki{
				{
					Slug:     "home",
					Title:    "Home",
					Content:  "# Home\n\nWelcome!",
					Encoding: "UTF-8",
					Format:   gitlab.WikiFormatMarkdown,
				},
			},
		},
		{
			name: "empty wiki",
			args: map[string]any{
				"project_id": "test/project",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					ListWikis(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusOK},
					}, nil)
			},
			want: []*gitlab.Wiki{},
		},
		{
			name: "API error",
			args: map[string]any{
				"project_id": "test/project",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					ListWikis(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusNotFound},
					}, gitlab.ErrNotFound)
			},
			wantError: true,
		},
		{
			name: "missing required project_id",
			args: map[string]any{},
			setupMock: func(_ *glabtest.MockWikisServiceInterface) {
				// No mock expectations - should fail at validation.
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockWikis)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "list_wiki_pages"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("listWikiPages error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			var got []*gitlab.Wiki
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("wiki pages mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestGetWikiPage(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockWikisServiceInterface)
		wantError bool
		want      *gitlab.Wiki
	}{
		{
			name: "get wiki page",
			args: map[string]any{
				"project_id": "test/project",
				"slug":       "Setup-Dokploy-Host",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					GetWikiPage(gomock.Eq("test/project"), gomock.Eq("Setup-Dokploy-Host"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ string, opts *gitlab.GetWikiPageOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Wiki, *gitlab.Response, error) {
						wantOpts := &gitlab.GetWikiPageOptions{
							RenderHTML: new(false),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("GetWikiPage() options mismatch (-want/+got):\n%s", diff)
						}

						return &gitlab.Wiki{
								Slug:     "Setup-Dokploy-Host",
								Title:    "Setup Dokploy Host",
								Content:  "# Setup\n\nSteps.",
								Encoding: "UTF-8",
								Format:   gitlab.WikiFormatMarkdown,
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
							}, nil
					})
			},
			want: &gitlab.Wiki{
				Slug:     "Setup-Dokploy-Host",
				Title:    "Setup Dokploy Host",
				Content:  "# Setup\n\nSteps.",
				Encoding: "UTF-8",
				Format:   gitlab.WikiFormatMarkdown,
			},
		},
		{
			name: "get wiki page of a subgroup path, rendered as HTML at a specific version",
			args: map[string]any{
				"project_id":  "17",
				"slug":        "group/subgroup-page",
				"version":     "c0ffee1234567890",
				"render_html": true,
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					GetWikiPage(gomock.Eq(int64(17)), gomock.Eq("group/subgroup-page"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ string, opts *gitlab.GetWikiPageOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Wiki, *gitlab.Response, error) {
						wantOpts := &gitlab.GetWikiPageOptions{
							RenderHTML: new(true),
							Version:    new("c0ffee1234567890"),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("GetWikiPage() options mismatch (-want/+got):\n%s", diff)
						}

						return &gitlab.Wiki{
								Slug:    "group/subgroup-page",
								Title:   "Subgroup page",
								Content: "<h1>Subgroup page</h1>",
								Format:  gitlab.WikiFormatMarkdown,
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
							}, nil
					})
			},
			want: &gitlab.Wiki{
				Slug:    "group/subgroup-page",
				Title:   "Subgroup page",
				Content: "<h1>Subgroup page</h1>",
				Format:  gitlab.WikiFormatMarkdown,
			},
		},
		{
			name: "wiki page not found",
			args: map[string]any{
				"project_id": "test/project",
				"slug":       "does-not-exist",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					GetWikiPage(gomock.Eq("test/project"), gomock.Eq("does-not-exist"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusNotFound},
					}, gitlab.ErrNotFound)
			},
			wantError: true,
		},
		{
			name: "missing required slug",
			args: map[string]any{
				"project_id": "test/project",
			},
			setupMock: func(_ *glabtest.MockWikisServiceInterface) {
				// No mock expectations - should fail at validation.
			},
			wantError: true,
		},
		{
			name: "missing required project_id",
			args: map[string]any{
				"slug": "home",
			},
			setupMock: func(_ *glabtest.MockWikisServiceInterface) {
				// No mock expectations - should fail at validation.
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockWikis)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "get_wiki_page"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("getWikiPage error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			var got *gitlab.Wiki
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("wiki page mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestCreateWikiPage(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockWikisServiceInterface)
		wantError bool
		want      *gitlab.Wiki
	}{
		{
			name: "create wiki page without explicit format",
			args: map[string]any{
				"project_id": "test/project",
				"title":      "Onboarding guide",
				"content":    "# Onboarding guide\n\nWelcome aboard.",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					CreateWikiPage(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateWikiPageOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Wiki, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateWikiPageOptions{
							Title:   new("Onboarding guide"),
							Content: new("# Onboarding guide\n\nWelcome aboard."),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateWikiPage() options mismatch (-want/+got):\n%s", diff)
						}

						return &gitlab.Wiki{
								Slug:    "Onboarding-guide",
								Title:   "Onboarding guide",
								Content: "# Onboarding guide\n\nWelcome aboard.",
								Format:  gitlab.WikiFormatMarkdown,
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusCreated},
							}, nil
					})
			},
			want: &gitlab.Wiki{
				Slug:    "Onboarding-guide",
				Title:   "Onboarding guide",
				Content: "# Onboarding guide\n\nWelcome aboard.",
				Format:  gitlab.WikiFormatMarkdown,
			},
		},
		{
			name: "create wiki page with explicit format",
			args: map[string]any{
				"project_id": "42",
				"title":      "Runbook",
				"content":    "= Runbook",
				"format":     "asciidoc",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					CreateWikiPage(gomock.Eq(int64(42)), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gitlab.CreateWikiPageOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Wiki, *gitlab.Response, error) {
						wantOpts := &gitlab.CreateWikiPageOptions{
							Title:   new("Runbook"),
							Content: new("= Runbook"),
							Format:  new(gitlab.WikiFormatASCIIDoc),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("CreateWikiPage() options mismatch (-want/+got):\n%s", diff)
						}

						return &gitlab.Wiki{
								Slug:    "Runbook",
								Title:   "Runbook",
								Content: "= Runbook",
								Format:  gitlab.WikiFormatASCIIDoc,
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusCreated},
							}, nil
					})
			},
			want: &gitlab.Wiki{
				Slug:    "Runbook",
				Title:   "Runbook",
				Content: "= Runbook",
				Format:  gitlab.WikiFormatASCIIDoc,
			},
		},
		{
			name: "invalid format",
			args: map[string]any{
				"project_id": "test/project",
				"title":      "Runbook",
				"content":    "Runbook",
				"format":     "textile",
			},
			setupMock: func(_ *glabtest.MockWikisServiceInterface) {
				// No mock expectations - the format is rejected before the API is called.
			},
			wantError: true,
		},
		{
			name: "missing required content",
			args: map[string]any{
				"project_id": "test/project",
				"title":      "Runbook",
			},
			setupMock: func(_ *glabtest.MockWikisServiceInterface) {
				// No mock expectations - should fail at validation.
			},
			wantError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"project_id": "test/project",
				"title":      "Runbook",
				"content":    "Runbook",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					CreateWikiPage(gomock.Eq("test/project"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusForbidden},
					}, fmt.Errorf("insufficient permissions"))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockWikis)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "create_wiki_page"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("createWikiPage error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			var got *gitlab.Wiki
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("wiki page mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestEditWikiPage(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockWikisServiceInterface)
		wantError bool
		want      *gitlab.Wiki
	}{
		{
			name: "edit title only",
			args: map[string]any{
				"project_id": "test/project",
				"slug":       "Onboarding-guide",
				"title":      "Onboarding",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					EditWikiPage(gomock.Eq("test/project"), gomock.Eq("Onboarding-guide"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ string, opts *gitlab.EditWikiPageOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Wiki, *gitlab.Response, error) {
						wantOpts := &gitlab.EditWikiPageOptions{
							Title: new("Onboarding"),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("EditWikiPage() options mismatch (-want/+got):\n%s", diff)
						}

						return &gitlab.Wiki{
								Slug:   "Onboarding",
								Title:  "Onboarding",
								Format: gitlab.WikiFormatMarkdown,
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
							}, nil
					})
			},
			want: &gitlab.Wiki{
				Slug:   "Onboarding",
				Title:  "Onboarding",
				Format: gitlab.WikiFormatMarkdown,
			},
		},
		{
			name: "edit content only",
			args: map[string]any{
				"project_id": "test/project",
				"slug":       "home",
				"content":    "# Home\n\nUpdated.",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					EditWikiPage(gomock.Eq("test/project"), gomock.Eq("home"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ string, opts *gitlab.EditWikiPageOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Wiki, *gitlab.Response, error) {
						wantOpts := &gitlab.EditWikiPageOptions{
							Content: new("# Home\n\nUpdated."),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("EditWikiPage() options mismatch (-want/+got):\n%s", diff)
						}

						return &gitlab.Wiki{
								Slug:    "home",
								Title:   "Home",
								Content: "# Home\n\nUpdated.",
								Format:  gitlab.WikiFormatMarkdown,
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
							}, nil
					})
			},
			want: &gitlab.Wiki{
				Slug:    "home",
				Title:   "Home",
				Content: "# Home\n\nUpdated.",
				Format:  gitlab.WikiFormatMarkdown,
			},
		},
		{
			name: "edit format only",
			args: map[string]any{
				"project_id": "7",
				"slug":       "home",
				"format":     "org",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					EditWikiPage(gomock.Eq(int64(7)), gomock.Eq("home"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ string, opts *gitlab.EditWikiPageOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Wiki, *gitlab.Response, error) {
						wantOpts := &gitlab.EditWikiPageOptions{
							Format: new(gitlab.WikiFormatOrg),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("EditWikiPage() options mismatch (-want/+got):\n%s", diff)
						}

						return &gitlab.Wiki{
								Slug:   "home",
								Title:  "Home",
								Format: gitlab.WikiFormatOrg,
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
							}, nil
					})
			},
			want: &gitlab.Wiki{
				Slug:   "home",
				Title:  "Home",
				Format: gitlab.WikiFormatOrg,
			},
		},
		{
			name: "edit title, content and format",
			args: map[string]any{
				"project_id": "test/project",
				"slug":       "home",
				"title":      "Start here",
				"content":    "Start here.",
				"format":     "rdoc",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					EditWikiPage(gomock.Eq("test/project"), gomock.Eq("home"), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ string, opts *gitlab.EditWikiPageOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Wiki, *gitlab.Response, error) {
						wantOpts := &gitlab.EditWikiPageOptions{
							Title:   new("Start here"),
							Content: new("Start here."),
							Format:  new(gitlab.WikiFormatRDoc),
						}

						if diff := cmp.Diff(wantOpts, opts); diff != "" {
							return nil, nil, fmt.Errorf("EditWikiPage() options mismatch (-want/+got):\n%s", diff)
						}

						return &gitlab.Wiki{
								Slug:    "Start-here",
								Title:   "Start here",
								Content: "Start here.",
								Format:  gitlab.WikiFormatRDoc,
							}, &gitlab.Response{
								Response: &http.Response{StatusCode: http.StatusOK},
							}, nil
					})
			},
			want: &gitlab.Wiki{
				Slug:    "Start-here",
				Title:   "Start here",
				Content: "Start here.",
				Format:  gitlab.WikiFormatRDoc,
			},
		},
		{
			name: "no fields to update",
			args: map[string]any{
				"project_id": "test/project",
				"slug":       "home",
			},
			setupMock: func(_ *glabtest.MockWikisServiceInterface) {
				// No mock expectations - the request is rejected before the API is called.
			},
			wantError: true,
		},
		{
			name: "invalid format",
			args: map[string]any{
				"project_id": "test/project",
				"slug":       "home",
				"format":     "textile",
			},
			setupMock: func(_ *glabtest.MockWikisServiceInterface) {
				// No mock expectations - the format is rejected before the API is called.
			},
			wantError: true,
		},
		{
			name: "missing required slug",
			args: map[string]any{
				"project_id": "test/project",
				"title":      "Start here",
			},
			setupMock: func(_ *glabtest.MockWikisServiceInterface) {
				// No mock expectations - should fail at validation.
			},
			wantError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"project_id": "test/project",
				"slug":       "home",
				"title":      "Start here",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					EditWikiPage(gomock.Eq("test/project"), gomock.Eq("home"), gomock.Any(), gomock.Any()).
					Return(nil, &gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusInternalServerError},
					}, fmt.Errorf("API error"))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockWikis)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "edit_wiki_page"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("editWikiPage error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			var got *gitlab.Wiki
			if err := unmarshalResult(result, &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("wiki page mismatch (-want/+got):\n%s", diff)
			}
		})
	}
}

func TestDeleteWikiPage(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		setupMock func(*glabtest.MockWikisServiceInterface)
		wantError bool
		want      string
	}{
		{
			name: "delete wiki page by project path",
			args: map[string]any{
				"project_id": "test/project",
				"slug":       "Onboarding-guide",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					DeleteWikiPage(gomock.Eq("test/project"), gomock.Eq("Onboarding-guide"), gomock.Any()).
					Return(&gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusNoContent},
					}, nil)
			},
			want: "success",
		},
		{
			name: "delete wiki page by numeric project ID",
			args: map[string]any{
				"project_id": "42",
				"slug":       "home",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					DeleteWikiPage(gomock.Eq(int64(42)), gomock.Eq("home"), gomock.Any()).
					Return(&gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusNoContent},
					}, nil)
			},
			want: "success",
		},
		{
			name: "wiki page not found",
			args: map[string]any{
				"project_id": "test/project",
				"slug":       "does-not-exist",
			},
			setupMock: func(mockWikis *glabtest.MockWikisServiceInterface) {
				mockWikis.EXPECT().
					DeleteWikiPage(gomock.Eq("test/project"), gomock.Eq("does-not-exist"), gomock.Any()).
					Return(&gitlab.Response{
						Response: &http.Response{StatusCode: http.StatusNotFound},
					}, gitlab.ErrNotFound)
			},
			wantError: true,
		},
		{
			name: "missing required slug",
			args: map[string]any{
				"project_id": "test/project",
			},
			setupMock: func(_ *glabtest.MockWikisServiceInterface) {
				// No mock expectations - should fail at validation.
			},
			wantError: true,
		},
		{
			name: "missing required project_id",
			args: map[string]any{
				"slug": "home",
			},
			setupMock: func(_ *glabtest.MockWikisServiceInterface) {
				// No mock expectations - should fail at validation.
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabClient := glabtest.NewTestClient(t)
			ctx := glctx.WithClient(t.Context(), gitlabClient.Client)

			tt.setupMock(gitlabClient.MockWikis)

			srv := mcptest.NewUnstartedServer(t)
			AddTo(srv)

			if err := srv.Start(ctx); err != nil {
				t.Fatalf("failed to start MCP test server: %v", err)
			}
			defer srv.Close()

			var req mcp.CallToolRequest

			req.Params.Name = "delete_wiki_page"
			req.Params.Arguments = tt.args

			result, err := srv.Client().CallTool(t.Context(), req)

			if gotErr := err != nil; gotErr != tt.wantError {
				t.Errorf("deleteWikiPage error mismatch, got: %v, want error: %v", err, tt.wantError)
			}

			if err != nil {
				return
			}

			got, err := resultToString(result)
			if err != nil {
				t.Fatalf("reading result: %v", err)
			}

			if got != tt.want {
				t.Errorf("deleteWikiPage result = %q, want %q", got, tt.want)
			}
		})
	}
}
