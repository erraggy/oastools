package validator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// validateSchemaName checks if a schema name is valid (non-empty, non-whitespace).
func (v *Validator) validateSchemaName(name, pathPrefix string, result *ValidationResult) {
	if name == "" {
		v.addError(result, pathPrefix, "schema name cannot be empty",
			withField("name"),
			withValue(""),
		)
		return
	}
	if strings.TrimSpace(name) == "" {
		v.addError(result, pathPrefix+"."+name,
			fmt.Sprintf("schema name cannot be whitespace-only: %q", name),
			withField("name"),
			withValue(name),
		)
	}
}

// validateSchema performs basic schema validation
func (v *Validator) validateSchema(schema *parser.Schema, path string, result *ValidationResult) {
	v.validateSchemaWithVisited(schema, path, result, make(map[*parser.Schema]bool), 0)
}

// validateSchemaWithVisited performs basic schema validation with cycle detection
func (v *Validator) validateSchemaWithVisited(schema *parser.Schema, path string, result *ValidationResult, visited map[*parser.Schema]bool, depth int) {
	if schema == nil {
		return
	}

	// Check for circular references
	if visited[schema] {
		return
	}
	visited[schema] = true

	// Check for excessive nesting depth to prevent resource exhaustion
	if depth > maxSchemaNestingDepth {
		v.addError(result, path,
			fmt.Sprintf("Schema nesting depth (%d) exceeds maximum allowed (%d)", depth, maxSchemaNestingDepth),
			withSpecRef(getJSONSchemaRef()),
		)
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
	v.validateNestedSchemas(schema, path, result, visited, depth)
}

// validateEnumValues validates that enum values match the schema type
func (v *Validator) validateEnumValues(schema *parser.Schema, path string, result *ValidationResult) {
	for i, enumVal := range schema.Enum {
		enumPath := path + ".enum[" + strconv.Itoa(i) + "]"

		switch schema.Type {
		case "string":
			if _, ok := enumVal.(string); !ok {
				v.addError(result, enumPath,
					fmt.Sprintf("Enum value must be a string (found %T)", enumVal),
					withSpecRef(getJSONSchemaRef()),
					withField("enum"),
					withValue(enumVal),
				)
			}
		case "integer":
			// Check if it's an integer (can be int, int32, int64, or float64 with no decimal part)
			switch ev := enumVal.(type) {
			case int, int32, int64:
				// Valid integer
			case float64:
				if ev != float64(int64(ev)) {
					v.addError(result, enumPath,
						fmt.Sprintf("Enum value must be an integer (found %v)", enumVal),
						withSpecRef(getJSONSchemaRef()),
						withField("enum"),
						withValue(enumVal),
					)
				}
			default:
				v.addError(result, enumPath,
					fmt.Sprintf("Enum value must be an integer (found %T)", enumVal),
					withSpecRef(getJSONSchemaRef()),
					withField("enum"),
					withValue(enumVal),
				)
			}
		case "number":
			// Check if it's a number (int or float)
			switch enumVal.(type) {
			case int, int32, int64, float32, float64:
				// Valid number
			default:
				v.addError(result, enumPath,
					fmt.Sprintf("Enum value must be a number (found %T)", enumVal),
					withSpecRef(getJSONSchemaRef()),
					withField("enum"),
					withValue(enumVal),
				)
			}
		case "boolean":
			if _, ok := enumVal.(bool); !ok {
				v.addError(result, enumPath,
					fmt.Sprintf("Enum value must be a boolean (found %T)", enumVal),
					withSpecRef(getJSONSchemaRef()),
					withField("enum"),
					withValue(enumVal),
				)
			}
		case "null":
			if enumVal != nil {
				v.addError(result, enumPath,
					"Enum value must be null",
					withSpecRef(getJSONSchemaRef()),
					withField("enum"),
					withValue(enumVal),
				)
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
			v.addError(result, path,
				"Array schema must have 'items' defined",
				withSpecRef(getJSONSchemaRef()),
				withField("items"),
			)
		}
	case "string":
		// Validate min/max length
		if schema.MinLength != nil && schema.MaxLength != nil && *schema.MinLength > *schema.MaxLength {
			v.addError(result, path,
				fmt.Sprintf("minLength (%d) cannot be greater than maxLength (%d)", *schema.MinLength, *schema.MaxLength),
				withSpecRef(getJSONSchemaRef()),
			)
		}
	case "number", "integer":
		// Validate minimum/maximum
		if schema.Minimum != nil && schema.Maximum != nil && *schema.Minimum > *schema.Maximum {
			v.addError(result, path,
				fmt.Sprintf("minimum (%v) cannot be greater than maximum (%v)", *schema.Minimum, *schema.Maximum),
				withSpecRef(getJSONSchemaRef()),
			)
		}
	}
}

// validateRequiredFields validates that required fields exist in properties
func (v *Validator) validateRequiredFields(schema *parser.Schema, path string, result *ValidationResult) {
	for _, reqField := range schema.Required {
		if _, exists := schema.Properties[reqField]; !exists {
			v.addError(result, path,
				fmt.Sprintf("Required field '%s' not found in properties", reqField),
				withSpecRef(getJSONSchemaRef()),
				withField("required"),
				withValue(reqField),
			)
		}
	}
}

// validateNestedSchemas validates all nested schemas (properties, items, allOf, oneOf, anyOf, not)
func (v *Validator) validateNestedSchemas(schema *parser.Schema, path string, result *ValidationResult, visited map[*parser.Schema]bool, depth int) {
	nextDepth := depth + 1

	// Validate properties
	for propName, propSchema := range schema.Properties {
		if propSchema == nil {
			continue
		}
		v.validateSchemaWithVisited(propSchema, path+".properties."+propName, result, visited, nextDepth)
	}

	// Validate additionalProperties (can be *Schema or bool)
	if schema.AdditionalProperties != nil {
		if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			v.validateSchemaWithVisited(addProps, path+".additionalProperties", result, visited, nextDepth)
		}
	}

	// Validate items (can be *Schema or bool)
	if schema.Items != nil {
		if items, ok := schema.Items.(*parser.Schema); ok {
			v.validateSchemaWithVisited(items, path+".items", result, visited, nextDepth)
		}
	}

	// Validate allOf
	for i, subSchema := range schema.AllOf {
		if subSchema == nil {
			continue
		}
		v.validateSchemaWithVisited(subSchema, path+".allOf["+strconv.Itoa(i)+"]", result, visited, nextDepth)
	}

	// Validate oneOf
	for i, subSchema := range schema.OneOf {
		if subSchema == nil {
			continue
		}
		v.validateSchemaWithVisited(subSchema, path+".oneOf["+strconv.Itoa(i)+"]", result, visited, nextDepth)
	}

	// Validate anyOf
	for i, subSchema := range schema.AnyOf {
		if subSchema == nil {
			continue
		}
		v.validateSchemaWithVisited(subSchema, path+".anyOf["+strconv.Itoa(i)+"]", result, visited, nextDepth)
	}

	// Validate not
	if schema.Not != nil {
		v.validateSchemaWithVisited(schema.Not, path+".not", result, visited, nextDepth)
	}
}
