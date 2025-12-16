package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Generic Names Utility Tests
// =============================================================================

// TestHasInvalidSchemaNameChars tests detection of invalid characters in schema names
func TestHasInvalidSchemaNameChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid names
		{name: "simple name", input: "User", expected: false},
		{name: "underscore name", input: "User_Profile", expected: false},
		{name: "hyphen name", input: "user-profile", expected: false},
		{name: "dot name", input: "User.Profile", expected: false},
		{name: "numeric suffix", input: "User123", expected: false},
		{name: "PascalCase", input: "UserProfileData", expected: false},
		{name: "camelCase", input: "userProfileData", expected: false},

		// Invalid names (generic type syntax)
		{name: "square brackets", input: "Response[User]", expected: true},
		{name: "angle brackets", input: "List<Item>", expected: true},
		{name: "nested brackets", input: "Response[List[User]]", expected: true},
		{name: "comma separated", input: "Map[string,int]", expected: true},

		// Invalid names (other special chars)
		{name: "space in name", input: "User Profile", expected: true},
		{name: "curly braces", input: "User{data}", expected: true},
		{name: "pipe character", input: "User|Admin", expected: true},
		{name: "backslash", input: "User\\Admin", expected: true},
		{name: "caret", input: "User^2", expected: true},
		{name: "backtick", input: "User`s", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasInvalidSchemaNameChars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsGenericStyleName tests detection of generic type names
func TestIsGenericStyleName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Generic style names
		{name: "square brackets", input: "Response[User]", expected: true},
		{name: "angle brackets", input: "List<Item>", expected: true},
		{name: "nested square", input: "Response[List[User]]", expected: true},
		{name: "nested angle", input: "Map<String, List<Int>>", expected: true},
		{name: "mixed brackets", input: "Response<List[User]>", expected: true},

		// Non-generic names
		{name: "simple name", input: "User", expected: false},
		{name: "underscore name", input: "User_Response", expected: false},
		{name: "hyphen name", input: "user-response", expected: false},
		{name: "dot name", input: "user.response", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGenericStyleName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseGenericName tests parsing generic names into base and parameters
func TestParseGenericName(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedBase   string
		expectedParams []string
		expectedStyle  rune
	}{
		// Square bracket generics
		{
			name:           "simple generic",
			input:          "Response[User]",
			expectedBase:   "Response",
			expectedParams: []string{"User"},
			expectedStyle:  '[',
		},
		{
			name:           "multiple params",
			input:          "Map[string,int]",
			expectedBase:   "Map",
			expectedParams: []string{"string", "int"},
			expectedStyle:  '[',
		},
		{
			name:           "nested generic",
			input:          "Response[List[User]]",
			expectedBase:   "Response",
			expectedParams: []string{"List[User]"},
			expectedStyle:  '[',
		},
		{
			name:           "deeply nested",
			input:          "Outer[Middle[Inner]]",
			expectedBase:   "Outer",
			expectedParams: []string{"Middle[Inner]"},
			expectedStyle:  '[',
		},
		{
			name:           "multiple nested params",
			input:          "Map[List[K],List[V]]",
			expectedBase:   "Map",
			expectedParams: []string{"List[K]", "List[V]"},
			expectedStyle:  '[',
		},

		// Angle bracket generics
		{
			name:           "angle bracket simple",
			input:          "List<Item>",
			expectedBase:   "List",
			expectedParams: []string{"Item"},
			expectedStyle:  '<',
		},
		{
			name:           "angle bracket multiple",
			input:          "Map<K,V>",
			expectedBase:   "Map",
			expectedParams: []string{"K", "V"},
			expectedStyle:  '<',
		},

		// Non-generic names
		{
			name:           "plain name",
			input:          "UserProfile",
			expectedBase:   "UserProfile",
			expectedParams: nil,
			expectedStyle:  0,
		},
		{
			name:           "underscore name",
			input:          "User_Profile",
			expectedBase:   "User_Profile",
			expectedParams: nil,
			expectedStyle:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, params, style := parseGenericName(tt.input)
			assert.Equal(t, tt.expectedBase, base)
			assert.Equal(t, tt.expectedParams, params)
			assert.Equal(t, tt.expectedStyle, style)
		})
	}
}

// TestSplitTypeParams tests splitting type parameters
func TestSplitTypeParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single param",
			input:    "User",
			expected: []string{"User"},
		},
		{
			name:     "two params",
			input:    "string,int",
			expected: []string{"string", "int"},
		},
		{
			name:     "three params",
			input:    "A,B,C",
			expected: []string{"A", "B", "C"},
		},
		{
			name:     "nested bracket",
			input:    "List[User],int",
			expected: []string{"List[User]", "int"},
		},
		{
			name:     "multiple nested",
			input:    "Map[K,V],List[T]",
			expected: []string{"Map[K,V]", "List[T]"},
		},
		{
			name:     "deeply nested",
			input:    "A[B[C]],D",
			expected: []string{"A[B[C]]", "D"},
		},
		{
			name:     "with spaces",
			input:    " User , Item ",
			expected: []string{"User", "Item"},
		},
		{
			name:     "angle brackets nested",
			input:    "List<User>,Map<K,V>",
			expected: []string{"List<User>", "Map<K,V>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitTypeParams(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTransformSchemaName tests name transformation with all strategies
func TestTransformSchemaName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		config   GenericNamingConfig
		expected string
	}{
		// Underscore strategy
		{
			name:  "underscore simple",
			input: "Response[User]",
			config: GenericNamingConfig{
				Strategy:  GenericNamingUnderscore,
				Separator: "_",
			},
			expected: "Response_User_",
		},
		{
			name:  "underscore multiple params",
			input: "Map[string,int]",
			config: GenericNamingConfig{
				Strategy:       GenericNamingUnderscore,
				Separator:      "_",
				ParamSeparator: "_",
			},
			expected: "Map_String_Int_",
		},
		{
			name:  "underscore nested",
			input: "Response[List[User]]",
			config: GenericNamingConfig{
				Strategy:  GenericNamingUnderscore,
				Separator: "_",
			},
			// List[User] transforms to "List_User_", then PascalCase makes it "ListUser"
			expected: "Response_ListUser_",
		},

		// Of strategy
		{
			name:  "of simple",
			input: "Response[User]",
			config: GenericNamingConfig{
				Strategy: GenericNamingOf,
			},
			expected: "ResponseOfUser",
		},
		{
			name:  "of multiple params",
			input: "Map[string,int]",
			config: GenericNamingConfig{
				Strategy:       GenericNamingOf,
				ParamSeparator: "_",
			},
			expected: "MapOfString_OfInt",
		},
		{
			name:  "of nested",
			input: "Response[List[User]]",
			config: GenericNamingConfig{
				Strategy: GenericNamingOf,
			},
			expected: "ResponseOfListOfUser",
		},

		// For strategy
		{
			name:  "for simple",
			input: "Handler[Request]",
			config: GenericNamingConfig{
				Strategy: GenericNamingFor,
			},
			expected: "HandlerForRequest",
		},
		{
			name:  "for multiple",
			input: "Mapper[Input,Output]",
			config: GenericNamingConfig{
				Strategy:       GenericNamingFor,
				ParamSeparator: "_",
			},
			expected: "MapperForInput_ForOutput",
		},

		// Flattened strategy
		{
			name:  "flattened simple",
			input: "Response[User]",
			config: GenericNamingConfig{
				Strategy: GenericNamingFlattened,
			},
			expected: "ResponseUser",
		},
		{
			name:  "flattened multiple",
			input: "Map[K,V]",
			config: GenericNamingConfig{
				Strategy: GenericNamingFlattened,
			},
			expected: "MapKV",
		},
		{
			name:  "flattened nested",
			input: "Response[List[User]]",
			config: GenericNamingConfig{
				Strategy: GenericNamingFlattened,
			},
			expected: "ResponseListUser",
		},

		// Dot strategy
		{
			name:  "dot simple",
			input: "Response[User]",
			config: GenericNamingConfig{
				Strategy: GenericNamingDot,
			},
			expected: "Response.User",
		},
		{
			name:  "dot multiple",
			input: "Map[K,V]",
			config: GenericNamingConfig{
				Strategy: GenericNamingDot,
			},
			expected: "Map.K.V",
		},

		// Preserve casing
		{
			name:  "preserve casing",
			input: "Response[user]",
			config: GenericNamingConfig{
				Strategy:       GenericNamingOf,
				PreserveCasing: true,
			},
			expected: "ResponseOfuser",
		},

		// Non-generic names (just sanitized)
		{
			name:  "plain name unchanged",
			input: "UserProfile",
			config: GenericNamingConfig{
				Strategy: GenericNamingOf,
			},
			expected: "UserProfile",
		},

		// Angle brackets
		{
			name:  "angle brackets",
			input: "List<Item>",
			config: GenericNamingConfig{
				Strategy: GenericNamingOf,
			},
			expected: "ListOfItem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformSchemaName(tt.input, tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSanitizeSchemaName tests the sanitization of schema names
func TestSanitizeSchemaName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid name unchanged",
			input:    "UserProfile",
			expected: "UserProfile",
		},
		{
			name:     "spaces replaced",
			input:    "User Profile",
			expected: "User_Profile",
		},
		{
			name:     "multiple spaces collapsed",
			input:    "User   Profile",
			expected: "User_Profile",
		},
		{
			name:     "leading trailing trimmed",
			input:    " User ",
			expected: "User",
		},
		{
			name:     "pipe replaced",
			input:    "User|Admin",
			expected: "User_Admin",
		},
		{
			name:     "brackets replaced",
			input:    "Response[User]",
			expected: "Response_User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeSchemaName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestToPascalCase tests the toPascalCase function
func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "lowercase", input: "user", expected: "User"},
		{name: "uppercase", input: "USER", expected: "USER"},
		{name: "already pascal", input: "UserProfile", expected: "UserProfile"},
		{name: "snake_case", input: "user_profile", expected: "UserProfile"},
		{name: "kebab-case", input: "user-profile", expected: "UserProfile"},
		{name: "dot.case", input: "user.profile", expected: "UserProfile"},
		{name: "slash/case", input: "user/profile", expected: "UserProfile"},
		{name: "space case", input: "user profile", expected: "UserProfile"},
		{name: "mixed", input: "user_profile-data.item", expected: "UserProfileDataItem"},
		{name: "single char", input: "u", expected: "U"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toPascalCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGenericNamingStrategy_String tests the String method of GenericNamingStrategy
func TestGenericNamingStrategy_String(t *testing.T) {
	tests := []struct {
		strategy GenericNamingStrategy
		expected string
	}{
		{GenericNamingUnderscore, "underscore"},
		{GenericNamingOf, "of"},
		{GenericNamingFor, "for"},
		{GenericNamingFlattened, "flattened"},
		{GenericNamingDot, "dot"},
		{GenericNamingStrategy(999), "unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.strategy.String())
		})
	}
}

// TestParseGenericNamingStrategy tests parsing strings into strategies
func TestParseGenericNamingStrategy(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    GenericNamingStrategy
		expectError bool
	}{
		{name: "underscore", input: "underscore", expected: GenericNamingUnderscore},
		{name: "underscore symbol", input: "_", expected: GenericNamingUnderscore},
		{name: "of", input: "of", expected: GenericNamingOf},
		{name: "for", input: "for", expected: GenericNamingFor},
		{name: "flattened", input: "flattened", expected: GenericNamingFlattened},
		{name: "flat shorthand", input: "flat", expected: GenericNamingFlattened},
		{name: "dot", input: "dot", expected: GenericNamingDot},
		{name: "dot symbol", input: ".", expected: GenericNamingDot},
		{name: "uppercase", input: "OF", expected: GenericNamingOf},
		{name: "with spaces", input: "  of  ", expected: GenericNamingOf},
		{name: "invalid", input: "invalid", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGenericNamingStrategy(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestDefaultGenericNamingConfig tests the default configuration
func TestDefaultGenericNamingConfig(t *testing.T) {
	config := DefaultGenericNamingConfig()

	assert.Equal(t, GenericNamingUnderscore, config.Strategy)
	assert.Equal(t, "_", config.Separator)
	assert.Equal(t, "_", config.ParamSeparator)
	assert.False(t, config.PreserveCasing)
}

// =============================================================================
// Schema Ref Rewriting Tests
// =============================================================================

// TestRewriteSchemaRefs tests rewriting $ref values in schemas
func TestRewriteSchemaRefs(t *testing.T) {
	// Create a schema with various ref locations
	schema := &parser.Schema{
		Ref: "#/components/schemas/OldName",
		Properties: map[string]*parser.Schema{
			"nested": {
				Ref: "#/components/schemas/OldName",
			},
		},
		AllOf: []*parser.Schema{
			{Ref: "#/components/schemas/OldName"},
		},
		Items: &parser.Schema{
			Ref: "#/components/schemas/OldName",
		},
	}

	renames := map[string]string{
		"#/components/schemas/OldName": "#/components/schemas/NewName",
	}

	rewriteSchemaRefs(schema, renames)

	assert.Equal(t, "#/components/schemas/NewName", schema.Ref)
	assert.Equal(t, "#/components/schemas/NewName", schema.Properties["nested"].Ref)
	assert.Equal(t, "#/components/schemas/NewName", schema.AllOf[0].Ref)

	// Items is interface{}, need type assertion
	items := schema.Items.(*parser.Schema)
	assert.Equal(t, "#/components/schemas/NewName", items.Ref)
}

// TestRewriteSchemaRefs_NilHandling tests that nil schemas are handled
func TestRewriteSchemaRefs_NilHandling(t *testing.T) {
	// Should not panic
	rewriteSchemaRefs(nil, map[string]string{"old": "new"})
	rewriteSchemaRefs(&parser.Schema{}, nil)
	rewriteSchemaRefs(&parser.Schema{}, map[string]string{})
}

// TestRewriteSchemaRefs_CircularRef tests that circular refs are handled
func TestRewriteSchemaRefs_CircularRef(t *testing.T) {
	schema := &parser.Schema{
		Ref:        "#/components/schemas/OldName",
		Properties: map[string]*parser.Schema{},
	}
	// Create circular reference
	schema.Properties["self"] = schema

	renames := map[string]string{
		"#/components/schemas/OldName": "#/components/schemas/NewName",
	}

	// Should not infinite loop
	rewriteSchemaRefs(schema, renames)

	assert.Equal(t, "#/components/schemas/NewName", schema.Ref)
}

// TestRewriteSchemaRefs_AdditionalProperties tests rewriting in additionalProperties
func TestRewriteSchemaRefs_AdditionalProperties(t *testing.T) {
	schema := &parser.Schema{
		AdditionalProperties: &parser.Schema{
			Ref: "#/components/schemas/OldName",
		},
	}

	renames := map[string]string{
		"#/components/schemas/OldName": "#/components/schemas/NewName",
	}

	rewriteSchemaRefs(schema, renames)

	addProps := schema.AdditionalProperties.(*parser.Schema)
	assert.Equal(t, "#/components/schemas/NewName", addProps.Ref)
}

// TestRewriteSchemaRefs_Discriminator tests rewriting discriminator mapping
func TestRewriteSchemaRefs_Discriminator(t *testing.T) {
	schema := &parser.Schema{
		Discriminator: &parser.Discriminator{
			PropertyName: "type",
			Mapping: map[string]string{
				"dog": "#/components/schemas/OldDog",
				"cat": "#/components/schemas/Cat", // unchanged
			},
		},
	}

	renames := map[string]string{
		"#/components/schemas/OldDog": "#/components/schemas/NewDog",
	}

	rewriteSchemaRefs(schema, renames)

	assert.Equal(t, "#/components/schemas/NewDog", schema.Discriminator.Mapping["dog"])
	assert.Equal(t, "#/components/schemas/Cat", schema.Discriminator.Mapping["cat"])
}

// TestRewriteSchemaRefs_AnyOfOneOf tests rewriting in anyOf and oneOf
func TestRewriteSchemaRefs_AnyOfOneOf(t *testing.T) {
	schema := &parser.Schema{
		AnyOf: []*parser.Schema{
			{Ref: "#/components/schemas/OldA"},
		},
		OneOf: []*parser.Schema{
			{Ref: "#/components/schemas/OldB"},
		},
		Not: &parser.Schema{
			Ref: "#/components/schemas/OldC",
		},
	}

	renames := map[string]string{
		"#/components/schemas/OldA": "#/components/schemas/NewA",
		"#/components/schemas/OldB": "#/components/schemas/NewB",
		"#/components/schemas/OldC": "#/components/schemas/NewC",
	}

	rewriteSchemaRefs(schema, renames)

	assert.Equal(t, "#/components/schemas/NewA", schema.AnyOf[0].Ref)
	assert.Equal(t, "#/components/schemas/NewB", schema.OneOf[0].Ref)
	assert.Equal(t, "#/components/schemas/NewC", schema.Not.Ref)
}

// TestExtractSchemaNameFromRefPath tests extracting names from ref paths
func TestExtractSchemaNameFromRefPath(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "OAS 3.x schema",
			ref:      "#/components/schemas/User",
			expected: "User",
		},
		{
			name:     "OAS 2.0 definition",
			ref:      "#/definitions/User",
			expected: "User",
		},
		{
			name:     "non-schema ref",
			ref:      "#/components/parameters/Param",
			expected: "",
		},
		{
			name:     "external ref",
			ref:      "external.yaml#/components/schemas/User",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSchemaNameFromRefPath(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsValidSchemaNameChar tests validation of schema name characters
func TestIsValidSchemaNameChar(t *testing.T) {
	// Valid characters
	assert.True(t, isValidSchemaNameChar('a'))
	assert.True(t, isValidSchemaNameChar('Z'))
	assert.True(t, isValidSchemaNameChar('0'))
	assert.True(t, isValidSchemaNameChar('_'))
	assert.True(t, isValidSchemaNameChar('-'))
	assert.True(t, isValidSchemaNameChar('.'))

	// Invalid characters
	assert.False(t, isValidSchemaNameChar('['))
	assert.False(t, isValidSchemaNameChar(']'))
	assert.False(t, isValidSchemaNameChar('<'))
	assert.False(t, isValidSchemaNameChar('>'))
	assert.False(t, isValidSchemaNameChar(' '))
	assert.False(t, isValidSchemaNameChar(','))
	assert.False(t, isValidSchemaNameChar('|'))
}
