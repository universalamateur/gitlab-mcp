package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

const (
	wikiFormatMarkdown = "markdown"
	wikiFormatRDoc     = "rdoc"
	wikiFormatASCIIDoc = "asciidoc"
	wikiFormatOrg      = "org"
)

// WikisServiceInterface defines the interface for wiki-related GitLab operations.
// It provides methods for listing, retrieving, creating, updating, and deleting wiki pages.
type WikisServiceInterface interface {
	// AddTo registers all wiki-related tools with the provided MCPServer.
	AddTo(srv ToolAdder)

	// ListWikiPages returns a tool for listing a project's wiki pages.
	ListWikiPages() server.ServerTool

	// GetWikiPage returns a tool for fetching a specific wiki page by its slug.
	GetWikiPage() server.ServerTool

	// CreateWikiPage returns a tool for creating a new wiki page.
	CreateWikiPage() server.ServerTool

	// EditWikiPage returns a tool for editing an existing wiki page.
	EditWikiPage() server.ServerTool

	// DeleteWikiPage returns a tool for deleting a wiki page.
	DeleteWikiPage() server.ServerTool
}

// WikisTools implements tools for interacting with GitLab project wikis.
// It provides methods for listing, retrieving, creating, updating, and deleting wiki pages.
type WikisTools struct{}

// AddTo registers all wiki-related tools with the provided MCPServer.
func (w *WikisTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		w.ListWikiPages(),
		w.GetWikiPage(),
		w.CreateWikiPage(),
		w.EditWikiPage(),
		w.DeleteWikiPage(),
	)
}

type listWikiPagesArgs struct {
	ProjectID   mcpargs.ID `mcp_desc:"The project to list wiki pages from, using either owner/project format or numeric project ID" mcp_required:"true"`
	WithContent bool       `mcp_desc:"Include the content of each wiki page in the results. Default is false (metadata only)."`
}

// ListWikiPages returns a ServerTool for listing all wiki pages of a project.
func (w *WikisTools) ListWikiPages() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(w.listWikiPages),
		Tool: mcpargs.NewTool("list_wiki_pages", listWikiPagesArgs{},
			mcp.WithDescription("List all wiki pages of a project"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (w *WikisTools) listWikiPages(ctx context.Context, client *gitlab.Client, args listWikiPagesArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.ListWikisOptions{
		WithContent: &args.WithContent,
	}

	wikis, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Wiki, *gitlab.Response, error) {
		return client.Wikis.ListWikis(args.ProjectID.Value(), opt, gitlab.WithContext(ctx), p)
	})
	if err != nil {
		return nil, fmt.Errorf("ListWikis(%q): %w", args.ProjectID.Value(), err)
	}

	return newToolResultJSON(wikis)
}

type getWikiPageArgs struct {
	ProjectID  mcpargs.ID `mcp_desc:"The project the wiki page belongs to, using either owner/project format or numeric project ID" mcp_required:"true"`
	Slug       string     `mcp_desc:"The slug (URL-friendly identifier) of the wiki page, e.g. 'Setup-Dokploy-Host' or 'group/subgroup-page'" mcp_required:"true"`
	Version    string     `mcp_desc:"The SHA of a specific version of the wiki page to retrieve. Defaults to the latest version."`
	RenderHTML bool       `mcp_desc:"Render the wiki page content to HTML instead of returning it in its original markup format."`
}

// GetWikiPage returns a ServerTool for fetching a specific wiki page.
func (w *WikisTools) GetWikiPage() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(w.getWikiPage),
		Tool: mcpargs.NewTool("get_wiki_page", getWikiPageArgs{},
			mcp.WithDescription("Get a wiki page, including its content"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (w *WikisTools) getWikiPage(ctx context.Context, client *gitlab.Client, args getWikiPageArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.GetWikiPageOptions{
		RenderHTML: &args.RenderHTML,
	}

	if args.Version != "" {
		opt.Version = &args.Version
	}

	wiki, _, err := client.Wikis.GetWikiPage(args.ProjectID.Value(), args.Slug, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetWikiPage(%q, %q): %w", args.ProjectID.Value(), args.Slug, err)
	}

	return newToolResultJSON(wiki)
}

type createWikiPageArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"The project to create the wiki page in, using either owner/project format or numeric project ID" mcp_required:"true"`
	Title     string     `mcp_desc:"The title of the wiki page. Determines its slug - e.g. a title of 'Onboarding guide' creates the page at slug 'Onboarding-guide'." mcp_required:"true"`
	Content   string     `mcp_desc:"The content of the wiki page" mcp_required:"true"`
	Format    string     `mcp_desc:"The markup format of the wiki page. Default is 'markdown'." mcp_enum:"markdown,rdoc,asciidoc,org"`
}

// CreateWikiPage returns a ServerTool for creating a new wiki page.
func (w *WikisTools) CreateWikiPage() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(w.createWikiPage),
		Tool: mcpargs.NewTool("create_wiki_page", createWikiPageArgs{},
			mcp.WithDescription("Create a new wiki page. When creating a wiki page, include its 'web_url' equivalent "+
				"(the project's wiki base URL plus the returned slug) in your response."),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (w *WikisTools) createWikiPage(ctx context.Context, client *gitlab.Client, args createWikiPageArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.CreateWikiPageOptions{
		Title:   new(args.Title),
		Content: new(args.Content),
	}

	if args.Format != "" {
		format, err := wikiFormatValue(args.Format)
		if err != nil {
			return nil, err
		}

		opt.Format = format
	}

	wiki, _, err := client.Wikis.CreateWikiPage(args.ProjectID.Value(), opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CreateWikiPage(%q): %w", args.ProjectID.Value(), err)
	}

	return newToolResultJSON(wiki)
}

type editWikiPageArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"The project the wiki page belongs to, using either owner/project format or numeric project ID" mcp_required:"true"`
	Slug      string     `mcp_desc:"The slug of the wiki page to edit" mcp_required:"true"`
	Title     string     `mcp_desc:"The new title of the wiki page. Leave empty to keep the current title."`
	Content   string     `mcp_desc:"The new content of the wiki page. Leave empty to keep the current content."`
	Format    string     `mcp_desc:"The new markup format of the wiki page. Leave empty to keep the current format." mcp_enum:"markdown,rdoc,asciidoc,org"`
}

// EditWikiPage returns a ServerTool for editing an existing wiki page.
func (w *WikisTools) EditWikiPage() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(w.editWikiPage),
		Tool: mcpargs.NewTool("edit_wiki_page", editWikiPageArgs{},
			mcp.WithDescription("Edit an existing wiki page. At least one of title, content, or format must be given."),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (w *WikisTools) editWikiPage(ctx context.Context, client *gitlab.Client, args editWikiPageArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.EditWikiPageOptions{}

	if args.Title != "" {
		opt.Title = new(args.Title)
	}

	if args.Content != "" {
		opt.Content = new(args.Content)
	}

	if args.Format != "" {
		format, err := wikiFormatValue(args.Format)
		if err != nil {
			return nil, err
		}

		opt.Format = format
	}

	if opt.Title == nil && opt.Content == nil && opt.Format == nil {
		return nil, fmt.Errorf("%w: at least one of title, content, or format must be given", ErrInvalidArgument)
	}

	wiki, _, err := client.Wikis.EditWikiPage(args.ProjectID.Value(), args.Slug, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("EditWikiPage(%q, %q): %w", args.ProjectID.Value(), args.Slug, err)
	}

	return newToolResultJSON(wiki)
}

type deleteWikiPageArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"The project the wiki page belongs to, using either owner/project format or numeric project ID" mcp_required:"true"`
	Slug      string     `mcp_desc:"The slug of the wiki page to delete" mcp_required:"true"`
}

// DeleteWikiPage returns a ServerTool for deleting a wiki page.
func (w *WikisTools) DeleteWikiPage() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(w.deleteWikiPage),
		Tool: mcpargs.NewTool("delete_wiki_page", deleteWikiPageArgs{},
			mcp.WithDescription("Delete a wiki page"),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (w *WikisTools) deleteWikiPage(ctx context.Context, client *gitlab.Client, args deleteWikiPageArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, err := client.Wikis.DeleteWikiPage(args.ProjectID.Value(), args.Slug, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("DeleteWikiPage(%q, %q): %w", args.ProjectID.Value(), args.Slug, err)
	}

	return mcp.NewToolResultText("success"), nil
}

var wikiFormats = map[string]*gitlab.WikiFormatValue{
	string(gitlab.WikiFormatMarkdown): new(gitlab.WikiFormatMarkdown),
	string(gitlab.WikiFormatRDoc):     new(gitlab.WikiFormatRDoc),
	string(gitlab.WikiFormatASCIIDoc): new(gitlab.WikiFormatASCIIDoc),
	string(gitlab.WikiFormatOrg):      new(gitlab.WikiFormatOrg),
}

func wikiFormatValue(s string) (*gitlab.WikiFormatValue, error) {
	f, ok := wikiFormats[s]
	if !ok {
		return nil, fmt.Errorf("%w: invalid wiki format: %q", ErrInvalidArgument, s)
	}

	return f, nil
}
