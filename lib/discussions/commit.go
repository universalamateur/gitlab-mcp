package discussions

import (
	"context"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// CommitDiscussion provides methods for managing discussions on GitLab commits.
type CommitDiscussion struct {
	client    *gitlab.Client
	projectID interface{}
	commitID  string
}

// NewCommitDiscussion creates a new instance for managing discussions on a specific GitLab commit.
func NewCommitDiscussion(client *gitlab.Client, projectID interface{}, commitID string) (*CommitDiscussion, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	if commitID == "" {
		return nil, fmt.Errorf("commitID cannot be empty")
	}

	return &CommitDiscussion{
		client:    client,
		projectID: projectID,
		commitID:  commitID,
	}, nil
}

// List returns all discussions for the commit.
func (d *CommitDiscussion) List(ctx context.Context) ([]*gitlab.Discussion, error) {
	var (
		opt = &gitlab.ListCommitDiscussionsOptions{
			PerPage: maxPerPage,
		}
		allDiscussions []*gitlab.Discussion
	)

	for {
		discussions, resp, err := d.client.Discussions.ListCommitDiscussions(
			d.projectID, d.commitID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("failed to list commit discussions: %w", err)
		}

		allDiscussions = append(allDiscussions, discussions...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allDiscussions, nil
}

// NewDiscussion creates a new discussion thread on the commit.
func (d *CommitDiscussion) NewDiscussion(ctx context.Context, body string) (*gitlab.Discussion, error) {
	opt := &gitlab.CreateCommitDiscussionOptions{
		Body: gitlab.Ptr(body),
	}

	discussion, _, err := d.client.Discussions.CreateCommitDiscussion(
		d.projectID, d.commitID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to create commit discussion: %w", err)
	}

	return discussion, nil
}

var _ PositionedDiscussionManager = (*CommitDiscussion)(nil)

// NewPositionDiscussion creates a new discussion on a specific position in the merge request diff.
func (d *CommitDiscussion) NewPositionDiscussion(ctx context.Context, body string, pos *gitlab.PositionOptions, commitID string) (*gitlab.Discussion, error) {

	opt := &gitlab.CreateCommitDiscussionOptions{
		Body:     gitlab.Ptr(body),
		Position: toNotePosition(pos),
	}

	discussion, _, err := d.client.Discussions.CreateCommitDiscussion(
		d.projectID, d.commitID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to create diff discussion: %w", err)
	}

	return discussion, nil
}

func toNotePosition(pOpts *gitlab.PositionOptions) *gitlab.NotePosition {
	return &gitlab.NotePosition{
		BaseSHA:      derefOptionalField(pOpts.BaseSHA),
		StartSHA:     derefOptionalField(pOpts.StartSHA),
		HeadSHA:      derefOptionalField(pOpts.HeadSHA),
		PositionType: derefOptionalField(pOpts.PositionType),
		NewPath:      derefOptionalField(pOpts.NewPath),
		NewLine:      derefOptionalField(pOpts.NewLine),
		OldPath:      derefOptionalField(pOpts.OldPath),
		OldLine:      derefOptionalField(pOpts.OldLine),
		// LineRange is not supported.
	}
}

func derefOptionalField[T any](p *T) T {
	var zero T

	if p != nil {
		return *p
	}

	return zero
}

// NewDiffDiscussion creates a new discussion on a specific position in the commit diff.
func (d *CommitDiscussion) NewDiffDiscussion(ctx context.Context, body string, position *gitlab.NotePosition) (*gitlab.Discussion, error) {
	opt := &gitlab.CreateCommitDiscussionOptions{
		Body:     gitlab.Ptr(body),
		Position: position,
	}

	discussion, _, err := d.client.Discussions.CreateCommitDiscussion(
		d.projectID, d.commitID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to create diff discussion: %w", err)
	}

	return discussion, nil
}

// AddNote adds a new note to an existing discussion thread.
func (d *CommitDiscussion) AddNote(ctx context.Context, discussionID string, body string) (*gitlab.Note, error) {
	opt := &gitlab.AddCommitDiscussionNoteOptions{
		Body: gitlab.Ptr(body),
	}

	note, _, err := d.client.Discussions.AddCommitDiscussionNote(
		d.projectID, d.commitID, discussionID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to add note to commit discussion: %w", err)
	}

	return note, nil
}

// ModifyNote updates an existing note in a discussion thread.
func (d *CommitDiscussion) ModifyNote(ctx context.Context, discussionID string, noteID int, body string) (*gitlab.Note, error) {
	opt := &gitlab.UpdateCommitDiscussionNoteOptions{
		Body: gitlab.Ptr(body),
	}

	note, _, err := d.client.Discussions.UpdateCommitDiscussionNote(
		d.projectID, d.commitID, discussionID, noteID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to modify commit note: %w", err)
	}

	return note, nil
}

// DeleteNote removes a note from a discussion thread.
func (d *CommitDiscussion) DeleteNote(ctx context.Context, discussionID string, noteID int) error {
	_, err := d.client.Discussions.DeleteCommitDiscussionNote(
		d.projectID, d.commitID, discussionID, noteID, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to delete commit note: %w", err)
	}

	return nil
}
