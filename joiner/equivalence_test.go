package joiner

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareSchemas_NoneMode(t *testing.T) {
	left := &parser.Schema{Type: "string"}
	right := &parser.Schema{Type: "string"}

	result := CompareSchemas(left, right, EquivalenceModeNone)

	assert.False(t, result.Equivalent)
	assert.Equal(t, 0, len(result.Differences))
}

func TestCompareSchemas_IdenticalSchemas(t *testing.T) {
	left := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
		Required: []string{"name"},
	}
	right := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
		Required: []string{"name"},
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.True(t, result.Equivalent)
	assert.Equal(t, 0, len(result.Differences))
}

func TestCompareSchemas_TypeMismatch(t *testing.T) {
	left := &parser.Schema{Type: "string"}
	right := &parser.Schema{Type: "integer"}

	result := CompareSchemas(left, right, EquivalenceModeShallow)

	assert.False(t, result.Equivalent)
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "type", result.Differences[0].Path)
	assert.Equal(t, "type mismatch", result.Differences[0].Description)
}

func TestCompareSchemas_FormatMismatch(t *testing.T) {
	left := &parser.Schema{Type: "string", Format: "date"}
	right := &parser.Schema{Type: "string", Format: "date-time"}

	result := CompareSchemas(left, right, EquivalenceModeShallow)

	assert.False(t, result.Equivalent)
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "format", result.Differences[0].Path)
}

func TestCompareSchemas_RequiredOrderIndependent(t *testing.T) {
	left := &parser.Schema{
		Type:     "object",
		Required: []string{"name", "email", "age"},
	}
	right := &parser.Schema{
		Type:     "object",
		Required: []string{"age", "name", "email"},
	}

	result := CompareSchemas(left, right, EquivalenceModeShallow)

	assert.True(t, result.Equivalent, "required arrays should be order-independent")
	assert.Equal(t, 0, len(result.Differences))
}

func TestCompareSchemas_RequiredMismatch(t *testing.T) {
	left := &parser.Schema{
		Type:     "object",
		Required: []string{"name", "email"},
	}
	right := &parser.Schema{
		Type:     "object",
		Required: []string{"name"},
	}

	result := CompareSchemas(left, right, EquivalenceModeShallow)

	assert.False(t, result.Equivalent)
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "required", result.Differences[0].Path)
}

func TestCompareSchemas_EnumMismatch(t *testing.T) {
	left := &parser.Schema{
		Type: "string",
		Enum: []any{"red", "green", "blue"},
	}
	right := &parser.Schema{
		Type: "string",
		Enum: []any{"red", "blue", "green"},
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.False(t, result.Equivalent, "enum order matters")
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "enum", result.Differences[0].Path)
}

func TestCompareSchemas_PropertyNamesMismatch(t *testing.T) {
	left := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name":  {Type: "string"},
			"email": {Type: "string"},
		},
	}
	right := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
	}

	result := CompareSchemas(left, right, EquivalenceModeShallow)

	assert.False(t, result.Equivalent)
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "properties", result.Differences[0].Path)
}

func TestCompareSchemas_DeepPropertyComparison(t *testing.T) {
	left := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
			"address": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"street": {Type: "string"},
					"city":   {Type: "string"},
				},
			},
		},
	}
	right := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
			"address": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"street": {Type: "string"},
					"city":   {Type: "integer"}, // Different type
				},
			},
		},
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.False(t, result.Equivalent)
	assert.Greater(t, len(result.Differences), 0)
	// Check that the difference is in the nested property
	found := false
	for _, diff := range result.Differences {
		if diff.Path == "properties.address.properties.city.type" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find difference in nested property")
}

func TestCompareSchemas_NumericConstraints(t *testing.T) {
	min5 := 5.0
	max10 := 10.0
	max20 := 20.0

	left := &parser.Schema{
		Type:    "integer",
		Minimum: &min5,
		Maximum: &max10,
	}
	right := &parser.Schema{
		Type:    "integer",
		Minimum: &min5,
		Maximum: &max20,
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.False(t, result.Equivalent)
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "maximum", result.Differences[0].Path)
}

func TestCompareSchemas_StringConstraints(t *testing.T) {
	minLen := 5
	maxLen10 := 10
	maxLen20 := 20

	left := &parser.Schema{
		Type:      "string",
		MinLength: &minLen,
		MaxLength: &maxLen10,
		Pattern:   "^[a-z]+$",
	}
	right := &parser.Schema{
		Type:      "string",
		MinLength: &minLen,
		MaxLength: &maxLen20,
		Pattern:   "^[a-z]+$",
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.False(t, result.Equivalent)
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "maxLength", result.Differences[0].Path)
}

func TestCompareSchemas_ArrayItems(t *testing.T) {
	left := &parser.Schema{
		Type:  "array",
		Items: &parser.Schema{Type: "string"},
	}
	right := &parser.Schema{
		Type:  "array",
		Items: &parser.Schema{Type: "integer"},
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.False(t, result.Equivalent)
	found := false
	for _, diff := range result.Differences {
		if diff.Path == "items.type" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find difference in items type")
}

func TestCompareSchemas_AdditionalProperties(t *testing.T) {
	left := &parser.Schema{
		Type:                 "object",
		AdditionalProperties: &parser.Schema{Type: "string"},
	}
	right := &parser.Schema{
		Type:                 "object",
		AdditionalProperties: &parser.Schema{Type: "integer"},
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.False(t, result.Equivalent)
	found := false
	for _, diff := range result.Differences {
		if diff.Path == "additionalProperties.type" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find difference in additionalProperties type")
}

func TestCompareSchemas_AdditionalPropertiesBoolean(t *testing.T) {
	left := &parser.Schema{
		Type:                 "object",
		AdditionalProperties: true,
	}
	right := &parser.Schema{
		Type:                 "object",
		AdditionalProperties: false,
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.False(t, result.Equivalent)
	assert.Greater(t, len(result.Differences), 0)
}

func TestCompareSchemas_AllOfComposition(t *testing.T) {
	left := &parser.Schema{
		AllOf: []*parser.Schema{
			{Type: "object", Properties: map[string]*parser.Schema{"name": {Type: "string"}}},
			{Type: "object", Properties: map[string]*parser.Schema{"age": {Type: "integer"}}},
		},
	}
	right := &parser.Schema{
		AllOf: []*parser.Schema{
			{Type: "object", Properties: map[string]*parser.Schema{"name": {Type: "string"}}},
			{Type: "object", Properties: map[string]*parser.Schema{"age": {Type: "integer"}}},
		},
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.True(t, result.Equivalent)
}

func TestCompareSchemas_AllOfLengthMismatch(t *testing.T) {
	left := &parser.Schema{
		AllOf: []*parser.Schema{
			{Type: "object"},
			{Type: "object"},
		},
	}
	right := &parser.Schema{
		AllOf: []*parser.Schema{
			{Type: "object"},
		},
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.False(t, result.Equivalent)
	found := false
	for _, diff := range result.Differences {
		if diff.Path == "allOf" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestCompareSchemas_DocsIncludedByDefault(t *testing.T) {
	// Schemas that differ only in documentation fields are NOT equivalent
	// under the default (strict) policy. Callers that want the old loose
	// behavior can opt in with EquivalenceDocsIgnore.
	left := &parser.Schema{
		Type:        "string",
		Title:       "User Name",
		Description: "The name of the user",
		Example:     "John Doe",
	}
	right := &parser.Schema{
		Type:        "string",
		Title:       "Full Name",
		Description: "User's full name",
		Example:     "Jane Smith",
	}

	// Default: docs included -> not equivalent
	result := CompareSchemas(left, right, EquivalenceModeDeep)
	assert.False(t, result.Equivalent, "schemas differing only in docs should NOT be equivalent under default")
	assert.NotEmpty(t, result.Differences, "expected differences to be recorded")

	// Opt-in loose mode: docs ignored -> equivalent
	looseResult := CompareSchemasWithOptions(left, right, CompareOptions{
		Mode: EquivalenceModeDeep,
		Docs: EquivalenceDocsIgnore,
	})
	assert.True(t, looseResult.Equivalent, "schemas should be equivalent when docs are explicitly ignored")
}

func TestCompareSchemas_TitleOnlyDifference(t *testing.T) {
	left := &parser.Schema{Type: "string", Title: "A"}
	right := &parser.Schema{Type: "string", Title: "B"}

	// Default (strict): titles differ -> not equivalent
	strict := CompareSchemas(left, right, EquivalenceModeDeep)
	assert.False(t, strict.Equivalent)
	require.NotEmpty(t, strict.Differences)
	assert.Equal(t, "title", strict.Differences[0].Path)

	// Loose mode: titles ignored -> equivalent
	loose := CompareSchemasWithOptions(left, right, CompareOptions{
		Mode: EquivalenceModeDeep,
		Docs: EquivalenceDocsIgnore,
	})
	assert.True(t, loose.Equivalent)
}

func TestCompareSchemas_DescriptionOnlyDifference(t *testing.T) {
	left := &parser.Schema{Type: "string", Description: "the first"}
	right := &parser.Schema{Type: "string", Description: "the second"}

	strict := CompareSchemas(left, right, EquivalenceModeDeep)
	assert.False(t, strict.Equivalent)
	require.NotEmpty(t, strict.Differences)
	assert.Equal(t, "description", strict.Differences[0].Path)

	loose := CompareSchemasWithOptions(left, right, CompareOptions{
		Mode: EquivalenceModeDeep,
		Docs: EquivalenceDocsIgnore,
	})
	assert.True(t, loose.Equivalent)
}

func TestCompareSchemas_ExampleOnlyDifference(t *testing.T) {
	left := &parser.Schema{Type: "string", Example: "hello"}
	right := &parser.Schema{Type: "string", Example: "world"}

	strict := CompareSchemas(left, right, EquivalenceModeDeep)
	assert.False(t, strict.Equivalent)
	require.NotEmpty(t, strict.Differences)
	assert.Equal(t, "example", strict.Differences[0].Path)

	loose := CompareSchemasWithOptions(left, right, CompareOptions{
		Mode: EquivalenceModeDeep,
		Docs: EquivalenceDocsIgnore,
	})
	assert.True(t, loose.Equivalent)
}

func TestCompareSchemas_NestedDocDifference(t *testing.T) {
	// Two object schemas, identical at the top level, but a nested property
	// differs only in description. Default behavior: not equivalent.
	left := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"id": {Type: "string", Description: "the user id"},
		},
	}
	right := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"id": {Type: "string", Description: "the account id"},
		},
	}

	strict := CompareSchemas(left, right, EquivalenceModeDeep)
	assert.False(t, strict.Equivalent, "nested description differences should bubble up")
	require.NotEmpty(t, strict.Differences)
	assert.Equal(t, "properties.id.description", strict.Differences[0].Path)

	loose := CompareSchemasWithOptions(left, right, CompareOptions{
		Mode: EquivalenceModeDeep,
		Docs: EquivalenceDocsIgnore,
	})
	assert.True(t, loose.Equivalent, "nested docs should be ignored in loose mode")
}

func TestCompareSchemas_IdenticalSchemasAlwaysEquivalent(t *testing.T) {
	// When schemas are fully identical (structure + docs), both modes agree.
	left := &parser.Schema{
		Type:        "string",
		Title:       "Name",
		Description: "A name",
		Example:     "Alice",
	}
	right := &parser.Schema{
		Type:        "string",
		Title:       "Name",
		Description: "A name",
		Example:     "Alice",
	}

	assert.True(t, CompareSchemas(left, right, EquivalenceModeDeep).Equivalent)
	assert.True(t, CompareSchemasWithOptions(left, right, CompareOptions{
		Mode: EquivalenceModeDeep,
		Docs: EquivalenceDocsIgnore,
	}).Equivalent)
}

func TestIsValidEquivalenceDocs(t *testing.T) {
	assert.True(t, IsValidEquivalenceDocs("include"))
	assert.True(t, IsValidEquivalenceDocs("ignore"))
	assert.False(t, IsValidEquivalenceDocs(""))
	assert.False(t, IsValidEquivalenceDocs("strict"))
	assert.False(t, IsValidEquivalenceDocs("INCLUDE"))
	assert.Equal(t, []string{"include", "ignore"}, ValidEquivalenceDocs())
}

func TestCompareSchemasWithOptions_EmptyDocsDefaultsToInclude(t *testing.T) {
	// Empty Docs is treated as "include" (strict).
	left := &parser.Schema{Type: "string", Description: "one"}
	right := &parser.Schema{Type: "string", Description: "two"}

	result := CompareSchemasWithOptions(left, right, CompareOptions{Mode: EquivalenceModeDeep})
	assert.False(t, result.Equivalent, "empty Docs should default to include (strict)")
}

func TestCompareSchemas_CircularReferences(t *testing.T) {
	// Create circular reference: Node -> children -> Node
	node := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	}
	node.Properties["children"] = &parser.Schema{
		Type:  "array",
		Items: node, // Circular reference
	}

	otherNode := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	}
	otherNode.Properties["children"] = &parser.Schema{
		Type:  "array",
		Items: otherNode,
	}

	// Should not panic or infinite loop
	result := CompareSchemas(node, otherNode, EquivalenceModeDeep)

	// Result may vary, but should complete without error
	assert.NotNil(t, result)
}

func TestCompareSchemas_TypeArray_OAS31(t *testing.T) {
	// OAS 3.1+ allows type as array
	left := &parser.Schema{
		Type: []string{"string", "null"},
	}
	right := &parser.Schema{
		Type: []string{"string", "null"},
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.True(t, result.Equivalent)
}

func TestCompareSchemas_TypeArrayMismatch(t *testing.T) {
	left := &parser.Schema{
		Type: []string{"string", "null"},
	}
	right := &parser.Schema{
		Type: []string{"integer", "null"},
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.False(t, result.Equivalent)
	assert.Greater(t, len(result.Differences), 0)
}

func TestCompareSchemas_EmptySchemas(t *testing.T) {
	left := &parser.Schema{}
	right := &parser.Schema{}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	// Empty schemas are semantically distinct - never equivalent.
	// They serve different purposes depending on context (placeholders,
	// "any type" markers, context-specific wildcards).
	assert.False(t, result.Equivalent, "empty schemas should NOT be equivalent")
	assert.Empty(t, result.Differences, "no structural differences for empty schemas")
}

func TestCompareSchemas_NilSchemas(t *testing.T) {
	// Both nil
	result := CompareSchemas(nil, nil, EquivalenceModeDeep)
	assert.True(t, result.Equivalent, "nil schemas should be equivalent")

	// Left nil
	result = CompareSchemas(nil, &parser.Schema{Type: "string"}, EquivalenceModeDeep)
	assert.False(t, result.Equivalent)
	assert.Equal(t, 1, len(result.Differences))

	// Right nil
	result = CompareSchemas(&parser.Schema{Type: "string"}, nil, EquivalenceModeDeep)
	assert.False(t, result.Equivalent)
	assert.Equal(t, 1, len(result.Differences))
}

func TestEqualTypes_StringTypes(t *testing.T) {
	assert.True(t, equalTypes("string", "string"))
	assert.False(t, equalTypes("string", "integer"))
	assert.False(t, equalTypes("string", nil))
	assert.True(t, equalTypes(nil, nil))
}

func TestEqualTypes_ArrayTypes(t *testing.T) {
	assert.True(t, equalTypes([]string{"string", "null"}, []string{"null", "string"}))
	assert.False(t, equalTypes([]string{"string", "null"}, []string{"integer", "null"}))
	assert.False(t, equalTypes([]string{"string"}, []string{"string", "null"}))
}

func TestPathJoin(t *testing.T) {
	assert.Equal(t, "type", pathJoin("", "type"))
	assert.Equal(t, "properties.name", pathJoin("properties", "name"))
	assert.Equal(t, "properties.address.city", pathJoin("properties.address", "city"))
}

func TestValidEquivalenceModes(t *testing.T) {
	modes := ValidEquivalenceModes()
	assert.Equal(t, 3, len(modes))
	assert.Contains(t, modes, "none")
	assert.Contains(t, modes, "shallow")
	assert.Contains(t, modes, "deep")
}

func TestIsValidEquivalenceMode(t *testing.T) {
	tests := []struct {
		mode     string
		expected bool
	}{
		{"none", true},
		{"shallow", true},
		{"deep", true},
		{"", false},
		{"invalid", false},
		{"DEEP", false}, // case-sensitive
		{"None", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsValidEquivalenceMode(tt.mode))
		})
	}
}

// TestCompareSchemas_JSONSchema2020_12 tests JSON Schema Draft 2020-12 fields
func TestCompareSchemas_JSONSchema2020_12(t *testing.T) {
	t.Run("contentEncoding mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "string", ContentEncoding: "base64"}
		right := &parser.Schema{Type: "string", ContentEncoding: "quoted-printable"}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("contentMediaType mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "string", ContentMediaType: "application/json"}
		right := &parser.Schema{Type: "string", ContentMediaType: "text/plain"}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("contentSchema mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "string", ContentSchema: &parser.Schema{Type: "object"}}
		right := &parser.Schema{Type: "string", ContentSchema: &parser.Schema{Type: "array"}}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("contentSchema presence mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "string", ContentSchema: &parser.Schema{Type: "object"}}
		right := &parser.Schema{Type: "string"}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("content fields same", func(t *testing.T) {
		left := &parser.Schema{
			Type:             "string",
			ContentEncoding:  "base64",
			ContentMediaType: "application/json",
			ContentSchema:    &parser.Schema{Type: "object"},
		}
		right := &parser.Schema{
			Type:             "string",
			ContentEncoding:  "base64",
			ContentMediaType: "application/json",
			ContentSchema:    &parser.Schema{Type: "object"},
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.True(t, result.Equivalent)
	})
}

// TestCompareSchemas_PrefixItems tests prefixItems comparison
func TestCompareSchemas_PrefixItems(t *testing.T) {
	t.Run("prefixItems length mismatch", func(t *testing.T) {
		left := &parser.Schema{
			Type:        "array",
			PrefixItems: []*parser.Schema{{Type: "string"}},
		}
		right := &parser.Schema{
			Type:        "array",
			PrefixItems: []*parser.Schema{{Type: "string"}, {Type: "integer"}},
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("prefixItems content mismatch", func(t *testing.T) {
		left := &parser.Schema{
			Type:        "array",
			PrefixItems: []*parser.Schema{{Type: "string"}},
		}
		right := &parser.Schema{
			Type:        "array",
			PrefixItems: []*parser.Schema{{Type: "integer"}},
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("prefixItems same", func(t *testing.T) {
		left := &parser.Schema{
			Type:        "array",
			PrefixItems: []*parser.Schema{{Type: "string"}, {Type: "integer"}},
		}
		right := &parser.Schema{
			Type:        "array",
			PrefixItems: []*parser.Schema{{Type: "string"}, {Type: "integer"}},
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.True(t, result.Equivalent)
	})
}

// TestCompareSchemas_Contains tests contains comparison
func TestCompareSchemas_Contains(t *testing.T) {
	t.Run("contains presence mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "array", Contains: &parser.Schema{Type: "string"}}
		right := &parser.Schema{Type: "array"}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("contains content mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "array", Contains: &parser.Schema{Type: "string"}}
		right := &parser.Schema{Type: "array", Contains: &parser.Schema{Type: "integer"}}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("contains same", func(t *testing.T) {
		left := &parser.Schema{Type: "array", Contains: &parser.Schema{Type: "string"}}
		right := &parser.Schema{Type: "array", Contains: &parser.Schema{Type: "string"}}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.True(t, result.Equivalent)
	})
}

// TestCompareSchemas_PropertyNames tests propertyNames comparison
func TestCompareSchemas_PropertyNames(t *testing.T) {
	t.Run("propertyNames presence mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "object", PropertyNames: &parser.Schema{Pattern: "^[a-z]+$"}}
		right := &parser.Schema{Type: "object"}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("propertyNames content mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "object", PropertyNames: &parser.Schema{Pattern: "^[a-z]+$"}}
		right := &parser.Schema{Type: "object", PropertyNames: &parser.Schema{Pattern: "^[A-Z]+$"}}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("propertyNames same", func(t *testing.T) {
		left := &parser.Schema{Type: "object", PropertyNames: &parser.Schema{Pattern: "^[a-z]+$"}}
		right := &parser.Schema{Type: "object", PropertyNames: &parser.Schema{Pattern: "^[a-z]+$"}}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.True(t, result.Equivalent)
	})
}

// TestCompareSchemas_DependentSchemas tests dependentSchemas comparison
func TestCompareSchemas_DependentSchemas(t *testing.T) {
	t.Run("dependentSchemas keys mismatch", func(t *testing.T) {
		left := &parser.Schema{
			Type: "object",
			DependentSchemas: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
		}
		right := &parser.Schema{
			Type: "object",
			DependentSchemas: map[string]*parser.Schema{
				"email": {Type: "object"},
			},
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("dependentSchemas content mismatch", func(t *testing.T) {
		left := &parser.Schema{
			Type: "object",
			DependentSchemas: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
		}
		right := &parser.Schema{
			Type: "object",
			DependentSchemas: map[string]*parser.Schema{
				"name": {Type: "array"},
			},
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("dependentSchemas same", func(t *testing.T) {
		left := &parser.Schema{
			Type: "object",
			DependentSchemas: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
		}
		right := &parser.Schema{
			Type: "object",
			DependentSchemas: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.True(t, result.Equivalent)
	})
}

// TestCompareSchemas_UnevaluatedProperties tests unevaluatedProperties comparison
func TestCompareSchemas_UnevaluatedProperties(t *testing.T) {
	t.Run("unevaluatedProperties bool mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "object", UnevaluatedProperties: true}
		right := &parser.Schema{Type: "object", UnevaluatedProperties: false}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("unevaluatedProperties type mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "object", UnevaluatedProperties: true}
		right := &parser.Schema{Type: "object", UnevaluatedProperties: &parser.Schema{Type: "string"}}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("unevaluatedProperties presence mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "object", UnevaluatedProperties: false}
		right := &parser.Schema{Type: "object"}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("unevaluatedProperties schema mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "object", UnevaluatedProperties: &parser.Schema{Type: "string"}}
		right := &parser.Schema{Type: "object", UnevaluatedProperties: &parser.Schema{Type: "integer"}}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("unevaluatedProperties same bool", func(t *testing.T) {
		left := &parser.Schema{Type: "object", UnevaluatedProperties: false}
		right := &parser.Schema{Type: "object", UnevaluatedProperties: false}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.True(t, result.Equivalent)
	})

	t.Run("unevaluatedProperties same schema", func(t *testing.T) {
		left := &parser.Schema{Type: "object", UnevaluatedProperties: &parser.Schema{Type: "string"}}
		right := &parser.Schema{Type: "object", UnevaluatedProperties: &parser.Schema{Type: "string"}}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.True(t, result.Equivalent)
	})
}

// TestCompareSchemas_UnevaluatedItems tests unevaluatedItems comparison
func TestCompareSchemas_UnevaluatedItems(t *testing.T) {
	t.Run("unevaluatedItems bool mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "array", UnevaluatedItems: true}
		right := &parser.Schema{Type: "array", UnevaluatedItems: false}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("unevaluatedItems presence mismatch", func(t *testing.T) {
		left := &parser.Schema{Type: "array", UnevaluatedItems: true}
		right := &parser.Schema{Type: "array"}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent)
	})

	t.Run("unevaluatedItems same", func(t *testing.T) {
		left := &parser.Schema{Type: "array", UnevaluatedItems: &parser.Schema{Type: "string"}}
		right := &parser.Schema{Type: "array", UnevaluatedItems: &parser.Schema{Type: "string"}}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.True(t, result.Equivalent)
	})
}

func TestIsEmptySchema(t *testing.T) {
	minVal := 1.0
	minLen := 1
	minItems := 0

	tests := []struct {
		name   string
		schema *parser.Schema
		want   bool
	}{
		{
			name:   "nil schema",
			schema: nil,
			want:   false,
		},
		{
			name:   "truly empty schema",
			schema: &parser.Schema{},
			want:   true,
		},
		{
			name:   "with title only (metadata, not a constraint)",
			schema: &parser.Schema{Title: "My Schema"},
			want:   true,
		},
		{
			name:   "with description only (metadata, not a constraint)",
			schema: &parser.Schema{Description: "A description"},
			want:   true,
		},
		{
			name:   "with both title and description",
			schema: &parser.Schema{Title: "Title", Description: "Desc"},
			want:   true,
		},
		{
			name:   "with example (metadata, not a constraint)",
			schema: &parser.Schema{Example: "sample"},
			want:   true,
		},
		{
			name:   "with deprecated (metadata, not a constraint)",
			schema: &parser.Schema{Deprecated: true},
			want:   true,
		},
		{
			name:   "with type string",
			schema: &parser.Schema{Type: "string"},
			want:   false,
		},
		{
			name:   "with type object",
			schema: &parser.Schema{Type: "object"},
			want:   false,
		},
		{
			name:   "with type array ([]string)",
			schema: &parser.Schema{Type: []string{"string", "null"}},
			want:   false,
		},
		{
			name:   "with format",
			schema: &parser.Schema{Format: "date-time"},
			want:   false,
		},
		{
			name:   "with enum",
			schema: &parser.Schema{Enum: []any{"a", "b"}},
			want:   false,
		},
		{
			name:   "with const",
			schema: &parser.Schema{Const: "fixed"},
			want:   false,
		},
		{
			name:   "with pattern",
			schema: &parser.Schema{Pattern: "^[a-z]+$"},
			want:   false,
		},
		{
			name:   "with required",
			schema: &parser.Schema{Required: []string{"name"}},
			want:   false,
		},
		{
			name:   "with properties",
			schema: &parser.Schema{Properties: map[string]*parser.Schema{"name": {Type: "string"}}},
			want:   false,
		},
		{
			name:   "with additionalProperties bool",
			schema: &parser.Schema{AdditionalProperties: false},
			want:   false,
		},
		{
			name:   "with additionalProperties schema",
			schema: &parser.Schema{AdditionalProperties: &parser.Schema{Type: "string"}},
			want:   false,
		},
		{
			name:   "with items",
			schema: &parser.Schema{Items: &parser.Schema{Type: "string"}},
			want:   false,
		},
		{
			name:   "with minimum",
			schema: &parser.Schema{Minimum: &minVal},
			want:   false,
		},
		{
			name:   "with maximum",
			schema: &parser.Schema{Maximum: &minVal},
			want:   false,
		},
		{
			name:   "with minLength",
			schema: &parser.Schema{MinLength: &minLen},
			want:   false,
		},
		{
			name:   "with maxLength",
			schema: &parser.Schema{MaxLength: &minLen},
			want:   false,
		},
		{
			name:   "with minItems",
			schema: &parser.Schema{MinItems: &minItems},
			want:   false,
		},
		{
			name:   "with maxItems",
			schema: &parser.Schema{MaxItems: &minItems},
			want:   false,
		},
		{
			name:   "with uniqueItems",
			schema: &parser.Schema{UniqueItems: true},
			want:   false,
		},
		{
			name:   "with minProperties",
			schema: &parser.Schema{MinProperties: &minLen},
			want:   false,
		},
		{
			name:   "with maxProperties",
			schema: &parser.Schema{MaxProperties: &minLen},
			want:   false,
		},
		{
			name:   "with allOf",
			schema: &parser.Schema{AllOf: []*parser.Schema{{Type: "object"}}},
			want:   false,
		},
		{
			name:   "with anyOf",
			schema: &parser.Schema{AnyOf: []*parser.Schema{{Type: "string"}}},
			want:   false,
		},
		{
			name:   "with oneOf",
			schema: &parser.Schema{OneOf: []*parser.Schema{{Type: "string"}}},
			want:   false,
		},
		{
			name:   "with not",
			schema: &parser.Schema{Not: &parser.Schema{Type: "null"}},
			want:   false,
		},
		{
			name:   "with unevaluatedProperties",
			schema: &parser.Schema{UnevaluatedProperties: false},
			want:   false,
		},
		{
			name:   "with unevaluatedItems",
			schema: &parser.Schema{UnevaluatedItems: &parser.Schema{Type: "string"}},
			want:   false,
		},
		{
			name:   "with contentEncoding",
			schema: &parser.Schema{ContentEncoding: "base64"},
			want:   false,
		},
		{
			name:   "with contentMediaType",
			schema: &parser.Schema{ContentMediaType: "application/json"},
			want:   false,
		},
		{
			name:   "with contentSchema",
			schema: &parser.Schema{ContentSchema: &parser.Schema{Type: "object"}},
			want:   false,
		},
		{
			name:   "with prefixItems",
			schema: &parser.Schema{PrefixItems: []*parser.Schema{{Type: "string"}}},
			want:   false,
		},
		{
			name:   "with contains",
			schema: &parser.Schema{Contains: &parser.Schema{Type: "string"}},
			want:   false,
		},
		{
			name:   "with propertyNames",
			schema: &parser.Schema{PropertyNames: &parser.Schema{Pattern: "^[a-z]+$"}},
			want:   false,
		},
		{
			name:   "with dependentSchemas",
			schema: &parser.Schema{DependentSchemas: map[string]*parser.Schema{"name": {Type: "object"}}},
			want:   false,
		},
		{
			name:   "with multipleOf",
			schema: &parser.Schema{MultipleOf: &minVal},
			want:   false,
		},
		{
			name:   "with exclusiveMinimum",
			schema: &parser.Schema{ExclusiveMinimum: 1.0},
			want:   false,
		},
		{
			name:   "with exclusiveMaximum",
			schema: &parser.Schema{ExclusiveMaximum: 100.0},
			want:   false,
		},
		{
			name:   "with additionalItems",
			schema: &parser.Schema{AdditionalItems: &parser.Schema{Type: "string"}},
			want:   false,
		},
		{
			name:   "with maxContains",
			schema: &parser.Schema{MaxContains: &minLen},
			want:   false,
		},
		{
			name:   "with minContains",
			schema: &parser.Schema{MinContains: &minLen},
			want:   false,
		},
		{
			name:   "with patternProperties",
			schema: &parser.Schema{PatternProperties: map[string]*parser.Schema{"^x-": {Type: "string"}}},
			want:   false,
		},
		{
			name:   "with dependentRequired",
			schema: &parser.Schema{DependentRequired: map[string][]string{"name": {"email"}}},
			want:   false,
		},
		{
			name:   "with if",
			schema: &parser.Schema{If: &parser.Schema{Type: "object"}},
			want:   false,
		},
		{
			name:   "with then",
			schema: &parser.Schema{Then: &parser.Schema{Type: "object"}},
			want:   false,
		},
		{
			name:   "with else",
			schema: &parser.Schema{Else: &parser.Schema{Type: "object"}},
			want:   false,
		},
		{
			name:   "with nullable true",
			schema: &parser.Schema{Nullable: true},
			want:   false,
		},
		{
			name:   "with readOnly true",
			schema: &parser.Schema{ReadOnly: true},
			want:   false,
		},
		{
			name:   "with writeOnly true",
			schema: &parser.Schema{WriteOnly: true},
			want:   false,
		},
		{
			name:   "with collectionFormat",
			schema: &parser.Schema{CollectionFormat: "csv"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEmptySchema(tt.schema)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompareSchemas_EmptySchemasNonEquivalent(t *testing.T) {
	tests := []struct {
		name  string
		left  *parser.Schema
		right *parser.Schema
	}{
		{
			name:  "both empty",
			left:  &parser.Schema{},
			right: &parser.Schema{},
		},
		{
			name:  "left empty right has title",
			left:  &parser.Schema{},
			right: &parser.Schema{Title: "Any"},
		},
		{
			name:  "left has description right empty",
			left:  &parser.Schema{Description: "placeholder"},
			right: &parser.Schema{},
		},
		{
			name:  "both have metadata but no constraints",
			left:  &parser.Schema{Title: "Schema A"},
			right: &parser.Schema{Title: "Schema B"},
		},
		{
			name:  "left empty right has type",
			left:  &parser.Schema{},
			right: &parser.Schema{Type: "string"},
		},
		{
			name:  "left has type right empty",
			left:  &parser.Schema{Type: "object"},
			right: &parser.Schema{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareSchemas(tt.left, tt.right, EquivalenceModeDeep)
			assert.False(t, result.Equivalent, "empty schemas should never be equivalent")
			assert.Empty(t, result.Differences, "no structural differences for empty schema short-circuit")
		})
	}
}

func TestCompareSchemas_EmptySchemaShallowMode(t *testing.T) {
	left := &parser.Schema{}
	right := &parser.Schema{}

	result := CompareSchemas(left, right, EquivalenceModeShallow)

	assert.False(t, result.Equivalent, "empty schemas should not be equivalent in shallow mode")
	assert.Empty(t, result.Differences)
}

func TestEquivalenceResult_String(t *testing.T) {
	tests := []struct {
		name     string
		result   EquivalenceResult
		contains string
	}{
		{
			name:     "equivalent",
			result:   EquivalenceResult{Equivalent: true},
			contains: "Schemas are equivalent",
		},
		{
			name: "non-equivalent with no differences (empty schema)",
			result: EquivalenceResult{
				Equivalent:  false,
				Differences: []SchemaDifference{},
			},
			contains: "empty schemas are semantically distinct",
		},
		{
			name: "non-equivalent with differences",
			result: EquivalenceResult{
				Equivalent: false,
				Differences: []SchemaDifference{
					{Path: "type", Description: "type mismatch"},
					{Path: "format", Description: "format mismatch"},
				},
			},
			contains: "Schemas differ:",
		},
		{
			name: "non-equivalent with nil differences (from EquivalenceModeNone)",
			result: EquivalenceResult{
				Equivalent:  false,
				Differences: nil,
			},
			contains: "Schemas differ:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.String()
			assert.Contains(t, got, tt.contains)
		})
	}
}

func TestCompareSchemas_EmptySchemasInCompositions(t *testing.T) {
	t.Run("empty schema in allOf", func(t *testing.T) {
		left := &parser.Schema{
			AllOf: []*parser.Schema{
				{}, // empty schema
				{Type: "object"},
			},
		}
		right := &parser.Schema{
			AllOf: []*parser.Schema{
				{Type: "string"}, // non-empty schema
				{Type: "object"},
			},
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent, "schemas with different allOf children should not be equivalent")
	})

	t.Run("empty schema in anyOf", func(t *testing.T) {
		left := &parser.Schema{
			AnyOf: []*parser.Schema{
				{}, // empty schema
				{Type: "integer"},
			},
		}
		right := &parser.Schema{
			AnyOf: []*parser.Schema{
				{Type: "string"}, // non-empty schema
				{Type: "integer"},
			},
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent, "schemas with different anyOf children should not be equivalent")
	})

	t.Run("empty schema as items", func(t *testing.T) {
		left := &parser.Schema{
			Type:  "array",
			Items: &parser.Schema{}, // empty schema as items
		}
		right := &parser.Schema{
			Type:  "array",
			Items: &parser.Schema{Type: "string"}, // non-empty items
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent, "schemas with different items (empty vs non-empty) should not be equivalent")
	})

	t.Run("parent schemas with different empty children are non-equivalent", func(t *testing.T) {
		// Both parents have allOf with empty schemas in different positions
		left := &parser.Schema{
			AllOf: []*parser.Schema{
				{}, // empty
				{Type: "object", Properties: map[string]*parser.Schema{"name": {Type: "string"}}},
			},
		}
		right := &parser.Schema{
			AllOf: []*parser.Schema{
				{Type: "object", Properties: map[string]*parser.Schema{"name": {Type: "string"}}},
				{}, // empty in different position
			},
		}

		result := CompareSchemas(left, right, EquivalenceModeDeep)
		assert.False(t, result.Equivalent, "schemas with empty children in different positions should not be equivalent")
	})
}

func TestEquivalenceResult_String_WithDifferences(t *testing.T) {
	result := EquivalenceResult{
		Equivalent: false,
		Differences: []SchemaDifference{
			{Path: "type", Description: "type mismatch"},
		},
	}

	s := result.String()
	assert.Contains(t, s, "type")
	assert.Contains(t, s, "type mismatch")
}

// TestCompareSchemas_XMLMismatch covers the XML Object, which deep comparison
// ignored entirely.
//
// XML decides the element name, namespace, and node kind a value serializes to, so
// two schemas whose XML differs describe different payloads. Reporting them as
// equivalent let semantic deduplication merge them and silently change the wire
// format — and the structural hash omitted XML too, so the two gaps hid each
// other: such schemas rarely reached this comparison at all.
func TestCompareSchemas_XMLMismatch(t *testing.T) {
	tests := []struct {
		name        string
		left, right *parser.XML
	}{
		{"nodeType differs", &parser.XML{NodeType: "attribute"}, &parser.XML{NodeType: "text"}},
		{"xml present vs absent", &parser.XML{Name: "tag"}, nil},
		{"name differs", &parser.XML{Name: "tag"}, &parser.XML{Name: "label"}},
		{"namespace differs", &parser.XML{Namespace: "https://a.example"}, &parser.XML{Namespace: "https://b.example"}},
		{"prefix differs", &parser.XML{Prefix: "a"}, &parser.XML{Prefix: "b"}},
		{"attribute differs", &parser.XML{Attribute: true}, &parser.XML{}},
		{"wrapped differs", &parser.XML{Wrapped: true}, &parser.XML{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := &parser.Schema{Type: "string", XML: tt.left}
			right := &parser.Schema{Type: "string", XML: tt.right}

			result := CompareSchemas(left, right, EquivalenceModeDeep)

			assert.False(t, result.Equivalent)
			require.Len(t, result.Differences, 1)
			assert.Equal(t, "xml", result.Differences[0].Path)
			assert.Equal(t, "xml metadata mismatch", result.Differences[0].Description)
		})
	}
}

// TestCompareSchemas_XMLEquivalent is the control: identical XML must not be made
// to differ, or deduplication stops consolidating schemas it always could.
func TestCompareSchemas_XMLEquivalent(t *testing.T) {
	xml := func() *parser.XML {
		return &parser.XML{Name: "tag", Namespace: "https://example.com/ns", NodeType: "attribute"}
	}
	left := &parser.Schema{Type: "string", XML: xml()}
	right := &parser.Schema{Type: "string", XML: xml()}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	assert.True(t, result.Equivalent, "identical xml must compare equal")
	assert.Empty(t, result.Differences)

	t.Run("both absent", func(t *testing.T) {
		plainLeft := &parser.Schema{Type: "string"}
		plainRight := &parser.Schema{Type: "string"}
		assert.True(t, CompareSchemas(plainLeft, plainRight, EquivalenceModeDeep).Equivalent)
	})
}

// TestCompareSchemas_NilNestedSchemas covers the nil values a schema map or a
// schema-or-bool field can legitimately hold.
//
// compareDeep dereferences both operands with no nil guard of its own, so every
// nested walk has to keep nils away from it. `patternProperties: {"^a": }` parses
// to a nil entry, and a caller can store a typed nil `(*parser.Schema)(nil)` in
// additionalItems, which passes an interface nil check and asserts cleanly.
// Issues #416 and #417 have the rest.
//
// Cases are one-directional except for properties, which is checked both ways
// because compareSchemaMaps ranges over the left map alone. The guard itself is
// symmetric, so the remaining fields would report the same verdict at the same
// path reversed.
func TestCompareSchemas_NilNestedSchemas(t *testing.T) {
	base := func() map[string]*parser.Schema {
		return map[string]*parser.Schema{"p": {Type: "string"}}
	}

	tests := []struct {
		name        string
		left, right *parser.Schema
		equivalent  bool
		wantPath    string
	}{
		{
			name:       "nil patternProperties entry against a present one",
			left:       &parser.Schema{Type: "object", Properties: base(), PatternProperties: map[string]*parser.Schema{"^a": nil}},
			right:      &parser.Schema{Type: "object", Properties: base(), PatternProperties: map[string]*parser.Schema{"^a": {Type: "string"}}},
			equivalent: false,
			wantPath:   "patternProperties.^a",
		},
		{
			name:       "nil $defs entry against a present one",
			left:       &parser.Schema{Type: "object", Properties: base(), Defs: map[string]*parser.Schema{"A": nil}},
			right:      &parser.Schema{Type: "object", Properties: base(), Defs: map[string]*parser.Schema{"A": {Type: "string"}}},
			equivalent: false,
			wantPath:   "$defs.A",
		},
		{
			name:       "both entries nil",
			left:       &parser.Schema{Type: "object", Properties: base(), Defs: map[string]*parser.Schema{"A": nil}},
			right:      &parser.Schema{Type: "object", Properties: base(), Defs: map[string]*parser.Schema{"A": nil}},
			equivalent: true,
		},
		{
			name: "nil properties entry against a present one",
			left: &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{
				"p": {Type: "string"}, "name": nil}},
			right: &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{
				"p": {Type: "string"}, "name": {Type: "string"}}},
			equivalent: false,
			wantPath:   "properties.name",
		},
		{
			name: "present properties entry against a nil one",
			left: &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{
				"p": {Type: "string"}, "name": {Type: "string"}}},
			right: &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{
				"p": {Type: "string"}, "name": nil}},
			equivalent: false,
			wantPath:   "properties.name",
		},
		{
			name:       "nil dependentSchemas entry against a present one",
			left:       &parser.Schema{Type: "object", Properties: base(), DependentSchemas: map[string]*parser.Schema{"a": nil}},
			right:      &parser.Schema{Type: "object", Properties: base(), DependentSchemas: map[string]*parser.Schema{"a": {Type: "string"}}},
			equivalent: false,
			wantPath:   "dependentSchemas.a",
		},
		{
			name:       "nil allOf element against a present one",
			left:       &parser.Schema{Type: "object", Properties: base(), AllOf: []*parser.Schema{nil}},
			right:      &parser.Schema{Type: "object", Properties: base(), AllOf: []*parser.Schema{{Type: "string"}}},
			equivalent: false,
			wantPath:   "allOf.[0]",
		},
		{
			name:       "nil anyOf element against a present one",
			left:       &parser.Schema{Type: "object", Properties: base(), AnyOf: []*parser.Schema{nil}},
			right:      &parser.Schema{Type: "object", Properties: base(), AnyOf: []*parser.Schema{{Type: "string"}}},
			equivalent: false,
			wantPath:   "anyOf.[0]",
		},
		{
			name:       "nil oneOf element against a present one",
			left:       &parser.Schema{Type: "object", Properties: base(), OneOf: []*parser.Schema{nil}},
			right:      &parser.Schema{Type: "object", Properties: base(), OneOf: []*parser.Schema{{Type: "string"}}},
			equivalent: false,
			wantPath:   "oneOf.[0]",
		},
		{
			name:       "nil prefixItems element against a present one",
			left:       &parser.Schema{Type: "array", Properties: base(), PrefixItems: []*parser.Schema{nil}},
			right:      &parser.Schema{Type: "array", Properties: base(), PrefixItems: []*parser.Schema{{Type: "string"}}},
			equivalent: false,
			wantPath:   "prefixItems.[0]",
		},
		{
			name:       "both allOf elements nil",
			left:       &parser.Schema{Type: "object", Properties: base(), AllOf: []*parser.Schema{nil}},
			right:      &parser.Schema{Type: "object", Properties: base(), AllOf: []*parser.Schema{nil}},
			equivalent: true,
		},
		{
			name:       "typed-nil additionalItems against a present schema",
			left:       &parser.Schema{Type: "array", Properties: base(), AdditionalItems: (*parser.Schema)(nil)},
			right:      &parser.Schema{Type: "array", Properties: base(), AdditionalItems: &parser.Schema{Type: "string"}},
			equivalent: false,
			wantPath:   "additionalItems",
		},
		{
			name:       "typed-nil additionalProperties against a present schema",
			left:       &parser.Schema{Type: "object", Properties: base(), AdditionalProperties: (*parser.Schema)(nil)},
			right:      &parser.Schema{Type: "object", Properties: base(), AdditionalProperties: &parser.Schema{Type: "string"}},
			equivalent: false,
			wantPath:   "additionalProperties",
		},
		{
			name:       "both additionalItems typed-nil",
			left:       &parser.Schema{Type: "array", Properties: base(), AdditionalItems: (*parser.Schema)(nil)},
			right:      &parser.Schema{Type: "array", Properties: base(), AdditionalItems: (*parser.Schema)(nil)},
			equivalent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareSchemas(tt.left, tt.right, EquivalenceModeDeep)

			assert.Equal(t, tt.equivalent, result.Equivalent)
			if tt.wantPath == "" {
				return
			}
			var paths []string
			for _, d := range result.Differences {
				paths = append(paths, d.Path)
			}
			assert.Contains(t, paths, tt.wantPath,
				"the difference should be reported at the field it belongs to")
		})
	}
}

// TestCompareSchemas_AdditionalItemsNamesItsOwnKeyword pins the keyword a
// schema-or-bool difference is reported under.
//
// additionalItems and additionalProperties share a comparison, which used to
// hardcode the additionalProperties path, so an item difference was labelled as
// an object one. The comparison was right and only the diagnostic was wrong,
// which is why no equivalence assertion caught it.
func TestCompareSchemas_AdditionalItemsNamesItsOwnKeyword(t *testing.T) {
	props := map[string]*parser.Schema{"p": {Type: "string"}}
	left := &parser.Schema{Type: "array", Properties: props, AdditionalItems: &parser.Schema{Type: "string"}}
	right := &parser.Schema{Type: "array", Properties: props, AdditionalItems: &parser.Schema{Type: "integer"}}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	require.False(t, result.Equivalent)
	require.NotEmpty(t, result.Differences)
	for _, d := range result.Differences {
		assert.True(t, strings.HasPrefix(d.Path, "additionalItems"),
			"expected the difference under additionalItems, got %q", d.Path)
	}
}

// TestCompareSchemas_SchemaOrBoolMismatch covers the remaining shapes a
// schema-or-bool field can take, since additionalItems and additionalProperties
// accept either and a document may change which.
func TestCompareSchemas_SchemaOrBoolMismatch(t *testing.T) {
	props := map[string]*parser.Schema{"p": {Type: "string"}}

	t.Run("boolean values differ", func(t *testing.T) {
		left := &parser.Schema{Type: "object", Properties: props, AdditionalProperties: true}
		right := &parser.Schema{Type: "object", Properties: props, AdditionalProperties: false}
		assert.False(t, CompareSchemas(left, right, EquivalenceModeDeep).Equivalent)
	})

	t.Run("a schema against a boolean", func(t *testing.T) {
		left := &parser.Schema{Type: "object", Properties: props, AdditionalProperties: &parser.Schema{Type: "string"}}
		right := &parser.Schema{Type: "object", Properties: props, AdditionalProperties: false}
		assert.False(t, CompareSchemas(left, right, EquivalenceModeDeep).Equivalent)
	})

	t.Run("present against absent", func(t *testing.T) {
		left := &parser.Schema{Type: "object", Properties: props, AdditionalProperties: true}
		right := &parser.Schema{Type: "object", Properties: props}
		assert.False(t, CompareSchemas(left, right, EquivalenceModeDeep).Equivalent)
	})
}

// TestCompareSchemas_NilAndEmptyMapsAgreeWithParser pins deep comparison to the
// convention parser's equality already documents: "Nil and empty maps are
// considered equal."
//
// reflect.DeepEqual splits those two, so using it for a typed map made this
// package disagree with parser about the same pair of schemas. The last case
// covers why the fix is maps.EqualFunc rather than maps.Equal for Extra:
// specification extensions hold arbitrary JSON, and == on a slice value panics.
func TestCompareSchemas_NilAndEmptyMapsAgreeWithParser(t *testing.T) {
	base := func() map[string]*parser.Schema {
		return map[string]*parser.Schema{"p": {Type: "string"}}
	}

	tests := []struct {
		name        string
		left, right *parser.Schema
	}{
		{
			name:  "$vocabulary absent against declared empty",
			left:  &parser.Schema{Type: "object", Properties: base()},
			right: &parser.Schema{Type: "object", Properties: base(), Vocabulary: map[string]bool{}},
		},
		{
			name: "discriminator mapping absent against declared empty",
			left: &parser.Schema{Type: "object", Properties: base(),
				Discriminator: &parser.Discriminator{PropertyName: "kind"}},
			right: &parser.Schema{Type: "object", Properties: base(),
				Discriminator: &parser.Discriminator{PropertyName: "kind", Mapping: map[string]string{}}},
		},
		{
			name: "dependentRequired absent against declared empty",
			left: &parser.Schema{Type: "object", Properties: base()},
			right: &parser.Schema{Type: "object", Properties: base(),
				DependentRequired: map[string][]string{}},
		},
		{
			name: "externalDocs extensions holding an uncomparable value",
			left: &parser.Schema{Type: "object", Properties: base(),
				ExternalDocs: &parser.ExternalDocs{URL: "https://example.com", Extra: map[string]any{"x-a": []any{1, 2}}}},
			right: &parser.Schema{Type: "object", Properties: base(),
				ExternalDocs: &parser.ExternalDocs{URL: "https://example.com", Extra: map[string]any{"x-a": []any{1, 2}}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, CompareSchemas(tt.left, tt.right, EquivalenceModeDeep).Equivalent,
				"deep comparison should treat these as equivalent")
			assert.True(t, tt.left.Equals(tt.right),
				"parser agrees, and the two must not diverge")
		})
	}
}
