package builder

import (
	"reflect"
	"text/template"

	"github.com/erraggy/oastools/parser"
)

// BuilderOption configures a Builder instance.
// Options are applied when creating a new Builder with New().
type BuilderOption func(*builderConfig)

// SchemaFieldProcessor is called for each struct field during schema generation.
// It receives the generated schema and the struct field, and returns a potentially
// modified schema. The processor is called after oastools applies any oas:"..." tags.
//
// This hook enables libraries to support custom tag formats alongside the standard
// oas:"..." tags. For example, a library could support legacy standalone tags:
//
//	type User struct {
//	    Name string `json:"name" description:"User's full name"`
//	}
//
// The processor can read these tags and apply them to the schema:
//
//	func(schema *parser.Schema, field reflect.StructField) *parser.Schema {
//	    if desc := field.Tag.Get("description"); desc != "" {
//	        schema.Description = desc
//	    }
//	    return schema
//	}
type SchemaFieldProcessor func(schema *parser.Schema, field reflect.StructField) *parser.Schema

// builderConfig holds builder configuration applied via options.
type builderConfig struct {
	namingStrategy        SchemaNamingStrategy
	namingTemplate        *template.Template
	namingFunc            SchemaNameFunc
	genericConfig         GenericNamingConfig
	templateError         error // Stores template parse errors for Build() to return
	semanticDeduplication bool  // Enable semantic schema deduplication
	schemaFieldProcessor  SchemaFieldProcessor
}

// defaultBuilderConfig returns a new builderConfig with default values.
// The defaults produce backward-compatible behavior: SchemaNamingDefault
// produces "package.TypeName" format, and GenericNamingUnderscore replaces
// brackets with underscores.
//
// This function is used by New() when processing BuilderOptions.
func defaultBuilderConfig() *builderConfig {
	return &builderConfig{
		namingStrategy: SchemaNamingDefault,
		genericConfig:  DefaultGenericNamingConfig(),
	}
}

// WithSchemaNaming sets a built-in schema naming strategy.
// The default is SchemaNamingDefault which produces "package.TypeName" format.
//
// Available strategies:
//   - SchemaNamingDefault: "package.TypeName" (e.g., models.User)
//   - SchemaNamingPascalCase: "PackageTypeName" (e.g., ModelsUser)
//   - SchemaNamingCamelCase: "packageTypeName" (e.g., modelsUser)
//   - SchemaNamingSnakeCase: "package_type_name" (e.g., models_user)
//   - SchemaNamingKebabCase: "package-type-name" (e.g., models-user)
//   - SchemaNamingTypeOnly: "TypeName" (e.g., User) - may cause conflicts
//   - SchemaNamingFullPath: "full_path_TypeName" (e.g., github.com_org_models_User)
//
// Setting a naming strategy clears any previously set template or custom function.
func WithSchemaNaming(strategy SchemaNamingStrategy) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.namingStrategy = strategy
		cfg.namingTemplate = nil
		cfg.namingFunc = nil
		cfg.templateError = nil
	}
}

// WithSchemaNameTemplate sets a custom Go text/template for schema naming.
// Template receives SchemaNameContext with type metadata.
// Template parse errors are returned by Build*() methods.
//
// Available template functions: pascal, camel, snake, kebab, upper, lower,
// title, sanitize, trimPrefix, trimSuffix, replace, join.
//
// Available context fields:
//   - .Type: Go type name without package (e.g., "User", "Response[T]")
//   - .TypeSanitized: Type with generic brackets replaced per GenericNamingStrategy
//   - .TypeBase: Base type name without generic parameters (e.g., "Response")
//   - .Package: Package base name (e.g., "models")
//   - .PackagePath: Full import path (e.g., "github.com/org/models")
//   - .PackagePathSanitized: PackagePath with slashes replaced
//   - .IsGeneric: Whether the type has type parameters
//   - .GenericParams: Type parameter names if IsGeneric is true
//   - .GenericParamsSanitized: Sanitized type parameter names
//   - .GenericSuffix: Formatted generic parameters portion
//   - .IsAnonymous: Whether this is an anonymous struct type
//   - .IsPointer: Whether the original type was a pointer
//   - .Kind: The reflect.Kind as a string
//
// Example:
//
//	WithSchemaNameTemplate(`{{pascal .Package}}{{pascal .Type}}`)
//
// Template parse errors are returned by BuildOAS3() or BuildOAS2().
// If template execution fails at runtime for a specific type (e.g., due to
// accessing an invalid field), the naming falls back to the default
// "package.TypeName" format silently. Ensure templates are tested with
// representative types to avoid unexpected fallback behavior.
//
// Setting a template clears any previously set custom function.
func WithSchemaNameTemplate(tmpl string) BuilderOption {
	return func(cfg *builderConfig) {
		t, err := parseSchemaNameTemplate(tmpl)
		if err != nil {
			cfg.templateError = err
			cfg.namingTemplate = nil
			return
		}
		cfg.namingTemplate = t
		cfg.namingFunc = nil
		cfg.templateError = nil
	}
}

// WithSchemaNameFunc sets a custom function for schema naming.
// Provides maximum flexibility for programmatic naming logic.
// The function receives SchemaNameContext and returns the schema name.
//
// Example:
//
//	WithSchemaNameFunc(func(ctx builder.SchemaNameContext) string {
//	    if ctx.IsAnonymous {
//	        return "AnonymousType"
//	    }
//	    return strings.ToUpper(ctx.Package) + "_" + ctx.Type
//	})
//
// Setting a custom function clears any previously set template.
func WithSchemaNameFunc(fn SchemaNameFunc) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.namingFunc = fn
		cfg.namingTemplate = nil
		cfg.templateError = nil
	}
}

// WithGenericNaming sets the strategy for handling generic type names.
// The default is GenericNamingUnderscore which produces "Response_User_" format.
//
// Available strategies:
//   - GenericNamingUnderscore: "Response_User_" (default)
//   - GenericNamingOf: "ResponseOfUser"
//   - GenericNamingFor: "ResponseForUser"
//   - GenericNamingAngleBrackets: "Response<User>" (URI-encoded in $ref)
//   - GenericNamingFlattened: "ResponseUser"
func WithGenericNaming(strategy GenericNamingStrategy) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.genericConfig.Strategy = strategy
	}
}

// WithGenericNamingConfig provides fine-grained control over generic type naming.
// This replaces any previous generic naming settings.
//
// Example:
//
//	WithGenericNamingConfig(builder.GenericNamingConfig{
//	    Strategy:        builder.GenericNamingOf,
//	    ParamSeparator:  "And",
//	    ApplyBaseCasing: true,
//	})
func WithGenericNamingConfig(config GenericNamingConfig) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.genericConfig = config
	}
}

// WithGenericSeparator sets the separator used for generic type parameters.
// Only applies to GenericNamingUnderscore strategy.
// Default is "_".
//
// Example:
//
//	WithGenericSeparator("__")
//	// Response[User] becomes Response__User__
func WithGenericSeparator(sep string) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.genericConfig.Separator = sep
	}
}

// WithGenericParamSeparator sets the separator between multiple type parameters.
// Default is "_".
//
// Example:
//
//	WithGenericParamSeparator("And")
//	// Map[string,int] with GenericNamingOf becomes MapOfStringAndOfInt
func WithGenericParamSeparator(sep string) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.genericConfig.ParamSeparator = sep
	}
}

// WithGenericIncludePackage includes package names in generic type parameters.
// When true, Response[models.User] becomes Response_models_User_.
// Default is false.
func WithGenericIncludePackage(include bool) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.genericConfig.IncludePackage = include
	}
}

// WithGenericApplyBaseCasing applies the base naming strategy to type parameters.
// When true with SchemaNamingPascalCase, Response[user_profile] becomes ResponseOfUserProfile.
// Default is false.
func WithGenericApplyBaseCasing(apply bool) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.genericConfig.ApplyBaseCasing = apply
	}
}

// WithSemanticDeduplication enables semantic schema deduplication.
// When enabled, the builder identifies schemas that are structurally identical
// and consolidates them to a single canonical schema. All references to
// duplicate schemas are rewritten to point to the canonical schema.
//
// The canonical schema name is selected alphabetically (e.g., if "Address"
// and "Location" are identical, "Address" becomes canonical).
//
// This option reduces document size when multiple types converge to the
// same structure. It is disabled by default.
//
// Example:
//
//	spec := builder.New(parser.OASVersion320,
//	    builder.WithSemanticDeduplication(true),
//	)
func WithSemanticDeduplication(enabled bool) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.semanticDeduplication = enabled
	}
}

// WithSchemaFieldProcessor sets a custom function for processing struct field tags.
// This enables libraries to support custom tag formats alongside the standard oas:"..." tags.
//
// The processor is called for each struct field after the base schema is generated
// and after any oas:"..." tags are applied. It receives the generated schema and
// the reflect.StructField, allowing access to all struct tags.
//
// Example - supporting legacy standalone tags:
//
//	spec := builder.New(parser.OASVersion320,
//	    builder.WithSchemaFieldProcessor(func(schema *parser.Schema, field reflect.StructField) *parser.Schema {
//	        // Skip if oas tag is present (already processed)
//	        if field.Tag.Get("oas") != "" {
//	            return schema
//	        }
//	        // Apply legacy description tag
//	        if desc := field.Tag.Get("description"); desc != "" {
//	            schema.Description = desc
//	        }
//	        // Apply legacy enum tag
//	        if enumStr := field.Tag.Get("enum"); enumStr != "" {
//	            values := strings.Split(enumStr, "|")
//	            schema.Enum = make([]any, len(values))
//	            for i, v := range values {
//	                schema.Enum[i] = strings.TrimSpace(v)
//	            }
//	        }
//	        return schema
//	    }),
//	)
//
// Use cases:
//   - Migration support from other OpenAPI libraries with different tag formats
//   - Custom validation tag integration (e.g., reading from validator tags)
//   - Documentation generators pulling from godoc-style comments or other sources
//   - Framework integration with existing struct tag conventions
func WithSchemaFieldProcessor(fn SchemaFieldProcessor) BuilderOption {
	return func(cfg *builderConfig) {
		cfg.schemaFieldProcessor = fn
	}
}
