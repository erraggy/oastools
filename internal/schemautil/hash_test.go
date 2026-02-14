package schemautil

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaHasher_Hash_Consistency(t *testing.T) {
	hasher := NewSchemaHasher()

	schema := &parser.Schema{
		Type:   "object",
		Format: "",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer", Format: "int32"},
		},
		Required: []string{"name"},
	}

	hash1 := hasher.Hash(schema)
	hash2 := hasher.Hash(schema)

	assert.Equal(t, hash1, hash2, "Hash is not consistent")
}

func TestSchemaHasher_Hash_IdenticalSchemas(t *testing.T) {
	hasher := NewSchemaHasher()

	schema1 := &parser.Schema{
		Type:   "object",
		Format: "",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer", Format: "int32"},
		},
		Required: []string{"name"},
	}

	schema2 := &parser.Schema{
		Type:   "object",
		Format: "",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer", Format: "int32"},
		},
		Required: []string{"name"},
	}

	hash1 := hasher.Hash(schema1)
	hash2 := hasher.Hash(schema2)

	assert.Equal(t, hash1, hash2, "Identical schemas should have same hash")
}

func TestSchemaHasher_Hash_DifferentSchemas(t *testing.T) {
	hasher := NewSchemaHasher()

	tests := []struct {
		name    string
		schema1 *parser.Schema
		schema2 *parser.Schema
	}{
		{
			name:    "different types",
			schema1: &parser.Schema{Type: "string"},
			schema2: &parser.Schema{Type: "integer"},
		},
		{
			name:    "different formats",
			schema1: &parser.Schema{Type: "string", Format: "email"},
			schema2: &parser.Schema{Type: "string", Format: "uri"},
		},
		{
			name: "different properties",
			schema1: &parser.Schema{
				Type:       "object",
				Properties: map[string]*parser.Schema{"foo": {Type: "string"}},
			},
			schema2: &parser.Schema{
				Type:       "object",
				Properties: map[string]*parser.Schema{"bar": {Type: "string"}},
			},
		},
		{
			name: "different required",
			schema1: &parser.Schema{
				Type:     "object",
				Required: []string{"foo"},
			},
			schema2: &parser.Schema{
				Type:     "object",
				Required: []string{"bar"},
			},
		},
		{
			name:    "different enum",
			schema1: &parser.Schema{Type: "string", Enum: []any{"a", "b"}},
			schema2: &parser.Schema{Type: "string", Enum: []any{"x", "y"}},
		},
		{
			name:    "different pattern",
			schema1: &parser.Schema{Type: "string", Pattern: "^[a-z]+$"},
			schema2: &parser.Schema{Type: "string", Pattern: "^[0-9]+$"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hasher.Hash(tt.schema1)
			hash2 := hasher.Hash(tt.schema2)
			assert.NotEqual(t, hash1, hash2, "Different schemas should have different hashes (hash collision)")
		})
	}
}

func TestSchemaHasher_Hash_RequiredOrderIndependent(t *testing.T) {
	hasher := NewSchemaHasher()

	schema1 := &parser.Schema{
		Type:     "object",
		Required: []string{"a", "b", "c"},
	}

	schema2 := &parser.Schema{
		Type:     "object",
		Required: []string{"c", "a", "b"},
	}

	hash1 := hasher.Hash(schema1)
	hash2 := hasher.Hash(schema2)

	assert.Equal(t, hash1, hash2, "Required order should not affect hash")
}

func TestSchemaHasher_Hash_PropertyOrderIndependent(t *testing.T) {
	hasher := NewSchemaHasher()

	// Create schemas with properties in different insertion order
	schema1 := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"alpha": {Type: "string"},
			"beta":  {Type: "integer"},
			"gamma": {Type: "boolean"},
		},
	}

	schema2 := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"gamma": {Type: "boolean"},
			"alpha": {Type: "string"},
			"beta":  {Type: "integer"},
		},
	}

	hash1 := hasher.Hash(schema1)
	hash2 := hasher.Hash(schema2)

	assert.Equal(t, hash1, hash2, "Property order should not affect hash")
}

func TestSchemaHasher_Hash_CircularReference(t *testing.T) {
	hasher := NewSchemaHasher()

	// Create a circular reference: schema -> property -> back to schema
	schema := &parser.Schema{
		Type:       "object",
		Properties: map[string]*parser.Schema{},
	}
	schema.Properties["self"] = schema

	// Should not panic or infinite loop
	hash := hasher.Hash(schema)
	assert.NotEqual(t, uint64(0), hash, "Hash should be non-zero for circular schema")

	// Verify consistency even with circular reference
	hash2 := hasher.Hash(schema)
	assert.Equal(t, hash, hash2, "Hash should be consistent for circular schema")
}

func TestSchemaHasher_Hash_NilSchema(t *testing.T) {
	hasher := NewSchemaHasher()
	hash := hasher.Hash(nil)
	// Should not panic
	assert.NotEqual(t, uint64(0), hash, "Nil schema should still produce a hash")
}

func TestSchemaHasher_Hash_RefSchema(t *testing.T) {
	hasher := NewSchemaHasher()

	schema1 := &parser.Schema{Ref: "#/components/schemas/User"}
	schema2 := &parser.Schema{Ref: "#/components/schemas/User"}
	schema3 := &parser.Schema{Ref: "#/components/schemas/Address"}

	hash1 := hasher.Hash(schema1)
	hash2 := hasher.Hash(schema2)
	hash3 := hasher.Hash(schema3)

	assert.Equal(t, hash1, hash2, "Same $ref should have same hash")
	assert.NotEqual(t, hash1, hash3, "Different $ref should have different hash")
}

func TestSchemaHasher_Hash_OAS31TypeArray(t *testing.T) {
	hasher := NewSchemaHasher()

	// OAS 3.1 can have type as array
	schema1 := &parser.Schema{Type: []any{"string", "null"}}
	schema2 := &parser.Schema{Type: []any{"null", "string"}} // Different order
	schema3 := &parser.Schema{Type: []any{"integer", "null"}}

	hash1 := hasher.Hash(schema1)
	hash2 := hasher.Hash(schema2)
	hash3 := hasher.Hash(schema3)

	assert.Equal(t, hash1, hash2, "Type array order should not affect hash")
	assert.NotEqual(t, hash1, hash3, "Different type arrays should have different hash")
}

func TestSchemaHasher_Hash_Composition(t *testing.T) {
	hasher := NewSchemaHasher()

	schema1 := &parser.Schema{
		AllOf: []*parser.Schema{
			{Type: "object"},
			{Type: "string"},
		},
	}

	schema2 := &parser.Schema{
		AllOf: []*parser.Schema{
			{Type: "object"},
			{Type: "string"},
		},
	}

	schema3 := &parser.Schema{
		AnyOf: []*parser.Schema{
			{Type: "object"},
			{Type: "string"},
		},
	}

	hash1 := hasher.Hash(schema1)
	hash2 := hasher.Hash(schema2)
	hash3 := hasher.Hash(schema3)

	assert.Equal(t, hash1, hash2, "Identical allOf should have same hash")
	assert.NotEqual(t, hash1, hash3, "allOf and anyOf should have different hash")
}

func TestSchemaHasher_Hash_NumericConstraints(t *testing.T) {
	hasher := NewSchemaHasher()

	min1, min2 := 0.0, 1.0
	max1, max2 := 100.0, 200.0

	schema1 := &parser.Schema{Type: "integer", Minimum: &min1, Maximum: &max1}
	schema2 := &parser.Schema{Type: "integer", Minimum: &min1, Maximum: &max1}
	schema3 := &parser.Schema{Type: "integer", Minimum: &min2, Maximum: &max2}

	hash1 := hasher.Hash(schema1)
	hash2 := hasher.Hash(schema2)
	hash3 := hasher.Hash(schema3)

	assert.Equal(t, hash1, hash2, "Same constraints should have same hash")
	assert.NotEqual(t, hash1, hash3, "Different constraints should have different hash")
}

func TestSchemaHasher_Hash_ArrayItems(t *testing.T) {
	hasher := NewSchemaHasher()

	schema1 := &parser.Schema{
		Type:  "array",
		Items: &parser.Schema{Type: "string"},
	}
	schema2 := &parser.Schema{
		Type:  "array",
		Items: &parser.Schema{Type: "string"},
	}
	schema3 := &parser.Schema{
		Type:  "array",
		Items: &parser.Schema{Type: "integer"},
	}

	hash1 := hasher.Hash(schema1)
	hash2 := hasher.Hash(schema2)
	hash3 := hasher.Hash(schema3)

	assert.Equal(t, hash1, hash2, "Same items should have same hash")
	assert.NotEqual(t, hash1, hash3, "Different items should have different hash")
}

func TestSchemaHasher_Hash_AdditionalPropertiesBool(t *testing.T) {
	hasher := NewSchemaHasher()

	schema1 := &parser.Schema{Type: "object", AdditionalProperties: true}
	schema2 := &parser.Schema{Type: "object", AdditionalProperties: true}
	schema3 := &parser.Schema{Type: "object", AdditionalProperties: false}

	hash1 := hasher.Hash(schema1)
	hash2 := hasher.Hash(schema2)
	hash3 := hasher.Hash(schema3)

	assert.Equal(t, hash1, hash2, "Same additionalProperties should have same hash")
	assert.NotEqual(t, hash1, hash3, "Different additionalProperties should have different hash")
}

func TestSchemaHasher_GroupByHash(t *testing.T) {
	hasher := NewSchemaHasher()

	schemas := map[string]*parser.Schema{
		"User": {
			Type: "object",
			Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
			},
		},
		"Person": { // Identical to User
			Type: "object",
			Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
			},
		},
		"Address": { // Different
			Type: "object",
			Properties: map[string]*parser.Schema{
				"street": {Type: "string"},
			},
		},
	}

	groups := hasher.GroupByHash(schemas)

	// Should have 2 groups: one with User+Person, one with Address
	assert.Len(t, groups, 2)

	// Find the group with multiple schemas
	foundDuplicateGroup := false
	for _, names := range groups {
		if len(names) == 2 {
			foundDuplicateGroup = true
			assert.ElementsMatch(t, []string{"User", "Person"}, names)
		}
	}
	require.True(t, foundDuplicateGroup, "Should find a group with 2 identical schemas")
}

func TestSchemaHasher_Hash_MetadataIgnored(t *testing.T) {
	hasher := NewSchemaHasher()

	// Schemas that differ only in metadata should have the same hash
	schema1 := &parser.Schema{
		Type:        "string",
		Title:       "User Name",
		Description: "The name of the user",
	}

	schema2 := &parser.Schema{
		Type:        "string",
		Title:       "Different Title",
		Description: "Completely different description",
	}

	hash1 := hasher.Hash(schema1)
	hash2 := hasher.Hash(schema2)

	assert.Equal(t, hash1, hash2, "Metadata-only differences should not affect hash")
}
