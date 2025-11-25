package differ

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// schemaVisited tracks visited schemas during recursive traversal to detect cycles.
// It uses pointer-based identity to detect when we encounter the same schema instance.
type schemaVisited struct {
	visited map[*parser.Schema]string // schema pointer -> first occurrence path
}

// newSchemaVisited creates a new visited tracker for schema traversal.
func newSchemaVisited() *schemaVisited {
	return &schemaVisited{
		visited: make(map[*parser.Schema]string),
	}
}

// enter marks a schema as visited at the given path.
// Returns true if the schema was already visited.
func (v *schemaVisited) enter(schema *parser.Schema, path string) bool {
	if _, exists := v.visited[schema]; exists {
		return true
	}
	v.visited[schema] = path
	return false
}

// leave removes a schema from the visited set.
// This should be called when exiting a schema's traversal to allow revisiting in different contexts.
func (v *schemaVisited) leave(schema *parser.Schema) {
	delete(v.visited, schema)
}

// schemaItemsType represents the possible types for the Items field
type schemaItemsType int

const (
	schemaItemsTypeNil schemaItemsType = iota
	schemaItemsTypeSchema
	schemaItemsTypeBool
	schemaItemsTypeUnknown
)

// getSchemaItemsType determines the type of a schema Items field value
func getSchemaItemsType(items any) schemaItemsType {
	if items == nil {
		return schemaItemsTypeNil
	}
	switch items.(type) {
	case *parser.Schema:
		return schemaItemsTypeSchema
	case bool:
		return schemaItemsTypeBool
	default:
		return schemaItemsTypeUnknown
	}
}

// schemaAdditionalPropsType represents the possible types for AdditionalProperties/AdditionalItems
type schemaAdditionalPropsType int

const (
	schemaAdditionalPropsTypeNil schemaAdditionalPropsType = iota
	schemaAdditionalPropsTypeSchema
	schemaAdditionalPropsTypeBool
	schemaAdditionalPropsTypeUnknown
)

// getSchemaAdditionalPropsType determines the type of AdditionalProperties or AdditionalItems
func getSchemaAdditionalPropsType(additionalProps any) schemaAdditionalPropsType {
	if additionalProps == nil {
		return schemaAdditionalPropsTypeNil
	}
	switch additionalProps.(type) {
	case *parser.Schema:
		return schemaAdditionalPropsTypeSchema
	case bool:
		return schemaAdditionalPropsTypeBool
	default:
		return schemaAdditionalPropsTypeUnknown
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
