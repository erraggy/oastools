package httpvalidator

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// ParamDeserializer handles deserialization of HTTP parameters according to
// OpenAPI serialization styles. Each parameter location has default styles:
//
// | Location | Default Style | Default Explode |
// |----------|---------------|-----------------|
// | path     | simple        | false           |
// | query    | form          | true            |
// | header   | simple        | false           |
// | cookie   | form          | false           |
type ParamDeserializer struct{}

// NewParamDeserializer creates a new parameter deserializer.
func NewParamDeserializer() *ParamDeserializer {
	return &ParamDeserializer{}
}

// DeserializePathParam deserializes a path parameter value according to its style.
// Path parameters default to style "simple" with explode=false.
//
// Styles supported:
//   - simple (default): comma-separated values, e.g., "a,b,c"
//   - label: dot-prefixed values, e.g., ".a.b.c"
//   - matrix: semicolon-prefixed key=value, e.g., ";id=5"
func (d *ParamDeserializer) DeserializePathParam(value string, param *parser.Parameter) any {
	style := param.Style
	if style == "" {
		style = "simple"
	}

	// Default explode is false for path params
	explode := false
	if param.Explode != nil {
		explode = *param.Explode
	}

	schema := param.Schema

	switch style {
	case "simple":
		return d.deserializeSimple(value, schema, explode)
	case "label":
		return d.deserializeLabel(value, schema, explode)
	case "matrix":
		return d.deserializeMatrix(value, param.Name, schema, explode)
	default:
		// Unknown style, return raw value
		return value
	}
}

// DeserializeQueryParam deserializes query parameter values according to their style.
// Query parameters default to style "form" with explode=true.
//
// Styles supported:
//   - form (default): standard query string format
//   - spaceDelimited: space-separated values
//   - pipeDelimited: pipe-separated values
//   - deepObject: nested object notation, e.g., "filter[status]=active"
func (d *ParamDeserializer) DeserializeQueryParam(values []string, param *parser.Parameter) any {
	style := param.Style
	if style == "" {
		style = "form"
	}

	// Default explode is true for query params with form style
	explode := true
	if param.Explode != nil {
		explode = *param.Explode
	}

	schema := param.Schema

	switch style {
	case "form":
		return d.deserializeForm(values, schema, explode)
	case "spaceDelimited":
		return d.deserializeDelimited(values, " ", schema)
	case "pipeDelimited":
		return d.deserializeDelimited(values, "|", schema)
	case "deepObject":
		// deepObject is handled at a higher level with the full query string
		// Here we just return the values as-is
		if len(values) == 1 {
			return values[0]
		}
		return values
	default:
		if len(values) == 1 {
			return values[0]
		}
		return values
	}
}

// DeserializeQueryParamsDeepObject deserializes query parameters using deepObject style.
// This handles nested object notation like "filter[status]=active&filter[type]=user".
//
// Returns a map representing the nested object structure.
func (d *ParamDeserializer) DeserializeQueryParamsDeepObject(queryValues url.Values, paramName string, schema *parser.Schema) map[string]any {
	prefix := paramName + "["
	result := make(map[string]any)

	for key, values := range queryValues {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Extract property name from filter[property]
		propEnd := strings.Index(key[len(prefix):], "]")
		if propEnd == -1 {
			continue
		}
		propName := key[len(prefix) : len(prefix)+propEnd]

		if len(values) == 1 {
			result[propName] = d.coerceValue(values[0], d.getPropertySchema(schema, propName))
		} else {
			result[propName] = values
		}
	}

	return result
}

// DeserializeHeaderParam deserializes a header parameter value.
// Header parameters default to style "simple" with explode=false.
func (d *ParamDeserializer) DeserializeHeaderParam(value string, param *parser.Parameter) any {
	// Header params use simple style by default
	// Currently only simple style is implemented for headers

	// Default explode is false for header params
	explode := false
	if param.Explode != nil {
		explode = *param.Explode
	}

	return d.deserializeSimple(value, param.Schema, explode)
}

// DeserializeCookieParam deserializes a cookie parameter value.
// Cookie parameters default to style "form" with explode=false.
func (d *ParamDeserializer) DeserializeCookieParam(value string, param *parser.Parameter) any {
	// Cookie params use form style by default
	// Currently only form style is implemented for cookies

	schema := param.Schema

	// For cookies, we only get a single value string
	// Form style without explode is comma-separated for arrays
	if isArraySchema(schema) {
		return d.deserializeSimple(value, schema, false)
	}

	return d.coerceValue(value, schema)
}

// deserializeSimple handles the "simple" style (comma-separated).
// Used by path and header parameters by default.
func (d *ParamDeserializer) deserializeSimple(value string, schema *parser.Schema, explode bool) any {
	if schema == nil {
		return value
	}

	if isArraySchema(schema) {
		// Split by comma
		parts := strings.Split(value, ",")
		return d.coerceArray(parts, getItemsSchema(schema))
	}

	if isObjectSchema(schema) {
		return d.deserializeSimpleObject(value, schema, explode)
	}

	return d.coerceValue(value, schema)
}

// deserializeSimpleObject handles object deserialization in simple style.
func (d *ParamDeserializer) deserializeSimpleObject(value string, schema *parser.Schema, explode bool) map[string]any {
	result := make(map[string]any)
	parts := strings.Split(value, ",")

	if explode {
		// explode=true: key=value,key2=value2
		for _, part := range parts {
			if idx := strings.Index(part, "="); idx > 0 {
				key := part[:idx]
				val := part[idx+1:]
				result[key] = d.coerceValue(val, d.getPropertySchema(schema, key))
			}
		}
	} else {
		// explode=false: key,value,key2,value2
		for i := 0; i+1 < len(parts); i += 2 {
			key := parts[i]
			val := parts[i+1]
			result[key] = d.coerceValue(val, d.getPropertySchema(schema, key))
		}
	}

	return result
}

// deserializeLabel handles the "label" style (dot-prefixed).
func (d *ParamDeserializer) deserializeLabel(value string, schema *parser.Schema, explode bool) any {
	// Label style starts with a dot
	if !strings.HasPrefix(value, ".") {
		return value
	}
	value = value[1:] // Remove leading dot

	if schema == nil {
		return value
	}

	if isArraySchema(schema) {
		var parts []string
		if explode {
			// explode=true: .a.b.c
			parts = strings.Split(value, ".")
		} else {
			// explode=false: .a,b,c
			parts = strings.Split(value, ",")
		}
		return d.coerceArray(parts, getItemsSchema(schema))
	}

	if isObjectSchema(schema) {
		return d.deserializeLabelObject(value, schema, explode)
	}

	return d.coerceValue(value, schema)
}

// deserializeLabelObject handles object deserialization in label style.
func (d *ParamDeserializer) deserializeLabelObject(value string, schema *parser.Schema, explode bool) map[string]any {
	result := make(map[string]any)

	if explode {
		// explode=true: .key=value.key2=value2
		parts := strings.Split(value, ".")
		for _, part := range parts {
			if part == "" {
				continue
			}
			if idx := strings.Index(part, "="); idx > 0 {
				key := part[:idx]
				val := part[idx+1:]
				result[key] = d.coerceValue(val, d.getPropertySchema(schema, key))
			}
		}
	} else {
		// explode=false: .key,value,key2,value2
		parts := strings.Split(value, ",")
		for i := 0; i+1 < len(parts); i += 2 {
			key := parts[i]
			val := parts[i+1]
			result[key] = d.coerceValue(val, d.getPropertySchema(schema, key))
		}
	}

	return result
}

// deserializeMatrix handles the "matrix" style (semicolon-prefixed).
func (d *ParamDeserializer) deserializeMatrix(value, paramName string, schema *parser.Schema, explode bool) any {
	// Matrix style starts with semicolon
	if !strings.HasPrefix(value, ";") {
		return value
	}
	value = value[1:] // Remove leading semicolon

	if schema == nil {
		// Try to extract value from ;name=value
		if strings.HasPrefix(value, paramName+"=") {
			return value[len(paramName)+1:]
		}
		return value
	}

	if isArraySchema(schema) {
		return d.deserializeMatrixArray(value, paramName, schema, explode)
	}

	if isObjectSchema(schema) {
		return d.deserializeMatrixObject(value, paramName, schema, explode)
	}

	// Primitive: ;name=value
	if strings.HasPrefix(value, paramName+"=") {
		return d.coerceValue(value[len(paramName)+1:], schema)
	}
	return d.coerceValue(value, schema)
}

// deserializeMatrixArray handles array deserialization in matrix style.
func (d *ParamDeserializer) deserializeMatrixArray(value, paramName string, schema *parser.Schema, explode bool) []any {
	if explode {
		// explode=true: ;id=3;id=4;id=5
		var values []string
		parts := strings.Split(value, ";")
		prefix := paramName + "="
		for _, part := range parts {
			if strings.HasPrefix(part, prefix) {
				values = append(values, part[len(prefix):])
			}
		}
		return d.coerceArray(values, getItemsSchema(schema))
	}

	// explode=false: ;id=3,4,5
	prefix := paramName + "="
	if strings.HasPrefix(value, prefix) {
		parts := strings.Split(value[len(prefix):], ",")
		return d.coerceArray(parts, getItemsSchema(schema))
	}
	return nil
}

// deserializeMatrixObject handles object deserialization in matrix style.
func (d *ParamDeserializer) deserializeMatrixObject(value, paramName string, schema *parser.Schema, explode bool) map[string]any {
	result := make(map[string]any)

	if explode {
		// explode=true: ;role=admin;firstName=Alex
		parts := strings.Split(value, ";")
		for _, part := range parts {
			if part == "" {
				continue
			}
			if idx := strings.Index(part, "="); idx > 0 {
				key := part[:idx]
				val := part[idx+1:]
				result[key] = d.coerceValue(val, d.getPropertySchema(schema, key))
			}
		}
	} else {
		// explode=false: ;id=role,admin,firstName,Alex
		prefix := paramName + "="
		if strings.HasPrefix(value, prefix) {
			parts := strings.Split(value[len(prefix):], ",")
			for i := 0; i+1 < len(parts); i += 2 {
				key := parts[i]
				val := parts[i+1]
				result[key] = d.coerceValue(val, d.getPropertySchema(schema, key))
			}
		}
	}

	return result
}

// deserializeForm handles the "form" style (standard query string format).
func (d *ParamDeserializer) deserializeForm(values []string, schema *parser.Schema, explode bool) any {
	if schema == nil {
		if len(values) == 1 {
			return values[0]
		}
		return values
	}

	if isArraySchema(schema) {
		if explode {
			// explode=true: multiple values (id=3&id=4&id=5)
			return d.coerceArray(values, getItemsSchema(schema))
		}
		// explode=false: comma-separated in single value (id=3,4,5)
		if len(values) == 1 {
			parts := strings.Split(values[0], ",")
			return d.coerceArray(parts, getItemsSchema(schema))
		}
		return d.coerceArray(values, getItemsSchema(schema))
	}

	if isObjectSchema(schema) {
		if explode {
			// explode=true: separate keys (role=admin&firstName=Alex)
			// This is handled at a higher level with the full query string
			if len(values) == 1 {
				return values[0]
			}
			return values
		}
		// explode=false: comma-separated key,value pairs (id=role,admin,firstName,Alex)
		if len(values) == 1 {
			parts := strings.Split(values[0], ",")
			result := make(map[string]any)
			for i := 0; i+1 < len(parts); i += 2 {
				key := parts[i]
				val := parts[i+1]
				result[key] = d.coerceValue(val, d.getPropertySchema(schema, key))
			}
			return result
		}
	}

	// Primitive type
	if len(values) == 1 {
		return d.coerceValue(values[0], schema)
	}
	return values
}

// deserializeDelimited handles space and pipe delimited styles.
func (d *ParamDeserializer) deserializeDelimited(values []string, delimiter string, schema *parser.Schema) any {
	// Join all values and split by delimiter
	joined := strings.Join(values, delimiter)
	parts := strings.Split(joined, delimiter)

	if isArraySchema(schema) {
		return d.coerceArray(parts, getItemsSchema(schema))
	}

	if len(parts) == 1 {
		return d.coerceValue(parts[0], schema)
	}
	return parts
}

// coerceValue converts a string value to the appropriate Go type based on the schema.
func (d *ParamDeserializer) coerceValue(value string, schema *parser.Schema) any {
	if schema == nil {
		return value
	}

	schemaType := getSchemaType(schema)

	switch schemaType {
	case "integer":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
		return value
	case "number":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
		return value
	case "boolean":
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
		return value
	default:
		return value
	}
}

// coerceArray converts string values to a slice of appropriately typed values.
func (d *ParamDeserializer) coerceArray(values []string, itemSchema *parser.Schema) []any {
	result := make([]any, len(values))
	for i, v := range values {
		result[i] = d.coerceValue(v, itemSchema)
	}
	return result
}

// getPropertySchema returns the schema for a property of an object schema.
func (d *ParamDeserializer) getPropertySchema(schema *parser.Schema, propName string) *parser.Schema {
	if schema == nil || schema.Properties == nil {
		return nil
	}
	return schema.Properties[propName]
}

// getSchemaType extracts the type from a schema, handling both string and []string types.
func getSchemaType(schema *parser.Schema) string {
	if schema == nil {
		return ""
	}

	switch t := schema.Type.(type) {
	case string:
		return t
	case []string:
		// For type arrays, use the first non-null type
		for _, typ := range t {
			if typ != "null" {
				return typ
			}
		}
		if len(t) > 0 {
			return t[0]
		}
	case []any:
		for _, typ := range t {
			if s, ok := typ.(string); ok && s != "null" {
				return s
			}
		}
		if len(t) > 0 {
			if s, ok := t[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

// isArraySchema checks if the schema type is "array".
func isArraySchema(schema *parser.Schema) bool {
	return getSchemaType(schema) == "array"
}

// isObjectSchema checks if the schema type is "object".
func isObjectSchema(schema *parser.Schema) bool {
	return getSchemaType(schema) == "object"
}

// getItemsSchema returns the items schema for an array schema.
// Schema.Items is `any` in OAS 3.1+ (can be *Schema or bool).
func getItemsSchema(schema *parser.Schema) *parser.Schema {
	if schema == nil {
		return nil
	}
	if items, ok := schema.Items.(*parser.Schema); ok {
		return items
	}
	return nil
}
