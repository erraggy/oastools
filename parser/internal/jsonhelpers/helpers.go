// Package jsonhelpers provides helper functions for JSON marshaling and unmarshaling
// with support for extension fields (x-* properties) in OpenAPI specifications.
//
// This package reduces boilerplate code in custom JSON marshal/unmarshal implementations
// while preserving extension fields that are not part of the OpenAPI schema.
package jsonhelpers

import (
	"encoding/json"
	"maps"
)

// MarshalWithExtras marshals a base map while merging in extension fields.
// This is used in custom MarshalJSON implementations to combine known fields
// with unknown extension fields (typically x-* properties).
//
// Example:
//
//	func (s *Schema) MarshalJSON() ([]byte, error) {
//	    base := map[string]any{
//	        "type": s.Type,
//	        "format": s.Format,
//	    }
//	    return jsonhelpers.MarshalWithExtras(base, s.Extra)
//	}
func MarshalWithExtras(base map[string]any, extras map[string]any) ([]byte, error) {
	maps.Copy(base, extras)
	return json.Marshal(base)
}

// UnmarshalExtras extracts extension fields from a JSON object after known fields
// have been removed. This is used in custom UnmarshalJSON implementations.
//
// The knownFields map should contain all known field names as keys. Any fields
// not in this map will be returned as extension fields.
//
// Example:
//
//	func (s *Schema) UnmarshalJSON(data []byte) error {
//	    var temp map[string]any
//	    if err := json.Unmarshal(data, &temp); err != nil {
//	        return err
//	    }
//
//	    knownFields := map[string]bool{
//	        "type": true,
//	        "format": true,
//	    }
//
//	    // Extract known fields...
//	    s.Type = GetString(temp, "type")
//	    s.Format = GetString(temp, "format")
//
//	    // Store remaining as extras
//	    s.Extra = UnmarshalExtras(temp, knownFields)
//	    return nil
//	}
func UnmarshalExtras(data map[string]any, knownFields map[string]bool) map[string]any {
	extras := make(map[string]any)
	for k, v := range data {
		if !knownFields[k] {
			extras[k] = v
		}
	}
	if len(extras) == 0 {
		return nil
	}
	return extras
}

// GetString safely extracts a string value from a map and removes it.
// Returns empty string if the key doesn't exist or value is not a string.
func GetString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		delete(m, key)
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetBool safely extracts a boolean value from a map and removes it.
// Returns false if the key doesn't exist or value is not a boolean.
func GetBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		delete(m, key)
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// GetInt safely extracts an integer value from a map and removes it.
// Returns 0 if the key doesn't exist or value is not a number.
// JSON numbers are unmarshaled as float64, so this handles the conversion.
func GetInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		delete(m, key)
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return 0
}

// GetFloat64 safely extracts a float64 value from a map and removes it.
// Returns 0.0 if the key doesn't exist or value is not a number.
func GetFloat64(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		delete(m, key)
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0.0
}

// GetStringSlice safely extracts a []string value from a map and removes it.
// Returns nil if the key doesn't exist or value is not a string array.
func GetStringSlice(m map[string]any, key string) []string {
	if v, ok := m[key]; ok {
		delete(m, key)
		if arr, ok := v.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// GetStringMap safely extracts a map[string]string value from a map and removes it.
// Returns nil if the key doesn't exist or value is not a string map.
func GetStringMap(m map[string]any, key string) map[string]string {
	if v, ok := m[key]; ok {
		delete(m, key)
		if obj, ok := v.(map[string]any); ok {
			result := make(map[string]string, len(obj))
			for k, val := range obj {
				if s, ok := val.(string); ok {
					result[k] = s
				}
			}
			return result
		}
	}
	return nil
}

// GetAny safely extracts a value of any type from a map and removes it.
// Returns nil if the key doesn't exist.
func GetAny(m map[string]any, key string) any {
	if v, ok := m[key]; ok {
		delete(m, key)
		return v
	}
	return nil
}

// SetIfNotEmpty sets a field in the map only if the value is not empty.
// This is useful for MarshalJSON to avoid adding empty fields to JSON output.
func SetIfNotEmpty(m map[string]any, key string, value string) {
	if value != "" {
		m[key] = value
	}
}

// SetIfNotNil sets a field in the map only if the value is not nil.
// This is useful for MarshalJSON to avoid adding nil fields to JSON output.
func SetIfNotNil(m map[string]any, key string, value any) {
	if value != nil {
		m[key] = value
	}
}

// SetIfNotZero sets a field in the map only if the value is not zero.
// This is useful for MarshalJSON to avoid adding zero-value numeric fields.
func SetIfNotZero(m map[string]any, key string, value int) {
	if value != 0 {
		m[key] = value
	}
}

// SetIfTrue sets a boolean field in the map only if the value is true.
// This is useful for MarshalJSON to avoid adding false boolean fields.
func SetIfTrue(m map[string]any, key string, value bool) {
	if value {
		m[key] = value
	}
}

// SetIfSliceNotEmpty sets a slice field in the map only if the slice has length > 0.
// This is useful for MarshalJSON to avoid adding empty slice fields.
// Note: In Go, both nil slices and empty slices should be omitted from JSON output.
func SetIfSliceNotEmpty[T any](m map[string]any, key string, value []T) {
	if len(value) > 0 {
		m[key] = value
	}
}

// SetIfMapNotEmpty sets a map field in the map only if the map has length > 0.
// This is useful for MarshalJSON to avoid adding empty map fields.
// Note: In Go, both nil maps and empty maps should be omitted from JSON output.
func SetIfMapNotEmpty[K comparable, V any](m map[string]any, key string, value map[K]V) {
	if len(value) > 0 {
		m[key] = value
	}
}

// OAS2PrimitiveFields holds OAS 2.0 primitive type fields shared across
// Parameter, Items, and Header types in Swagger 2.0 specifications.
type OAS2PrimitiveFields struct {
	Type             string
	Format           string
	Items            any
	CollectionFormat string
	Default          any
}

// SetOAS2PrimitiveFields adds OAS 2.0 primitive type fields to a map.
// This is used by Parameter, Items, and Header MarshalJSON to reduce duplication.
// Note: For Items, Type should be set separately as a required field.
func SetOAS2PrimitiveFields(m map[string]any, f OAS2PrimitiveFields) {
	SetIfNotEmpty(m, "type", f.Type)
	SetIfNotEmpty(m, "format", f.Format)
	SetIfNotNil(m, "items", f.Items)
	SetIfNotEmpty(m, "collectionFormat", f.CollectionFormat)
	SetIfNotNil(m, "default", f.Default)
}

// SchemaConstraints holds JSON Schema validation constraint fields.
// This is used for shared marshaling of constraint fields across
// Parameter, Items, and Header types.
type SchemaConstraints struct {
	Maximum          *float64
	ExclusiveMaximum bool
	Minimum          *float64
	ExclusiveMinimum bool
	MaxLength        *int
	MinLength        *int
	Pattern          string
	MaxItems         *int
	MinItems         *int
	UniqueItems      bool
	Enum             []any
	MultipleOf       *float64
}

// SetSchemaConstraints adds JSON Schema validation constraint fields to a map.
// This is used by Parameter, Items, and Header MarshalJSON to reduce duplication.
func SetSchemaConstraints(m map[string]any, c SchemaConstraints) {
	SetIfNotNil(m, "maximum", c.Maximum)
	SetIfTrue(m, "exclusiveMaximum", c.ExclusiveMaximum)
	SetIfNotNil(m, "minimum", c.Minimum)
	SetIfTrue(m, "exclusiveMinimum", c.ExclusiveMinimum)
	SetIfNotNil(m, "maxLength", c.MaxLength)
	SetIfNotNil(m, "minLength", c.MinLength)
	SetIfNotEmpty(m, "pattern", c.Pattern)
	SetIfNotNil(m, "maxItems", c.MaxItems)
	SetIfNotNil(m, "minItems", c.MinItems)
	SetIfTrue(m, "uniqueItems", c.UniqueItems)
	SetIfNotNil(m, "enum", c.Enum)
	SetIfNotNil(m, "multipleOf", c.MultipleOf)
}

// ExtractExtensions extracts specification extension fields (x-* properties)
// from JSON data. This is the common pattern used in all UnmarshalJSON methods
// to capture extension fields.
//
// Returns nil if no extensions are found or if the data cannot be parsed.
// This function never returns an error - parsing failures result in nil extensions.
//
// Example:
//
//	func (c *Contact) UnmarshalJSON(data []byte) error {
//	    type Alias Contact
//	    if err := json.Unmarshal(data, (*Alias)(c)); err != nil {
//	        return err
//	    }
//	    c.Extra = jsonhelpers.ExtractExtensions(data)
//	    return nil
//	}
func ExtractExtensions(data []byte) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}

	var extra map[string]any
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			if extra == nil {
				extra = make(map[string]any)
			}
			extra[k] = v
		}
	}
	return extra
}
