package builder

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// parseJSONTag parses a struct field's json tag.
// Returns the field name and options (like "omitempty").
func parseJSONTag(tag string) (name string, opts []string) {
	if tag == "" {
		return "", nil
	}

	parts := strings.Split(tag, ",")
	name = parts[0]
	if len(parts) > 1 {
		opts = parts[1:]
	}
	return name, opts
}

// hasOmitempty checks if json tag options include omitempty.
func hasOmitempty(opts []string) bool {
	for _, opt := range opts {
		if opt == "omitempty" {
			return true
		}
	}
	return false
}

// isFieldRequired determines if a struct field should be marked as required.
// Rules:
//  1. Non-pointer fields without omitempty are required
//  2. Fields with oas:"required=true" are explicitly required
//  3. Fields with oas:"required=false" are explicitly optional
//  4. Pointer fields are optional by default
func isFieldRequired(field reflect.StructField, jsonOpts []string) bool {
	// Check for explicit required setting in oas tag
	oasTag := field.Tag.Get("oas")
	if oasTag != "" {
		opts := parseOASTag(oasTag)
		if val, ok := opts["required"]; ok {
			return val == "true"
		}
	}

	// Pointer fields are optional by default
	if field.Type.Kind() == reflect.Ptr {
		return false
	}

	// Non-pointer fields without omitempty are required
	return !hasOmitempty(jsonOpts)
}

// parseOASTag parses the oas struct tag into a map of key-value pairs.
// Supports formats like: oas:"description=User ID,minLength=1,maxLength=100"
func parseOASTag(tag string) map[string]string {
	result := make(map[string]string)
	if tag == "" {
		return result
	}

	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle key=value pairs
		if idx := strings.Index(part, "="); idx > 0 {
			key := strings.TrimSpace(part[:idx])
			value := strings.TrimSpace(part[idx+1:])
			result[key] = value
		} else {
			// Handle boolean flags (e.g., "deprecated" without =true)
			result[part] = "true"
		}
	}

	return result
}

// applyOASTag applies oas tag options to a schema.
// Returns a new schema with the tag options applied.
func applyOASTag(schema *parser.Schema, tag string) *parser.Schema {
	opts := parseOASTag(tag)
	if len(opts) == 0 {
		return schema
	}

	// Create a copy to avoid modifying the original
	result := copySchema(schema)

	for key, value := range opts {
		switch key {
		case "description":
			result.Description = value

		case "format":
			result.Format = value

		case "enum":
			// Parse pipe-separated enum values
			enumValues := strings.Split(value, "|")
			result.Enum = make([]any, len(enumValues))
			for i, v := range enumValues {
				result.Enum[i] = strings.TrimSpace(v)
			}

		case "minimum":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				result.Minimum = &f
			}

		case "maximum":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				result.Maximum = &f
			}

		case "minLength":
			if n, err := strconv.Atoi(value); err == nil {
				result.MinLength = &n
			}

		case "maxLength":
			if n, err := strconv.Atoi(value); err == nil {
				result.MaxLength = &n
			}

		case "pattern":
			result.Pattern = value

		case "minItems":
			if n, err := strconv.Atoi(value); err == nil {
				result.MinItems = &n
			}

		case "maxItems":
			if n, err := strconv.Atoi(value); err == nil {
				result.MaxItems = &n
			}

		case "readOnly":
			result.ReadOnly = value == "true"

		case "writeOnly":
			result.WriteOnly = value == "true"

		case "nullable":
			result.Nullable = value == "true"

		case "deprecated":
			result.Deprecated = value == "true"

		case "example":
			// Try to parse as JSON, otherwise use as string
			result.Example = value

		case "title":
			result.Title = value

		case "default":
			// Try to parse as appropriate type based on schema type
			result.Default = parseDefaultValue(value, result.Type)
		}
	}

	return result
}

// copySchema creates a shallow copy of a schema.
// Note: This is sufficient for tag application since we only modify top-level fields.
func copySchema(s *parser.Schema) *parser.Schema {
	if s == nil {
		return nil
	}

	// Copy the basic fields
	result := &parser.Schema{
		Ref:         s.Ref,
		Type:        s.Type,
		Format:      s.Format,
		Title:       s.Title,
		Description: s.Description,
		Default:     s.Default,
		Nullable:    s.Nullable,
		ReadOnly:    s.ReadOnly,
		WriteOnly:   s.WriteOnly,
		Deprecated:  s.Deprecated,
		Pattern:     s.Pattern,
		UniqueItems: s.UniqueItems,
	}

	// Deep copy pointer fields
	if s.Minimum != nil {
		minCopy := *s.Minimum
		result.Minimum = &minCopy
	}
	if s.Maximum != nil {
		maxCopy := *s.Maximum
		result.Maximum = &maxCopy
	}
	if s.MinLength != nil {
		minLenCopy := *s.MinLength
		result.MinLength = &minLenCopy
	}
	if s.MaxLength != nil {
		maxLenCopy := *s.MaxLength
		result.MaxLength = &maxLenCopy
	}
	if s.MinItems != nil {
		minItemsCopy := *s.MinItems
		result.MinItems = &minItemsCopy
	}
	if s.MaxItems != nil {
		maxItemsCopy := *s.MaxItems
		result.MaxItems = &maxItemsCopy
	}
	if s.MinProperties != nil {
		minPropsCopy := *s.MinProperties
		result.MinProperties = &minPropsCopy
	}
	if s.MaxProperties != nil {
		maxPropsCopy := *s.MaxProperties
		result.MaxProperties = &maxPropsCopy
	}
	if s.MultipleOf != nil {
		multOfCopy := *s.MultipleOf
		result.MultipleOf = &multOfCopy
	}
	if s.ExclusiveMaximum != nil {
		result.ExclusiveMaximum = s.ExclusiveMaximum
	}
	if s.ExclusiveMinimum != nil {
		result.ExclusiveMinimum = s.ExclusiveMinimum
	}

	// Deep copy slices that might be modified
	if s.Enum != nil {
		result.Enum = make([]any, len(s.Enum))
		copy(result.Enum, s.Enum)
	}
	if s.Required != nil {
		result.Required = make([]string, len(s.Required))
		copy(result.Required, s.Required)
	}
	// Copy slices (shallow reference for immutable schemas)
	result.Examples = s.Examples
	result.AllOf = s.AllOf
	result.AnyOf = s.AnyOf
	result.OneOf = s.OneOf

	// Copy maps (shallow reference)
	result.Properties = s.Properties
	result.PatternProperties = s.PatternProperties
	result.DependentRequired = s.DependentRequired
	result.DependentSchemas = s.DependentSchemas

	// Copy other fields
	result.Items = s.Items
	result.AdditionalProperties = s.AdditionalProperties
	result.AdditionalItems = s.AdditionalItems
	result.Not = s.Not
	result.If = s.If
	result.Then = s.Then
	result.Else = s.Else
	result.Discriminator = s.Discriminator
	result.XML = s.XML
	result.ExternalDocs = s.ExternalDocs
	result.Example = s.Example

	return result
}

// parseDefaultValue attempts to parse a default value string
// based on the schema type.
func parseDefaultValue(value string, schemaType any) any {
	typeStr, ok := schemaType.(string)
	if !ok {
		return value
	}

	switch typeStr {
	case "integer":
		if n, err := strconv.ParseInt(value, 10, 64); err == nil {
			return n
		}
	case "number":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	case "boolean":
		return value == "true"
	}

	return value
}
