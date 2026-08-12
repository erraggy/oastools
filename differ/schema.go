package differ

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// schemaPair represents a pair of schemas being compared.
// Used as a map key for cycle detection.
type schemaPair struct {
	source *parser.Schema
	target *parser.Schema
}

// schemaVisited tracks visited schema pairs during recursive traversal to detect cycles.
// It uses pointer-based identity to detect when we encounter the same comparison pair.
type schemaVisited struct {
	visited map[schemaPair]string // schema pair -> first occurrence path
}

// newSchemaVisited creates a new visited tracker for schema traversal.
func newSchemaVisited() *schemaVisited {
	return &schemaVisited{
		visited: make(map[schemaPair]string),
	}
}

// enter marks a schema pair as visited at the given path.
// Returns true if this exact pair was already visited.
func (v *schemaVisited) enter(source, target *parser.Schema, path string) bool {
	pair := schemaPair{source: source, target: target}
	if _, exists := v.visited[pair]; exists {
		return true
	}
	v.visited[pair] = path
	return false
}

// leave removes a schema pair from the visited set.
// This should be called when exiting a schema pair's traversal to allow revisiting in different contexts.
func (v *schemaVisited) leave(source, target *parser.Schema) {
	pair := schemaPair{source: source, target: target}
	delete(v.visited, pair)
}

// schemaOrBoolKind names the shape held by a schema-or-bool field: Items,
// AdditionalProperties, AdditionalItems, UnevaluatedItems or
// UnevaluatedProperties. Those fields are typed any and every decode path
// yields a *parser.Schema, a []*parser.Schema (the OAS 2.0 tuple form) or a
// bool (#502).
type schemaOrBoolKind int

const (
	schemaOrBoolNil schemaOrBoolKind = iota
	schemaOrBoolSchema
	schemaOrBoolTuple
	schemaOrBoolBool
	schemaOrBoolUnknown
)

// getSchemaOrBoolKind classifies the value held by a schema-or-bool field.
func getSchemaOrBoolKind(value any) schemaOrBoolKind {
	if value == nil {
		return schemaOrBoolNil
	}
	switch value.(type) {
	case *parser.Schema:
		return schemaOrBoolSchema
	case []*parser.Schema:
		return schemaOrBoolTuple
	case bool:
		return schemaOrBoolBool
	default:
		return schemaOrBoolUnknown
	}
}

// schemaOrBoolShapeName renders a kind in OAS terms for change messages.
func schemaOrBoolShapeName(kind schemaOrBoolKind) string {
	switch kind {
	case schemaOrBoolSchema:
		return "schema"
	case schemaOrBoolTuple:
		return "tuple of schemas"
	case schemaOrBoolBool:
		return "boolean"
	case schemaOrBoolNil, schemaOrBoolUnknown:
		return "unrecognized value"
	default:
		return "unrecognized value"
	}
}

// formatSchemaType converts a schema Type field to a string representation
func formatSchemaType(schemaType any) string {
	if schemaType == nil {
		return ""
	}
	return fmt.Sprintf("%v", schemaType)
}

// isPropertyRequired checks if a property name is in the required list
func isPropertyRequired(propertyName string, required []string) bool {
	for _, req := range required {
		if req == propertyName {
			return true
		}
	}
	return false
}
