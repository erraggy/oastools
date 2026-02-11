package httpvalidator

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/erraggy/oastools/internal/testutil"
	"github.com/erraggy/oastools/parser"
)

func TestDeserializePathParam_Simple(t *testing.T) {
	d := NewParamDeserializer()

	tests := []struct {
		name     string
		value    string
		param    *parser.Parameter
		expected any
	}{
		{
			name:  "simple primitive string",
			value: "hello",
			param: &parser.Parameter{
				Name:   "name",
				In:     "path",
				Schema: &parser.Schema{Type: "string"},
			},
			expected: "hello",
		},
		{
			name:  "simple primitive integer",
			value: "42",
			param: &parser.Parameter{
				Name:   "id",
				In:     "path",
				Schema: &parser.Schema{Type: "integer"},
			},
			expected: int64(42),
		},
		{
			name:  "simple primitive number",
			value: "3.14",
			param: &parser.Parameter{
				Name:   "value",
				In:     "path",
				Schema: &parser.Schema{Type: "number"},
			},
			expected: 3.14,
		},
		{
			name:  "simple primitive boolean",
			value: "true",
			param: &parser.Parameter{
				Name:   "flag",
				In:     "path",
				Schema: &parser.Schema{Type: "boolean"},
			},
			expected: true,
		},
		{
			name:  "simple array",
			value: "a,b,c",
			param: &parser.Parameter{
				Name:   "items",
				In:     "path",
				Schema: &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
			},
			expected: []any{"a", "b", "c"},
		},
		{
			name:  "simple array integers",
			value: "1,2,3",
			param: &parser.Parameter{
				Name:   "ids",
				In:     "path",
				Schema: &parser.Schema{Type: "array", Items: &parser.Schema{Type: "integer"}},
			},
			expected: []any{int64(1), int64(2), int64(3)},
		},
		{
			name:  "simple object no explode",
			value: "role,admin,name,alex",
			param: &parser.Parameter{
				Name:    "obj",
				In:      "path",
				Explode: testutil.Ptr(false),
				Schema: &parser.Schema{
					Type: "object",
					Properties: map[string]*parser.Schema{
						"role": {Type: "string"},
						"name": {Type: "string"},
					},
				},
			},
			expected: map[string]any{"role": "admin", "name": "alex"},
		},
		{
			name:  "simple object with explode",
			value: "role=admin,name=alex",
			param: &parser.Parameter{
				Name:    "obj",
				In:      "path",
				Explode: testutil.Ptr(true),
				Schema: &parser.Schema{
					Type: "object",
					Properties: map[string]*parser.Schema{
						"role": {Type: "string"},
						"name": {Type: "string"},
					},
				},
			},
			expected: map[string]any{"role": "admin", "name": "alex"},
		},
		{
			name:  "no schema returns raw value",
			value: "test",
			param: &parser.Parameter{
				Name: "raw",
				In:   "path",
			},
			expected: "test",
		},
		{
			name:  "unknown style returns raw value",
			value: "test",
			param: &parser.Parameter{
				Name:   "raw",
				In:     "path",
				Style:  "unknown",
				Schema: &parser.Schema{Type: "string"},
			},
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.DeserializePathParam(tt.value, tt.param)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestDeserializePathParam_Label(t *testing.T) {
	d := NewParamDeserializer()

	tests := []struct {
		name     string
		value    string
		param    *parser.Parameter
		expected any
	}{
		{
			name:  "label primitive",
			value: ".hello",
			param: &parser.Parameter{
				Name:   "name",
				In:     "path",
				Style:  "label",
				Schema: &parser.Schema{Type: "string"},
			},
			expected: "hello",
		},
		{
			name:  "label array no explode",
			value: ".a,b,c",
			param: &parser.Parameter{
				Name:    "items",
				In:      "path",
				Style:   "label",
				Explode: testutil.Ptr(false),
				Schema:  &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
			},
			expected: []any{"a", "b", "c"},
		},
		{
			name:  "label array with explode",
			value: ".a.b.c",
			param: &parser.Parameter{
				Name:    "items",
				In:      "path",
				Style:   "label",
				Explode: testutil.Ptr(true),
				Schema:  &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
			},
			expected: []any{"a", "b", "c"},
		},
		{
			name:  "label object no explode",
			value: ".role,admin,name,alex",
			param: &parser.Parameter{
				Name:    "obj",
				In:      "path",
				Style:   "label",
				Explode: testutil.Ptr(false),
				Schema: &parser.Schema{
					Type: "object",
					Properties: map[string]*parser.Schema{
						"role": {Type: "string"},
						"name": {Type: "string"},
					},
				},
			},
			expected: map[string]any{"role": "admin", "name": "alex"},
		},
		{
			name:  "label object with explode",
			value: ".role=admin.name=alex",
			param: &parser.Parameter{
				Name:    "obj",
				In:      "path",
				Style:   "label",
				Explode: testutil.Ptr(true),
				Schema: &parser.Schema{
					Type: "object",
					Properties: map[string]*parser.Schema{
						"role": {Type: "string"},
						"name": {Type: "string"},
					},
				},
			},
			expected: map[string]any{"role": "admin", "name": "alex"},
		},
		{
			name:  "label without leading dot returns raw",
			value: "hello",
			param: &parser.Parameter{
				Name:   "name",
				In:     "path",
				Style:  "label",
				Schema: &parser.Schema{Type: "string"},
			},
			expected: "hello",
		},
		{
			name:  "label no schema",
			value: ".hello",
			param: &parser.Parameter{
				Name:  "name",
				In:    "path",
				Style: "label",
			},
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.DeserializePathParam(tt.value, tt.param)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestDeserializePathParam_Matrix(t *testing.T) {
	d := NewParamDeserializer()

	tests := []struct {
		name     string
		value    string
		param    *parser.Parameter
		expected any
	}{
		{
			name:  "matrix primitive",
			value: ";id=5",
			param: &parser.Parameter{
				Name:   "id",
				In:     "path",
				Style:  "matrix",
				Schema: &parser.Schema{Type: "integer"},
			},
			expected: int64(5),
		},
		{
			name:  "matrix array no explode",
			value: ";id=3,4,5",
			param: &parser.Parameter{
				Name:    "id",
				In:      "path",
				Style:   "matrix",
				Explode: testutil.Ptr(false),
				Schema:  &parser.Schema{Type: "array", Items: &parser.Schema{Type: "integer"}},
			},
			expected: []any{int64(3), int64(4), int64(5)},
		},
		{
			name:  "matrix array with explode",
			value: ";id=3;id=4;id=5",
			param: &parser.Parameter{
				Name:    "id",
				In:      "path",
				Style:   "matrix",
				Explode: testutil.Ptr(true),
				Schema:  &parser.Schema{Type: "array", Items: &parser.Schema{Type: "integer"}},
			},
			expected: []any{int64(3), int64(4), int64(5)},
		},
		{
			name:  "matrix object no explode",
			value: ";id=role,admin,name,alex",
			param: &parser.Parameter{
				Name:    "id",
				In:      "path",
				Style:   "matrix",
				Explode: testutil.Ptr(false),
				Schema: &parser.Schema{
					Type: "object",
					Properties: map[string]*parser.Schema{
						"role": {Type: "string"},
						"name": {Type: "string"},
					},
				},
			},
			expected: map[string]any{"role": "admin", "name": "alex"},
		},
		{
			name:  "matrix object with explode",
			value: ";role=admin;name=alex",
			param: &parser.Parameter{
				Name:    "id",
				In:      "path",
				Style:   "matrix",
				Explode: testutil.Ptr(true),
				Schema: &parser.Schema{
					Type: "object",
					Properties: map[string]*parser.Schema{
						"role": {Type: "string"},
						"name": {Type: "string"},
					},
				},
			},
			expected: map[string]any{"role": "admin", "name": "alex"},
		},
		{
			name:  "matrix without leading semicolon returns raw",
			value: "id=5",
			param: &parser.Parameter{
				Name:   "id",
				In:     "path",
				Style:  "matrix",
				Schema: &parser.Schema{Type: "integer"},
			},
			expected: "id=5",
		},
		{
			name:  "matrix no schema extracts value",
			value: ";id=hello",
			param: &parser.Parameter{
				Name:  "id",
				In:    "path",
				Style: "matrix",
			},
			expected: "hello",
		},
		{
			name:  "matrix no schema no match returns raw",
			value: ";other=hello",
			param: &parser.Parameter{
				Name:  "id",
				In:    "path",
				Style: "matrix",
			},
			expected: "other=hello",
		},
		{
			name:  "matrix primitive with schema no match",
			value: ";other=hello",
			param: &parser.Parameter{
				Name:   "id",
				In:     "path",
				Style:  "matrix",
				Schema: &parser.Schema{Type: "string"},
			},
			expected: "other=hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.DeserializePathParam(tt.value, tt.param)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestDeserializeQueryParam_Form(t *testing.T) {
	d := NewParamDeserializer()

	tests := []struct {
		name     string
		values   []string
		param    *parser.Parameter
		expected any
	}{
		{
			name:   "form primitive single value",
			values: []string{"hello"},
			param: &parser.Parameter{
				Name:   "name",
				In:     "query",
				Schema: &parser.Schema{Type: "string"},
			},
			expected: "hello",
		},
		{
			name:   "form array with explode (default)",
			values: []string{"a", "b", "c"},
			param: &parser.Parameter{
				Name:   "items",
				In:     "query",
				Schema: &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
			},
			expected: []any{"a", "b", "c"},
		},
		{
			name:   "form array no explode comma-separated",
			values: []string{"a,b,c"},
			param: &parser.Parameter{
				Name:    "items",
				In:      "query",
				Explode: testutil.Ptr(false),
				Schema:  &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
			},
			expected: []any{"a", "b", "c"},
		},
		{
			name:   "form object no explode comma-separated",
			values: []string{"role,admin,name,alex"},
			param: &parser.Parameter{
				Name:    "obj",
				In:      "query",
				Explode: testutil.Ptr(false),
				Schema: &parser.Schema{
					Type: "object",
					Properties: map[string]*parser.Schema{
						"role": {Type: "string"},
						"name": {Type: "string"},
					},
				},
			},
			expected: map[string]any{"role": "admin", "name": "alex"},
		},
		{
			name:   "form object with explode returns raw (handled at higher level)",
			values: []string{"admin"},
			param: &parser.Parameter{
				Name:    "role",
				In:      "query",
				Explode: testutil.Ptr(true),
				Schema:  &parser.Schema{Type: "object"},
			},
			expected: "admin",
		},
		{
			name:   "no schema single value",
			values: []string{"hello"},
			param: &parser.Parameter{
				Name: "raw",
				In:   "query",
			},
			expected: "hello",
		},
		{
			name:   "no schema multiple values",
			values: []string{"a", "b"},
			param: &parser.Parameter{
				Name: "raw",
				In:   "query",
			},
			expected: []string{"a", "b"},
		},
		{
			name:   "primitive multiple values returns as-is",
			values: []string{"hello", "world"},
			param: &parser.Parameter{
				Name:   "name",
				In:     "query",
				Schema: &parser.Schema{Type: "string"},
			},
			expected: []string{"hello", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.DeserializeQueryParam(tt.values, tt.param)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestDeserializeQueryParam_SpaceDelimited(t *testing.T) {
	d := NewParamDeserializer()

	result := d.DeserializeQueryParam(
		[]string{"a b c"},
		&parser.Parameter{
			Name:   "items",
			In:     "query",
			Style:  "spaceDelimited",
			Schema: &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
		},
	)

	expected := []any{"a", "b", "c"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestDeserializeQueryParam_PipeDelimited(t *testing.T) {
	d := NewParamDeserializer()

	result := d.DeserializeQueryParam(
		[]string{"a|b|c"},
		&parser.Parameter{
			Name:   "items",
			In:     "query",
			Style:  "pipeDelimited",
			Schema: &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
		},
	)

	expected := []any{"a", "b", "c"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestDeserializeQueryParam_DeepObject(t *testing.T) {
	d := NewParamDeserializer()

	// DeepObject is handled at a higher level
	result := d.DeserializeQueryParam(
		[]string{"active"},
		&parser.Parameter{
			Name:   "filter",
			In:     "query",
			Style:  "deepObject",
			Schema: &parser.Schema{Type: "object"},
		},
	)

	if result != "active" {
		t.Errorf("got %v, want 'active'", result)
	}

	// Multiple values
	result = d.DeserializeQueryParam(
		[]string{"a", "b"},
		&parser.Parameter{
			Name:   "filter",
			In:     "query",
			Style:  "deepObject",
			Schema: &parser.Schema{Type: "object"},
		},
	)

	expected := []string{"a", "b"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestDeserializeQueryParam_UnknownStyle(t *testing.T) {
	d := NewParamDeserializer()

	// Single value
	result := d.DeserializeQueryParam(
		[]string{"hello"},
		&parser.Parameter{
			Name:   "name",
			In:     "query",
			Style:  "unknownStyle",
			Schema: &parser.Schema{Type: "string"},
		},
	)
	if result != "hello" {
		t.Errorf("got %v, want 'hello'", result)
	}

	// Multiple values
	result = d.DeserializeQueryParam(
		[]string{"a", "b"},
		&parser.Parameter{
			Name:   "name",
			In:     "query",
			Style:  "unknownStyle",
			Schema: &parser.Schema{Type: "string"},
		},
	)
	expected := []string{"a", "b"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestDeserializeQueryParamsDeepObject(t *testing.T) {
	d := NewParamDeserializer()

	queryValues := url.Values{
		"filter[status]": []string{"active"},
		"filter[type]":   []string{"user"},
		"filter[count]":  []string{"10"},
		"other":          []string{"ignored"},
	}

	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"status": {Type: "string"},
			"type":   {Type: "string"},
			"count":  {Type: "integer"},
		},
	}

	result := d.DeserializeQueryParamsDeepObject(queryValues, "filter", schema)

	if result["status"] != "active" {
		t.Errorf("status = %v, want 'active'", result["status"])
	}
	if result["type"] != "user" {
		t.Errorf("type = %v, want 'user'", result["type"])
	}
	if result["count"] != int64(10) {
		t.Errorf("count = %v (%T), want int64(10)", result["count"], result["count"])
	}
	if _, ok := result["other"]; ok {
		t.Error("'other' should not be in result")
	}
}

func TestDeserializeQueryParamsDeepObject_MultipleValues(t *testing.T) {
	d := NewParamDeserializer()

	queryValues := url.Values{
		"filter[tags]": []string{"a", "b", "c"},
	}

	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"tags": {Type: "array"},
		},
	}

	result := d.DeserializeQueryParamsDeepObject(queryValues, "filter", schema)

	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(result["tags"], expected) {
		t.Errorf("tags = %v, want %v", result["tags"], expected)
	}
}

func TestDeserializeQueryParamsDeepObject_InvalidFormat(t *testing.T) {
	d := NewParamDeserializer()

	// Missing closing bracket
	queryValues := url.Values{
		"filter[status": []string{"active"},
	}

	schema := &parser.Schema{Type: "object"}
	result := d.DeserializeQueryParamsDeepObject(queryValues, "filter", schema)

	if len(result) != 0 {
		t.Errorf("expected empty result for invalid format, got %v", result)
	}
}

func TestDeserializeHeaderParam(t *testing.T) {
	d := NewParamDeserializer()

	tests := []struct {
		name     string
		value    string
		param    *parser.Parameter
		expected any
	}{
		{
			name:  "header primitive",
			value: "hello",
			param: &parser.Parameter{
				Name:   "X-Custom",
				In:     "header",
				Schema: &parser.Schema{Type: "string"},
			},
			expected: "hello",
		},
		{
			name:  "header integer",
			value: "42",
			param: &parser.Parameter{
				Name:   "X-Count",
				In:     "header",
				Schema: &parser.Schema{Type: "integer"},
			},
			expected: int64(42),
		},
		{
			name:  "header array",
			value: "a,b,c",
			param: &parser.Parameter{
				Name:   "X-Items",
				In:     "header",
				Schema: &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
			},
			expected: []any{"a", "b", "c"},
		},
		{
			name:  "header with explode",
			value: "key=value,key2=value2",
			param: &parser.Parameter{
				Name:    "X-Object",
				In:      "header",
				Explode: testutil.Ptr(true),
				Schema: &parser.Schema{
					Type: "object",
					Properties: map[string]*parser.Schema{
						"key":  {Type: "string"},
						"key2": {Type: "string"},
					},
				},
			},
			expected: map[string]any{"key": "value", "key2": "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.DeserializeHeaderParam(tt.value, tt.param)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestDeserializeCookieParam(t *testing.T) {
	d := NewParamDeserializer()

	tests := []struct {
		name     string
		value    string
		param    *parser.Parameter
		expected any
	}{
		{
			name:  "cookie primitive",
			value: "hello",
			param: &parser.Parameter{
				Name:   "session",
				In:     "cookie",
				Schema: &parser.Schema{Type: "string"},
			},
			expected: "hello",
		},
		{
			name:  "cookie integer",
			value: "42",
			param: &parser.Parameter{
				Name:   "count",
				In:     "cookie",
				Schema: &parser.Schema{Type: "integer"},
			},
			expected: int64(42),
		},
		{
			name:  "cookie array",
			value: "a,b,c",
			param: &parser.Parameter{
				Name:   "items",
				In:     "cookie",
				Schema: &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
			},
			expected: []any{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.DeserializeCookieParam(tt.value, tt.param)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestCoerceValue(t *testing.T) {
	d := NewParamDeserializer()

	tests := []struct {
		name     string
		value    string
		schema   *parser.Schema
		expected any
	}{
		{"string", "hello", &parser.Schema{Type: "string"}, "hello"},
		{"integer valid", "42", &parser.Schema{Type: "integer"}, int64(42)},
		{"integer invalid", "not-a-number", &parser.Schema{Type: "integer"}, "not-a-number"},
		{"number valid", "3.14", &parser.Schema{Type: "number"}, 3.14},
		{"number invalid", "not-a-number", &parser.Schema{Type: "number"}, "not-a-number"},
		{"boolean true", "true", &parser.Schema{Type: "boolean"}, true},
		{"boolean false", "false", &parser.Schema{Type: "boolean"}, false},
		{"boolean invalid", "not-a-bool", &parser.Schema{Type: "boolean"}, "not-a-bool"},
		{"nil schema", "hello", nil, "hello"},
		{"unknown type", "hello", &parser.Schema{Type: "unknown"}, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.coerceValue(tt.value, tt.schema)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestDeserializeDelimited_SingleValue(t *testing.T) {
	d := NewParamDeserializer()

	// Non-array with single value
	result := d.deserializeDelimited([]string{"hello"}, " ", &parser.Schema{Type: "string"})
	if result != "hello" {
		t.Errorf("got %v, want 'hello'", result)
	}
}

func TestDeserializeDelimited_MultipleNonArray(t *testing.T) {
	d := NewParamDeserializer()

	// Multiple values without array schema
	result := d.deserializeDelimited([]string{"a b", "c d"}, " ", &parser.Schema{Type: "string"})
	expected := []string{"a", "b", "c", "d"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestGetPropertySchema(t *testing.T) {
	d := NewParamDeserializer()

	// Nil schema
	if d.getPropertySchema(nil, "foo") != nil {
		t.Error("expected nil for nil schema")
	}

	// No properties
	if d.getPropertySchema(&parser.Schema{Type: "object"}, "foo") != nil {
		t.Error("expected nil for schema without properties")
	}

	// Property exists
	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	}
	propSchema := d.getPropertySchema(schema, "name")
	if propSchema == nil || getSchemaType(propSchema) != "string" {
		t.Errorf("expected string schema, got %v", propSchema)
	}
}

func TestIsArraySchema(t *testing.T) {
	if isArraySchema(&parser.Schema{Type: "array"}) != true {
		t.Error("expected true for array type")
	}
	if isArraySchema(&parser.Schema{Type: "string"}) != false {
		t.Error("expected false for string type")
	}
	if isArraySchema(nil) != false {
		t.Error("expected false for nil schema")
	}
}

func TestIsObjectSchema(t *testing.T) {
	if isObjectSchema(&parser.Schema{Type: "object"}) != true {
		t.Error("expected true for object type")
	}
	if isObjectSchema(&parser.Schema{Type: "string"}) != false {
		t.Error("expected false for string type")
	}
	if isObjectSchema(nil) != false {
		t.Error("expected false for nil schema")
	}
}

func TestGetItemsSchema(t *testing.T) {
	// Nil schema
	if getItemsSchema(nil) != nil {
		t.Error("expected nil for nil schema")
	}

	// No items
	if getItemsSchema(&parser.Schema{Type: "array"}) != nil {
		t.Error("expected nil for schema without items")
	}

	// Items as *Schema
	itemSchema := &parser.Schema{Type: "string"}
	schema := &parser.Schema{Type: "array", Items: itemSchema}
	if getItemsSchema(schema) != itemSchema {
		t.Error("expected items schema")
	}

	// Items as bool (OAS 3.1+)
	schema = &parser.Schema{Type: "array", Items: true}
	if getItemsSchema(schema) != nil {
		t.Error("expected nil for bool items")
	}
}

func TestFormArrayMultipleValuesNoExplode(t *testing.T) {
	d := NewParamDeserializer()

	// Test edge case: multiple values passed but explode=false
	// This typically shouldn't happen, but we should handle it gracefully
	result := d.DeserializeQueryParam(
		[]string{"a", "b", "c"},
		&parser.Parameter{
			Name:    "items",
			In:      "query",
			Explode: testutil.Ptr(false),
			Schema:  &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
		},
	)

	expected := []any{"a", "b", "c"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestFormObjectExplodeMultipleValues(t *testing.T) {
	d := NewParamDeserializer()

	// Multiple values with explode object
	result := d.DeserializeQueryParam(
		[]string{"a", "b"},
		&parser.Parameter{
			Name:    "obj",
			In:      "query",
			Explode: testutil.Ptr(true),
			Schema:  &parser.Schema{Type: "object"},
		},
	)

	expected := []string{"a", "b"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}
