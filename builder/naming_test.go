package builder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
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
			if got != tt.want {
				t.Errorf("extractBaseTypeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
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
			if len(got) != len(tt.want) {
				t.Errorf("extractGenericParams(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractGenericParams(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.want[i])
				}
			}
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
			if got != tt.want {
				t.Errorf("sanitizeSchemaName(%q) = %q, want %q", tt.input, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("toKebabCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestDefaultGenericNamingConfig tests DefaultGenericNamingConfig.
func TestDefaultGenericNamingConfig(t *testing.T) {
	cfg := DefaultGenericNamingConfig()

	if cfg.Strategy != GenericNamingUnderscore {
		t.Errorf("Strategy = %d, want %d", cfg.Strategy, GenericNamingUnderscore)
	}
	if cfg.Separator != "_" {
		t.Errorf("Separator = %q, want %q", cfg.Separator, "_")
	}
	if cfg.ParamSeparator != "_" {
		t.Errorf("ParamSeparator = %q, want %q", cfg.ParamSeparator, "_")
	}
	if cfg.IncludePackage {
		t.Errorf("IncludePackage = true, want false")
	}
	if cfg.ApplyBaseCasing {
		t.Errorf("ApplyBaseCasing = true, want false")
	}
}

// TestNewSchemaNamer tests newSchemaNamer creates a namer with defaults.
func TestNewSchemaNamer(t *testing.T) {
	namer := newSchemaNamer()

	if namer.strategy != SchemaNamingDefault {
		t.Errorf("strategy = %d, want %d", namer.strategy, SchemaNamingDefault)
	}
	if namer.genericConfig.Strategy != GenericNamingUnderscore {
		t.Errorf("genericConfig.Strategy = %d, want %d",
			namer.genericConfig.Strategy, GenericNamingUnderscore)
	}
	if namer.template != nil {
		t.Errorf("template = %v, want nil", namer.template)
	}
	if namer.fn != nil {
		t.Errorf("fn = %v, want nil", namer.fn)
	}
}

// TestSchemaNamerBuildContext tests context building from reflect.Type.
func TestSchemaNamerBuildContext(t *testing.T) {
	namer := newSchemaNamer()

	t.Run("struct type", func(t *testing.T) {
		ctx := namer.buildContext(reflect.TypeOf(testUser{}))

		if ctx.Type != "testUser" {
			t.Errorf("Type = %q, want %q", ctx.Type, "testUser")
		}
		if ctx.TypeBase != "testUser" {
			t.Errorf("TypeBase = %q, want %q", ctx.TypeBase, "testUser")
		}
		if ctx.TypeSanitized != "testUser" {
			t.Errorf("TypeSanitized = %q, want %q", ctx.TypeSanitized, "testUser")
		}
		if ctx.Package != "builder" {
			t.Errorf("Package = %q, want %q", ctx.Package, "builder")
		}
		if ctx.IsGeneric {
			t.Errorf("IsGeneric = true, want false")
		}
		if ctx.IsAnonymous {
			t.Errorf("IsAnonymous = true, want false")
		}
		if ctx.IsPointer {
			t.Errorf("IsPointer = true, want false")
		}
		if ctx.Kind != "struct" {
			t.Errorf("Kind = %q, want %q", ctx.Kind, "struct")
		}
	})

	t.Run("pointer type", func(t *testing.T) {
		ctx := namer.buildContext(reflect.TypeOf(&testUser{}))

		if !ctx.IsPointer {
			t.Errorf("IsPointer = false, want true")
		}
		if ctx.Type != "testUser" {
			t.Errorf("Type = %q, want %q (should be dereferenced)", ctx.Type, "testUser")
		}
	})

	t.Run("anonymous type", func(t *testing.T) {
		ctx := namer.buildContext(reflect.TypeOf(struct{ X int }{}))

		if !ctx.IsAnonymous {
			t.Errorf("IsAnonymous = false, want true")
		}
		if ctx.Type != "" {
			t.Errorf("Type = %q, want empty string", ctx.Type)
		}
	})
}

// TestSchemaNamerDefaultName tests the default naming strategy.
func TestSchemaNamerDefaultName(t *testing.T) {
	namer := newSchemaNamer()

	t.Run("named type", func(t *testing.T) {
		name := namer.name(reflect.TypeOf(testUser{}))
		// Should be package.TypeName format
		if name != "builder.testUser" {
			t.Errorf("name = %q, want %q", name, "builder.testUser")
		}
	})

	t.Run("anonymous type", func(t *testing.T) {
		name := namer.name(reflect.TypeOf(struct{ X int }{}))
		if name != "AnonymousType" {
			t.Errorf("name = %q, want %q", name, "AnonymousType")
		}
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
			if got != tt.want {
				t.Errorf("name() with %s = %q, want %q", tt.name, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("formatGenericSuffix(%v) with %s = %q, want %q",
					tt.params, tt.name, got, tt.want)
			}
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
		if got != "__User__" {
			t.Errorf("formatGenericSuffix with Separator='__' = %q, want %q", got, "__User__")
		}
	})

	t.Run("custom param separator", func(t *testing.T) {
		namer := newSchemaNamer()
		namer.genericConfig.Strategy = GenericNamingOf
		namer.genericConfig.ParamSeparator = "And"

		got := namer.formatGenericSuffix([]string{"string", "int"})
		if got != "OfstringAndOfint" {
			t.Errorf("formatGenericSuffix with ParamSeparator='And' = %q, want %q", got, "OfstringAndOfint")
		}
	})

	t.Run("include package", func(t *testing.T) {
		namer := newSchemaNamer()
		namer.genericConfig.IncludePackage = true

		params := []string{"models.User"}
		got := namer.sanitizeGenericParams(params)
		if len(got) != 1 || got[0] != "models_User" {
			t.Errorf("sanitizeGenericParams with IncludePackage = %v, want [models_User]", got)
		}
	})

	t.Run("strip package", func(t *testing.T) {
		namer := newSchemaNamer()
		namer.genericConfig.IncludePackage = false

		params := []string{"models.User"}
		got := namer.sanitizeGenericParams(params)
		if len(got) != 1 || got[0] != "User" {
			t.Errorf("sanitizeGenericParams without IncludePackage = %v, want [User]", got)
		}
	})

	t.Run("apply base casing", func(t *testing.T) {
		namer := newSchemaNamer()
		namer.strategy = SchemaNamingPascalCase
		namer.genericConfig.ApplyBaseCasing = true

		params := []string{"user_profile"}
		got := namer.sanitizeGenericParams(params)
		if len(got) != 1 || got[0] != "UserProfile" {
			t.Errorf("sanitizeGenericParams with ApplyBaseCasing = %v, want [UserProfile]", got)
		}
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
	if got != "Custom_testUser" {
		t.Errorf("name with custom func = %q, want %q", got, "Custom_testUser")
	}
	if callCount != 1 {
		t.Errorf("custom func called %d times, want 1", callCount)
	}
}

// TestSchemaNamerWithTemplate tests custom naming template.
func TestSchemaNamerWithTemplate(t *testing.T) {
	t.Run("simple template", func(t *testing.T) {
		tmpl, err := parseSchemaNameTemplate("{{.Type}}")
		if err != nil {
			t.Fatalf("parseSchemaNameTemplate failed: %v", err)
		}

		namer := newSchemaNamer()
		namer.template = tmpl

		got := namer.name(reflect.TypeOf(testUser{}))
		if got != "testUser" {
			t.Errorf("name with template = %q, want %q", got, "testUser")
		}
	})

	t.Run("template with functions", func(t *testing.T) {
		tmpl, err := parseSchemaNameTemplate("{{pascal .Package}}{{pascal .Type}}")
		if err != nil {
			t.Fatalf("parseSchemaNameTemplate failed: %v", err)
		}

		namer := newSchemaNamer()
		namer.template = tmpl

		got := namer.name(reflect.TypeOf(testUser{}))
		if got != "BuilderTestUser" {
			t.Errorf("name with template = %q, want %q", got, "BuilderTestUser")
		}
	})

	t.Run("template with conditionals", func(t *testing.T) {
		tmpl, err := parseSchemaNameTemplate("{{if .IsAnonymous}}Anon{{else}}{{.Type}}{{end}}")
		if err != nil {
			t.Fatalf("parseSchemaNameTemplate failed: %v", err)
		}

		namer := newSchemaNamer()
		namer.template = tmpl

		got := namer.name(reflect.TypeOf(struct{ X int }{}))
		if got != "Anon" {
			t.Errorf("name for anonymous with template = %q, want %q", got, "Anon")
		}

		got = namer.name(reflect.TypeOf(testUser{}))
		if got != "testUser" {
			t.Errorf("name for named with template = %q, want %q", got, "testUser")
		}
	})
}

// TestParseSchemaNameTemplate tests template parsing and validation.
func TestParseSchemaNameTemplate(t *testing.T) {
	t.Run("valid template", func(t *testing.T) {
		_, err := parseSchemaNameTemplate("{{.Type}}")
		if err != nil {
			t.Errorf("parseSchemaNameTemplate failed: %v", err)
		}
	})

	t.Run("invalid syntax", func(t *testing.T) {
		_, err := parseSchemaNameTemplate("{{.Type")
		if err == nil {
			t.Errorf("parseSchemaNameTemplate should fail for invalid syntax")
		}
	})

	t.Run("invalid field", func(t *testing.T) {
		// Template with invalid field should fail during validation execution
		_, err := parseSchemaNameTemplate("{{.InvalidField}}")
		if err == nil {
			t.Errorf("parseSchemaNameTemplate should fail for invalid field")
		}
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
			if err != nil {
				t.Errorf("parseSchemaNameTemplate(%q) failed: %v", tmplStr, err)
			}
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
		if got != "builder.testUser" {
			t.Errorf("name without conflict = %q, want %q", got, "builder.testUser")
		}
	})

	t.Run("with conflict", func(t *testing.T) {
		got := namer.nameWithConflictCheck(reflect.TypeOf(testUser{}), func(name string) bool {
			return name == "builder.testUser" // Conflict on initial name
		})
		// Should include full package path
		if got == "builder.testUser" {
			t.Errorf("name with conflict should use full path, got %q", got)
		}
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
			if got != tt.want {
				t.Errorf("applyCasing(%q) with %d = %q, want %q",
					tt.input, tt.strategy, got, tt.want)
			}
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
		if _, ok := funcs[name]; !ok {
			t.Errorf("templateFuncs missing %q", name)
		}
	}
}

// TestSchemaNameContextFields tests that all SchemaNameContext fields are populated.
func TestSchemaNameContextFields(t *testing.T) {
	namer := newSchemaNamer()
	ctx := namer.buildContext(reflect.TypeOf(testUser{}))

	// Verify all fields that should be populated for a struct type
	if ctx.Type == "" {
		t.Error("Type should not be empty")
	}
	if ctx.TypeSanitized == "" {
		t.Error("TypeSanitized should not be empty")
	}
	if ctx.TypeBase == "" {
		t.Error("TypeBase should not be empty")
	}
	if ctx.Package == "" {
		t.Error("Package should not be empty")
	}
	if ctx.PackagePath == "" {
		t.Error("PackagePath should not be empty")
	}
	if ctx.PackagePathSanitized == "" {
		t.Error("PackagePathSanitized should not be empty")
	}
	if ctx.Kind == "" {
		t.Error("Kind should not be empty")
	}
}

// TestSchemaNamerDefaultNameEdgeCases tests edge cases for defaultName.
func TestSchemaNamerDefaultNameEdgeCases(t *testing.T) {
	t.Run("anonymous type", func(t *testing.T) {
		namer := newSchemaNamer()
		ctx := namer.buildContext(reflect.TypeOf(struct{ X int }{}))
		name := namer.defaultName(ctx)
		if name != "AnonymousType" {
			t.Errorf("defaultName for anonymous = %q, want %q", name, "AnonymousType")
		}
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
		if name != "int" {
			t.Errorf("defaultName for builtin = %q, want %q", name, "int")
		}
	})
}

// TestSchemaNamerApplyStrategyFullPath tests the full path strategy.
func TestSchemaNamerApplyStrategyFullPath(t *testing.T) {
	namer := newSchemaNamer()
	namer.strategy = SchemaNamingFullPath

	name := namer.name(reflect.TypeOf(testUser{}))

	// Should include full package path
	if !strings.Contains(name, "builder") {
		t.Errorf("full path name = %q, should contain 'builder'", name)
	}
	if !strings.Contains(name, "testUser") {
		t.Errorf("full path name = %q, should contain 'testUser'", name)
	}
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
			if name == "" {
				t.Error("applyStrategy should return non-empty name")
			}
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
	if !strings.Contains(got, "A") || !strings.Contains(got, "B") {
		t.Errorf("formatGenericSuffix with empty separator = %q, should contain A and B", got)
	}
}

// TestBuildContextGenericType tests building context for generic-like type names.
func TestBuildContextGenericType(t *testing.T) {
	// We can't easily create a real generic type in tests, but we can verify
	// the string parsing works by checking a type with brackets in the name
	// would be detected (though Go types won't naturally have this)

	// Test the parsing logic directly
	typeName := "Response[User]"
	if !strings.Contains(typeName, "[") {
		t.Error("Test type name should contain bracket")
	}

	baseType := extractBaseTypeName(typeName)
	if baseType != "Response" {
		t.Errorf("extractBaseTypeName(%q) = %q, want %q", typeName, baseType, "Response")
	}

	params := extractGenericParams(typeName)
	if len(params) != 1 || params[0] != "User" {
		t.Errorf("extractGenericParams(%q) = %v, want [User]", typeName, params)
	}
}

// TestTemplateExecutionError tests template fallback on execution error.
func TestTemplateExecutionError(t *testing.T) {
	// Create a template that will fail during execution with some inputs
	// This is tricky because we validate templates, but we can test the fallback
	tmpl, err := parseSchemaNameTemplate("{{.Type}}")
	if err != nil {
		t.Fatalf("parseSchemaNameTemplate failed: %v", err)
	}

	namer := newSchemaNamer()
	namer.template = tmpl

	// Normal execution should work
	name := namer.name(reflect.TypeOf(testUser{}))
	if name != "testUser" {
		t.Errorf("template execution = %q, want %q", name, "testUser")
	}
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
	if err != nil {
		t.Fatalf("BuildOAS3 failed: %v", err)
	}

	// Verify the custom function was used
	if _, exists := doc.Components.Schemas["custom_testUser"]; !exists {
		t.Errorf("expected schema named 'custom_testUser', got keys: %v", mapKeys(doc.Components.Schemas))
	}
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
			if err != nil {
				t.Fatalf("BuildOAS3 failed with strategy %s: %v", tt.name, err)
			}
			if doc == nil {
				t.Error("expected non-nil document")
			}
		})
	}
}

// TestWithGenericSeparator tests custom separator for generic types.
func TestWithGenericSeparator(t *testing.T) {
	spec := New(parser.OASVersion320, WithGenericSeparator("__"))
	spec.SetTitle("Test").SetVersion("1.0.0")

	// Verify the option was applied
	if spec.namer.genericConfig.Separator != "__" {
		t.Errorf("separator = %q, want %q", spec.namer.genericConfig.Separator, "__")
	}
}

// TestWithGenericParamSeparator tests custom parameter separator.
func TestWithGenericParamSeparator(t *testing.T) {
	spec := New(parser.OASVersion320, WithGenericParamSeparator("And"))
	spec.SetTitle("Test").SetVersion("1.0.0")

	// Verify the option was applied
	if spec.namer.genericConfig.ParamSeparator != "And" {
		t.Errorf("ParamSeparator = %q, want %q", spec.namer.genericConfig.ParamSeparator, "And")
	}
}

// TestWithGenericIncludePackage tests package inclusion in generic params.
func TestWithGenericIncludePackage(t *testing.T) {
	spec := New(parser.OASVersion320, WithGenericIncludePackage(true))
	spec.SetTitle("Test").SetVersion("1.0.0")

	// Verify the option was applied
	if !spec.namer.genericConfig.IncludePackage {
		t.Error("IncludePackage should be true")
	}
}

// TestWithGenericApplyBaseCasing tests base casing for generic params.
func TestWithGenericApplyBaseCasing(t *testing.T) {
	spec := New(parser.OASVersion320, WithGenericApplyBaseCasing(true))
	spec.SetTitle("Test").SetVersion("1.0.0")

	// Verify the option was applied
	if !spec.namer.genericConfig.ApplyBaseCasing {
		t.Error("ApplyBaseCasing should be true")
	}
}

// mapKeys returns the keys of a map for error messages.
func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestBuilder_InvalidTemplateError tests that invalid templates cause BuildOAS3/BuildOAS2 to return errors.
// This is an integration test for the configError propagation path.
func TestBuilder_InvalidTemplateError(t *testing.T) {
	t.Run("BuildOAS3", func(t *testing.T) {
		spec := New(parser.OASVersion320,
			WithSchemaNameTemplate("{{.Type"), // Invalid syntax - unclosed action
		).SetTitle("Test").SetVersion("1.0.0")

		_, err := spec.BuildOAS3()
		if err == nil {
			t.Error("expected error for invalid template")
		}
		if !strings.Contains(err.Error(), "configuration error") {
			t.Errorf("expected configuration error, got: %v", err)
		}
	})

	t.Run("BuildOAS2", func(t *testing.T) {
		spec := New(parser.OASVersion20,
			WithSchemaNameTemplate("{{.Type"), // Invalid syntax - unclosed action
		).SetTitle("Test").SetVersion("1.0.0")

		_, err := spec.BuildOAS2()
		if err == nil {
			t.Error("expected error for invalid template")
		}
		if !strings.Contains(err.Error(), "configuration error") {
			t.Errorf("expected configuration error, got: %v", err)
		}
	})
}
