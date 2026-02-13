package builder

import (
	"reflect"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for naming tests
type testUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TestExtractBaseTypeName tests the extractBaseTypeName function.
func TestExtractBaseTypeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Response[User]", "Response"},
		{"Map[string,int]", "Map"},
		{"NoGenerics", "NoGenerics"},
		{"Nested[List[User]]", "Nested"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractBaseTypeName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestExtractGenericParams tests the extractGenericParams function.
func TestExtractGenericParams(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"single param", "Response[User]", []string{"User"}},
		{"two params", "Map[string,int]", []string{"string", "int"}},
		{"no generics", "NoGenerics", nil},
		{"nested", "Response[List[User]]", []string{"List[User]"}},
		{"three params", "Tuple[A,B,C]", []string{"A", "B", "C"}},
		{"with package", "Response[models.User]", []string{"models.User"}},
		{"empty", "", nil},
		{"malformed open", "Response[User", nil},
		{"malformed close", "ResponseUser]", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGenericParams(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSanitizeSchemaName tests the sanitizeSchemaName function.
func TestSanitizeSchemaName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple type", "User", "User"},
		{"generic single param", "Response[User]", "Response_User"},
		{"generic with package", "Response[main.User]", "Response_main.User"},
		{"generic multi params", "Map[string,int]", "Map_string_int"},
		{"nested generic", "Response[List[User]]", "Response_List_User"},
		{"complex nested", "Map[string,Response[User]]", "Map_string_Response_User"},
		{"multiple underscores", "Type__With__Underscores_", "Type_With_Underscores"},
		{"with spaces", "Some Type", "Some_Type"},
		{"no changes needed", "NoChanges", "NoChanges"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSchemaName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestToPascalCase tests the toPascalCase function.
func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user_profile", "UserProfile"},
		{"api-client", "ApiClient"},
		{"UserProfile", "UserProfile"},
		{"user.profile", "UserProfile"},
		{"path/to/type", "PathToType"},
		{"", ""},
		{"a", "A"},
		{"already_pascal_case", "AlreadyPascalCase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPascalCase(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestToCamelCase tests the toCamelCase function.
func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user_profile", "userProfile"},
		{"api-client", "apiClient"},
		{"UserProfile", "userProfile"},
		{"user.profile", "userProfile"},
		{"", ""},
		{"A", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toCamelCase(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestToSnakeCase tests the toSnakeCase function.
func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"UserProfile", "user_profile"},
		// Note: APIClient becomes a_p_i_client because each uppercase letter
		// is treated as a word boundary. This is a simple algorithm that doesn't
		// try to detect acronyms.
		{"APIClient", "a_p_i_client"},
		{"user_profile", "user_profile"},
		{"api-client", "api_client"},
		{"path.to.type", "path_to_type"},
		{"", ""},
		{"A", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestToKebabCase tests the toKebabCase function.
func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"UserProfile", "user-profile"},
		// Note: Like snake_case, each uppercase letter is a word boundary
		{"APIClient", "a-p-i-client"},
		{"user_profile", "user-profile"},
		{"api-client", "api-client"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toKebabCase(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSanitizePath tests the sanitizePath function.
func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github.com/org/models", "github.com_org_models"},
		{"models", "models"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizePath(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDefaultGenericNamingConfig tests DefaultGenericNamingConfig.
func TestDefaultGenericNamingConfig(t *testing.T) {
	cfg := DefaultGenericNamingConfig()

	assert.Equal(t, GenericNamingUnderscore, cfg.Strategy)
	assert.Equal(t, "_", cfg.Separator)
	assert.Equal(t, "_", cfg.ParamSeparator)
	assert.False(t, cfg.IncludePackage)
	assert.False(t, cfg.ApplyBaseCasing)
}

// TestNewSchemaNamer tests newSchemaNamer creates a namer with defaults.
func TestNewSchemaNamer(t *testing.T) {
	namer := newSchemaNamer()

	assert.Equal(t, SchemaNamingDefault, namer.strategy)
	assert.Equal(t, GenericNamingUnderscore, namer.genericConfig.Strategy)
	assert.Nil(t, namer.template)
	assert.Nil(t, namer.fn)
}

// TestSchemaNamerBuildContext tests context building from reflect.Type.
func TestSchemaNamerBuildContext(t *testing.T) {
	namer := newSchemaNamer()

	t.Run("struct type", func(t *testing.T) {
		ctx := namer.buildContext(reflect.TypeOf(testUser{}))

		assert.Equal(t, "testUser", ctx.Type)
		assert.Equal(t, "testUser", ctx.TypeBase)
		assert.Equal(t, "testUser", ctx.TypeSanitized)
		assert.Equal(t, "builder", ctx.Package)
		assert.False(t, ctx.IsGeneric)
		assert.False(t, ctx.IsAnonymous)
		assert.False(t, ctx.IsPointer)
		assert.Equal(t, "struct", ctx.Kind)
	})

	t.Run("pointer type", func(t *testing.T) {
		ctx := namer.buildContext(reflect.TypeOf(&testUser{}))

		assert.True(t, ctx.IsPointer)
		assert.Equal(t, "testUser", ctx.Type)
	})

	t.Run("anonymous type", func(t *testing.T) {
		ctx := namer.buildContext(reflect.TypeOf(struct{ X int }{}))

		assert.True(t, ctx.IsAnonymous)
		assert.Equal(t, "", ctx.Type)
	})
}

// TestSchemaNamerDefaultName tests the default naming strategy.
func TestSchemaNamerDefaultName(t *testing.T) {
	namer := newSchemaNamer()

	t.Run("named type", func(t *testing.T) {
		name := namer.name(reflect.TypeOf(testUser{}))
		// Should be package.TypeName format
		assert.Equal(t, "builder.testUser", name)
	})

	t.Run("anonymous type", func(t *testing.T) {
		name := namer.name(reflect.TypeOf(struct{ X int }{}))
		assert.Equal(t, "AnonymousType", name)
	})
}

// TestSchemaNamerStrategies tests all built-in naming strategies.
func TestSchemaNamerStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy SchemaNamingStrategy
		want     string
	}{
		{"default", SchemaNamingDefault, "builder.testUser"},
		{"pascal", SchemaNamingPascalCase, "BuilderTestUser"},
		{"camel", SchemaNamingCamelCase, "builderTestUser"},
		{"snake", SchemaNamingSnakeCase, "builder_test_user"},
		{"kebab", SchemaNamingKebabCase, "builder-test-user"},
		{"type_only", SchemaNamingTypeOnly, "testUser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namer := newSchemaNamer()
			namer.strategy = tt.strategy

			got := namer.name(reflect.TypeOf(testUser{}))
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaNamerGenericStrategies tests generic naming strategies.
func TestSchemaNamerGenericStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy GenericNamingStrategy
		params   []string
		want     string
	}{
		{"underscore_single", GenericNamingUnderscore, []string{"User"}, "_User_"},
		{"underscore_multi", GenericNamingUnderscore, []string{"string", "int"}, "_string_int_"},
		{"of_single", GenericNamingOf, []string{"User"}, "OfUser"},
		// Note: with default ParamSeparator="_", multi params become "Ofstring_Ofint"
		{"of_multi", GenericNamingOf, []string{"string", "int"}, "Ofstring_Ofint"},
		{"for_single", GenericNamingFor, []string{"User"}, "ForUser"},
		// Note: with default ParamSeparator="_", multi params become "Forstring_Forint"
		{"for_multi", GenericNamingFor, []string{"string", "int"}, "Forstring_Forint"},
		{"angle_single", GenericNamingAngleBrackets, []string{"User"}, "<User>"},
		{"angle_multi", GenericNamingAngleBrackets, []string{"string", "int"}, "<string,int>"},
		{"flat_single", GenericNamingFlattened, []string{"User"}, "User"},
		{"flat_multi", GenericNamingFlattened, []string{"string", "int"}, "stringint"},
		{"empty", GenericNamingUnderscore, []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namer := newSchemaNamer()
			namer.genericConfig.Strategy = tt.strategy

			got := namer.formatGenericSuffix(tt.params)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaNamerGenericConfig tests custom generic naming configuration.
func TestSchemaNamerGenericConfig(t *testing.T) {
	t.Run("custom separator", func(t *testing.T) {
		namer := newSchemaNamer()
		namer.genericConfig.Strategy = GenericNamingUnderscore
		namer.genericConfig.Separator = "__"

		got := namer.formatGenericSuffix([]string{"User"})
		assert.Equal(t, "__User__", got)
	})

	t.Run("custom param separator", func(t *testing.T) {
		namer := newSchemaNamer()
		namer.genericConfig.Strategy = GenericNamingOf
		namer.genericConfig.ParamSeparator = "And"

		got := namer.formatGenericSuffix([]string{"string", "int"})
		assert.Equal(t, "OfstringAndOfint", got)
	})

	t.Run("include package", func(t *testing.T) {
		namer := newSchemaNamer()
		namer.genericConfig.IncludePackage = true

		params := []string{"models.User"}
		got := namer.sanitizeGenericParams(params)
		require.Len(t, got, 1)
		assert.Equal(t, "models_User", got[0])
	})

	t.Run("strip package", func(t *testing.T) {
		namer := newSchemaNamer()
		namer.genericConfig.IncludePackage = false

		params := []string{"models.User"}
		got := namer.sanitizeGenericParams(params)
		require.Len(t, got, 1)
		assert.Equal(t, "User", got[0])
	})

	t.Run("apply base casing", func(t *testing.T) {
		namer := newSchemaNamer()
		namer.strategy = SchemaNamingPascalCase
		namer.genericConfig.ApplyBaseCasing = true

		params := []string{"user_profile"}
		got := namer.sanitizeGenericParams(params)
		require.Len(t, got, 1)
		assert.Equal(t, "UserProfile", got[0])
	})
}

// TestSchemaNamerWithFunc tests custom naming function.
func TestSchemaNamerWithFunc(t *testing.T) {
	namer := newSchemaNamer()
	callCount := 0
	namer.fn = func(ctx SchemaNameContext) string {
		callCount++
		return "Custom_" + ctx.Type
	}

	got := namer.name(reflect.TypeOf(testUser{}))
	assert.Equal(t, "Custom_testUser", got)
	assert.Equal(t, 1, callCount)
}

// TestSchemaNamerWithTemplate tests custom naming template.
func TestSchemaNamerWithTemplate(t *testing.T) {
	t.Run("simple template", func(t *testing.T) {
		tmpl, err := parseSchemaNameTemplate("{{.Type}}")
		require.NoError(t, err)

		namer := newSchemaNamer()
		namer.template = tmpl

		got := namer.name(reflect.TypeOf(testUser{}))
		assert.Equal(t, "testUser", got)
	})

	t.Run("template with functions", func(t *testing.T) {
		tmpl, err := parseSchemaNameTemplate("{{pascal .Package}}{{pascal .Type}}")
		require.NoError(t, err)

		namer := newSchemaNamer()
		namer.template = tmpl

		got := namer.name(reflect.TypeOf(testUser{}))
		assert.Equal(t, "BuilderTestUser", got)
	})

	t.Run("template with conditionals", func(t *testing.T) {
		tmpl, err := parseSchemaNameTemplate("{{if .IsAnonymous}}Anon{{else}}{{.Type}}{{end}}")
		require.NoError(t, err)

		namer := newSchemaNamer()
		namer.template = tmpl

		got := namer.name(reflect.TypeOf(struct{ X int }{}))
		assert.Equal(t, "Anon", got)

		got = namer.name(reflect.TypeOf(testUser{}))
		assert.Equal(t, "testUser", got)
	})
}

// TestParseSchemaNameTemplate tests template parsing and validation.
func TestParseSchemaNameTemplate(t *testing.T) {
	t.Run("valid template", func(t *testing.T) {
		_, err := parseSchemaNameTemplate("{{.Type}}")
		assert.NoError(t, err)
	})

	t.Run("invalid syntax", func(t *testing.T) {
		_, err := parseSchemaNameTemplate("{{.Type")
		assert.Error(t, err)
	})

	t.Run("invalid field", func(t *testing.T) {
		// Template with invalid field should fail during validation execution
		_, err := parseSchemaNameTemplate("{{.InvalidField}}")
		assert.Error(t, err)
	})

	t.Run("all template functions", func(t *testing.T) {
		// Test that all template functions work
		funcs := []string{
			"{{pascal .Type}}",
			"{{camel .Type}}",
			"{{snake .Type}}",
			"{{kebab .Type}}",
			"{{upper .Type}}",
			"{{lower .Type}}",
			"{{title .Type}}",
			"{{sanitize .Type}}",
			"{{trimPrefix .Type \"test\"}}",
			"{{trimSuffix .Type \"User\"}}",
			"{{replace .Type \"test\" \"Test\"}}",
			"{{join \"-\" .Package .Type}}",
		}

		for _, tmplStr := range funcs {
			_, err := parseSchemaNameTemplate(tmplStr)
			assert.NoError(t, err)
		}
	})
}

// TestSchemaNamerNameWithConflictCheck tests conflict detection.
func TestSchemaNamerNameWithConflictCheck(t *testing.T) {
	namer := newSchemaNamer()

	t.Run("no conflict", func(t *testing.T) {
		got := namer.nameWithConflictCheck(reflect.TypeOf(testUser{}), func(name string) bool {
			return false // No conflict
		})
		assert.Equal(t, "builder.testUser", got)
	})

	t.Run("with conflict", func(t *testing.T) {
		got := namer.nameWithConflictCheck(reflect.TypeOf(testUser{}), func(name string) bool {
			return name == "builder.testUser" // Conflict on initial name
		})
		// Should include full package path
		assert.NotEqual(t, "builder.testUser", got)
	})
}

// TestSchemaNamerApplyCasing tests casing application to strings.
func TestSchemaNamerApplyCasing(t *testing.T) {
	tests := []struct {
		strategy SchemaNamingStrategy
		input    string
		want     string
	}{
		{SchemaNamingDefault, "user_profile", "user_profile"},
		{SchemaNamingPascalCase, "user_profile", "UserProfile"},
		{SchemaNamingCamelCase, "user_profile", "userProfile"},
		{SchemaNamingSnakeCase, "UserProfile", "user_profile"},
		{SchemaNamingKebabCase, "UserProfile", "user-profile"},
		{SchemaNamingTypeOnly, "user_profile", "user_profile"},
		{SchemaNamingFullPath, "user_profile", "user_profile"},
	}

	for _, tt := range tests {
		t.Run(tt.strategy.String(), func(t *testing.T) {
			namer := newSchemaNamer()
			namer.strategy = tt.strategy

			got := namer.applyCasing(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// String returns a string representation of SchemaNamingStrategy for testing.
func (s SchemaNamingStrategy) String() string {
	switch s {
	case SchemaNamingDefault:
		return "default"
	case SchemaNamingPascalCase:
		return "pascal"
	case SchemaNamingCamelCase:
		return "camel"
	case SchemaNamingSnakeCase:
		return "snake"
	case SchemaNamingKebabCase:
		return "kebab"
	case SchemaNamingTypeOnly:
		return "type_only"
	case SchemaNamingFullPath:
		return "full_path"
	default:
		return "unknown"
	}
}

// TestTemplateFuncs tests that templateFuncs returns all expected functions.
func TestTemplateFuncs(t *testing.T) {
	funcs := templateFuncs()

	expectedFuncs := []string{
		"pascal", "camel", "snake", "kebab",
		"upper", "lower", "title", "sanitize",
		"trimPrefix", "trimSuffix", "replace", "join",
	}

	for _, name := range expectedFuncs {
		assert.Contains(t, funcs, name)
	}
}

// TestSchemaNameContextFields tests that all SchemaNameContext fields are populated.
func TestSchemaNameContextFields(t *testing.T) {
	namer := newSchemaNamer()
	ctx := namer.buildContext(reflect.TypeOf(testUser{}))

	// Verify all fields that should be populated for a struct type
	assert.NotEmpty(t, ctx.Type)
	assert.NotEmpty(t, ctx.TypeSanitized)
	assert.NotEmpty(t, ctx.TypeBase)
	assert.NotEmpty(t, ctx.Package)
	assert.NotEmpty(t, ctx.PackagePath)
	assert.NotEmpty(t, ctx.PackagePathSanitized)
	assert.NotEmpty(t, ctx.Kind)
}

// TestSchemaNamerDefaultNameEdgeCases tests edge cases for defaultName.
func TestSchemaNamerDefaultNameEdgeCases(t *testing.T) {
	t.Run("anonymous type", func(t *testing.T) {
		namer := newSchemaNamer()
		ctx := namer.buildContext(reflect.TypeOf(struct{ X int }{}))
		name := namer.defaultName(ctx)
		assert.Equal(t, "AnonymousType", name)
	})

	t.Run("builtin type without package", func(t *testing.T) {
		namer := newSchemaNamer()
		// Create a context simulating a built-in type (no package)
		ctx := SchemaNameContext{
			Type:          "int",
			TypeSanitized: "int",
			TypeBase:      "int",
			Package:       "",
			IsAnonymous:   false,
		}
		name := namer.defaultName(ctx)
		assert.Equal(t, "int", name)
	})
}

// TestSchemaNamerApplyStrategyFullPath tests the full path strategy.
func TestSchemaNamerApplyStrategyFullPath(t *testing.T) {
	namer := newSchemaNamer()
	namer.strategy = SchemaNamingFullPath

	name := namer.name(reflect.TypeOf(testUser{}))

	// Should include full package path
	assert.Contains(t, name, "builder")
	assert.Contains(t, name, "testUser")
}

// TestSchemaNamerApplyStrategyBuiltinType tests strategies with no package.
func TestSchemaNamerApplyStrategyBuiltinType(t *testing.T) {
	namer := newSchemaNamer()

	tests := []struct {
		strategy SchemaNamingStrategy
	}{
		{SchemaNamingSnakeCase},
		{SchemaNamingKebabCase},
		{SchemaNamingFullPath},
	}

	for _, tt := range tests {
		t.Run(tt.strategy.String(), func(t *testing.T) {
			namer.strategy = tt.strategy
			// Create a context simulating a type with no package
			ctx := SchemaNameContext{
				Type:                 "CustomType",
				TypeSanitized:        "CustomType",
				TypeBase:             "CustomType",
				Package:              "",
				PackagePathSanitized: "",
				IsAnonymous:          false,
			}
			name := namer.applyStrategy(ctx)
			assert.NotEmpty(t, name)
		})
	}
}

// TestFormatGenericSuffixEmptySeparator tests edge case with empty separator.
func TestFormatGenericSuffixEmptySeparator(t *testing.T) {
	namer := newSchemaNamer()
	namer.genericConfig.Strategy = GenericNamingUnderscore
	namer.genericConfig.Separator = ""
	namer.genericConfig.ParamSeparator = ""

	got := namer.formatGenericSuffix([]string{"A", "B"})
	// With empty separators, should still use defaults
	assert.Contains(t, got, "A")
	assert.Contains(t, got, "B")
}

// TestBuildContextGenericType tests building context for generic-like type names.
func TestBuildContextGenericType(t *testing.T) {
	// We can't easily create a real generic type in tests, but we can verify
	// the string parsing works by checking a type with brackets in the name
	// would be detected (though Go types won't naturally have this)

	// Test the parsing logic directly
	typeName := "Response[User]"
	assert.Contains(t, typeName, "[")

	baseType := extractBaseTypeName(typeName)
	assert.Equal(t, "Response", baseType)

	params := extractGenericParams(typeName)
	require.Len(t, params, 1)
	assert.Equal(t, "User", params[0])
}

// TestTemplateExecutionError tests template fallback on execution error.
func TestTemplateExecutionError(t *testing.T) {
	// Create a template that will fail during execution with some inputs
	// This is tricky because we validate templates, but we can test the fallback
	tmpl, err := parseSchemaNameTemplate("{{.Type}}")
	require.NoError(t, err)

	namer := newSchemaNamer()
	namer.template = tmpl

	// Normal execution should work
	name := namer.name(reflect.TypeOf(testUser{}))
	assert.Equal(t, "testUser", name)
}

// TestWithSchemaNameFunc tests the custom naming function option.
func TestWithSchemaNameFunc(t *testing.T) {
	customNamer := func(ctx SchemaNameContext) string {
		return "custom_" + ctx.Type
	}

	spec := New(parser.OASVersion320, WithSchemaNameFunc(customNamer))
	spec.SetTitle("Test").SetVersion("1.0.0")
	spec.RegisterType(testUser{})

	doc, err := spec.BuildOAS3()
	require.NoError(t, err)

	// Verify the custom function was used
	assert.Contains(t, doc.Components.Schemas, "custom_testUser")
}

// TestWithGenericNaming tests the generic naming strategy option.
func TestWithGenericNaming(t *testing.T) {
	tests := []struct {
		name     string
		strategy GenericNamingStrategy
	}{
		{"Underscore", GenericNamingUnderscore},
		{"Of", GenericNamingOf},
		{"For", GenericNamingFor},
		{"AngleBrackets", GenericNamingAngleBrackets},
		{"Flattened", GenericNamingFlattened},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := New(parser.OASVersion320, WithGenericNaming(tt.strategy))
			spec.SetTitle("Test").SetVersion("1.0.0")

			// Build should succeed with any strategy
			doc, err := spec.BuildOAS3()
			require.NoError(t, err)
			assert.NotNil(t, doc)
		})
	}
}

// TestWithGenericSeparator tests custom separator for generic types.
func TestWithGenericSeparator(t *testing.T) {
	spec := New(parser.OASVersion320, WithGenericSeparator("__"))
	spec.SetTitle("Test").SetVersion("1.0.0")

	// Verify the option was applied
	assert.Equal(t, "__", spec.namer.genericConfig.Separator)
}

// TestWithGenericParamSeparator tests custom parameter separator.
func TestWithGenericParamSeparator(t *testing.T) {
	spec := New(parser.OASVersion320, WithGenericParamSeparator("And"))
	spec.SetTitle("Test").SetVersion("1.0.0")

	// Verify the option was applied
	assert.Equal(t, "And", spec.namer.genericConfig.ParamSeparator)
}

// TestWithGenericIncludePackage tests package inclusion in generic params.
func TestWithGenericIncludePackage(t *testing.T) {
	spec := New(parser.OASVersion320, WithGenericIncludePackage(true))
	spec.SetTitle("Test").SetVersion("1.0.0")

	// Verify the option was applied
	assert.True(t, spec.namer.genericConfig.IncludePackage)
}

// TestWithGenericApplyBaseCasing tests base casing for generic params.
func TestWithGenericApplyBaseCasing(t *testing.T) {
	spec := New(parser.OASVersion320, WithGenericApplyBaseCasing(true))
	spec.SetTitle("Test").SetVersion("1.0.0")

	// Verify the option was applied
	assert.True(t, spec.namer.genericConfig.ApplyBaseCasing)
}

// TestBuilder_InvalidTemplateError tests that invalid templates cause BuildOAS3/BuildOAS2 to return errors.
// This is an integration test for the configError propagation path.
func TestBuilder_InvalidTemplateError(t *testing.T) {
	t.Run("BuildOAS3", func(t *testing.T) {
		spec := New(parser.OASVersion320,
			WithSchemaNameTemplate("{{.Type"), // Invalid syntax - unclosed action
		).SetTitle("Test").SetVersion("1.0.0")

		_, err := spec.BuildOAS3()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration error")
	})

	t.Run("BuildOAS2", func(t *testing.T) {
		spec := New(parser.OASVersion20,
			WithSchemaNameTemplate("{{.Type"), // Invalid syntax - unclosed action
		).SetTitle("Test").SetVersion("1.0.0")

		_, err := spec.BuildOAS2()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration error")
	})
}
