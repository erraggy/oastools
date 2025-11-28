package builder

import (
	"reflect"
	"time"

	"github.com/erraggy/oastools/parser"
)

// SchemaFrom generates an OpenAPI schema from a Go type using reflection.
// This is a convenience function for one-off schema generation.
//
// Example:
//
//	type User struct {
//		ID   int64  `json:"id"`
//		Name string `json:"name"`
//	}
//	schema := builder.SchemaFrom(User{})
func SchemaFrom(v any) *parser.Schema {
	b := New(parser.OASVersion320)
	return b.generateSchema(v)
}

// SchemaFromType generates a schema from a reflect.Type.
func SchemaFromType(t reflect.Type) *parser.Schema {
	b := New(parser.OASVersion320)
	return b.generateSchemaFromType(t)
}

// generateSchema converts a Go type to an OpenAPI schema.
func (b *Builder) generateSchema(v any) *parser.Schema {
	if v == nil {
		return &parser.Schema{} // Empty schema for nil
	}

	t := reflect.TypeOf(v)
	return b.generateSchemaFromType(t)
}

// generateSchemaInternal generates a schema with a custom name override.
func (b *Builder) generateSchemaInternal(v any, nameOverride string) *parser.Schema {
	if v == nil {
		return &parser.Schema{}
	}

	t := reflect.TypeOf(v)
	return b.generateSchemaFromTypeWithName(t, nameOverride)
}

// generateSchemaFromType generates a schema from a reflect.Type.
func (b *Builder) generateSchemaFromType(t reflect.Type) *parser.Schema {
	return b.generateSchemaFromTypeWithName(t, "")
}

// generateSchemaFromTypeWithName generates a schema with optional name override.
func (b *Builder) generateSchemaFromTypeWithName(t reflect.Type, nameOverride string) *parser.Schema {
	// Dereference pointers
	isPointer := false
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
		isPointer = true
	}

	// Check for special types first (before cache check)
	// This handles time.Time and similar special types
	if specialSchema := b.generateSpecialTypeSchema(t); specialSchema != nil {
		if isPointer {
			specialSchema.Nullable = true
		}
		return specialSchema
	}

	// Check cache first
	if schema := b.schemaCache.get(t); schema != nil {
		// Return a reference to the cached schema
		if name := b.schemaCache.getNameForType(t); name != "" {
			return b.refToSchema(name)
		}
		return schema
	}

	// Check for circular reference
	if b.schemaCache.isInProgress(t) {
		// Return a reference - the schema will be completed later
		name := b.schemaName(t)
		if nameOverride != "" {
			name = nameOverride
		}
		return b.refToSchema(name)
	}

	// Generate schema based on kind
	var schema *parser.Schema
	switch t.Kind() {
	case reflect.Struct:
		// Mark as in-progress for circular reference detection
		b.schemaCache.markInProgress(t)
		defer b.schemaCache.clearInProgress(t)

		schema = b.generateStructSchema(t)

		// Register named types in components.schemas
		name := b.schemaName(t)
		if nameOverride != "" {
			name = nameOverride
		}
		b.schemas[name] = schema
		b.schemaCache.set(t, name, schema)
		return b.refToSchema(name)

	case reflect.Slice, reflect.Array:
		schema = b.generateArraySchema(t)

	case reflect.Map:
		schema = b.generateMapSchema(t)

	default:
		schema = b.generatePrimitiveSchema(t)
	}

	// Handle pointer nullability
	if isPointer && schema != nil {
		schema.Nullable = true
	}

	return schema
}

// generateSpecialTypeSchema handles special types like time.Time
func (b *Builder) generateSpecialTypeSchema(t reflect.Type) *parser.Schema {
	// Handle time.Time
	if t == reflect.TypeOf(time.Time{}) {
		return &parser.Schema{
			Type:   "string",
			Format: "date-time",
		}
	}

	// Handle uuid.UUID (check by type name since we don't want to import the uuid package)
	if t.String() == "uuid.UUID" {
		return &parser.Schema{
			Type:   "string",
			Format: "uuid",
		}
	}

	return nil
}

// generateStructSchema reflects on a struct type to generate an object schema.
func (b *Builder) generateStructSchema(t reflect.Type) *parser.Schema {
	properties := make(map[string]*parser.Schema)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle embedded structs
		if field.Anonymous {
			embeddedSchema := b.generateSchemaFromType(field.Type)
			// If embedded schema is a ref, resolve it for inlining
			if embeddedSchema.Ref != "" {
				// Get the referenced schema and merge its properties
				refName := extractRefName(embeddedSchema.Ref)
				if refSchema, ok := b.schemas[refName]; ok {
					for propName, propSchema := range refSchema.Properties {
						if _, exists := properties[propName]; !exists {
							properties[propName] = propSchema
						}
					}
					for _, req := range refSchema.Required {
						if !contains(required, req) {
							required = append(required, req)
						}
					}
				}
			} else if embeddedSchema.Properties != nil {
				// Inline the properties
				for propName, propSchema := range embeddedSchema.Properties {
					if _, exists := properties[propName]; !exists {
						properties[propName] = propSchema
					}
				}
				for _, req := range embeddedSchema.Required {
					if !contains(required, req) {
						required = append(required, req)
					}
				}
			}
			continue
		}

		// Parse json tag for field name and options
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue // Explicitly excluded
		}

		name, jsonOpts := parseJSONTag(jsonTag)
		if name == "" {
			name = field.Name
		}

		// Generate schema for field type
		fieldSchema := b.generateSchemaFromType(field.Type)

		// Apply oas tag customizations
		oasTag := field.Tag.Get("oas")
		if oasTag != "" {
			fieldSchema = applyOASTag(fieldSchema, oasTag)
		}

		properties[name] = fieldSchema

		// Determine if required
		if isFieldRequired(field, jsonOpts) {
			required = append(required, name)
		}
	}

	return &parser.Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// generateArraySchema generates a schema for slice/array types.
func (b *Builder) generateArraySchema(t reflect.Type) *parser.Schema {
	elemType := t.Elem()
	itemsSchema := b.generateSchemaFromType(elemType)

	return &parser.Schema{
		Type:  "array",
		Items: itemsSchema,
	}
}

// generateMapSchema generates a schema for map types.
func (b *Builder) generateMapSchema(t reflect.Type) *parser.Schema {
	// Maps with string keys become objects with additionalProperties
	valueType := t.Elem()
	valueSchema := b.generateSchemaFromType(valueType)

	return &parser.Schema{
		Type:                 "object",
		AdditionalProperties: valueSchema,
	}
}

// generatePrimitiveSchema generates a schema for primitive types.
func (b *Builder) generatePrimitiveSchema(t reflect.Type) *parser.Schema {
	// Handle basic types
	switch t.Kind() {
	case reflect.String:
		return &parser.Schema{Type: "string"}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return &parser.Schema{Type: "integer", Format: "int32"}

	case reflect.Int64:
		return &parser.Schema{Type: "integer", Format: "int64"}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return &parser.Schema{Type: "integer", Format: "int32"}

	case reflect.Uint64:
		return &parser.Schema{Type: "integer", Format: "int64"}

	case reflect.Float32:
		return &parser.Schema{Type: "number", Format: "float"}

	case reflect.Float64:
		return &parser.Schema{Type: "number", Format: "double"}

	case reflect.Bool:
		return &parser.Schema{Type: "boolean"}

	case reflect.Interface:
		// interface{} / any becomes an empty schema (accepts anything)
		return &parser.Schema{}

	default:
		// Unknown type - return empty schema
		return &parser.Schema{}
	}
}

// schemaName generates a schema name from a type.
func (b *Builder) schemaName(t reflect.Type) string {
	// Use the type name (without package prefix)
	name := t.Name()
	if name == "" {
		// Anonymous type - generate a unique name
		name = "AnonymousType"
	}
	return name
}

// refToSchema creates a schema with a $ref to a named schema.
func (b *Builder) refToSchema(name string) *parser.Schema {
	return &parser.Schema{
		Ref: "#/components/schemas/" + name,
	}
}

// extractRefName extracts the schema name from a $ref string.
func extractRefName(ref string) string {
	const prefix = "#/components/schemas/"
	if len(ref) > len(prefix) {
		return ref[len(prefix):]
	}
	return ""
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
