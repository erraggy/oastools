package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, 1, len(result.Differences))
	assert.Equal(t, "type", result.Differences[0].Path)
	assert.Equal(t, "type mismatch", result.Differences[0].Description)
}

func TestCompareSchemas_FormatMismatch(t *testing.T) {
	left := &parser.Schema{Type: "string", Format: "date"}
	right := &parser.Schema{Type: "string", Format: "date-time"}

	result := CompareSchemas(left, right, EquivalenceModeShallow)

	assert.False(t, result.Equivalent)
	assert.Equal(t, 1, len(result.Differences))
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
	assert.Equal(t, 1, len(result.Differences))
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
	assert.Equal(t, 1, len(result.Differences))
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
	assert.Equal(t, 1, len(result.Differences))
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
	assert.Equal(t, 1, len(result.Differences))
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
	assert.Equal(t, 1, len(result.Differences))
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

func TestCompareSchemas_IgnoresMetadata(t *testing.T) {
	left := &parser.Schema{
		Type:        "string",
		Title:       "User Name",
		Description: "The name of the user",
		Example:     "John Doe",
		Deprecated:  false,
	}
	right := &parser.Schema{
		Type:        "string",
		Title:       "Full Name",
		Description: "User's full name",
		Example:     "Jane Smith",
		Deprecated:  true,
	}

	result := CompareSchemas(left, right, EquivalenceModeDeep)

	// Should be equivalent because metadata is ignored
	assert.True(t, result.Equivalent, "schemas should be equivalent when only metadata differs")
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

	assert.True(t, result.Equivalent, "empty schemas should be equivalent")
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
