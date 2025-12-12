package generator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

func TestToTypeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pet", "Pet"},
		{"Pet", "Pet"},
		{"pet-store", "PetStore"},
		{"pet_store", "PetStore"},
		{"pet.store", "PetStore"},
		{"PetStore", "PetStore"},
		{"123abc", "T123abc"},
		{"", "Type"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toTypeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToParamName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"petId", "petId"},
		{"PetId", "petId"},
		{"pet-id", "petId"},
		{"pet_id", "petId"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toParamName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToFieldName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"petId", "PetId"},
		{"pet_id", "PetId"},
		{"pet-id", "PetId"},
		{"PET_ID", "PETID"},
		{"break", "Break_"},
		{"pet", "Pet"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toFieldName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOperationToMethodName(t *testing.T) {
	tests := []struct {
		op       *parser.Operation
		path     string
		method   string
		expected string
	}{
		{&parser.Operation{OperationID: "listPets"}, "/pets", "get", "ListPets"},
		{&parser.Operation{OperationID: "get-pet-by-id"}, "/pets/{id}", "get", "GetPetById"},
		{&parser.Operation{}, "/pets", "get", "GetPets"},
		{&parser.Operation{}, "/pets/{petId}", "get", "GetPetsByPetId"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := operationToMethodName(tt.op, tt.path, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringFormatToGoType(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"date-time", "time.Time"},
		{"date", "string"},
		{"byte", "[]byte"},
		{"binary", "[]byte"},
		{"", "string"},
		{"unknown", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := stringFormatToGoType(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntegerFormatToGoType(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"int32", "int32"},
		{"int64", "int64"},
		{"", "int64"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := integerFormatToGoType(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNumberFormatToGoType(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"float", "float32"},
		{"double", "float64"},
		{"", "float64"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := numberFormatToGoType(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSchemaType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected string
	}{
		{"nil schema", nil, ""},
		{"string type", &parser.Schema{Type: "string"}, "string"},
		{"object type", &parser.Schema{Type: "object"}, "object"},
		{"array type", &parser.Schema{Type: "array"}, "array"},
		{"properties infer object", &parser.Schema{Properties: map[string]*parser.Schema{}}, "object"},
		{"items infer array", &parser.Schema{Items: &parser.Schema{}}, "array"},
		{"enum infer string", &parser.Schema{Enum: []any{"a", "b"}}, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSchemaType(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRequired(t *testing.T) {
	required := []string{"id", "name", "email"}

	assert.True(t, isRequired(required, "id"))
	assert.True(t, isRequired(required, "name"))
	assert.False(t, isRequired(required, "optional"))
	assert.False(t, isRequired(nil, "any"))
}

func TestCleanDescription(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple description", "Simple description"},
		{"Multi\nline\ndescription", "Multi line description"},
		{"  Whitespace  ", "Whitespace"},
		{strings.Repeat("a", 300), strings.Repeat("a", 197) + "..."},
	}

	for _, tt := range tests {
		name := tt.input
		if len(name) > 10 {
			name = name[:10]
		}
		t.Run(name, func(t *testing.T) {
			result := cleanDescription(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestZeroValue(t *testing.T) {
	tests := []struct {
		typeName string
		expected string
	}{
		{"", "nil"},
		{"*http.Response", "nil"},
		{"*Pet", "nil"},
		{"[]Pet", "nil"},
		{"map[string]Pet", "nil"},
		{"Pet", "Pet{}"},
		{"string", "string{}"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result := zeroValue(tt.typeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Note: isTypeNullable tests moved to internal/schemautil/type_test.go as IsNullable

func TestNeedsTimeImport(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected bool
	}{
		{"nil schema", nil, false},
		{"date-time format", &parser.Schema{Type: "string", Format: "date-time"}, true},
		{"no format", &parser.Schema{Type: "string"}, false},
		{"nested date-time", &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"created": {Type: "string", Format: "date-time"},
			},
		}, true},
		{"array with date-time items", &parser.Schema{
			Type:  "array",
			Items: &parser.Schema{Type: "string", Format: "date-time"},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsTimeImport(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeReservedWord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"break", "break_"},
		{"type", "type_"},
		{"Package", "Package_"},
		{"Error", "Error"},
		{"func", "func_"},
		{"interface", "interface_"},
		{"pet", "pet"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeReservedWord(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaTypeFromMap(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		expected string
	}{
		{
			name:     "string type",
			schema:   map[string]interface{}{"type": "string"},
			expected: "string",
		},
		{
			name:     "number type",
			schema:   map[string]interface{}{"type": "number"},
			expected: "float64",
		},
		{
			name:     "integer type",
			schema:   map[string]interface{}{"type": "integer"},
			expected: "int64",
		},
		{
			name:     "boolean type",
			schema:   map[string]interface{}{"type": "boolean"},
			expected: "bool",
		},
		{
			name:     "object type",
			schema:   map[string]interface{}{"type": "object"},
			expected: "map[string]any",
		},
		{
			name:     "array type",
			schema:   map[string]interface{}{"type": "array"},
			expected: "[]any",
		},
		{
			name:     "missing type",
			schema:   map[string]interface{}{},
			expected: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := schemaTypeFromMap(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildDefaultUserAgent tests the buildDefaultUserAgent helper function.
// This test covers empty/nil title cases which cannot be tested via integration
// tests (e.g., GenerateWithOptions) because the parser rejects specs with empty
// info.title as invalid before reaching user agent generation.
func TestBuildDefaultUserAgent(t *testing.T) {
	tests := []struct {
		name     string
		info     *parser.Info
		expected string
	}{
		{
			name:     "with title",
			info:     &parser.Info{Title: "PetStore"},
			expected: "oastools/" + oastools.Version() + "/generated/PetStore",
		},
		{
			name:     "with complex title",
			info:     &parser.Info{Title: "My Complex API"},
			expected: "oastools/" + oastools.Version() + "/generated/My Complex API",
		},
		{
			name:     "with empty title",
			info:     &parser.Info{Title: ""},
			expected: "oastools/" + oastools.Version() + "/generated/API Client",
		},
		{
			name:     "with nil info",
			info:     nil,
			expected: "oastools/" + oastools.Version() + "/generated/API Client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDefaultUserAgent(tt.info)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSelfReference(t *testing.T) {
	tests := []struct {
		name           string
		propSchema     *parser.Schema
		parentTypeName string
		expected       bool
	}{
		{
			name:           "nil schema",
			propSchema:     nil,
			parentTypeName: "User",
			expected:       false,
		},
		{
			name:           "no ref",
			propSchema:     &parser.Schema{Type: "string"},
			parentTypeName: "User",
			expected:       false,
		},
		{
			name:           "self reference OAS3",
			propSchema:     &parser.Schema{Ref: "#/components/schemas/User"},
			parentTypeName: "User",
			expected:       true,
		},
		{
			name:           "self reference OAS2",
			propSchema:     &parser.Schema{Ref: "#/definitions/User"},
			parentTypeName: "User",
			expected:       true,
		},
		{
			name:           "different reference",
			propSchema:     &parser.Schema{Ref: "#/components/schemas/Pet"},
			parentTypeName: "User",
			expected:       false,
		},
		{
			name:           "case sensitive - different case",
			propSchema:     &parser.Schema{Ref: "#/components/schemas/user"},
			parentTypeName: "User",
			expected:       true, // toTypeName normalizes to same name
		},
		{
			name:           "underscore naming",
			propSchema:     &parser.Schema{Ref: "#/components/schemas/user_group"},
			parentTypeName: "UserGroup",
			expected:       true, // toTypeName("user_group") == "UserGroup"
		},
		{
			name: "allOf self reference",
			propSchema: &parser.Schema{
				AllOf: []*parser.Schema{
					{Ref: "#/components/schemas/User"},
				},
			},
			parentTypeName: "User",
			expected:       true,
		},
		{
			name: "allOf no self reference",
			propSchema: &parser.Schema{
				AllOf: []*parser.Schema{
					{Ref: "#/components/schemas/Pet"},
				},
			},
			parentTypeName: "User",
			expected:       false,
		},
		{
			name: "nested allOf self reference",
			propSchema: &parser.Schema{
				AllOf: []*parser.Schema{
					{
						AllOf: []*parser.Schema{
							{Ref: "#/components/schemas/TreeNode"},
						},
					},
				},
			},
			parentTypeName: "TreeNode",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSelfReference(tt.propSchema, tt.parentTypeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatAndFixImports(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		contains []string
	}{
		{
			name: "adds missing import",
			input: `package test

func foo() {
	fmt.Println("hello")
}
`,
			wantErr:  false,
			contains: []string{`"fmt"`},
		},
		{
			name: "removes unused import",
			input: `package test

import "fmt"
import "strings"

func foo() {
	fmt.Println("hello")
}
`,
			wantErr:  false,
			contains: []string{`"fmt"`},
		},
		{
			name: "invalid Go code",
			input: `package test

func foo( {
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatAndFixImports("test.go", []byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			for _, s := range tt.contains {
				assert.Contains(t, string(result), s)
			}
		})
	}

	// Special test: verify unused import is removed
	t.Run("unused import removed", func(t *testing.T) {
		input := `package test

import "fmt"
import "strings"

func foo() {
	fmt.Println("hello")
}
`
		result, err := formatAndFixImports("test.go", []byte(input))
		assert.NoError(t, err)
		assert.NotContains(t, string(result), `"strings"`)
	})
}
