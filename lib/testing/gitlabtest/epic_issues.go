package gitlabtest

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// FakeEpicIssuesService is a partial implementation of the gitlab.EpicIssuesServiceInterface to be used in unit tests.
// Calling unimplemented methods will panic.
type FakeEpicIssuesService struct {
	gitlab.EpicIssuesServiceInterface

	Epics  []*gitlab.Epic
	Issues []*gitlab.Issue
}

func (f *FakeEpicIssuesService) ListEpicIssues(groupID any, epicIID int64, _ *gitlab.ListOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
	id, ok := groupID.(int64)
	if !ok {
		return nil, nil, ErrNumericIDRequired
	}

	epicID := int64(-1)

	for _, epic := range f.Epics {
		if epic.GroupID == id && epic.IID == epicIID {
			epicID = epic.ID
			break
		}
	}

	if epicID == -1 {
		return nil, nil, fmt.Errorf("%w: epicIID=%d, groupID=%d", ErrNotFound, epicIID, id)
	}

	var issues []*gitlab.Issue

	for _, issue := range f.Issues {
		if issue.Epic != nil && issue.Epic.ID == epicID {
			issues = append(issues, issue)
		}
	}

	return issues, &gitlab.Response{}, nil
}
