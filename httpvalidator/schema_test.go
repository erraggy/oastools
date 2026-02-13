package httpvalidator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

func TestSchemaValidator_Validate_NilSchema(t *testing.T) {
	v := NewSchemaValidator()
	errors := v.Validate("test", nil, "path")
	assert.Empty(t, errors, "expected no errors for nil schema")
}

func TestSchemaValidator_Validate_NullValue(t *testing.T) {
	v := NewSchemaValidator()

	// Non-nullable schema
	schema := &parser.Schema{Type: "string"}
	errors := v.Validate(nil, schema, "path")
	assert.Len(t, errors, 1, "expected 1 error for null value on non-nullable schema")

	// Nullable via nullable field (OAS 3.0)
	schema = &parser.Schema{Type: "string", Nullable: true}
	errors = v.Validate(nil, schema, "path")
	assert.Empty(t, errors, "expected no errors for null value on nullable schema")

	// Nullable via type array (OAS 3.1+)
	schema = &parser.Schema{Type: []any{"string", "null"}}
	errors = v.Validate(nil, schema, "path")
	assert.Empty(t, errors, "expected no errors for null value on type array with null")
}

func TestSchemaValidator_ValidateType(t *testing.T) {
	v := NewSchemaValidator()

	tests := []struct {
		name        string
		data        any
		schemaType  any
		expectError bool
	}{
		{"string matches string", "hello", "string", false},
		{"number matches number", 3.14, "number", false},
		{"integer matches integer", int64(42), "integer", false},
		{"float64 whole number matches integer", float64(42), "integer", false},
		{"float64 with decimal fails integer", float64(42.5), "integer", true},
		{"boolean matches boolean", true, "boolean", false},
		{"array matches array", []any{1, 2, 3}, "array", false},
		{"object matches object", map[string]any{"a": 1}, "object", false},
		{"string does not match number", "hello", "number", true},
		{"integer matches number (subset)", int64(42), "number", false},
		{"no type accepts anything", "hello", nil, false},
		{"type array accepts matching", "hello", []any{"string", "number"}, false},
		{"type array rejects non-matching", true, []any{"string", "number"}, true},
		{"type []string accepts matching", "hello", []string{"string", "number"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &parser.Schema{Type: tt.schemaType}
			errors := v.Validate(tt.data, schema, "path")
			hasError := len(errors) > 0
			assert.Equal(t, tt.expectError, hasError, "errors: %v", errors)
		})
	}
}

func TestSchemaValidator_ValidateString(t *testing.T) {
	v := NewSchemaValidator()

	minLen := 3
	maxLen := 10

	tests := []struct {
		name        string
		data        string
		schema      *parser.Schema
		expectError bool
	}{
		{
			name:        "valid string within length bounds",
			data:        "hello",
			schema:      &parser.Schema{Type: "string", MinLength: &minLen, MaxLength: &maxLen},
			expectError: false,
		},
		{
			name:        "string too short",
			data:        "hi",
			schema:      &parser.Schema{Type: "string", MinLength: &minLen},
			expectError: true,
		},
		{
			name:        "string too long",
			data:        "hello world!",
			schema:      &parser.Schema{Type: "string", MaxLength: &maxLen},
			expectError: true,
		},
		{
			name:        "string matches pattern",
			data:        "abc123",
			schema:      &parser.Schema{Type: "string", Pattern: "^[a-z]+[0-9]+$"},
			expectError: false,
		},
		{
			name:        "string does not match pattern",
			data:        "123abc",
			schema:      &parser.Schema{Type: "string", Pattern: "^[a-z]+[0-9]+$"},
			expectError: true,
		},
		{
			name:        "invalid regex pattern",
			data:        "test",
			schema:      &parser.Schema{Type: "string", Pattern: "[invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := v.Validate(tt.data, tt.schema, "path")
			hasError := len(errors) > 0
			assert.Equal(t, tt.expectError, hasError, "errors: %v", errors)
		})
	}
}

func TestSchemaValidator_ValidateFormat(t *testing.T) {
	v := NewSchemaValidator()

	tests := []struct {
		name        string
		data        string
		format      string
		expectError bool
	}{
		{"valid email", "test@example.com", "email", false},
		{"invalid email", "not-an-email", "email", true},
		{"valid uri http", "http://example.com", "uri", false},
		{"valid uri https", "https://example.com", "uri", false},
		{"valid uri custom scheme", "ftp://files.example.com", "uri", false},
		{"invalid uri", "not a uri", "uri", true},
		{"valid uri-reference", "https://example.com/path", "uri-reference", false},
		{"valid date", "2024-01-15", "date", false},
		{"invalid date", "01-15-2024", "date", true},
		{"valid date-time", "2024-01-15T10:30:00Z", "date-time", false},
		{"invalid date-time", "2024-01-15 10:30:00", "date-time", true},
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000", "uuid", false},
		{"invalid uuid", "not-a-uuid", "uuid", true},
		{"unknown format ignored", "anything", "unknown-format", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &parser.Schema{Type: "string", Format: tt.format}
			errors := v.Validate(tt.data, schema, "path")
			hasError := len(errors) > 0
			assert.Equal(t, tt.expectError, hasError, "errors: %v", errors)
		})
	}
}

func TestSchemaValidator_ValidateNumber(t *testing.T) {
	v := NewSchemaValidator()

	min := float64(0)
	max := float64(100)
	multipleOf := float64(5)

	tests := []struct {
		name        string
		data        float64
		schema      *parser.Schema
		expectError bool
	}{
		{
			name:        "valid number in range",
			data:        50,
			schema:      &parser.Schema{Type: "number", Minimum: &min, Maximum: &max},
			expectError: false,
		},
		{
			name:        "number below minimum",
			data:        -10,
			schema:      &parser.Schema{Type: "number", Minimum: &min},
			expectError: true,
		},
		{
			name:        "number above maximum",
			data:        150,
			schema:      &parser.Schema{Type: "number", Maximum: &max},
			expectError: true,
		},
		{
			name:        "number at minimum (inclusive)",
			data:        0,
			schema:      &parser.Schema{Type: "number", Minimum: &min},
			expectError: false,
		},
		{
			name:        "number at maximum (inclusive)",
			data:        100,
			schema:      &parser.Schema{Type: "number", Maximum: &max},
			expectError: false,
		},
		{
			name:        "exclusive minimum - value at bound fails",
			data:        0,
			schema:      &parser.Schema{Type: "number", Minimum: &min, ExclusiveMinimum: true},
			expectError: true,
		},
		{
			name:        "exclusive maximum - value at bound fails",
			data:        100,
			schema:      &parser.Schema{Type: "number", Maximum: &max, ExclusiveMaximum: true},
			expectError: true,
		},
		{
			name:        "valid multipleOf",
			data:        25,
			schema:      &parser.Schema{Type: "number", MultipleOf: &multipleOf},
			expectError: false,
		},
		{
			name:        "invalid multipleOf",
			data:        23,
			schema:      &parser.Schema{Type: "number", MultipleOf: &multipleOf},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := v.Validate(tt.data, tt.schema, "path")
			hasError := len(errors) > 0
			assert.Equal(t, tt.expectError, hasError, "errors: %v", errors)
		})
	}
}

func TestSchemaValidator_ValidateInteger(t *testing.T) {
	v := NewSchemaValidator()

	// Test with int and int64 types
	schema := &parser.Schema{Type: "integer"}

	errors := v.Validate(int(42), schema, "path")
	assert.Empty(t, errors, "expected no errors for int")

	errors = v.Validate(int64(42), schema, "path")
	assert.Empty(t, errors, "expected no errors for int64")
}

func TestSchemaValidator_ValidateArray(t *testing.T) {
	v := NewSchemaValidator()

	minItems := 2
	maxItems := 5

	tests := []struct {
		name        string
		data        []any
		schema      *parser.Schema
		expectError bool
	}{
		{
			name:        "valid array within item bounds",
			data:        []any{1, 2, 3},
			schema:      &parser.Schema{Type: "array", MinItems: &minItems, MaxItems: &maxItems},
			expectError: false,
		},
		{
			name:        "array too few items",
			data:        []any{1},
			schema:      &parser.Schema{Type: "array", MinItems: &minItems},
			expectError: true,
		},
		{
			name:        "array too many items",
			data:        []any{1, 2, 3, 4, 5, 6},
			schema:      &parser.Schema{Type: "array", MaxItems: &maxItems},
			expectError: true,
		},
		{
			name:        "unique items valid",
			data:        []any{1, 2, 3},
			schema:      &parser.Schema{Type: "array", UniqueItems: true},
			expectError: false,
		},
		{
			name:        "unique items with duplicates",
			data:        []any{1, 2, 1},
			schema:      &parser.Schema{Type: "array", UniqueItems: true},
			expectError: true,
		},
		{
			name: "items schema validation passes",
			data: []any{"a", "b", "c"},
			schema: &parser.Schema{
				Type:  "array",
				Items: &parser.Schema{Type: "string"},
			},
			expectError: false,
		},
		{
			name: "items schema validation fails",
			data: []any{"a", 123, "c"},
			schema: &parser.Schema{
				Type:  "array",
				Items: &parser.Schema{Type: "string"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := v.Validate(tt.data, tt.schema, "path")
			hasError := len(errors) > 0
			assert.Equal(t, tt.expectError, hasError, "errors: %v", errors)
		})
	}
}

func TestSchemaValidator_ValidateObject(t *testing.T) {
	v := NewSchemaValidator()

	minProps := 2
	maxProps := 5

	tests := []struct {
		name        string
		data        map[string]any
		schema      *parser.Schema
		expectError bool
	}{
		{
			name:        "valid object",
			data:        map[string]any{"name": "test", "value": 123},
			schema:      &parser.Schema{Type: "object"},
			expectError: false,
		},
		{
			name:        "required property present",
			data:        map[string]any{"name": "test"},
			schema:      &parser.Schema{Type: "object", Required: []string{"name"}},
			expectError: false,
		},
		{
			name:        "required property missing",
			data:        map[string]any{"value": 123},
			schema:      &parser.Schema{Type: "object", Required: []string{"name"}},
			expectError: true,
		},
		{
			name:        "too few properties",
			data:        map[string]any{"a": 1},
			schema:      &parser.Schema{Type: "object", MinProperties: &minProps},
			expectError: true,
		},
		{
			name:        "too many properties",
			data:        map[string]any{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5, "f": 6},
			schema:      &parser.Schema{Type: "object", MaxProperties: &maxProps},
			expectError: true,
		},
		{
			name: "property schema validation passes",
			data: map[string]any{"name": "test"},
			schema: &parser.Schema{
				Type: "object",
				Properties: map[string]*parser.Schema{
					"name": {Type: "string"},
				},
			},
			expectError: false,
		},
		{
			name: "property schema validation fails",
			data: map[string]any{"name": 123},
			schema: &parser.Schema{
				Type: "object",
				Properties: map[string]*parser.Schema{
					"name": {Type: "string"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := v.Validate(tt.data, tt.schema, "path")
			hasError := len(errors) > 0
			assert.Equal(t, tt.expectError, hasError, "errors: %v", errors)
		})
	}
}

func TestSchemaValidator_ValidateEnum(t *testing.T) {
	v := NewSchemaValidator()

	schema := &parser.Schema{
		Type: "string",
		Enum: []any{"red", "green", "blue"},
	}

	// Valid enum value
	errors := v.Validate("red", schema, "path")
	assert.Empty(t, errors, "expected no errors for valid enum value")

	// Invalid enum value
	errors = v.Validate("yellow", schema, "path")
	assert.Len(t, errors, 1, "expected 1 error for invalid enum value")
}

func TestSchemaValidator_ValidateAllOf(t *testing.T) {
	v := NewSchemaValidator()

	minLen := 3

	schema := &parser.Schema{
		AllOf: []*parser.Schema{
			{Type: "string"},
			{Type: "string", MinLength: &minLen},
		},
	}

	// Passes all schemas
	errors := v.Validate("hello", schema, "path")
	assert.Empty(t, errors, "expected no errors")

	// Fails one schema (too short)
	errors = v.Validate("hi", schema, "path")
	assert.NotEmpty(t, errors, "expected errors for failing allOf schema")
}

func TestSchemaValidator_ValidateAnyOf(t *testing.T) {
	v := NewSchemaValidator()

	schema := &parser.Schema{
		AnyOf: []*parser.Schema{
			{Type: "string"},
			{Type: "number"},
		},
	}

	// Matches first schema
	errors := v.Validate("hello", schema, "path")
	assert.Empty(t, errors, "expected no errors for matching anyOf")

	// Matches second schema
	errors = v.Validate(42.0, schema, "path")
	assert.Empty(t, errors, "expected no errors for matching anyOf")

	// Matches neither
	errors = v.Validate(true, schema, "path")
	assert.NotEmpty(t, errors, "expected error for not matching any anyOf schema")
}

func TestSchemaValidator_ValidateOneOf(t *testing.T) {
	v := NewSchemaValidator()

	min := float64(10)

	schema := &parser.Schema{
		OneOf: []*parser.Schema{
			{Type: "string"},
			{Type: "number", Minimum: &min},
		},
	}

	// Matches exactly one (string)
	errors := v.Validate("hello", schema, "path")
	assert.Empty(t, errors, "expected no errors for matching exactly one oneOf")

	// Matches exactly one (number >= 10)
	errors = v.Validate(15.0, schema, "path")
	assert.Empty(t, errors, "expected no errors for matching exactly one oneOf")

	// Matches none
	errors = v.Validate(true, schema, "path")
	assert.NotEmpty(t, errors, "expected error for matching zero oneOf schemas")

	// Test multiple matches scenario
	schemaMultiple := &parser.Schema{
		OneOf: []*parser.Schema{
			{Type: "number"},
			{Type: "integer"},
		},
	}
	// float64(42) matches both number and integer (since 42.0 is a whole number)
	errors = v.Validate(float64(42), schemaMultiple, "path")
	assert.NotEmpty(t, errors, "expected error for matching multiple oneOf schemas")
}

func TestSchemaValidator_ValidateBoolean(t *testing.T) {
	v := NewSchemaValidator()

	schema := &parser.Schema{Type: "boolean"}

	errors := v.Validate(true, schema, "path")
	assert.Empty(t, errors, "expected no errors for boolean true")

	errors = v.Validate(false, schema, "path")
	assert.Empty(t, errors, "expected no errors for boolean false")

	errors = v.Validate("true", schema, "path")
	assert.NotEmpty(t, errors, "expected error for string instead of boolean")
}

func TestGetDataType(t *testing.T) {
	tests := []struct {
		data     any
		expected string
	}{
		{nil, "null"},
		{"hello", "string"},
		{float64(3.14), "number"},
		{int(42), "integer"},
		{int32(42), "integer"},
		{int64(42), "integer"},
		{uint(42), "integer"},
		{uint32(42), "integer"},
		{uint64(42), "integer"},
		{true, "boolean"},
		{[]any{1, 2}, "array"},
		{map[string]any{"a": 1}, "object"},
		// Test reflect-based detection
		{[]string{"a", "b"}, "array"},
		{map[int]string{1: "a"}, "object"},
		{int8(42), "integer"},
		{int16(42), "integer"},
		{uint8(42), "integer"},
		{uint16(42), "integer"},
		{float32(3.14), "number"},
	}

	for _, tt := range tests {
		result := getDataType(tt.data)
		assert.Equal(t, tt.expected, result, "getDataType(%T)", tt.data)
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input    any
		expected float64
	}{
		{int(42), 42.0},
		{int32(42), 42.0},
		{int64(42), 42.0},
		{float32(3.14), float64(float32(3.14))},
		{float64(3.14), 3.14},
		{"invalid", 0}, // Non-numeric returns 0
	}

	for _, tt := range tests {
		result := toFloat64(tt.input)
		assert.Equal(t, tt.expected, result, "toFloat64(%T(%v))", tt.input, tt.input)
	}
}

func TestHasDuplicates(t *testing.T) {
	tests := []struct {
		arr      []any
		expected bool
	}{
		{[]any{1, 2, 3}, false},
		{[]any{1, 2, 1}, true},
		{[]any{"a", "b", "c"}, false},
		{[]any{"a", "b", "a"}, true},
		{[]any{}, false},
		{[]any{1}, false},
	}

	for _, tt := range tests {
		result := hasDuplicates(tt.arr)
		assert.Equal(t, tt.expected, result, "hasDuplicates(%v)", tt.arr)
	}
}

func TestPatternCache(t *testing.T) {
	v := NewSchemaValidator()

	// First call compiles and caches
	matched, err := v.matchPattern("^test$", "test")
	require.NoError(t, err, "unexpected error")
	assert.True(t, matched, "expected match")

	// Second call uses cache
	matched, err = v.matchPattern("^test$", "test")
	require.NoError(t, err, "unexpected error")
	assert.True(t, matched, "expected match from cache")
}

func TestIsExclusiveMinimum(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected bool
	}{
		{"nil", &parser.Schema{}, false},
		{"bool true", &parser.Schema{ExclusiveMinimum: true}, true},
		{"bool false", &parser.Schema{ExclusiveMinimum: false}, false},
		{"number (not bool)", &parser.Schema{ExclusiveMinimum: float64(5)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isExclusiveMinimum(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsExclusiveMaximum(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected bool
	}{
		{"nil", &parser.Schema{}, false},
		{"bool true", &parser.Schema{ExclusiveMaximum: true}, true},
		{"bool false", &parser.Schema{ExclusiveMaximum: false}, false},
		{"number (not bool)", &parser.Schema{ExclusiveMaximum: float64(100)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isExclusiveMaximum(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSchemaTypes(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected []string
	}{
		{"nil type", &parser.Schema{}, nil},
		{"string type", &parser.Schema{Type: "string"}, []string{"string"}},
		{"[]any type", &parser.Schema{Type: []any{"string", "null"}}, []string{"string", "null"}},
		{"[]string type", &parser.Schema{Type: []string{"string", "number"}}, []string{"string", "number"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSchemaTypes(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTypeMatches(t *testing.T) {
	tests := []struct {
		dataType   string
		schemaType string
		expected   bool
	}{
		{"string", "string", true},
		{"number", "number", true},
		{"integer", "integer", true},
		{"boolean", "boolean", true},
		{"array", "array", true},
		{"object", "object", true},
		{"integer", "number", true}, // integer is subset of number
		{"number", "integer", true}, // Will be validated separately for fractional part
		{"string", "number", false},
		{"boolean", "string", false},
	}

	for _, tt := range tests {
		result := typeMatches(tt.dataType, tt.schemaType)
		assert.Equal(t, tt.expected, result, "typeMatches(%q, %q)", tt.dataType, tt.schemaType)
	}
}
