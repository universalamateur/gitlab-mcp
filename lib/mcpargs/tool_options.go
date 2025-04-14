package mcpargs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolOptions accepts a struct and returns a slice of MCP tool options based on the struct's
// fields and tags.
//
// The `mcp_desc` tag is used to set the description of the tool option.
// The `mcp_required` tag is used to mark a tool option as required.
// The `mcp_enum` tag is used to set a comma separated list of allowed enum values for a string tool option.
func ToolOptions(v any) ([]mcp.ToolOption, error) {

	// Get the type of the struct
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", t.Kind())
	}

	var (
		toolOpts []mcp.ToolOption
		errs     error
	)

	// Iterate through all fields of the struct
	for i := range t.NumField() {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field type and name
		fieldType := field.Type
		fieldName := field.Name

		// Convert field name to snake_case for tool option name
		optName := toSnakeCase(fieldName)

		var propOpts []mcp.PropertyOption

		// Set the description.
		description := field.Tag.Get("mcp_desc")
		if description == "" {
			errs = errors.Join(errs, fmt.Errorf(`missing "mcp_desc" tag on field %q`, fieldName))
			continue
		}

		propOpts = append(propOpts, mcp.Description(description))

		// Set the required flag if appropriate.
		switch field.Tag.Get("mcp_required") {
		case "true":
			propOpts = append(propOpts, mcp.Required())
		case "false", "":
			// no-op
		default:
			errs = errors.Join(errs, fmt.Errorf(`invalid "mcp_required" tag on field %q: must be "true" or "false"`, fieldName))
			continue
		}

		// Set enum values if provided.
		if enumTag := field.Tag.Get("mcp_enum"); enumTag != "" {
			if fieldType.Kind() != reflect.String {
				errs = errors.Join(errs, fmt.Errorf(`invalid "mcp_enum" tag on field %q: must be a string`, fieldName))
				continue
			}

			enumValues := strings.Split(enumTag, ",")
			propOpts = append(propOpts, mcp.Enum(enumValues...))
		}

		// If the field implements the MCPTyper interface, use the MCPType method to determine the tool option type.
		var fieldValue reflect.Value
		if reflect.ValueOf(v).Kind() == reflect.Ptr {
			fieldValue = reflect.ValueOf(v).Elem().Field(i)
		} else {
			fieldValue = reflect.ValueOf(v).Field(i)
		}

		if typer, ok := fieldValue.Interface().(MCPTyper); ok {
			switch typer.MCPType() {
			case TypeString:
				toolOpts = append(toolOpts, mcp.WithString(optName, propOpts...))
			case TypeBoolean:
				toolOpts = append(toolOpts, mcp.WithBoolean(optName, propOpts...))
			case TypeNumber:
				toolOpts = append(toolOpts, mcp.WithNumber(optName, propOpts...))
			default:
				errs = errors.Join(errs, fmt.Errorf("unsupported field type %v for field %q", typer.MCPType(), fieldName))
				continue
			}

			continue
		}

		switch fieldType.Kind() {
		case reflect.String:
			toolOpts = append(toolOpts, mcp.WithString(optName, propOpts...))
		case reflect.Bool:
			toolOpts = append(toolOpts, mcp.WithBoolean(optName, propOpts...))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			toolOpts = append(toolOpts, mcp.WithNumber(optName, propOpts...))
		default:
			errs = errors.Join(errs, fmt.Errorf("unsupported field type %q for field %q", fieldType.Kind(), fieldName))
			continue
		}
	}

	if errs != nil {
		return nil, errs
	}

	return toolOpts, nil
}

type MCPType int

const (
	TypeString MCPType = iota
	TypeNumber
	TypeBoolean
)

// MCPTyper is an interface for types that can return their MCPType.
// This allows the type to be identified without requiring knowledge of the specific type.
// In particular, it allows MCP tool parameters to work with composite types.
type MCPTyper interface {
	MCPType() MCPType
}

// toSnakeCase converts a camelCase string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		// Check if current character is uppercase
		if unicode.IsUpper(r) {
			// Add underscore if:
			// 1. Not the first character, and
			// 2. Either the previous character is lowercase, or
			// 3. Not at the end and the next character is lowercase (end of acronym)
			if i > 0 && (unicode.IsLower(rune(s[i-1])) ||
				(i < len(s)-1 && unicode.IsLower(rune(s[i+1])))) {
				result.WriteRune('_')
			}
		}
		result.WriteRune(unicode.ToLower(r))
	}

	return result.String()
}
