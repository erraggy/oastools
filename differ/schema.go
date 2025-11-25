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
