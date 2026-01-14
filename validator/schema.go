package validator

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// validateSchemaName checks if a schema name is valid (non-empty, non-whitespace).
func (v *Validator) validateSchemaName(name, pathPrefix string, result *ValidationResult) {
	if name == "" {
		result.Errors = append(result.Errors, ValidationError{
			Path:     pathPrefix,
			Message:  "schema name cannot be empty",
			Severity: SeverityError,
			Field:    "name",
			Value:    "",
		})
		return
	}
	if strings.TrimSpace(name) == "" {
		result.Errors = append(result.Errors, ValidationError{
			Path:     fmt.Sprintf("%s.%s", pathPrefix, name),
			Message:  fmt.Sprintf("schema name cannot be whitespace-only: %q", name),
			Severity: SeverityError,
			Field:    "name",
			Value:    name,
		})
	}
}

// validateSchema performs basic schema validation
func (v *Validator) validateSchema(schema *parser.Schema, path string, result *ValidationResult) {
	v.validateSchemaWithVisited(schema, path, result, make(map[*parser.Schema]bool))
}

// validateSchemaWithVisited performs basic schema validation with cycle detection
func (v *Validator) validateSchemaWithVisited(schema *parser.Schema, path string, result *ValidationResult, visited map[*parser.Schema]bool) {
	if schema == nil {
		return
	}

	// Check for circular references
	if visited[schema] {
		return
	}
	visited[schema] = true

	// Check for excessive nesting depth to prevent resource exhaustion
	depth := strings.Count(path, ".")
	if depth > maxSchemaNestingDepth {
		result.Errors = append(result.Errors, ValidationError{
			Path:     path,
			Message:  fmt.Sprintf("Schema nesting depth (%d) exceeds maximum allowed (%d)", depth, maxSchemaNestingDepth),
			SpecRef:  getJSONSchemaRef(),
			Severity: SeverityError,
		})
		return
	}

	// Validate enum values match the schema type
	if len(schema.Enum) > 0 && schema.Type != "" {
		v.validateEnumValues(schema, path, result)
	}

	// Validate type-specific constraints
	v.validateSchemaTypeConstraints(schema, path, result)

	// Validate required fields
	v.validateRequiredFields(schema, path, result)

	// Validate nested schemas
	v.validateNestedSchemas(schema, path, result, visited)
}

// validateEnumValues validates that enum values match the schema type
func (v *Validator) validateEnumValues(schema *parser.Schema, path string, result *ValidationResult) {
	for i, enumVal := range schema.Enum {
		enumPath := fmt.Sprintf("%s.enum[%d]", path, i)

		switch schema.Type {
		case "string":
			if _, ok := enumVal.(string); !ok {
				result.Errors = append(result.Errors, ValidationError{
					Path:     enumPath,
					Message:  fmt.Sprintf("Enum value must be a string (found %T)", enumVal),
					SpecRef:  getJSONSchemaRef(),
					Severity: SeverityError,
					Field:    "enum",
					Value:    enumVal,
				})
			}
		case "integer":
			// Check if it's an integer (can be int, int32, int64, or float64 with no decimal part)
			switch v := enumVal.(type) {
			case int, int32, int64:
				// Valid integer
			case float64:
				if v != float64(int64(v)) {
					result.Errors = append(result.Errors, ValidationError{
						Path:     enumPath,
						Message:  fmt.Sprintf("Enum value must be an integer (found %v)", enumVal),
						SpecRef:  getJSONSchemaRef(),
						Severity: SeverityError,
						Field:    "enum",
						Value:    enumVal,
					})
				}
			default:
				result.Errors = append(result.Errors, ValidationError{
					Path:     enumPath,
					Message:  fmt.Sprintf("Enum value must be an integer (found %T)", enumVal),
					SpecRef:  getJSONSchemaRef(),
					Severity: SeverityError,
					Field:    "enum",
					Value:    enumVal,
				})
			}
		case "number":
			// Check if it's a number (int or float)
			switch enumVal.(type) {
			case int, int32, int64, float32, float64:
				// Valid number
			default:
				result.Errors = append(result.Errors, ValidationError{
					Path:     enumPath,
					Message:  fmt.Sprintf("Enum value must be a number (found %T)", enumVal),
					SpecRef:  getJSONSchemaRef(),
					Severity: SeverityError,
					Field:    "enum",
					Value:    enumVal,
				})
			}
		case "boolean":
			if _, ok := enumVal.(bool); !ok {
				result.Errors = append(result.Errors, ValidationError{
					Path:     enumPath,
					Message:  fmt.Sprintf("Enum value must be a boolean (found %T)", enumVal),
					SpecRef:  getJSONSchemaRef(),
					Severity: SeverityError,
					Field:    "enum",
					Value:    enumVal,
				})
			}
		case "null":
			if enumVal != nil {
				result.Errors = append(result.Errors, ValidationError{
					Path:     enumPath,
					Message:  "Enum value must be null",
					SpecRef:  getJSONSchemaRef(),
					Severity: SeverityError,
					Field:    "enum",
					Value:    enumVal,
				})
			}
		}
	}
}

// validateSchemaTypeConstraints validates type-specific constraints for a schema
func (v *Validator) validateSchemaTypeConstraints(schema *parser.Schema, path string, result *ValidationResult) {
	if schema.Type == "" {
		return
	}

	switch schema.Type {
	case "array":
		if schema.Items == nil {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  "Array schema must have 'items' defined",
				SpecRef:  getJSONSchemaRef(),
				Severity: SeverityError,
				Field:    "items",
			})
		}
	case "string":
		// Validate min/max length
		if schema.MinLength != nil && schema.MaxLength != nil && *schema.MinLength > *schema.MaxLength {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("minLength (%d) cannot be greater than maxLength (%d)", *schema.MinLength, *schema.MaxLength),
				SpecRef:  getJSONSchemaRef(),
				Severity: SeverityError,
			})
		}
	case "number", "integer":
		// Validate minimum/maximum
		if schema.Minimum != nil && schema.Maximum != nil && *schema.Minimum > *schema.Maximum {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("minimum (%v) cannot be greater than maximum (%v)", *schema.Minimum, *schema.Maximum),
				SpecRef:  getJSONSchemaRef(),
				Severity: SeverityError,
			})
		}
	}
}

// validateRequiredFields validates that required fields exist in properties
func (v *Validator) validateRequiredFields(schema *parser.Schema, path string, result *ValidationResult) {
	for _, reqField := range schema.Required {
		if _, exists := schema.Properties[reqField]; !exists {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("Required field '%s' not found in properties", reqField),
				SpecRef:  getJSONSchemaRef(),
				Severity: SeverityError,
				Field:    "required",
				Value:    reqField,
			})
		}
	}
}

// validateNestedSchemas validates all nested schemas (properties, items, allOf, oneOf, anyOf, not)
func (v *Validator) validateNestedSchemas(schema *parser.Schema, path string, result *ValidationResult, visited map[*parser.Schema]bool) {
	// Validate properties
	for propName, propSchema := range schema.Properties {
		if propSchema == nil {
			continue
		}
		propPath := fmt.Sprintf("%s.properties.%s", path, propName)
		v.validateSchemaWithVisited(propSchema, propPath, result, visited)
	}

	// Validate additionalProperties (can be *Schema or bool)
	if schema.AdditionalProperties != nil {
		if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			addPropsPath := fmt.Sprintf("%s.additionalProperties", path)
			v.validateSchemaWithVisited(addProps, addPropsPath, result, visited)
		}
	}

	// Validate items (can be *Schema or bool)
	if schema.Items != nil {
		if items, ok := schema.Items.(*parser.Schema); ok {
			itemsPath := fmt.Sprintf("%s.items", path)
			v.validateSchemaWithVisited(items, itemsPath, result, visited)
		}
	}

	// Validate allOf
	for i, subSchema := range schema.AllOf {
		if subSchema == nil {
			continue
		}
		subPath := fmt.Sprintf("%s.allOf[%d]", path, i)
		v.validateSchemaWithVisited(subSchema, subPath, result, visited)
	}

	// Validate oneOf
	for i, subSchema := range schema.OneOf {
		if subSchema == nil {
			continue
		}
		subPath := fmt.Sprintf("%s.oneOf[%d]", path, i)
		v.validateSchemaWithVisited(subSchema, subPath, result, visited)
	}

	// Validate anyOf
	for i, subSchema := range schema.AnyOf {
		if subSchema == nil {
			continue
		}
		subPath := fmt.Sprintf("%s.anyOf[%d]", path, i)
		v.validateSchemaWithVisited(subSchema, subPath, result, visited)
	}

	// Validate not
	if schema.Not != nil {
		notPath := fmt.Sprintf("%s.not", path)
		v.validateSchemaWithVisited(schema.Not, notPath, result, visited)
	}
}
