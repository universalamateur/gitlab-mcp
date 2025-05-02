package gitlabtest

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type FakeEpicsService struct {
	Epics []*gitlab.Epic

	gitlab.EpicsServiceInterface
}

func (f *FakeEpicsService) ListGroupEpics(groupID any, opt *gitlab.ListGroupEpicsOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.Epic, *gitlab.Response, error) {
	id, ok := groupID.(int)
	if !ok {
		return nil, nil, fmt.Errorf("the fake epic server only supports numeric group IDs")
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
		return nil, nil, fmt.Errorf("the fake epic server only supports numeric group IDs")
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
		return nil, nil, fmt.Errorf("the fake epic server only supports numeric group IDs")
	}

	epicID := -1
	for _, epic := range f.Epics {
		if epic.GroupID == id && epic.IID == epicIID {
			epicID = epic.ID
			break
		}
	}
	if epicID == -1 {
		return nil, nil, fmt.Errorf("epic %d not found in group %d", epicIID, groupID)
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
		return nil, nil, fmt.Errorf("the fake epic server only supports numeric group IDs")
	}

	epicID := -1
	for _, epic := range f.Epics {
		if epic.GroupID == id && epic.IID == epicIID {
			epicID = epic.ID
			break
		}
	}
	if epicID == -1 {
		return nil, nil, fmt.Errorf("epic %d not found in group %d", epicIID, groupID)
	}

	var issues []*gitlab.Issue
	for _, issue := range f.Issues {
		if issue.Epic != nil && issue.Epic.ID == epicID {
			issues = append(issues, issue)
		}
	}

	return issues, &gitlab.Response{}, nil
}
