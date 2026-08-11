package validator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/internal/schemautil"
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

	// A bare-boolean schema has no keywords, so it is the whole check when
	// present — nothing below applies to it.
	if _, isBool := schema.IsBool(); isBool {
		v.validateBoolSchemaVersion(schema, path, result)
		return
	}

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

	// Validate the discriminator meets the varying requirements of each OAS version
	v.validateDiscriminatorForm(schema, path, result)

	// Validate the OAS 3.2 schema-level fields (xml.nodeType, discriminator.defaultMapping)
	v.validateOAS32SchemaFields(schema, path, result)

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
		// No OAS version requires `items` on an array-typed *Schema Object*.
		// It is absent from schema.yaml, meta.yaml and dialect.yaml, and no
		// version's prose states it. OAS 2.0 does require it when `type` is
		// "array" — but only on the Items Object and on non-body Parameters
		// and Headers, which are `parser.Items`, not `parser.Schema`. That
		// rule lives in validateOAS2PrimitiveParameter,
		// validateOAS2ResponseHeaders and validateOAS2Items.
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
	case "null":
		// "null" is only a valid JSON Schema type in OAS 3.1+ (JSON Schema 2020-12).
		// In OAS 3.0.x, the only valid types are: array, boolean, integer, number,
		// object, string. Nullability is expressed via "nullable: true".
		// Note: this only catches the scalar form (schema.Type == "null"). OAS 3.1+
		// type arrays (e.g. ["string", "null"]) are represented as []any and bypass
		// this switch entirely, which is the correct behavior.
		if isOAS30x(v.oasVersion) {
			v.addError(result, path,
				`"null" is not a valid type for OpenAPI 3.0; valid types are: array, boolean, integer, number, object, string. Use "nullable: true" instead.`,
				withSpecRef("https://spec.openapis.org/oas/v3.0.0.html#data-types"),
				withField("type"),
				withValue("null"),
			)
		}
	}
}

// validateBoolSchemaVersion rejects the bare-boolean schema form for the
// versions that predate it.
//
// JSON Schema 2020-12 allows `true` and `false` wherever a schema is expected,
// and OAS 3.1 adopted that dialect wholesale. OAS 3.0 is based on an earlier
// draft where a schema is always an object, and OAS 2.0 more so. The parser
// accepts the form regardless of version — a Schema Object is decoded before
// the document version is known to it — which makes this check the only thing
// standing between a 3.0 document and a silently accepted boolean schema.
//
// The same division of labour as validateDiscriminatorForm.
func (v *Validator) validateBoolSchemaVersion(schema *parser.Schema, path string, result *ValidationResult) {
	// An unrecognized version says nothing about which forms are legal.
	if !v.oasVersion.IsValid() || v.oasVersion >= parser.OASVersion310 {
		return
	}
	b, _ := schema.IsBool()
	v.addError(result, path,
		fmt.Sprintf("Boolean schemas (%t) require OpenAPI 3.1 or later; in this version a schema must be an object", b),
		withSpecRef(getJSONSchemaRef()),
		withValue(b),
	)
}

// isOAS30x reports whether the given version is in the OAS 3.0.x family,
// where "null" is not a valid schema type.
func isOAS30x(version parser.OASVersion) bool {
	return version >= parser.OASVersion300 && version <= parser.OASVersion304
}

// validateDiscriminatorForm checks that a schema's discriminator uses the
// form its OAS version requires: a bare string in OAS 2.0, an object with
// propertyName in OAS 3.0+.
//
// The parser accepts both forms regardless of version, because a Schema Object
// is decoded before the document version is known to it. That makes this check
// the only thing standing between a document and a silently accepted
// cross-dialect discriminator.
func (v *Validator) validateDiscriminatorForm(schema *parser.Schema, path string, result *ValidationResult) {
	if schema.Discriminator == nil {
		return
	}

	// An unrecognized version says nothing about which form is correct, so
	// there is nothing to enforce. ValidateParsed always sets this; a
	// hand-assembled ParseResult may not.
	if !v.oasVersion.IsValid() {
		return
	}

	switch {
	case v.oasVersion == parser.OASVersion20 && !schema.Discriminator.StringForm:
		v.addError(result, path,
			"discriminator must be a string naming the property in OpenAPI 2.0; the object form with 'propertyName' was introduced in OpenAPI 3.0",
			withSpecRef("https://spec.openapis.org/oas/v2.0.html#schema-object"),
			withField("discriminator"),
			withValue(schema.Discriminator.PropertyName),
		)

	case v.oasVersion != parser.OASVersion20 && schema.Discriminator.StringForm:
		v.addError(result, path,
			"discriminator must be an object with 'propertyName' in OpenAPI 3.0+; the bare string form is OpenAPI 2.0 only",
			withSpecRef("https://spec.openapis.org/oas/v3.0.0.html#discriminator-object"),
			withField("discriminator"),
			withValue(schema.Discriminator.PropertyName),
		)
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

	// Validate additionalProperties (can be *Schema, []*Schema, or bool)
	for i, addProps := range schemautil.SchemaOrBoolSchemas(schema.AdditionalProperties) {
		v.validateSchemaWithVisited(addProps, path+".additionalProperties"+schemautil.IndexSuffix(i), result, visited, nextDepth)
	}

	// Validate items (can be *Schema, []*Schema, or bool)
	for i, items := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		v.validateSchemaWithVisited(items, path+".items"+schemautil.IndexSuffix(i), result, visited, nextDepth)
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
