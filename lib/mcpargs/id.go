package mcpargs

import (
	"fmt"
	"strconv"
)

// ID is a type that can represent an ID either as a string or an integer.
// It implements the Unmarshaler interface to unmarshal from either string or int types.
type ID struct {
	String  string
	Integer int
}

// Unmarshal sets the ID value from a string or integer.
// It implements the Unmarshaler interface.
func (id *ID) Unmarshal(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot unmarshal ID from %T", v)
	}

	if i, err := strconv.Atoi(s); err == nil {
		id.Integer = i
		id.String = ""

		return nil
	}

	id.String = s
	id.Integer = 0

	return nil
}

// MCPType implements the MCPTyper interface.
func (ID) MCPType() MCPType {
	return TypeString
}

func (id ID) Value() any {
	if id.Integer != 0 {
		return id.Integer
	}
	return id.String
}
