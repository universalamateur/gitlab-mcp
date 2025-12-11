package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/discussions"
	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// DiscussionServiceInterface defines the interface for discussion-related GitLab operations.
type DiscussionServiceInterface interface {
	// AddTo registers all discussion-related tools with the provided MCPServer.
	AddTo(srv ToolAdder)

	// Discussion tools
	NewDiscussion() server.ServerTool
	ListDiscussions() server.ServerTool
	AddDiscussionNote() server.ServerTool
	ModifyDiscussionNote() server.ServerTool
	DeleteDiscussionNote() server.ServerTool
	ResolveDiscussion() server.ServerTool
	NewPositionDiscussion() server.ServerTool
}

// DiscussionTools implements discussion-related MCP tools.
type DiscussionTools struct{}

// AddTo registers all discussion-related tools with the provided MCPServer.
func (d *DiscussionTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		d.NewDiscussion(),
		d.ListDiscussions(),
		d.AddDiscussionNote(),
		d.ModifyDiscussionNote(),
		d.DeleteDiscussionNote(),
		d.ResolveDiscussion(),
		d.NewPositionDiscussion(),
	)
}

// discussionResourceType represents the type of GitLab resource for discussions.
type discussionResourceType string

const (
	discussionResourceTypeIssue        discussionResourceType = "issue"
	discussionResourceTypeMergeRequest discussionResourceType = "merge_request"
	discussionResourceTypeEpic         discussionResourceType = "epic"
	discussionResourceTypeSnippet      discussionResourceType = "snippet"
	discussionResourceTypeCommit       discussionResourceType = "commit"
)

// discussionManager returns the appropriate discussion manager based on resource type.
//
//nolint:ireturn,wrapcheck
func (d *DiscussionTools) discussionManager(client *gitlab.Client, resourceType discussionResourceType, parentID mcpargs.ID, resourceID mcpargs.ID) (discussions.Manager, error) {
	switch resourceType {
	case discussionResourceTypeIssue:
		return discussions.NewIssueDiscussion(client, parentID, resourceID.Integer)

	case discussionResourceTypeMergeRequest:
		return discussions.NewMergeRequestDiscussion(client, parentID, resourceID.Integer)

	case discussionResourceTypeEpic:
		return discussions.NewEpicDiscussion(client, parentID, resourceID.Integer)

	case discussionResourceTypeSnippet:
		return discussions.NewSnippetDiscussion(client, parentID, resourceID.Integer)

	case discussionResourceTypeCommit:
		return discussions.NewCommitDiscussion(client, parentID, resourceID.String)

	default:
		return nil, fmt.Errorf("%w: unsupported resource type: %s", ErrInvalidArgument, resourceType)
	}
}

type newDiscussionArgs struct {
	ResourceType discussionResourceType `mcp_desc:"Type of GitLab resource (issue, merge_request, epic, snippet, commit)" mcp_required:"true" mcp_enum:"issue,merge_request,epic,snippet,commit"`
	ParentID     mcpargs.ID             `mcp_desc:"ID of the parent resource (project ID for issue/MR/snippet/commit, group ID for epic)" mcp_required:"true"`
	ResourceID   mcpargs.ID             `mcp_desc:"ID of the resource (IID for issue/MR, ID for epic/snippet, SHA for commit)" mcp_required:"true"`
	Body         string                 `mcp_desc:"The content of the discussion in GitLab Flavored Markdown" mcp_required:"true"`
}

func (d *DiscussionTools) NewDiscussion() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(d.newDiscussion),
		Tool: mcpargs.NewTool("discussion_new", newDiscussionArgs{},
			mcp.WithDescription("Creates a new discussion thread on a GitLab resource"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (d *DiscussionTools) newDiscussion(ctx context.Context, client *gitlab.Client, args newDiscussionArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manager, err := d.discussionManager(client, args.ResourceType, args.ParentID, args.ResourceID)
	if err != nil {
		return nil, err
	}

	discussion, err := manager.NewDiscussion(ctx, args.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create discussion: %w", err)
	}

	return newToolResultJSON(discussion)
}

type listDiscussionsArgs struct {
	ResourceType discussionResourceType `mcp_desc:"Type of GitLab resource (issue, merge_request, epic, snippet, commit)" mcp_required:"true" mcp_enum:"issue,merge_request,epic,snippet,commit"`
	ParentID     mcpargs.ID             `mcp_desc:"ID of the parent resource (project ID for issue/MR/snippet/commit, group ID for epic)" mcp_required:"true"`
	ResourceID   mcpargs.ID             `mcp_desc:"ID of the resource (IID for issue/MR, ID for epic/snippet, SHA for commit)" mcp_required:"true"`
	Confidential bool                   `mcp_desc:"Whether to include confidential discussions in the response. Only access confidential information when explicitly prompted. Defaults to 'false'."`
}

func (d *DiscussionTools) ListDiscussions() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(d.listDiscussions),
		Tool: mcpargs.NewTool("discussion_list", listDiscussionsArgs{},
			mcp.WithDescription("Lists all discussions for a GitLab resource"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (d *DiscussionTools) listDiscussions(ctx context.Context, client *gitlab.Client, args listDiscussionsArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manager, err := d.discussionManager(client, args.ResourceType, args.ParentID, args.ResourceID)
	if err != nil {
		return nil, err
	}

	discussions, err := manager.List(ctx, args.Confidential)
	if err != nil {
		return nil, fmt.Errorf("failed to list discussions: %w", err)
	}

	return newToolResultJSON(discussions)
}

type addDiscussionNoteArgs struct {
	ResourceType discussionResourceType `mcp_desc:"Type of GitLab resource (issue, merge_request, epic, snippet, commit)" mcp_required:"true" mcp_enum:"issue,merge_request,epic,snippet,commit"`
	ParentID     mcpargs.ID             `mcp_desc:"ID of the parent resource (project ID for issue/MR/snippet/commit, group ID for epic)" mcp_required:"true"`
	ResourceID   mcpargs.ID             `mcp_desc:"ID of the resource (IID for issue/MR, ID for epic/snippet, SHA for commit)" mcp_required:"true"`
	DiscussionID string                 `mcp_desc:"The ID of the discussion thread" mcp_required:"true"`
	Body         string                 `mcp_desc:"The content of the note in GitLab Flavored Markdown" mcp_required:"true"`
}

func (d *DiscussionTools) AddDiscussionNote() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(d.addDiscussionNote),
		Tool: mcpargs.NewTool("discussion_add_note", addDiscussionNoteArgs{},
			mcp.WithDescription("Adds a new note (i.e. a reply) to an existing discussion thread"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (d *DiscussionTools) addDiscussionNote(ctx context.Context, client *gitlab.Client, args addDiscussionNoteArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manager, err := d.discussionManager(client, args.ResourceType, args.ParentID, args.ResourceID)
	if err != nil {
		return nil, err
	}

	note, err := manager.AddNote(ctx, args.DiscussionID, args.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to add discussion note: %w", err)
	}

	return newToolResultJSON(note)
}

type modifyDiscussionNoteArgs struct {
	ResourceType discussionResourceType `mcp_desc:"Type of GitLab resource (issue, merge_request, epic, snippet, commit)" mcp_required:"true" mcp_enum:"issue,merge_request,epic,snippet,commit"`
	ParentID     mcpargs.ID             `mcp_desc:"ID of the parent resource (project ID for issue/MR/snippet/commit, group ID for epic)" mcp_required:"true"`
	ResourceID   mcpargs.ID             `mcp_desc:"ID of the resource (IID for issue/MR, ID for epic/snippet, SHA for commit)" mcp_required:"true"`
	DiscussionID string                 `mcp_desc:"The ID of the discussion thread" mcp_required:"true"`
	NoteID       int64                  `mcp_desc:"The ID of the note to modify" mcp_required:"true"`
	Body         string                 `mcp_desc:"The updated content of the note in GitLab Flavored Markdown" mcp_required:"true"`
}

func (d *DiscussionTools) ModifyDiscussionNote() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(d.modifyDiscussionNote),
		Tool: mcpargs.NewTool("discussion_modify_note", modifyDiscussionNoteArgs{},
			mcp.WithDescription("Modifies an existing note in a discussion thread"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (d *DiscussionTools) modifyDiscussionNote(ctx context.Context, client *gitlab.Client, args modifyDiscussionNoteArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manager, err := d.discussionManager(client, args.ResourceType, args.ParentID, args.ResourceID)
	if err != nil {
		return nil, err
	}

	note, err := manager.ModifyNote(ctx, args.DiscussionID, args.NoteID, args.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to modify discussion note: %w", err)
	}

	return newToolResultJSON(note)
}

type deleteDiscussionNoteArgs struct {
	ResourceType discussionResourceType `mcp_desc:"Type of GitLab resource (issue, merge_request, epic, snippet, commit)" mcp_required:"true" mcp_enum:"issue,merge_request,epic,snippet,commit"`
	ParentID     mcpargs.ID             `mcp_desc:"ID of the parent resource (project ID for issue/MR/snippet/commit, group ID for epic)" mcp_required:"true"`
	ResourceID   mcpargs.ID             `mcp_desc:"ID of the resource (IID for issue/MR, ID for epic/snippet, SHA for commit)" mcp_required:"true"`
	DiscussionID string                 `mcp_desc:"The ID of the discussion thread" mcp_required:"true"`
	NoteID       int64                  `mcp_desc:"The ID of the note to delete" mcp_required:"true"`
}

func (d *DiscussionTools) DeleteDiscussionNote() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(d.deleteDiscussionNote),
		Tool: mcpargs.NewTool("discussion_delete_note", deleteDiscussionNoteArgs{},
			mcp.WithDescription("Deletes a note from a discussion thread"),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (d *DiscussionTools) deleteDiscussionNote(ctx context.Context, client *gitlab.Client, args deleteDiscussionNoteArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manager, err := d.discussionManager(client, args.ResourceType, args.ParentID, args.ResourceID)
	if err != nil {
		return nil, err
	}

	err = manager.DeleteNote(ctx, args.DiscussionID, args.NoteID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete discussion note: %w", err)
	}

	return newToolResultJSON(map[string]any{"deleted": true})
}

type resolveDiscussionArgs struct {
	ProjectID       mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	MergeRequestIID int64      `mcp_desc:"The internal ID of the merge request" mcp_required:"true"`
	DiscussionID    string     `mcp_desc:"The ID of the discussion thread" mcp_required:"true"`
	Resolved        bool       `mcp_desc:"Whether to resolve (true) or unresolve (false) the discussion" mcp_required:"true"`
}

func (d *DiscussionTools) ResolveDiscussion() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(d.resolveDiscussion),
		Tool: mcpargs.NewTool("discussion_resolve", resolveDiscussionArgs{},
			mcp.WithDescription("Resolves or unresolves a discussion thread in a merge request"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (d *DiscussionTools) resolveDiscussion(ctx context.Context, client *gitlab.Client, args resolveDiscussionArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// This is specific to merge requests only
	manager, err := d.discussionManager(client, discussionResourceTypeMergeRequest,
		args.ProjectID, mcpargs.ID{Integer: args.MergeRequestIID})
	if err != nil {
		return nil, err
	}

	// Type assertion is needed here as Resolve is not part of the common interface
	resolvableMgr, ok := manager.(discussions.ResolvableManager)
	if !ok {
		return nil, fmt.Errorf("%w: expected merge request discussion manager, got another type", ErrInvalidArgument)
	}

	discussion, err := resolvableMgr.ResolveDiscussion(ctx, args.DiscussionID, args.Resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to %s discussion: %w", resolveActionText(args.Resolved), err)
	}

	return newToolResultJSON(discussion)
}

// resolveActionText returns the appropriate text based on the resolved status.
func resolveActionText(resolved bool) string {
	if resolved {
		return "resolve"
	}

	return "unresolve"
}

type newPositionDiscussionArgs struct {
	ResourceType discussionResourceType `mcp_desc:"Type of GitLab resource (merge_request, commit)" mcp_required:"true" mcp_enum:"merge_request,commit"`
	ProjectID    mcpargs.ID             `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	ResourceID   mcpargs.ID             `mcp_desc:"ID of the resource (IID for MR, SHA for commit)" mcp_required:"true"`
	Body         string                 `mcp_desc:"The content of the discussion in GitLab Flavored Markdown" mcp_required:"true"`
	BaseSHA      string                 `mcp_desc:"Base commit SHA in the source branch" mcp_required:"true"`
	HeadSHA      string                 `mcp_desc:"SHA referencing HEAD of this merge request" mcp_required:"true"`
	StartSHA     string                 `mcp_desc:"SHA referencing commit in target branch" mcp_required:"true"`
	OldPath      string                 `mcp_desc:"File path before change" mcp_required:"true"`
	NewPath      string                 `mcp_desc:"File path after change" mcp_required:"true"`
	PositionType string                 `mcp_desc:"Type of the position reference. Allowed values: text or file" mcp_required:"true" mcp_enum:"text,file"`
	OldLine      int64                  `mcp_desc:"Line number before change (for text positions)"`
	NewLine      int64                  `mcp_desc:"Line number after change (for text positions)"`
	CommitID     string                 `mcp_desc:"SHA referencing commit to start this thread on"`
}

func (d *DiscussionTools) NewPositionDiscussion() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(d.newDiffDiscussion),
		Tool: mcpargs.NewTool("discussion_new_with_position", newPositionDiscussionArgs{},
			mcp.WithDescription("Creates a new discussion on a specific position in a merge request diff"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (d *DiscussionTools) newDiffDiscussion(ctx context.Context, client *gitlab.Client, args newPositionDiscussionArgs, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Create merge request discussion manager
	manager, err := d.discussionManager(client, args.ResourceType, args.ProjectID, args.ResourceID)
	if err != nil {
		return nil, err
	}

	positionManager, ok := manager.(discussions.PositionedManager)
	if !ok {
		return nil, fmt.Errorf("%w: expected positioned discussion manager, got another type", ErrInvalidArgument)
	}

	// Create position object
	position := &gitlab.PositionOptions{
		BaseSHA:      gitlab.Ptr(args.BaseSHA),
		HeadSHA:      gitlab.Ptr(args.HeadSHA),
		StartSHA:     gitlab.Ptr(args.StartSHA),
		OldPath:      gitlab.Ptr(args.OldPath),
		NewPath:      gitlab.Ptr(args.NewPath),
		PositionType: gitlab.Ptr(args.PositionType),
	}

	// Add line numbers if provided
	if args.OldLine != 0 {
		position.OldLine = gitlab.Ptr(args.OldLine)
	}

	if args.NewLine != 0 {
		position.NewLine = gitlab.Ptr(args.NewLine)
	}

	// Create the diff discussion
	discussion, err := positionManager.NewPositionDiscussion(ctx, args.Body, position, args.CommitID)
	if err != nil {
		return nil, fmt.Errorf("failed to create diff discussion: %w", err)
	}

	return newToolResultJSON(discussion)
}
