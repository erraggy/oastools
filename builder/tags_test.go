package builder

import (
	"reflect"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJSONTag(t *testing.T) {
	testCases := []struct {
		tag  string
		name string
		opts []string
	}{
		{"", "", nil},
		{"name", "name", nil},
		{"name,omitempty", "name", []string{"omitempty"}},
		{"name,omitempty,string", "name", []string{"omitempty", "string"}},
		{"-", "-", nil},
		{",omitempty", "", []string{"omitempty"}},
	}

	for _, tc := range testCases {
		t.Run(tc.tag, func(t *testing.T) {
			name, opts := parseJSONTag(tc.tag)
			assert.Equal(t, tc.name, name)
			assert.Equal(t, tc.opts, opts)
		})
	}
}

func TestHasOmitempty(t *testing.T) {
	assert.True(t, hasOmitempty([]string{"omitempty"}))
	assert.True(t, hasOmitempty([]string{"string", "omitempty"}))
	assert.False(t, hasOmitempty([]string{}))
	assert.False(t, hasOmitempty(nil))
	assert.False(t, hasOmitempty([]string{"string"}))
}

func TestParseOASTag(t *testing.T) {
	testCases := []struct {
		tag    string
		result map[string]string
	}{
		{"", map[string]string{}},
		{"description=Test", map[string]string{"description": "Test"}},
		{"minLength=1,maxLength=100", map[string]string{"minLength": "1", "maxLength": "100"}},
		{"enum=a|b|c", map[string]string{"enum": "a|b|c"}},
		{"deprecated", map[string]string{"deprecated": "true"}},
		{"readOnly=true,writeOnly=false", map[string]string{"readOnly": "true", "writeOnly": "false"}},
		{" spaced = value ", map[string]string{"spaced": "value"}},
	}

	for _, tc := range testCases {
		t.Run(tc.tag, func(t *testing.T) {
			result := parseOASTag(tc.tag)
			assert.Equal(t, tc.result, result)
		})
	}
}

func TestApplyOASTag(t *testing.T) {
	t.Run("description", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "description=Test description")
		assert.Equal(t, "Test description", result.Description)
	})

	t.Run("format", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "format=email")
		assert.Equal(t, "email", result.Format)
	})

	t.Run("enum", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "enum=admin|user|guest")
		require.Len(t, result.Enum, 3)
		assert.Equal(t, "admin", result.Enum[0])
		assert.Equal(t, "user", result.Enum[1])
		assert.Equal(t, "guest", result.Enum[2])
	})

	t.Run("minimum/maximum", func(t *testing.T) {
		schema := &parser.Schema{Type: "integer"}
		result := applyOASTag(schema, "minimum=0,maximum=100")
		require.NotNil(t, result.Minimum)
		require.NotNil(t, result.Maximum)
		assert.Equal(t, float64(0), *result.Minimum)
		assert.Equal(t, float64(100), *result.Maximum)
	})

	t.Run("minLength/maxLength", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "minLength=1,maxLength=50")
		require.NotNil(t, result.MinLength)
		require.NotNil(t, result.MaxLength)
		assert.Equal(t, 1, *result.MinLength)
		assert.Equal(t, 50, *result.MaxLength)
	})

	t.Run("pattern", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "pattern=^[a-z]+$")
		assert.Equal(t, "^[a-z]+$", result.Pattern)
	})

	t.Run("minItems/maxItems", func(t *testing.T) {
		schema := &parser.Schema{Type: "array"}
		result := applyOASTag(schema, "minItems=1,maxItems=10")
		require.NotNil(t, result.MinItems)
		require.NotNil(t, result.MaxItems)
		assert.Equal(t, 1, *result.MinItems)
		assert.Equal(t, 10, *result.MaxItems)
	})

	t.Run("readOnly", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "readOnly=true")
		assert.True(t, result.ReadOnly)
	})

	t.Run("writeOnly", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "writeOnly=true")
		assert.True(t, result.WriteOnly)
	})

	t.Run("nullable", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "nullable=true")
		assert.True(t, result.Nullable)
	})

	t.Run("deprecated", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "deprecated=true")
		assert.True(t, result.Deprecated)
	})

	t.Run("title", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "title=My Title")
		assert.Equal(t, "My Title", result.Title)
	})

	t.Run("default string", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "default=hello")
		assert.Equal(t, "hello", result.Default)
	})

	t.Run("default integer", func(t *testing.T) {
		schema := &parser.Schema{Type: "integer"}
		result := applyOASTag(schema, "default=42")
		assert.Equal(t, int64(42), result.Default)
	})

	t.Run("default boolean", func(t *testing.T) {
		schema := &parser.Schema{Type: "boolean"}
		result := applyOASTag(schema, "default=true")
		assert.Equal(t, true, result.Default)
	})

	t.Run("example", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "example=test@example.com")
		assert.Equal(t, "test@example.com", result.Example)
	})

	t.Run("multiple options", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		result := applyOASTag(schema, "description=Email address,format=email,minLength=5,maxLength=100")
		assert.Equal(t, "Email address", result.Description)
		assert.Equal(t, "email", result.Format)
		require.NotNil(t, result.MinLength)
		assert.Equal(t, 5, *result.MinLength)
		require.NotNil(t, result.MaxLength)
		assert.Equal(t, 100, *result.MaxLength)
	})

	t.Run("does not modify original", func(t *testing.T) {
		schema := &parser.Schema{Type: "string", Description: "original"}
		result := applyOASTag(schema, "description=modified")
		assert.Equal(t, "original", schema.Description)
		assert.Equal(t, "modified", result.Description)
	})
}

func TestIsFieldRequired(t *testing.T) {
	type TestStruct struct {
		Required1   string  `json:"required1"`
		Required2   int     `json:"required2"`
		Optional1   string  `json:"optional1,omitempty"`
		Optional2   *string `json:"optional2"`
		ExplicitReq string  `json:"explicit_req" oas:"required=true"`
		ExplicitOpt string  `json:"explicit_opt" oas:"required=false"`
		OmitWithReq string  `json:"omit_req,omitempty" oas:"required=true"`
	}

	typ := reflect.TypeOf(TestStruct{})

	testCases := []struct {
		fieldName string
		required  bool
	}{
		{"Required1", true},
		{"Required2", true},
		{"Optional1", false},
		{"Optional2", false},
		{"ExplicitReq", true},
		{"ExplicitOpt", false},
		{"OmitWithReq", true}, // Explicit required=true overrides omitempty
	}

	for _, tc := range testCases {
		t.Run(tc.fieldName, func(t *testing.T) {
			field, ok := typ.FieldByName(tc.fieldName)
			require.True(t, ok)

			jsonTag := field.Tag.Get("json")
			_, jsonOpts := parseJSONTag(jsonTag)

			result := isFieldRequired(field, jsonOpts)
			assert.Equal(t, tc.required, result)
		})
	}
}

func TestCopySchema(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := copySchema(nil)
		assert.Nil(t, result)
	})

	t.Run("basic copy", func(t *testing.T) {
		min := 0.0
		max := 100.0
		original := &parser.Schema{
			Type:        "integer",
			Format:      "int32",
			Description: "Test",
			Minimum:     &min,
			Maximum:     &max,
		}

		result := copySchema(original)

		// Verify copy
		assert.Equal(t, "integer", result.Type)
		assert.Equal(t, "int32", result.Format)
		assert.Equal(t, "Test", result.Description)
		require.NotNil(t, result.Minimum)
		assert.Equal(t, 0.0, *result.Minimum)

		// Modify copy - should not affect original
		result.Description = "Modified"
		assert.Equal(t, "Test", original.Description)
	})

	t.Run("deep copy pointer fields", func(t *testing.T) {
		min := 1.0
		max := 100.0
		minLen := 5
		maxLen := 50
		minItems := 1
		maxItems := 10
		minProps := 2
		maxProps := 20
		multOf := 5.0

		original := &parser.Schema{
			Type:             "integer",
			Minimum:          &min,
			Maximum:          &max,
			MinLength:        &minLen,
			MaxLength:        &maxLen,
			MinItems:         &minItems,
			MaxItems:         &maxItems,
			MinProperties:    &minProps,
			MaxProperties:    &maxProps,
			MultipleOf:       &multOf,
			ExclusiveMaximum: 99.0, // interface{} type, not pointer
			ExclusiveMinimum: 2.0,  // interface{} type, not pointer
		}

		result := copySchema(original)

		// Verify all pointer fields are copied
		require.NotNil(t, result.Minimum)
		require.NotNil(t, result.Maximum)
		require.NotNil(t, result.MinLength)
		require.NotNil(t, result.MaxLength)
		require.NotNil(t, result.MinItems)
		require.NotNil(t, result.MaxItems)
		require.NotNil(t, result.MinProperties)
		require.NotNil(t, result.MaxProperties)
		require.NotNil(t, result.MultipleOf)
		require.NotNil(t, result.ExclusiveMaximum)
		require.NotNil(t, result.ExclusiveMinimum)

		// Verify values
		assert.Equal(t, 1.0, *result.Minimum)
		assert.Equal(t, 100.0, *result.Maximum)
		assert.Equal(t, 5, *result.MinLength)
		assert.Equal(t, 50, *result.MaxLength)
		assert.Equal(t, 1, *result.MinItems)
		assert.Equal(t, 10, *result.MaxItems)
		assert.Equal(t, 2, *result.MinProperties)
		assert.Equal(t, 20, *result.MaxProperties)
		assert.Equal(t, 5.0, *result.MultipleOf)
		assert.Equal(t, 99.0, result.ExclusiveMaximum) // interface{} type
		assert.Equal(t, 2.0, result.ExclusiveMinimum)  // interface{} type

		// Verify pointers are different (deep copy)
		assert.NotSame(t, original.Minimum, result.Minimum)
		assert.NotSame(t, original.Maximum, result.Maximum)
		assert.NotSame(t, original.MinLength, result.MinLength)
		assert.NotSame(t, original.MaxLength, result.MaxLength)
		assert.NotSame(t, original.MinItems, result.MinItems)
		assert.NotSame(t, original.MaxItems, result.MaxItems)
		assert.NotSame(t, original.MinProperties, result.MinProperties)
		assert.NotSame(t, original.MaxProperties, result.MaxProperties)
		assert.NotSame(t, original.MultipleOf, result.MultipleOf)

		// Modifying result should not affect original
		*result.Minimum = 999.0
		assert.Equal(t, 1.0, *original.Minimum)
	})

	t.Run("deep copy slices", func(t *testing.T) {
		original := &parser.Schema{
			Type:     "object",
			Enum:     []any{"a", "b", "c"},
			Required: []string{"field1", "field2"},
		}

		result := copySchema(original)

		// Verify slices are copied
		require.Len(t, result.Enum, 3)
		require.Len(t, result.Required, 2)

		// Verify values
		assert.Equal(t, "a", result.Enum[0])
		assert.Equal(t, "field1", result.Required[0])

		// Modifying result slices should not affect original
		result.Enum[0] = "modified"
		result.Required[0] = "modified"
		assert.Equal(t, "a", original.Enum[0])
		assert.Equal(t, "field1", original.Required[0])
	})
}

func TestIntegration_OASTagsOnStruct(t *testing.T) {
	type User struct {
		ID       int64  `json:"id" oas:"description=Unique identifier,readOnly=true"`
		Name     string `json:"name" oas:"minLength=1,maxLength=100"`
		Email    string `json:"email" oas:"format=email,description=Email address"`
		Role     string `json:"role" oas:"enum=admin|user|guest,default=user"`
		Age      int    `json:"age,omitempty" oas:"minimum=0,maximum=150"`
		Password string `json:"-"`
		IsActive bool   `json:"is_active" oas:"deprecated=true"`
	}

	b := New(parser.OASVersion320)
	b.generateSchema(User{})

	require.Contains(t, b.schemas, "User")
	schema := b.schemas["User"]

	// Check id field
	require.Contains(t, schema.Properties, "id")
	idProp := schema.Properties["id"]
	assert.Equal(t, "Unique identifier", idProp.Description)
	assert.True(t, idProp.ReadOnly)

	// Check name field
	require.Contains(t, schema.Properties, "name")
	nameProp := schema.Properties["name"]
	require.NotNil(t, nameProp.MinLength)
	assert.Equal(t, 1, *nameProp.MinLength)
	require.NotNil(t, nameProp.MaxLength)
	assert.Equal(t, 100, *nameProp.MaxLength)

	// Check email field
	require.Contains(t, schema.Properties, "email")
	emailProp := schema.Properties["email"]
	assert.Equal(t, "email", emailProp.Format)
	assert.Equal(t, "Email address", emailProp.Description)

	// Check role field
	require.Contains(t, schema.Properties, "role")
	roleProp := schema.Properties["role"]
	require.Len(t, roleProp.Enum, 3)
	assert.Equal(t, "user", roleProp.Default)

	// Check age field
	require.Contains(t, schema.Properties, "age")
	ageProp := schema.Properties["age"]
	require.NotNil(t, ageProp.Minimum)
	require.NotNil(t, ageProp.Maximum)
	assert.Equal(t, 0.0, *ageProp.Minimum)
	assert.Equal(t, 150.0, *ageProp.Maximum)

	// Password should be excluded
	assert.NotContains(t, schema.Properties, "Password")

	// Check deprecated field
	require.Contains(t, schema.Properties, "is_active")
	isActiveProp := schema.Properties["is_active"]
	assert.True(t, isActiveProp.Deprecated)

	// Check required fields
	assert.Contains(t, schema.Required, "id")
	assert.Contains(t, schema.Required, "name")
	assert.Contains(t, schema.Required, "email")
	assert.Contains(t, schema.Required, "role")
	assert.Contains(t, schema.Required, "is_active")
	assert.NotContains(t, schema.Required, "age") // Has omitempty
}

func TestParseDefaultValue(t *testing.T) {
	testCases := []struct {
		value      string
		schemaType any
		expected   any
	}{
		{"hello", "string", "hello"},
		{"42", "integer", int64(42)},
		{"3.14", "number", 3.14},
		{"true", "boolean", true},
		{"false", "boolean", false},
		{"123", nil, "123"}, // Unknown type returns string
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			result := parseDefaultValue(tc.value, tc.schemaType)
			assert.Equal(t, tc.expected, result)
		})
	}
}
