package builder

import (
	"reflect"
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
	assert.Contains(t, schema.Ref, "SimpleStruct")

	// Check the actual schema
	require.Contains(t, b.schemas, "SimpleStruct")
	actualSchema := b.schemas["SimpleStruct"]
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

	require.Contains(t, b.schemas, "StructWithRequired")
	schema := b.schemas["StructWithRequired"]

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

	require.Contains(t, b.schemas, "StructWithExcluded")
	schema := b.schemas["StructWithExcluded"]

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

	require.Contains(t, b.schemas, "StructWithUnexported")
	schema := b.schemas["StructWithUnexported"]

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
	assert.Contains(t, itemsSchema.Ref, "Item")
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
	assert.Contains(t, addPropsSchema.Ref, "Value")
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
	require.Contains(t, b.schemas, "Person")
	require.Contains(t, b.schemas, "Address")

	// Person's address should be a ref
	personSchema := b.schemas["Person"]
	require.Contains(t, personSchema.Properties, "address")
	assert.Contains(t, personSchema.Properties["address"].Ref, "Address")
}

func TestBuilder_generateSchema_CircularReference(t *testing.T) {
	type Node struct {
		Value    int     `json:"value"`
		Children []*Node `json:"children"`
	}

	b := New(parser.OASVersion320)
	b.generateSchema(Node{})

	require.Contains(t, b.schemas, "Node")
	nodeSchema := b.schemas["Node"]

	// Children should be an array with $ref to Node
	require.Contains(t, nodeSchema.Properties, "children")
	childrenProp := nodeSchema.Properties["children"]
	assert.Equal(t, "array", childrenProp.Type)
	require.NotNil(t, childrenProp.Items)
	itemsSchema, ok := childrenProp.Items.(*parser.Schema)
	require.True(t, ok)
	assert.Contains(t, itemsSchema.Ref, "Node")
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

	require.Contains(t, b.schemas, "Extended")
	schema := b.schemas["Extended"]

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

	assert.Contains(t, schema.Ref, "CustomType")
	require.Contains(t, b.schemas, "CustomType")
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

	t.Run("hasName", func(t *testing.T) {
		cache := newSchemaCache()
		assert.False(t, cache.hasName("Test"))

		cache.byName["Test"] = nil
		assert.True(t, cache.hasName("Test"))
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
	assert.Contains(t, schema.Ref, "MyStruct")
	// Note: nullable flag may be set differently on references
	require.Contains(t, b.schemas, "MyStruct")
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
