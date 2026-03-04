package discussions

import (
	"context"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go/v2"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

// SnippetDiscussion provides methods for managing discussions on GitLab snippets.
type SnippetDiscussion struct {
	client    *gitlab.Client
	projectID mcpargs.ID
	snippetID int64
}

var _ Manager = (*SnippetDiscussion)(nil)

// NewSnippetDiscussion creates a new instance for managing discussions on a specific GitLab snippet.
func NewSnippetDiscussion(client *gitlab.Client, projectID mcpargs.ID, snippetID int64) (*SnippetDiscussion, error) {
	if client == nil {
		return nil, fmt.Errorf("%w: client cannot be nil", ErrInvalidArgument)
	}

	if projectID.IsZero() {
		return nil, fmt.Errorf("%w: projectID cannot be zero", ErrInvalidArgument)
	}

	if snippetID == 0 {
		return nil, fmt.Errorf("%w: snippetID cannot be zero", ErrInvalidArgument)
	}

	return &SnippetDiscussion{
		client:    client,
		projectID: projectID,
		snippetID: snippetID,
	}, nil
}

// List returns all discussions for the snippet.
func (d *SnippetDiscussion) List(ctx context.Context, confidential bool) ([]*gitlab.Discussion, error) {
	var (
		opt = &gitlab.ListSnippetDiscussionsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: maxPerPage,
			},
		}
		allDiscussions []*gitlab.Discussion
	)

	for {
		discussions, resp, err := d.client.Discussions.ListSnippetDiscussions(d.projectID.Value(), d.snippetID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("failed to list snippet discussions: %w", err)
		}

		allDiscussions = append(allDiscussions, discussions...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return filterDiscussions(allDiscussions, confidential), nil
}

// NewDiscussion creates a new discussion thread on the snippet.
func (d *SnippetDiscussion) NewDiscussion(ctx context.Context, body string) (*gitlab.Discussion, error) {
	opt := &gitlab.CreateSnippetDiscussionOptions{
		Body: new(body),
	}

	discussion, _, err := d.client.Discussions.CreateSnippetDiscussion(d.projectID.Value(), d.snippetID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to create snippet discussion: %w", err)
	}

	return discussion, nil
}

// AddNote adds a new note to an existing discussion thread.
func (d *SnippetDiscussion) AddNote(ctx context.Context, discussionID string, body string) (*gitlab.Note, error) {
	opt := &gitlab.AddSnippetDiscussionNoteOptions{
		Body: new(body),
	}

	note, _, err := d.client.Discussions.AddSnippetDiscussionNote(d.projectID.Value(), d.snippetID, discussionID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to add note to snippet discussion: %w", err)
	}

	return note, nil
}

// ModifyNote updates an existing note in a discussion thread.
func (d *SnippetDiscussion) ModifyNote(ctx context.Context, discussionID string, noteID int64, body string) (*gitlab.Note, error) {
	opt := &gitlab.UpdateSnippetDiscussionNoteOptions{
		Body: new(body),
	}

	note, _, err := d.client.Discussions.UpdateSnippetDiscussionNote(d.projectID.Value(), d.snippetID, discussionID, noteID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to modify snippet note: %w", err)
	}

	return note, nil
}

// DeleteNote removes a note from a discussion thread.
func (d *SnippetDiscussion) DeleteNote(ctx context.Context, discussionID string, noteID int64) error {
	_, err := d.client.Discussions.DeleteSnippetDiscussionNote(d.projectID.Value(), d.snippetID, discussionID, noteID, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to delete snippet note: %w", err)
	}

	return nil
}
