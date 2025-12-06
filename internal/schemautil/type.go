// Package schemautil provides utilities for working with OpenAPI schema types.
//
// This package centralizes type assertion patterns for OAS version-specific fields,
// particularly handling the differences between OAS 2.0/3.0 (string types) and
// OAS 3.1+ (array types for nullable support).
package schemautil

import "github.com/erraggy/oastools/parser"

// GetSchemaTypes returns the type(s) from a schema, handling both
// string (OAS 2.0/3.0) and []any (OAS 3.1+) representations.
//
// Examples:
//   - OAS 3.0: {"type": "string"} returns ["string"]
//   - OAS 3.1: {"type": ["string", "null"]} returns ["string", "null"]
func GetSchemaTypes(schema *parser.Schema) []string {
	if schema == nil {
		return nil
	}
	switch t := schema.Type.(type) {
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	case []any:
		result := make([]string, 0, len(t))
		for _, v := range t {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return t
	}
	return nil
}

// GetPrimaryType returns the first non-null type from a schema.
// This is useful for OAS 3.1+ where type arrays may include "null".
//
// Returns an empty string if the schema is nil or has no types.
func GetPrimaryType(schema *parser.Schema) string {
	types := GetSchemaTypes(schema)
	for _, t := range types {
		if t != "null" {
			return t
		}
	}
	if len(types) > 0 {
		return types[0]
	}
	return ""
}

// IsNullable checks if the schema allows null values.
// In OAS 3.1+, this is indicated by "null" in the type array.
// In OAS 3.0, this is indicated by the nullable field (not checked here).
func IsNullable(schema *parser.Schema) bool {
	for _, t := range GetSchemaTypes(schema) {
		if t == "null" {
			return true
		}
	}
	return false
}

// HasType checks if the schema includes the specified type.
func HasType(schema *parser.Schema, targetType string) bool {
	for _, t := range GetSchemaTypes(schema) {
		if t == targetType {
			return true
		}
	}
	return false
}

// IsSingleType returns true if the schema has exactly one type (not counting null).
func IsSingleType(schema *parser.Schema) bool {
	types := GetSchemaTypes(schema)
	nonNullCount := 0
	for _, t := range types {
		if t != "null" {
			nonNullCount++
		}
	}
	return nonNullCount == 1
}
