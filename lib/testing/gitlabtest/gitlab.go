package gitlabtest

import (
	"errors"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// ErrNumericIDRequired indicates that the fake epic server only supports numeric group IDs.
var ErrNumericIDRequired = errors.New("the fake epic server only supports numeric group IDs")

// ErrNotFound indicates that the requested resource was not found.
var ErrNotFound = errors.New("not found")

type FakeEpicsService struct {
	Epics []*gitlab.Epic

	gitlab.EpicsServiceInterface
}

func (f *FakeEpicsService) ListGroupEpics(groupID any, opt *gitlab.ListGroupEpicsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Epic, *gitlab.Response, error) {
	id, ok := groupID.(int)
	if !ok {
		return nil, nil, ErrNumericIDRequired
	}

	var epics []*gitlab.Epic

	for _, epic := range f.Epics {
		if epic.GroupID != id {
			continue
		}

		if opt.State != nil && *opt.State != "" && epic.State != *opt.State {
			continue
		}

		epics = append(epics, epic)
	}

	return epics, &gitlab.Response{}, nil
}

func (f *FakeEpicsService) GetEpic(groupID any, epicIID int, _ ...gitlab.RequestOptionFunc) (*gitlab.Epic, *gitlab.Response, error) {
	id, ok := groupID.(int)
	if !ok {
		return nil, nil, ErrNumericIDRequired
	}

	for _, epic := range f.Epics {
		if epic.GroupID == id && epic.IID == epicIID {
			return epic, &gitlab.Response{}, nil
		}
	}

	return nil, &gitlab.Response{}, nil
}

func (f *FakeEpicsService) GetEpicLinks(groupID any, epicIID int, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Epic, *gitlab.Response, error) {
	id, ok := groupID.(int)
	if !ok {
		return nil, nil, ErrNumericIDRequired
	}

	epicID := -1

	for _, epic := range f.Epics {
		if epic.GroupID == id && epic.IID == epicIID {
			epicID = epic.ID
			break
		}
	}

	if epicID == -1 {
		return nil, nil, fmt.Errorf("%w: epicIID=%d, groupID=%d", ErrNotFound, epicIID, id)
	}

	var epics []*gitlab.Epic

	for _, epic := range f.Epics {
		if epic.ParentID != epicID {
			continue
		}

		epics = append(epics, epic)
	}

	return epics, &gitlab.Response{}, nil
}

type FakeEpicIssuesService struct {
	Epics  []*gitlab.Epic
	Issues []*gitlab.Issue

	gitlab.EpicIssuesServiceInterface
}

func (f *FakeEpicIssuesService) ListEpicIssues(groupID any, epicIID int, _ *gitlab.ListOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Issue, *gitlab.Response, error) {
	id, ok := groupID.(int)
	if !ok {
		return nil, nil, ErrNumericIDRequired
	}

	epicID := -1

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
