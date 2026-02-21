package httpvalidator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/erraggy/oastools/internal/stringutil"
	"github.com/erraggy/oastools/parser"
)

// SchemaValidator validates data values against OpenAPI schemas.
// It implements a minimal subset of JSON Schema validation suitable for
// validating HTTP request and response bodies.
type SchemaValidator struct {
	// patternCache caches compiled regex patterns (sync.Map[string, *regexp.Regexp])
	patternCache sync.Map

	// patternCount tracks the approximate number of cached patterns for size capping
	patternCount atomic.Int32

	// redactValues controls whether actual values appear in error messages.
	// When true, error messages describe the violation without exposing the value.
	// This should be enabled when validating potentially sensitive data like headers.
	redactValues bool
}

// NewSchemaValidator creates a new SchemaValidator.
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{}
}

// NewRedactingSchemaValidator creates a SchemaValidator that omits actual values
// from error messages. Use this when validating potentially sensitive data like
// HTTP headers that may contain credentials.
func NewRedactingSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		redactValues: true,
	}
}

// Validate validates data against an OpenAPI schema.
// Returns a slice of validation errors (empty if valid).
func (v *SchemaValidator) Validate(data any, schema *parser.Schema, path string) []ValidationError {
	if schema == nil {
		return nil
	}

	var errors []ValidationError

	// Handle nullable
	if data == nil {
		if v.isNullable(schema) {
			return nil
		}
		errors = append(errors, ValidationError{
			Path:     path,
			Message:  "value cannot be null",
			Severity: SeverityError,
		})
		return errors
	}

	// Validate type
	typeErrors := v.validateType(data, schema, path)
	errors = append(errors, typeErrors...)

	// If type validation failed, skip constraint validation
	if len(typeErrors) > 0 {
		return errors
	}

	// Validate constraints based on data type
	switch d := data.(type) {
	case string:
		errors = append(errors, v.validateString(d, schema, path)...)
	case float64:
		errors = append(errors, v.validateNumber(d, schema, path)...)
	case int, int64:
		num := toFloat64(d)
		errors = append(errors, v.validateNumber(num, schema, path)...)
	case bool:
		// No additional constraints for boolean
	case []any:
		errors = append(errors, v.validateArray(d, schema, path)...)
	case map[string]any:
		errors = append(errors, v.validateObject(d, schema, path)...)
	}

	// Validate enum
	if len(schema.Enum) > 0 {
		errors = append(errors, v.validateEnum(data, schema, path)...)
	}

	// Validate composition (allOf, anyOf, oneOf)
	errors = append(errors, v.validateComposition(data, schema, path)...)

	return errors
}

// isNullable checks if a schema allows null values.
func (v *SchemaValidator) isNullable(schema *parser.Schema) bool {
	// OAS 3.0 style: nullable: true
	if schema.Nullable {
		return true
	}

	// OAS 3.1+ style: type includes "null"
	types := getSchemaTypes(schema)
	for _, t := range types {
		if t == "null" {
			return true
		}
	}

	return false
}

// validateType validates that the data matches the schema type(s).
func (v *SchemaValidator) validateType(data any, schema *parser.Schema, path string) []ValidationError {
	types := getSchemaTypes(schema)
	if len(types) == 0 {
		// No type specified, any type is valid
		return nil
	}

	dataType := getDataType(data)

	for _, schemaType := range types {
		if typeMatches(dataType, schemaType) {
			// Additional check: if schema expects integer but data is a float64,
			// verify it has no fractional part
			if schemaType == "integer" && dataType == "number" {
				if f, ok := data.(float64); ok {
					if f != float64(int64(f)) {
						msg := "value must be an integer"
						if !v.redactValues {
							msg = fmt.Sprintf("value must be an integer, got %v", f)
						}
						return []ValidationError{{
							Path:     path,
							Message:  msg,
							Severity: SeverityError,
						}}
					}
				}
			}
			return nil
		}
	}

	return []ValidationError{{
		Path:     path,
		Message:  fmt.Sprintf("expected type %s but got %s", strings.Join(types, " or "), dataType),
		Severity: SeverityError,
	}}
}

// validateString validates string-specific constraints.
func (v *SchemaValidator) validateString(s string, schema *parser.Schema, path string) []ValidationError {
	var errors []ValidationError

	// minLength
	if schema.MinLength != nil && len(s) < *schema.MinLength {
		errors = append(errors, ValidationError{
			Path:     path,
			Message:  fmt.Sprintf("string length %d is less than minimum %d", len(s), *schema.MinLength),
			Severity: SeverityError,
		})
	}

	// maxLength
	if schema.MaxLength != nil && len(s) > *schema.MaxLength {
		errors = append(errors, ValidationError{
			Path:     path,
			Message:  fmt.Sprintf("string length %d exceeds maximum %d", len(s), *schema.MaxLength),
			Severity: SeverityError,
		})
	}

	// pattern
	if schema.Pattern != "" {
		matched, err := v.matchPattern(schema.Pattern, s)
		if err != nil {
			errors = append(errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("invalid pattern %q: %v", schema.Pattern, err),
				Severity: SeverityError,
			})
		} else if !matched {
			errors = append(errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("string does not match pattern %q", schema.Pattern),
				Severity: SeverityError,
			})
		}
	}

	// format (basic validation for common formats)
	if schema.Format != "" {
		errors = append(errors, v.validateFormat(s, schema.Format, path)...)
	}

	return errors
}

// validateNumber validates numeric constraints.
func (v *SchemaValidator) validateNumber(n float64, schema *parser.Schema, path string) []ValidationError {
	var errors []ValidationError

	// minimum
	if schema.Minimum != nil {
		excl := isExclusiveMinimum(schema)
		if excl && n <= *schema.Minimum {
			errors = append(errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("value %v must be greater than %v", n, *schema.Minimum),
				Severity: SeverityError,
			})
		} else if !excl && n < *schema.Minimum {
			errors = append(errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("value %v is less than minimum %v", n, *schema.Minimum),
				Severity: SeverityError,
			})
		}
	}

	// maximum
	if schema.Maximum != nil {
		excl := isExclusiveMaximum(schema)
		if excl && n >= *schema.Maximum {
			errors = append(errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("value %v must be less than %v", n, *schema.Maximum),
				Severity: SeverityError,
			})
		} else if !excl && n > *schema.Maximum {
			errors = append(errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("value %v exceeds maximum %v", n, *schema.Maximum),
				Severity: SeverityError,
			})
		}
	}

	// multipleOf
	if schema.MultipleOf != nil && *schema.MultipleOf != 0 {
		// Use modulo with tolerance for floating point precision
		remainder := n / *schema.MultipleOf
		if remainder != float64(int64(remainder)) {
			errors = append(errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("value %v is not a multiple of %v", n, *schema.MultipleOf),
				Severity: SeverityError,
			})
		}
	}

	return errors
}

// validateArray validates array-specific constraints.
func (v *SchemaValidator) validateArray(arr []any, schema *parser.Schema, path string) []ValidationError {
	var errors []ValidationError

	// minItems
	if schema.MinItems != nil && len(arr) < *schema.MinItems {
		errors = append(errors, ValidationError{
			Path:     path,
			Message:  fmt.Sprintf("array has %d items, minimum is %d", len(arr), *schema.MinItems),
			Severity: SeverityError,
		})
	}

	// maxItems
	if schema.MaxItems != nil && len(arr) > *schema.MaxItems {
		errors = append(errors, ValidationError{
			Path:     path,
			Message:  fmt.Sprintf("array has %d items, maximum is %d", len(arr), *schema.MaxItems),
			Severity: SeverityError,
		})
	}

	// uniqueItems
	if schema.UniqueItems && hasDuplicates(arr) {
		errors = append(errors, ValidationError{
			Path:     path,
			Message:  "array items must be unique",
			Severity: SeverityError,
		})
	}

	// items schema
	if itemSchema := getItemsSchema(schema); itemSchema != nil {
		for i, item := range arr {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			errors = append(errors, v.Validate(item, itemSchema, itemPath)...)
		}
	}

	return errors
}

// validateObject validates object-specific constraints.
func (v *SchemaValidator) validateObject(obj map[string]any, schema *parser.Schema, path string) []ValidationError {
	var errors []ValidationError

	// required properties
	for _, req := range schema.Required {
		if _, exists := obj[req]; !exists {
			errors = append(errors, ValidationError{
				Path:     path + "." + req,
				Message:  fmt.Sprintf("required property %q is missing", req),
				Severity: SeverityError,
			})
		}
	}

	// minProperties
	if schema.MinProperties != nil && len(obj) < *schema.MinProperties {
		errors = append(errors, ValidationError{
			Path:     path,
			Message:  fmt.Sprintf("object has %d properties, minimum is %d", len(obj), *schema.MinProperties),
			Severity: SeverityError,
		})
	}

	// maxProperties
	if schema.MaxProperties != nil && len(obj) > *schema.MaxProperties {
		errors = append(errors, ValidationError{
			Path:     path,
			Message:  fmt.Sprintf("object has %d properties, maximum is %d", len(obj), *schema.MaxProperties),
			Severity: SeverityError,
		})
	}

	// property schemas
	for name, value := range obj {
		if propSchema, ok := schema.Properties[name]; ok {
			propPath := path + "." + name
			errors = append(errors, v.Validate(value, propSchema, propPath)...)
		}
	}

	// additionalProperties enforcement
	if allowed, ok := schema.AdditionalProperties.(bool); ok && !allowed {
		for name := range obj {
			if _, defined := schema.Properties[name]; !defined {
				errors = append(errors, ValidationError{
					Path:     path + "." + name,
					Message:  fmt.Sprintf("additional property %q is not allowed", name),
					Severity: SeverityError,
				})
			}
		}
	}

	return errors
}

// validateEnum validates that the value is one of the allowed enum values.
func (v *SchemaValidator) validateEnum(data any, schema *parser.Schema, path string) []ValidationError {
	for _, allowed := range schema.Enum {
		if reflect.DeepEqual(data, allowed) {
			return nil
		}
	}

	msg := "value is not one of the allowed values"
	if !v.redactValues {
		msg = fmt.Sprintf("value %v is not one of the allowed values", data)
	}

	return []ValidationError{{
		Path:     path,
		Message:  msg,
		Severity: SeverityError,
	}}
}

// validateComposition validates allOf, anyOf, oneOf.
func (v *SchemaValidator) validateComposition(data any, schema *parser.Schema, path string) []ValidationError {
	var errors []ValidationError

	// allOf - all schemas must match
	if len(schema.AllOf) > 0 {
		for i, subSchema := range schema.AllOf {
			subErrors := v.Validate(data, subSchema, path)
			if len(subErrors) > 0 {
				errors = append(errors, ValidationError{
					Path:     path,
					Message:  fmt.Sprintf("allOf[%d] validation failed", i),
					Severity: SeverityError,
				})
				errors = append(errors, subErrors...)
			}
		}
	}

	// anyOf - at least one schema must match
	if len(schema.AnyOf) > 0 {
		matched := false
		for _, subSchema := range schema.AnyOf {
			if len(v.Validate(data, subSchema, path)) == 0 {
				matched = true
				break
			}
		}
		if !matched {
			errors = append(errors, ValidationError{
				Path:     path,
				Message:  "value does not match any of the anyOf schemas",
				Severity: SeverityError,
			})
		}
	}

	// oneOf - exactly one schema must match
	if len(schema.OneOf) > 0 {
		matchCount := 0
		for _, subSchema := range schema.OneOf {
			if len(v.Validate(data, subSchema, path)) == 0 {
				matchCount++
			}
		}
		if matchCount == 0 {
			errors = append(errors, ValidationError{
				Path:     path,
				Message:  "value does not match any of the oneOf schemas",
				Severity: SeverityError,
			})
		} else if matchCount > 1 {
			errors = append(errors, ValidationError{
				Path:     path,
				Message:  fmt.Sprintf("value matches %d oneOf schemas, expected exactly 1", matchCount),
				Severity: SeverityError,
			})
		}
	}

	return errors
}

// validateFormat validates common string formats.
func (v *SchemaValidator) validateFormat(s, format, path string) []ValidationError {
	switch format {
	case "email":
		if !isValidEmail(s) {
			msg := "value is not a valid email address"
			if !v.redactValues {
				msg = fmt.Sprintf("%q is not a valid email address", s)
			}
			return []ValidationError{{
				Path:     path,
				Message:  msg,
				Severity: SeverityWarning, // Format validation is typically a warning
			}}
		}
	case "uri", "uri-reference":
		if !isValidURI(s) {
			msg := "value is not a valid URI"
			if !v.redactValues {
				msg = fmt.Sprintf("%q is not a valid URI", s)
			}
			return []ValidationError{{
				Path:     path,
				Message:  msg,
				Severity: SeverityWarning,
			}}
		}
	case "date":
		if !isValidDate(s) {
			msg := "value is not a valid date (expected YYYY-MM-DD)"
			if !v.redactValues {
				msg = fmt.Sprintf("%q is not a valid date (expected YYYY-MM-DD)", s)
			}
			return []ValidationError{{
				Path:     path,
				Message:  msg,
				Severity: SeverityWarning,
			}}
		}
	case "date-time":
		if !isValidDateTime(s) {
			msg := "value is not a valid date-time (expected RFC 3339)"
			if !v.redactValues {
				msg = fmt.Sprintf("%q is not a valid date-time (expected RFC 3339)", s)
			}
			return []ValidationError{{
				Path:     path,
				Message:  msg,
				Severity: SeverityWarning,
			}}
		}
	case "uuid":
		if !isValidUUID(s) {
			msg := "value is not a valid UUID"
			if !v.redactValues {
				msg = fmt.Sprintf("%q is not a valid UUID", s)
			}
			return []ValidationError{{
				Path:     path,
				Message:  msg,
				Severity: SeverityWarning,
			}}
		}
	}
	// Unknown formats are ignored (as per JSON Schema spec)
	return nil
}

// maxPatternCacheSize is the upper bound on cached compiled regex patterns.
// When exceeded, the cache is cleared to prevent unbounded memory growth
// from specs with many unique patterns.
const maxPatternCacheSize = 1000

// matchPattern compiles and matches a regex pattern.
func (v *SchemaValidator) matchPattern(pattern, s string) (bool, error) {
	if cached, ok := v.patternCache.Load(pattern); ok {
		return cached.(*regexp.Regexp).MatchString(s), nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	// Size cap: if cache exceeds limit, clear and start fresh.
	// This prevents unbounded growth from specs with many unique patterns.
	// NOTE: The count check and clear are not atomic — under high concurrency,
	// multiple goroutines may clear simultaneously. This is acceptable because
	// the cache is a performance optimization; worst case is extra recompilation.
	if v.patternCount.Add(1) > maxPatternCacheSize {
		v.patternCache.Range(func(key, _ any) bool {
			v.patternCache.Delete(key)
			return true
		})
		v.patternCount.Store(1)
	}
	v.patternCache.Store(pattern, re)
	return re.MatchString(s), nil
}

// Helper functions

// getSchemaTypes returns the type(s) defined in a schema.
func getSchemaTypes(schema *parser.Schema) []string {
	if schema.Type == nil {
		return nil
	}

	switch t := schema.Type.(type) {
	case string:
		return []string{t}
	case []any:
		types := make([]string, 0, len(t))
		for _, v := range t {
			if s, ok := v.(string); ok {
				types = append(types, s)
			}
		}
		return types
	case []string:
		return t
	}
	return nil
}

// getDataType returns the JSON Schema type of a Go value.
func getDataType(data any) string {
	if data == nil {
		return "null"
	}

	switch data.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case int, int32, int64, uint, uint32, uint64:
		return "integer"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		rv := reflect.ValueOf(data)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			return "array"
		case reflect.Map:
			return "object"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return "integer"
		case reflect.Float32, reflect.Float64:
			return "number"
		case reflect.String:
			return "string"
		case reflect.Bool:
			return "boolean"
		}
		return "unknown"
	}
}

// typeMatches checks if a data type matches a schema type.
func typeMatches(dataType, schemaType string) bool {
	if dataType == schemaType {
		return true
	}
	// "integer" is a subset of "number"
	if schemaType == "number" && dataType == "integer" {
		return true
	}
	// JSON numbers that are whole numbers can match "integer"
	// This is a common case since JSON only has one number type
	if schemaType == "integer" && dataType == "number" {
		return true // Will be validated for fractional part separately
	}
	return false
}

// toFloat64 converts numeric types to float64.
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	}
	return 0
}

// isExclusiveMinimum checks if minimum is exclusive.
func isExclusiveMinimum(schema *parser.Schema) bool {
	if schema.ExclusiveMinimum == nil {
		return false
	}
	// OAS 3.0 uses bool, OAS 3.1+ uses number
	if b, ok := schema.ExclusiveMinimum.(bool); ok {
		return b
	}
	// If it's a number, the exclusiveMinimum field itself is the bound
	return false
}

// isExclusiveMaximum checks if maximum is exclusive.
func isExclusiveMaximum(schema *parser.Schema) bool {
	if schema.ExclusiveMaximum == nil {
		return false
	}
	if b, ok := schema.ExclusiveMaximum.(bool); ok {
		return b
	}
	return false
}

// hasDuplicates checks if an array has duplicate values.
func hasDuplicates(arr []any) bool {
	seen := make(map[string]bool)
	for _, item := range arr {
		key := fmt.Sprintf("%T:%v", item, item)
		if seen[key] {
			return true
		}
		seen[key] = true
	}
	return false
}

// Format validation helpers

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var dateTimeRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)

func isValidEmail(s string) bool {
	return stringutil.IsValidEmail(s)
}

func isValidURI(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.Contains(s, "://")
}

func isValidDate(s string) bool {
	return dateRegex.MatchString(s)
}

func isValidDateTime(s string) bool {
	return dateTimeRegex.MatchString(s)
}

func isValidUUID(s string) bool {
	return uuidRegex.MatchString(s)
}
