package tools

import (
	"reflect"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/testing/gitlabtest"
)

func TestListUserTodos(t *testing.T) {
	fakeTodos := []*gitlab.Todo{
		{ID: 1, State: "done"},
		{ID: 2, State: "pending"},
		{ID: 3, State: "done"},
		{ID: 4, State: "done"},
	}
	service := &TodosService{client: &gitlab.Client{}}
	service.client.Todos = &gitlabtest.FakeTodosService{Todos: fakeTodos}

	tests := []struct {
		name  string
		state string
		limit int
		want  []int
	}{
		{
			name:  "pending only",
			state: "pending",
			limit: 0,
			want:  []int{2},
		},
		{
			name:  "done only",
			state: "done",
			limit: 0,
			want:  []int{1, 3, 4},
		},
		{
			name:  "limit 1 done",
			state: "done",
			limit: 1,
			want:  []int{1},
		},
		{
			name:  "limit 2 done",
			state: "done",
			limit: 2,
			want:  []int{1, 3},
		},
		{
			name:  "limit greater than todos done",
			state: "done",
			limit: 10,
			want:  []int{1, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req mcp.CallToolRequest

			args := map[string]any{"state": tt.state}
			if tt.limit != 0 {
				args["limit"] = tt.limit
			}

			req.Params.Arguments = args

			result, err := service.listUserTodos(t.Context(), req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var todos []*gitlab.Todo
			if err := unmarshalResult(result, &todos); err != nil {
				t.Fatalf("decode error: %v", err)
			}

			var gotIDs []int
			for _, todo := range todos {
				gotIDs = append(gotIDs, todo.ID)
			}

			if !reflect.DeepEqual(gotIDs, tt.want) {
				t.Errorf("got %v, want %v", gotIDs, tt.want)
			}
		})
	}
}
