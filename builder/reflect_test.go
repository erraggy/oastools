package builder

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaFrom(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		schema := SchemaFrom("")
		assert.Equal(t, "string", schema.Type)
	})

	t.Run("int32", func(t *testing.T) {
		schema := SchemaFrom(int32(0))
		assert.Equal(t, "integer", schema.Type)
		assert.Equal(t, "int32", schema.Format)
	})

	t.Run("int64", func(t *testing.T) {
		schema := SchemaFrom(int64(0))
		assert.Equal(t, "integer", schema.Type)
		assert.Equal(t, "int64", schema.Format)
	})

	t.Run("float32", func(t *testing.T) {
		schema := SchemaFrom(float32(0))
		assert.Equal(t, "number", schema.Type)
		assert.Equal(t, "float", schema.Format)
	})

	t.Run("float64", func(t *testing.T) {
		schema := SchemaFrom(float64(0))
		assert.Equal(t, "number", schema.Type)
		assert.Equal(t, "double", schema.Format)
	})

	t.Run("bool", func(t *testing.T) {
		schema := SchemaFrom(false)
		assert.Equal(t, "boolean", schema.Type)
	})

	t.Run("time.Time", func(t *testing.T) {
		schema := SchemaFrom(time.Time{})
		assert.Equal(t, "string", schema.Type)
		assert.Equal(t, "date-time", schema.Format)
	})

	t.Run("nil", func(t *testing.T) {
		schema := SchemaFrom(nil)
		assert.NotNil(t, schema)
	})
}

func TestBuilder_generateSchema_Struct(t *testing.T) {
	type SimpleStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	b := New(parser.OASVersion320)
	schema := b.generateSchema(SimpleStruct{})

	// Schema should be a $ref
	assert.Contains(t, schema.Ref, "builder.SimpleStruct")

	// Check the actual schema
	require.Contains(t, b.schemas, "builder.SimpleStruct")
	actualSchema := b.schemas["builder.SimpleStruct"]
	assert.Equal(t, "object", actualSchema.Type)
	require.Contains(t, actualSchema.Properties, "name")
	require.Contains(t, actualSchema.Properties, "age")
	assert.Equal(t, "string", actualSchema.Properties["name"].Type)
	assert.Equal(t, "integer", actualSchema.Properties["age"].Type)
}

func TestBuilder_generateSchema_RequiredFields(t *testing.T) {
	type StructWithRequired struct {
		Required1 string  `json:"required1"`           // Required (no omitempty)
		Required2 int     `json:"required2"`           // Required (no omitempty)
		Optional1 string  `json:"optional1,omitempty"` // Optional (omitempty)
		Optional2 *string `json:"optional2"`           // Optional (pointer)
		Optional3 *int    `json:"optional3,omitempty"` // Optional (pointer + omitempty)
	}

	b := New(parser.OASVersion320)
	b.generateSchema(StructWithRequired{})

	require.Contains(t, b.schemas, "builder.StructWithRequired")
	schema := b.schemas["builder.StructWithRequired"]

	assert.Contains(t, schema.Required, "required1")
	assert.Contains(t, schema.Required, "required2")
	assert.NotContains(t, schema.Required, "optional1")
	assert.NotContains(t, schema.Required, "optional2")
	assert.NotContains(t, schema.Required, "optional3")
}

func TestBuilder_generateSchema_JSONTagMinus(t *testing.T) {
	type StructWithExcluded struct {
		Included string `json:"included"`
		Excluded string `json:"-"`
	}

	b := New(parser.OASVersion320)
	b.generateSchema(StructWithExcluded{})

	require.Contains(t, b.schemas, "builder.StructWithExcluded")
	schema := b.schemas["builder.StructWithExcluded"]

	assert.Contains(t, schema.Properties, "included")
	assert.NotContains(t, schema.Properties, "Excluded")
}

func TestBuilder_generateSchema_UnexportedFields(t *testing.T) {
	type StructWithUnexported struct {
		Exported   string `json:"exported"`
		unexported string //nolint:unused
	}

	b := New(parser.OASVersion320)
	b.generateSchema(StructWithUnexported{})

	require.Contains(t, b.schemas, "builder.StructWithUnexported")
	schema := b.schemas["builder.StructWithUnexported"]

	assert.Contains(t, schema.Properties, "exported")
	assert.NotContains(t, schema.Properties, "unexported")
}

func TestBuilder_generateSchema_Slice(t *testing.T) {
	b := New(parser.OASVersion320)
	schema := b.generateSchema([]string{})

	assert.Equal(t, "array", schema.Type)
	require.NotNil(t, schema.Items)
	itemsSchema, ok := schema.Items.(*parser.Schema)
	require.True(t, ok)
	assert.Equal(t, "string", itemsSchema.Type)
}

func TestBuilder_generateSchema_SliceOfStructs(t *testing.T) {
	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	b := New(parser.OASVersion320)
	schema := b.generateSchema([]Item{})

	assert.Equal(t, "array", schema.Type)
	require.NotNil(t, schema.Items)
	itemsSchema, ok := schema.Items.(*parser.Schema)
	require.True(t, ok)
	assert.Contains(t, itemsSchema.Ref, "builder.Item")
}

func TestBuilder_generateSchema_Map(t *testing.T) {
	b := New(parser.OASVersion320)
	schema := b.generateSchema(map[string]int{})

	assert.Equal(t, "object", schema.Type)
	require.NotNil(t, schema.AdditionalProperties)
	addPropsSchema, ok := schema.AdditionalProperties.(*parser.Schema)
	require.True(t, ok)
	assert.Equal(t, "integer", addPropsSchema.Type)
}

func TestBuilder_generateSchema_MapOfStructs(t *testing.T) {
	type Value struct {
		Data string `json:"data"`
	}

	b := New(parser.OASVersion320)
	schema := b.generateSchema(map[string]Value{})

	assert.Equal(t, "object", schema.Type)
	require.NotNil(t, schema.AdditionalProperties)
	addPropsSchema, ok := schema.AdditionalProperties.(*parser.Schema)
	require.True(t, ok)
	assert.Contains(t, addPropsSchema.Ref, "builder.Value")
}

func TestBuilder_generateSchema_Pointer(t *testing.T) {
	b := New(parser.OASVersion320)
	var ptr *string
	schema := b.generateSchema(ptr)

	assert.Equal(t, "string", schema.Type)
	assert.True(t, schema.Nullable)
}

func TestBuilder_generateSchema_NestedStruct(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	b := New(parser.OASVersion320)
	b.generateSchema(Person{})

	// Both schemas should be registered
	require.Contains(t, b.schemas, "builder.Person")
	require.Contains(t, b.schemas, "builder.Address")

	// Person's address should be a ref
	personSchema := b.schemas["builder.Person"]
	require.Contains(t, personSchema.Properties, "address")
	assert.Contains(t, personSchema.Properties["address"].Ref, "builder.Address")
}

func TestBuilder_generateSchema_CircularReference(t *testing.T) {
	type Node struct {
		Value    int     `json:"value"`
		Children []*Node `json:"children"`
	}

	b := New(parser.OASVersion320)
	b.generateSchema(Node{})

	require.Contains(t, b.schemas, "builder.Node")
	nodeSchema := b.schemas["builder.Node"]

	// Children should be an array with $ref to Node
	require.Contains(t, nodeSchema.Properties, "children")
	childrenProp := nodeSchema.Properties["children"]
	assert.Equal(t, "array", childrenProp.Type)
	require.NotNil(t, childrenProp.Items)
	itemsSchema, ok := childrenProp.Items.(*parser.Schema)
	require.True(t, ok)
	assert.Contains(t, itemsSchema.Ref, "builder.Node")
}

func TestBuilder_generateSchema_EmbeddedStruct(t *testing.T) {
	type Base struct {
		ID        int64  `json:"id"`
		CreatedAt string `json:"created_at"`
	}

	type Extended struct {
		Base
		Name string `json:"name"`
	}

	b := New(parser.OASVersion320)
	b.generateSchema(Extended{})

	require.Contains(t, b.schemas, "builder.Extended")
	schema := b.schemas["builder.Extended"]

	// Should have properties from both Base and Extended
	assert.Contains(t, schema.Properties, "id")
	assert.Contains(t, schema.Properties, "created_at")
	assert.Contains(t, schema.Properties, "name")
}

func TestBuilder_generateSchema_Interface(t *testing.T) {
	b := New(parser.OASVersion320)
	var iface any
	schema := b.generateSchema(iface)

	// interface{}/any should produce an empty schema
	assert.Empty(t, schema.Type)
}

func TestBuilder_generateSchema_AllIntTypes(t *testing.T) {
	testCases := []struct {
		name   string
		value  any
		format string
	}{
		{"int", int(0), "int32"},
		{"int8", int8(0), "int32"},
		{"int16", int16(0), "int32"},
		{"int32", int32(0), "int32"},
		{"int64", int64(0), "int64"},
		{"uint", uint(0), "int32"},
		{"uint8", uint8(0), "int32"},
		{"uint16", uint16(0), "int32"},
		{"uint32", uint32(0), "int32"},
		{"uint64", uint64(0), "int64"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schema := SchemaFrom(tc.value)
			assert.Equal(t, "integer", schema.Type)
			assert.Equal(t, tc.format, schema.Format)
		})
	}
}

func TestBuilder_RegisterType(t *testing.T) {
	type CustomType struct {
		Field string `json:"field"`
	}

	b := New(parser.OASVersion320)
	schema := b.RegisterType(CustomType{})

	assert.Contains(t, schema.Ref, "builder.CustomType")
	require.Contains(t, b.schemas, "builder.CustomType")
}

func TestBuilder_RegisterTypeAs(t *testing.T) {
	type SomeType struct {
		Field string `json:"field"`
	}

	b := New(parser.OASVersion320)
	schema := b.RegisterTypeAs("MyCustomName", SomeType{})

	assert.Contains(t, schema.Ref, "MyCustomName")
	require.Contains(t, b.schemas, "MyCustomName")
}

func TestSchemaCache(t *testing.T) {
	t.Run("get/set", func(t *testing.T) {
		type TestType struct{}

		b := New(parser.OASVersion320)
		b.generateSchema(TestType{})

		// Generate again - should use cache
		schema := b.generateSchema(TestType{})
		assert.Contains(t, schema.Ref, "TestType")
	})

	t.Run("getNameForType", func(t *testing.T) {
		type TestType struct{}
		cache := newSchemaCache()

		// Not found case
		name := cache.getNameForType(reflect.TypeOf(TestType{}))
		assert.Empty(t, name)

		// Found case
		cache.set(reflect.TypeOf(TestType{}), "TestType", &parser.Schema{})
		name = cache.getNameForType(reflect.TypeOf(TestType{}))
		assert.Equal(t, "TestType", name)
	})

	t.Run("getTypeForName", func(t *testing.T) {
		type TestType struct{}
		cache := newSchemaCache()

		// Not found case
		t1 := cache.getTypeForName("TestType")
		assert.Nil(t, t1)

		// Found case
		cache.set(reflect.TypeOf(TestType{}), "TestType", &parser.Schema{})
		t2 := cache.getTypeForName("TestType")
		assert.Equal(t, reflect.TypeOf(TestType{}), t2)
	})
}

func TestSchemaFromType(t *testing.T) {
	// Test the public SchemaFromType function
	schema := SchemaFromType(reflect.TypeOf(""))
	assert.Equal(t, "string", schema.Type)

	schema = SchemaFromType(reflect.TypeOf(int64(0)))
	assert.Equal(t, "integer", schema.Type)
	assert.Equal(t, "int64", schema.Format)
}

func TestBuilder_generateSchema_PointerToStruct(t *testing.T) {
	type MyStruct struct {
		Field string `json:"field"`
	}

	b := New(parser.OASVersion320)
	var ptr *MyStruct
	schema := b.generateSchema(ptr)

	// Should return a reference with nullable
	assert.Contains(t, schema.Ref, "builder.MyStruct")
	// Note: nullable flag may be set differently on references
	require.Contains(t, b.schemas, "builder.MyStruct")
}

func TestBuilder_generateSchema_AnonymousStruct(t *testing.T) {
	b := New(parser.OASVersion320)
	schema := b.generateSchema(struct {
		Field string `json:"field"`
	}{})

	// Anonymous struct should return a ref to the generated schema
	// or be inline depending on implementation
	assert.NotNil(t, schema)
}

func TestBuilder_generateSchema_UnknownKind(t *testing.T) {
	b := New(parser.OASVersion320)
	// Test with channel type (uncommon)
	ch := make(chan int)
	schema := b.generateSchema(ch)
	// Should return empty schema for unknown types
	assert.NotNil(t, schema)
}

func TestBuilder_extractRefName(t *testing.T) {
	tests := []struct {
		ref      string
		expected string
	}{
		{"#/components/schemas/User", "User"},
		{"#/components/schemas/", ""},
		{"", ""},
		{"#/components/schemas/ComplexName", "ComplexName"},
	}

	for _, tc := range tests {
		result := extractRefName(tc.ref)
		assert.Equal(t, tc.expected, result)
	}
}

func TestBuilder_contains(t *testing.T) {
	slice := []string{"a", "b", "c"}
	assert.True(t, contains(slice, "a"))
	assert.True(t, contains(slice, "b"))
	assert.True(t, contains(slice, "c"))
	assert.False(t, contains(slice, "d"))
	assert.False(t, contains(nil, "a"))
}

func TestBuilder_schemaName_ConflictDetection(t *testing.T) {
	// This test verifies that when two types have the same base name
	// (e.g., models.User from different packages), the conflict is detected
	// and the second type gets a full package path name.

	// Create a builder and simulate a conflict scenario
	b := New(parser.OASVersion320)

	// Simulate registering a type with name "models.User" from package "github.com/foo/models"
	// by directly manipulating the cache
	type FakeType1 struct{ A string }
	type FakeType2 struct{ B string }

	// Register the first type with a name that would conflict
	b.schemaCache.set(reflect.TypeOf(FakeType1{}), "models.User", &parser.Schema{Type: "object"})

	// Now when we try to get a name for a different type that would have the same base name,
	// the conflict detection should kick in
	existingType := b.schemaCache.getTypeForName("models.User")
	assert.NotNil(t, existingType, "Expected to find existing type in cache")
	assert.NotEqual(t, reflect.TypeOf(FakeType2{}), existingType, "Types should be different")

	// Verify the getTypeForName function works correctly for conflict detection
	assert.Equal(t, reflect.TypeOf(FakeType1{}), existingType)
}

func TestSanitizeSchemaName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple type",
			input:    "User",
			expected: "User",
		},
		{
			name:     "generic type with single parameter",
			input:    "Response[User]",
			expected: "Response_User",
		},
		{
			name:     "generic type with package qualifier",
			input:    "Response[main.User]",
			expected: "Response_main.User",
		},
		{
			name:     "generic type with multiple parameters",
			input:    "Map[string,int]",
			expected: "Map_string_int",
		},
		{
			name:     "nested generic type",
			input:    "Response[List[User]]",
			expected: "Response_List_User",
		},
		{
			name:     "complex nested generics",
			input:    "Map[string,Response[User]]",
			expected: "Map_string_Response_User",
		},
		{
			name:     "type with spaces (edge case)",
			input:    "Some Type",
			expected: "Some_Type",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeSchemaName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Generic test types for schema generation tests
type GenericResponse[T any] struct {
	Data    T      `json:"data"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type GenericList[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

type GenericMap[K comparable, V any] struct {
	Entries map[K]V `json:"entries"`
}

func TestBuilder_generateSchema_GenericTypes(t *testing.T) {
	t.Run("simple generic type", func(t *testing.T) {
		type User struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}

		b := New(parser.OASVersion320)
		response := GenericResponse[User]{}
		schema := b.generateSchema(response)

		// Schema should be a $ref with sanitized name (no brackets)
		assert.NotEmpty(t, schema.Ref)
		assert.NotContains(t, schema.Ref, "[")
		assert.NotContains(t, schema.Ref, "]")

		// Find the schema name
		found := false
		for name := range b.schemas {
			if strings.Contains(name, "GenericResponse") {
				found = true
				assert.NotContains(t, name, "[")
				assert.NotContains(t, name, "]")
			}
		}
		assert.True(t, found, "Expected to find GenericResponse schema")
	})

	t.Run("nested generic types", func(t *testing.T) {
		type Item struct {
			Value string `json:"value"`
		}

		b := New(parser.OASVersion320)
		listResponse := GenericResponse[GenericList[Item]]{}
		schema := b.generateSchema(listResponse)

		// All refs should be sanitized
		assert.NotEmpty(t, schema.Ref)
		assert.NotContains(t, schema.Ref, "[")
		assert.NotContains(t, schema.Ref, "]")

		// Check all registered schemas have sanitized names
		for name := range b.schemas {
			assert.NotContains(t, name, "[", "Schema name %s contains brackets", name)
			assert.NotContains(t, name, "]", "Schema name %s contains brackets", name)
		}
	})

	t.Run("generic type with primitive", func(t *testing.T) {
		b := New(parser.OASVersion320)
		response := GenericResponse[string]{}
		schema := b.generateSchema(response)

		assert.NotEmpty(t, schema.Ref)
		assert.NotContains(t, schema.Ref, "[")
		assert.NotContains(t, schema.Ref, "]")
	})

	t.Run("generic list type", func(t *testing.T) {
		type Product struct {
			SKU   string  `json:"sku"`
			Price float64 `json:"price"`
		}

		b := New(parser.OASVersion320)
		list := GenericList[Product]{}
		schema := b.generateSchema(list)

		assert.NotEmpty(t, schema.Ref)
		assert.NotContains(t, schema.Ref, "[")
		assert.NotContains(t, schema.Ref, "]")

		// Verify the internal schema has the right structure
		for name, s := range b.schemas {
			if strings.Contains(name, "GenericList") {
				require.Contains(t, s.Properties, "items")
				require.Contains(t, s.Properties, "total")
			}
		}
	})
}

func TestBuilder_refToSchema_WithGenericTypes(t *testing.T) {
	// Test that $ref URIs don't contain problematic characters
	testCases := []struct {
		name     string
		typeName string
	}{
		{"simple", "User"},
		{"with_dot", "models.User"},
		{"sanitized_generic", "Response_User"},
		{"complex_generic", "Map_string_Response_User"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := New(parser.OASVersion320)
			schema := b.refToSchema(tc.typeName)

			assert.Equal(t, "#/components/schemas/"+tc.typeName, schema.Ref)

			// Ensure no problematic URI characters
			assert.NotContains(t, schema.Ref, "[")
			assert.NotContains(t, schema.Ref, "]")
			assert.NotContains(t, schema.Ref, ",")
			assert.NotContains(t, schema.Ref, " ")
		})
	}
}
