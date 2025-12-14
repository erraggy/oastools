package builder

import (
	"fmt"
	"path"
	"reflect"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// SchemaNamingStrategy defines built-in schema naming conventions.
// Use these with WithSchemaNaming to control how schema names are generated
// from Go types.
type SchemaNamingStrategy int

const (
	// SchemaNamingDefault uses "package.TypeName" format (current behavior).
	// Example: models.User
	SchemaNamingDefault SchemaNamingStrategy = iota

	// SchemaNamingPascalCase uses "PackageTypeName" format.
	// Example: models.User -> ModelsUser
	SchemaNamingPascalCase

	// SchemaNamingCamelCase uses "packageTypeName" format.
	// Example: models.User -> modelsUser
	SchemaNamingCamelCase

	// SchemaNamingSnakeCase uses "package_type_name" format.
	// Example: models.User -> models_user
	SchemaNamingSnakeCase

	// SchemaNamingKebabCase uses "package-type-name" format.
	// Example: models.User -> models-user
	SchemaNamingKebabCase

	// SchemaNamingTypeOnly uses just "TypeName" without package.
	// Example: models.User -> User
	// Warning: May cause conflicts with same-named types in different packages.
	SchemaNamingTypeOnly

	// SchemaNamingFullPath uses full package path.
	// Example: models.User -> github.com_org_models_User
	SchemaNamingFullPath
)

// anonymousTypeName is the schema name used for anonymous struct types.
const anonymousTypeName = "AnonymousType"

// GenericNamingStrategy defines how generic type parameters are formatted
// in schema names.
type GenericNamingStrategy int

const (
	// GenericNamingUnderscore replaces brackets with underscores (default behavior).
	// Example: Response[User] -> Response_User_
	GenericNamingUnderscore GenericNamingStrategy = iota

	// GenericNamingOf uses "Of" separator between base type and parameters.
	// Example: Response[User] -> ResponseOfUser
	GenericNamingOf

	// GenericNamingFor uses "For" separator.
	// Example: Response[User] -> ResponseForUser
	GenericNamingFor

	// GenericNamingAngleBrackets uses angle brackets (URI-encoded in $ref).
	// Example: Response[User] -> Response<User>
	// Note: Produces Response%3CUser%3E in $ref URIs.
	GenericNamingAngleBrackets

	// GenericNamingFlattened removes brackets entirely.
	// Example: Response[User] -> ResponseUser
	GenericNamingFlattened
)

// GenericNamingConfig provides fine-grained control over generic type naming.
type GenericNamingConfig struct {
	// Strategy is the primary generic naming approach.
	Strategy GenericNamingStrategy

	// Separator is used between base type and parameters.
	// Only applies to GenericNamingUnderscore strategy.
	// Default: "_"
	Separator string

	// ParamSeparator is used between multiple type parameters.
	// Example with ParamSeparator="_": Map[string,int] -> Map_string_int
	// Example with ParamSeparator="And": Map[string,int] -> MapOfStringAndOfInt
	// Default: "_"
	ParamSeparator string

	// IncludePackage includes the type parameter's package in the name.
	// Example: Response[models.User] -> Response_models_User (true)
	// Example: Response[models.User] -> Response_User (false, default)
	IncludePackage bool

	// ApplyBaseCasing applies the base naming strategy to type parameters.
	// Example with SchemaNamingPascalCase: Response[user_profile] -> ResponseOfUserProfile
	ApplyBaseCasing bool
}

// DefaultGenericNamingConfig returns the default generic naming configuration.
// This matches the current behavior where brackets are replaced with underscores.
func DefaultGenericNamingConfig() GenericNamingConfig {
	return GenericNamingConfig{
		Strategy:        GenericNamingUnderscore,
		Separator:       "_",
		ParamSeparator:  "_",
		IncludePackage:  false,
		ApplyBaseCasing: false,
	}
}

// SchemaNameContext provides type metadata for custom naming templates
// and functions. All fields are populated before being passed to
// templates or custom naming functions.
type SchemaNameContext struct {
	// Type is the Go type name without package (e.g., "User", "Response[T]").
	Type string

	// TypeSanitized is Type with generic brackets replaced per GenericNamingStrategy.
	TypeSanitized string

	// TypeBase is the base type name without generic parameters (e.g., "Response").
	TypeBase string

	// Package is the package base name (e.g., "models").
	Package string

	// PackagePath is the full import path (e.g., "github.com/org/models").
	PackagePath string

	// PackagePathSanitized is PackagePath with slashes replaced
	// (e.g., "github.com_org_models").
	PackagePathSanitized string

	// IsGeneric indicates if the type has type parameters.
	IsGeneric bool

	// GenericParams contains the type parameter names if IsGeneric is true.
	GenericParams []string

	// GenericParamsSanitized contains sanitized type parameter names.
	GenericParamsSanitized []string

	// GenericSuffix is the formatted generic parameters portion.
	// Varies based on GenericNamingStrategy (e.g., "_User_", "OfUser", "<User>").
	GenericSuffix string

	// IsAnonymous indicates if this is an anonymous struct type.
	IsAnonymous bool

	// IsPointer indicates if the original type was a pointer.
	IsPointer bool

	// Kind is the reflect.Kind as a string (e.g., "struct", "slice", "map").
	Kind string
}

// SchemaNameFunc is the signature for custom schema naming functions.
// The function receives a SchemaNameContext with complete type metadata
// and should return the desired schema name.
type SchemaNameFunc func(ctx SchemaNameContext) string

// schemaNamer handles schema name generation with configurable strategies.
type schemaNamer struct {
	strategy      SchemaNamingStrategy
	genericConfig GenericNamingConfig
	template      *template.Template
	fn            SchemaNameFunc
}

// newSchemaNamer creates a namer with the default strategy.
// The default strategy produces "package.TypeName" format with underscores
// replacing generic brackets, matching the original behavior.
func newSchemaNamer() *schemaNamer {
	return &schemaNamer{
		strategy:      SchemaNamingDefault,
		genericConfig: DefaultGenericNamingConfig(),
	}
}

// name generates a schema name for the given type.
// The name is generated according to the configured strategy, template, or function.
// Priority: custom function > template > built-in strategy.
func (n *schemaNamer) name(t reflect.Type) string {
	ctx := n.buildContext(t)

	// Custom function takes highest priority
	if n.fn != nil {
		return n.fn(ctx)
	}

	// Template takes second priority
	if n.template != nil {
		var buf strings.Builder
		if err := n.template.Execute(&buf, ctx); err != nil {
			// Fall back to default on template error
			return n.defaultName(ctx)
		}
		return sanitizeSchemaName(buf.String())
	}

	// Built-in strategy
	return n.applyStrategy(ctx)
}

// nameWithConflictCheck generates a schema name with conflict detection.
// If the initial name causes a conflict (as determined by checkConflict),
// a disambiguated name using the full package path is returned.
func (n *schemaNamer) nameWithConflictCheck(t reflect.Type, checkConflict func(name string) bool) string {
	name := n.name(t)

	// Check for conflicts
	if checkConflict(name) {
		// Use full package path to disambiguate
		ctx := n.buildContext(t)
		if ctx.PackagePathSanitized != "" {
			name = ctx.PackagePathSanitized + "_" + ctx.TypeSanitized
		}
	}

	return name
}

// buildContext creates SchemaNameContext from a reflect.Type.
// This populates all fields needed for naming strategies, templates, and custom functions.
func (n *schemaNamer) buildContext(t reflect.Type) SchemaNameContext {
	// Unwrap pointer types
	isPointer := false
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
		isPointer = true
	}

	typeName := t.Name()
	pkgPath := t.PkgPath()

	ctx := SchemaNameContext{
		Type:                 typeName,
		Package:              path.Base(pkgPath),
		PackagePath:          pkgPath,
		PackagePathSanitized: sanitizePath(pkgPath),
		IsAnonymous:          typeName == "",
		IsPointer:            isPointer,
		Kind:                 t.Kind().String(),
	}

	// Handle anonymous types
	if ctx.IsAnonymous {
		ctx.TypeBase = ""
		ctx.TypeSanitized = ""
		return ctx
	}

	// Handle generic types
	if strings.Contains(typeName, "[") {
		ctx.IsGeneric = true
		ctx.TypeBase = extractBaseTypeName(typeName)
		ctx.GenericParams = extractGenericParams(typeName)
		ctx.GenericParamsSanitized = n.sanitizeGenericParams(ctx.GenericParams)
		ctx.GenericSuffix = n.formatGenericSuffix(ctx.GenericParamsSanitized)
		ctx.TypeSanitized = ctx.TypeBase + ctx.GenericSuffix
	} else {
		ctx.TypeBase = typeName
		ctx.TypeSanitized = typeName
	}

	return ctx
}

// sanitizeGenericParams processes generic parameters according to configuration.
// It removes package prefixes (unless IncludePackage is true), applies
// base casing if configured, and sanitizes any nested brackets.
func (n *schemaNamer) sanitizeGenericParams(params []string) []string {
	result := make([]string, len(params))
	for i, param := range params {
		// Remove package prefix if not including package
		if !n.genericConfig.IncludePackage {
			if idx := strings.LastIndex(param, "."); idx != -1 {
				param = param[idx+1:]
			}
		} else {
			// Sanitize package path in parameter (replace dots with underscores)
			param = strings.ReplaceAll(param, ".", "_")
		}

		// Sanitize any nested brackets in the parameter (e.g., List[User] -> List_User)
		param = sanitizeSchemaName(param)

		// Apply base casing if configured
		if n.genericConfig.ApplyBaseCasing {
			param = n.applyCasing(param)
		}

		result[i] = param
	}
	return result
}

// formatGenericSuffix creates the generic portion of the name based on
// the configured generic naming strategy.
func (n *schemaNamer) formatGenericSuffix(params []string) string {
	if len(params) == 0 {
		return ""
	}

	switch n.genericConfig.Strategy {
	case GenericNamingOf:
		return "Of" + strings.Join(params, n.genericConfig.ParamSeparator+"Of")

	case GenericNamingFor:
		return "For" + strings.Join(params, n.genericConfig.ParamSeparator+"For")

	case GenericNamingAngleBrackets:
		return "<" + strings.Join(params, ",") + ">"

	case GenericNamingFlattened:
		return strings.Join(params, "")

	default: // GenericNamingUnderscore
		sep := n.genericConfig.Separator
		if sep == "" {
			sep = "_"
		}
		paramSep := n.genericConfig.ParamSeparator
		if paramSep == "" {
			paramSep = "_"
		}
		return sep + strings.Join(params, paramSep) + sep
	}
}

// applyCasing applies the current naming strategy's casing to a string.
// This is used for generic type parameters when ApplyBaseCasing is true.
func (n *schemaNamer) applyCasing(s string) string {
	switch n.strategy {
	case SchemaNamingPascalCase:
		return toPascalCase(s)
	case SchemaNamingCamelCase:
		return toCamelCase(s)
	case SchemaNamingSnakeCase:
		return toSnakeCase(s)
	case SchemaNamingKebabCase:
		return toKebabCase(s)
	default:
		return s
	}
}

// applyStrategy applies a built-in naming strategy to generate the schema name.
func (n *schemaNamer) applyStrategy(ctx SchemaNameContext) string {
	if ctx.IsAnonymous {
		return anonymousTypeName
	}

	switch n.strategy {
	case SchemaNamingPascalCase:
		return toPascalCase(ctx.Package) + toPascalCase(ctx.TypeSanitized)

	case SchemaNamingCamelCase:
		return toCamelCase(ctx.Package) + toPascalCase(ctx.TypeSanitized)

	case SchemaNamingSnakeCase:
		base := toSnakeCase(ctx.Package)
		typePart := toSnakeCase(ctx.TypeSanitized)
		if base == "" {
			return typePart
		}
		return base + "_" + typePart

	case SchemaNamingKebabCase:
		base := toKebabCase(ctx.Package)
		typePart := toKebabCase(ctx.TypeSanitized)
		if base == "" {
			return typePart
		}
		return base + "-" + typePart

	case SchemaNamingTypeOnly:
		return ctx.TypeSanitized

	case SchemaNamingFullPath:
		if ctx.PackagePathSanitized == "" {
			return ctx.TypeSanitized
		}
		return ctx.PackagePathSanitized + "_" + ctx.TypeSanitized

	default: // SchemaNamingDefault
		return n.defaultName(ctx)
	}
}

// defaultName generates the default package.TypeName format.
// This matches the original schemaName() behavior for backward compatibility.
func (n *schemaNamer) defaultName(ctx SchemaNameContext) string {
	if ctx.IsAnonymous {
		return anonymousTypeName
	}
	if ctx.Package == "" {
		return ctx.TypeSanitized
	}
	return ctx.Package + "." + ctx.TypeSanitized
}

// extractBaseTypeName extracts the base type name from a generic type.
// Example: "Response[User]" -> "Response"
func extractBaseTypeName(name string) string {
	if idx := strings.Index(name, "["); idx != -1 {
		return name[:idx]
	}
	return name
}

// extractGenericParams extracts type parameters from a generic type name.
// It handles nested generics by counting bracket depth.
// Example: "Response[User]" -> ["User"]
// Example: "Map[string,int]" -> ["string", "int"]
// Example: "Response[List[User]]" -> ["List[User]"]
func extractGenericParams(name string) []string {
	start := strings.Index(name, "[")
	end := strings.LastIndex(name, "]")
	if start == -1 || end == -1 || end <= start {
		return nil
	}

	paramStr := name[start+1 : end]

	// Handle nested generics by counting bracket depth
	var params []string
	var current strings.Builder
	depth := 0

	for _, r := range paramStr {
		switch r {
		case '[':
			depth++
			current.WriteRune(r)
		case ']':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				// Top-level comma - end of parameter
				params = append(params, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				// Nested comma - part of parameter
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	// Add final parameter
	if current.Len() > 0 {
		params = append(params, strings.TrimSpace(current.String()))
	}

	return params
}

// sanitizePath replaces path separators with underscores.
// Example: "github.com/org/models" -> "github.com_org_models"
func sanitizePath(s string) string {
	return strings.ReplaceAll(s, "/", "_")
}

// sanitizeSchemaName replaces characters that are problematic in URIs.
// This is especially important for generic types which include brackets.
// Example: "Response[User]" -> "Response_User"
func sanitizeSchemaName(name string) string {
	// Replace brackets used in generic types
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	// Replace commas (multi-type generics)
	name = strings.ReplaceAll(name, ",", "_")
	// Replace spaces (shouldn't occur but be safe)
	name = strings.ReplaceAll(name, " ", "_")
	// Clean up multiple consecutive underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	// Remove trailing underscore
	name = strings.TrimSuffix(name, "_")
	return name
}

// toPascalCase converts a string to PascalCase.
// Separators (underscore, hyphen, dot, slash) trigger capitalization of the next letter.
// Example: "user_profile" -> "UserProfile"
// Example: "api-client" -> "ApiClient"
func toPascalCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	capitalizeNext := true

	for _, r := range s {
		if r == '_' || r == '-' || r == '.' || r == '/' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// toCamelCase converts a string to camelCase.
// Like PascalCase but with the first letter lowercase.
// Example: "user_profile" -> "userProfile"
// Example: "UserProfile" -> "userProfile"
func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if pascal == "" {
		return ""
	}
	runes := []rune(pascal)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// toSnakeCase converts a string to snake_case.
// Uppercase letters are prefixed with underscore and lowercased.
// Existing separators (hyphen, dot, slash) are converted to underscores.
// Example: "UserProfile" -> "user_profile"
// Example: "APIClient" -> "api_client"
func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else if r == '-' || r == '.' || r == '/' {
			result.WriteRune('_')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// toKebabCase converts a string to kebab-case.
// Like snake_case but with hyphens instead of underscores.
// Example: "UserProfile" -> "user-profile"
func toKebabCase(s string) string {
	return strings.ReplaceAll(toSnakeCase(s), "_", "-")
}

// templateFuncs returns the function map for schema name templates.
// These functions are available in templates passed to WithSchemaNameTemplate.
func templateFuncs() template.FuncMap {
	// Use golang.org/x/text/cases for proper title casing (strings.Title is deprecated)
	titleCaser := cases.Title(language.English)

	return template.FuncMap{
		"pascal":     toPascalCase,
		"camel":      toCamelCase,
		"snake":      toSnakeCase,
		"kebab":      toKebabCase,
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      titleCaser.String,
		"sanitize":   sanitizeSchemaName,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"replace":    strings.ReplaceAll,
		"join": func(sep string, parts ...string) string {
			return strings.Join(parts, sep)
		},
	}
}

// parseSchemaNameTemplate parses and validates a schema name template.
// The template is validated by executing it with a sample context.
// Returns an error if the template is syntactically invalid or fails execution.
func parseSchemaNameTemplate(tmpl string) (*template.Template, error) {
	t, err := template.New("schemaName").Funcs(templateFuncs()).Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("builder: invalid schema name template: %w", err)
	}

	// Validate template with sample context
	ctx := SchemaNameContext{
		Type:                   "TestType",
		TypeSanitized:          "TestType",
		TypeBase:               "TestType",
		Package:                "testpkg",
		PackagePath:            "github.com/test/testpkg",
		PackagePathSanitized:   "github.com_test_testpkg",
		IsGeneric:              false,
		GenericParams:          nil,
		GenericParamsSanitized: nil,
		GenericSuffix:          "",
		IsAnonymous:            false,
		IsPointer:              false,
		Kind:                   "struct",
	}
	var buf strings.Builder
	if err := t.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("builder: schema name template execution failed: %w", err)
	}

	return t, nil
}
