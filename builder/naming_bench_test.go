package builder

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/erraggy/oastools/parser"
)

// Test types for naming benchmarks
type benchUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type benchResponse[T any] struct {
	Data  T      `json:"data"`
	Error string `json:"error,omitempty"`
}

type benchOrder struct {
	ID     int      `json:"id"`
	UserID int      `json:"user_id"`
	Items  []string `json:"items"`
	Total  float64  `json:"total"`
}

type benchNested[T any] struct {
	Inner benchResponse[T] `json:"inner"`
}

// BenchmarkSchemaNaming benchmarks each built-in schema naming strategy.
func BenchmarkSchemaNaming(b *testing.B) {
	strategies := []struct {
		name     string
		strategy SchemaNamingStrategy
	}{
		{"default", SchemaNamingDefault},
		{"pascal", SchemaNamingPascalCase},
		{"camel", SchemaNamingCamelCase},
		{"snake", SchemaNamingSnakeCase},
		{"kebab", SchemaNamingKebabCase},
		{"type_only", SchemaNamingTypeOnly},
		{"full_path", SchemaNamingFullPath},
	}

	for _, s := range strategies {
		b.Run(s.name, func(b *testing.B) {
			for b.Loop() {
				spec := New(parser.OASVersion320,
					WithSchemaNaming(s.strategy),
				).
					SetTitle("Benchmark API").
					SetVersion("1.0.0").
					AddOperation(http.MethodGet, "/users",
						WithResponse(http.StatusOK, []benchUser{}),
					)
				_, _ = spec.BuildOAS3()
			}
		})
	}
}

// BenchmarkGenericNaming benchmarks each generic naming strategy.
func BenchmarkGenericNaming(b *testing.B) {
	strategies := []struct {
		name     string
		strategy GenericNamingStrategy
	}{
		{"underscore", GenericNamingUnderscore},
		{"of", GenericNamingOf},
		{"for", GenericNamingFor},
		{"flat", GenericNamingFlattened},
		{"angle_brackets", GenericNamingAngleBrackets},
	}

	for _, s := range strategies {
		b.Run(s.name, func(b *testing.B) {
			for b.Loop() {
				spec := New(parser.OASVersion320,
					WithGenericNaming(s.strategy),
				).
					SetTitle("Benchmark API").
					SetVersion("1.0.0").
					AddOperation(http.MethodGet, "/users",
						WithResponse(http.StatusOK, benchResponse[benchUser]{}),
					)
				_, _ = spec.BuildOAS3()
			}
		})
	}
}

// BenchmarkSchemaNameTemplate benchmarks custom template execution with varying complexity.
func BenchmarkSchemaNameTemplate(b *testing.B) {
	templates := []struct {
		name string
		tmpl string
	}{
		{"simple", `{{.Type}}`},
		{"pascal", `{{pascal .Package}}{{pascal .Type}}`},
		{"complex", `{{if .IsGeneric}}Generic_{{end}}{{snake .Package}}_{{pascal .Type}}`},
		{"generic_aware", `{{.TypeBase}}{{if .IsGeneric}}Of{{range .GenericParamsSanitized}}{{pascal .}}{{end}}{{end}}`},
	}

	for _, tt := range templates {
		b.Run(tt.name, func(b *testing.B) {
			for b.Loop() {
				spec := New(parser.OASVersion320,
					WithSchemaNameTemplate(tt.tmpl),
				).
					SetTitle("Benchmark API").
					SetVersion("1.0.0").
					AddOperation(http.MethodGet, "/users",
						WithResponse(http.StatusOK, benchResponse[benchUser]{}),
					)
				_, _ = spec.BuildOAS3()
			}
		})
	}
}

// BenchmarkCaseConversions benchmarks individual case conversion functions.
func BenchmarkCaseConversions(b *testing.B) {
	input := "UserProfileSettings"

	b.Run("toPascalCase", func(b *testing.B) {
		for b.Loop() {
			_ = toPascalCase(input)
		}
	})

	b.Run("toCamelCase", func(b *testing.B) {
		for b.Loop() {
			_ = toCamelCase(input)
		}
	})

	b.Run("toSnakeCase", func(b *testing.B) {
		for b.Loop() {
			_ = toSnakeCase(input)
		}
	})

	b.Run("toKebabCase", func(b *testing.B) {
		for b.Loop() {
			_ = toKebabCase(input)
		}
	})
}

// BenchmarkCaseConversions_Separators benchmarks case conversions with separator-heavy inputs.
func BenchmarkCaseConversions_Separators(b *testing.B) {
	input := "user_profile_settings_data"

	b.Run("toPascalCase", func(b *testing.B) {
		for b.Loop() {
			_ = toPascalCase(input)
		}
	})

	b.Run("toCamelCase", func(b *testing.B) {
		for b.Loop() {
			_ = toCamelCase(input)
		}
	})

	b.Run("toSnakeCase", func(b *testing.B) {
		for b.Loop() {
			_ = toSnakeCase(input)
		}
	})

	b.Run("toKebabCase", func(b *testing.B) {
		for b.Loop() {
			_ = toKebabCase(input)
		}
	})
}

// BenchmarkExtractGenericParams benchmarks generic parameter extraction with varying complexity.
func BenchmarkExtractGenericParams(b *testing.B) {
	inputs := []struct {
		name  string
		input string
	}{
		{"simple", "Response[User]"},
		{"multi", "Map[string,int]"},
		{"nested", "Response[List[User]]"},
		{"triple", "Tuple[A,B,C]"},
		{"deeply_nested", "Response[Map[string,List[User]]]"},
		{"none", "SimpleType"},
	}

	for _, tt := range inputs {
		b.Run(tt.name, func(b *testing.B) {
			for b.Loop() {
				_ = extractGenericParams(tt.input)
			}
		})
	}
}

// BenchmarkExtractBaseTypeName benchmarks base type name extraction.
func BenchmarkExtractBaseTypeName(b *testing.B) {
	inputs := []struct {
		name  string
		input string
	}{
		{"generic", "Response[User]"},
		{"non_generic", "SimpleType"},
		{"nested", "Response[List[User]]"},
	}

	for _, tt := range inputs {
		b.Run(tt.name, func(b *testing.B) {
			for b.Loop() {
				_ = extractBaseTypeName(tt.input)
			}
		})
	}
}

// BenchmarkSanitizeSchemaName benchmarks schema name sanitization.
func BenchmarkSanitizeSchemaName(b *testing.B) {
	inputs := []struct {
		name  string
		input string
	}{
		{"clean", "SimpleType"},
		{"brackets", "Response[User]"},
		{"complex", "Map[string,List[User]]"},
		{"spaces", "Some Type Name"},
	}

	for _, tt := range inputs {
		b.Run(tt.name, func(b *testing.B) {
			for b.Loop() {
				_ = sanitizeSchemaName(tt.input)
			}
		})
	}
}

// BenchmarkSchemaNamer_Name benchmarks the schemaNamer.name method directly.
func BenchmarkSchemaNamer_Name(b *testing.B) {
	// Get reflect.Type for our test types
	userType := reflect.TypeOf(benchUser{})
	responseType := reflect.TypeOf(benchResponse[benchUser]{})
	orderType := reflect.TypeOf(benchOrder{})

	b.Run("default_simple", func(b *testing.B) {
		namer := newSchemaNamer()
		for b.Loop() {
			_ = namer.name(userType)
		}
	})

	b.Run("default_generic", func(b *testing.B) {
		namer := newSchemaNamer()
		for b.Loop() {
			_ = namer.name(responseType)
		}
	})

	b.Run("pascal_simple", func(b *testing.B) {
		namer := newSchemaNamer()
		namer.strategy = SchemaNamingPascalCase
		for b.Loop() {
			_ = namer.name(userType)
		}
	})

	b.Run("pascal_generic", func(b *testing.B) {
		namer := newSchemaNamer()
		namer.strategy = SchemaNamingPascalCase
		for b.Loop() {
			_ = namer.name(responseType)
		}
	})

	b.Run("snake_nested", func(b *testing.B) {
		namer := newSchemaNamer()
		namer.strategy = SchemaNamingSnakeCase
		for b.Loop() {
			_ = namer.name(orderType)
		}
	})
}

// BenchmarkSchemaNamer_BuildContext benchmarks context building separately.
func BenchmarkSchemaNamer_BuildContext(b *testing.B) {
	userType := reflect.TypeOf(benchUser{})
	responseType := reflect.TypeOf(benchResponse[benchUser]{})
	nestedType := reflect.TypeOf(benchNested[benchUser]{})

	b.Run("simple", func(b *testing.B) {
		namer := newSchemaNamer()
		for b.Loop() {
			_ = namer.buildContext(userType)
		}
	})

	b.Run("generic", func(b *testing.B) {
		namer := newSchemaNamer()
		for b.Loop() {
			_ = namer.buildContext(responseType)
		}
	})

	b.Run("nested_generic", func(b *testing.B) {
		namer := newSchemaNamer()
		for b.Loop() {
			_ = namer.buildContext(nestedType)
		}
	})
}

// BenchmarkGenericNamingConfig benchmarks fine-grained generic naming configurations.
func BenchmarkGenericNamingConfig(b *testing.B) {
	configs := []struct {
		name   string
		config GenericNamingConfig
	}{
		{"default", DefaultGenericNamingConfig()},
		{"of_strategy", GenericNamingConfig{Strategy: GenericNamingOf}},
		{"include_package", GenericNamingConfig{Strategy: GenericNamingUnderscore, IncludePackage: true}},
		{"apply_casing", GenericNamingConfig{Strategy: GenericNamingOf, ApplyBaseCasing: true}},
		{"custom_separators", GenericNamingConfig{
			Strategy:       GenericNamingUnderscore,
			Separator:      "__",
			ParamSeparator: "_",
		}},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			for b.Loop() {
				spec := New(parser.OASVersion320,
					WithGenericNamingConfig(cfg.config),
				).
					SetTitle("Benchmark API").
					SetVersion("1.0.0").
					AddOperation(http.MethodGet, "/users",
						WithResponse(http.StatusOK, benchResponse[benchUser]{}),
					)
				_, _ = spec.BuildOAS3()
			}
		})
	}
}

// BenchmarkBuilderWithNamingOptions benchmarks complete builder workflows with different naming options.
func BenchmarkBuilderWithNamingOptions(b *testing.B) {
	b.Run("default", func(b *testing.B) {
		for b.Loop() {
			spec := New(parser.OASVersion320).
				SetTitle("Benchmark API").
				SetVersion("1.0.0").
				AddOperation(http.MethodGet, "/users",
					WithResponse(http.StatusOK, []benchUser{}),
				).
				AddOperation(http.MethodGet, "/orders",
					WithResponse(http.StatusOK, []benchOrder{}),
				).
				AddOperation(http.MethodGet, "/responses",
					WithResponse(http.StatusOK, benchResponse[benchUser]{}),
				)
			_, _ = spec.BuildOAS3()
		}
	})

	b.Run("with_pascal_naming", func(b *testing.B) {
		for b.Loop() {
			spec := New(parser.OASVersion320,
				WithSchemaNaming(SchemaNamingPascalCase),
			).
				SetTitle("Benchmark API").
				SetVersion("1.0.0").
				AddOperation(http.MethodGet, "/users",
					WithResponse(http.StatusOK, []benchUser{}),
				).
				AddOperation(http.MethodGet, "/orders",
					WithResponse(http.StatusOK, []benchOrder{}),
				).
				AddOperation(http.MethodGet, "/responses",
					WithResponse(http.StatusOK, benchResponse[benchUser]{}),
				)
			_, _ = spec.BuildOAS3()
		}
	})

	b.Run("with_template", func(b *testing.B) {
		for b.Loop() {
			spec := New(parser.OASVersion320,
				WithSchemaNameTemplate(`{{pascal .Package}}{{pascal .Type}}`),
			).
				SetTitle("Benchmark API").
				SetVersion("1.0.0").
				AddOperation(http.MethodGet, "/users",
					WithResponse(http.StatusOK, []benchUser{}),
				).
				AddOperation(http.MethodGet, "/orders",
					WithResponse(http.StatusOK, []benchOrder{}),
				).
				AddOperation(http.MethodGet, "/responses",
					WithResponse(http.StatusOK, benchResponse[benchUser]{}),
				)
			_, _ = spec.BuildOAS3()
		}
	})

	b.Run("with_custom_func", func(b *testing.B) {
		nameFn := func(ctx SchemaNameContext) string {
			if ctx.IsAnonymous {
				return "AnonymousType"
			}
			return toPascalCase(ctx.Package) + toPascalCase(ctx.TypeSanitized)
		}
		for b.Loop() {
			spec := New(parser.OASVersion320,
				WithSchemaNameFunc(nameFn),
			).
				SetTitle("Benchmark API").
				SetVersion("1.0.0").
				AddOperation(http.MethodGet, "/users",
					WithResponse(http.StatusOK, []benchUser{}),
				).
				AddOperation(http.MethodGet, "/orders",
					WithResponse(http.StatusOK, []benchOrder{}),
				).
				AddOperation(http.MethodGet, "/responses",
					WithResponse(http.StatusOK, benchResponse[benchUser]{}),
				)
			_, _ = spec.BuildOAS3()
		}
	})

	b.Run("combined_pascal_of", func(b *testing.B) {
		for b.Loop() {
			spec := New(parser.OASVersion320,
				WithSchemaNaming(SchemaNamingPascalCase),
				WithGenericNaming(GenericNamingOf),
			).
				SetTitle("Benchmark API").
				SetVersion("1.0.0").
				AddOperation(http.MethodGet, "/users",
					WithResponse(http.StatusOK, []benchUser{}),
				).
				AddOperation(http.MethodGet, "/orders",
					WithResponse(http.StatusOK, []benchOrder{}),
				).
				AddOperation(http.MethodGet, "/responses",
					WithResponse(http.StatusOK, benchResponse[benchUser]{}),
				)
			_, _ = spec.BuildOAS3()
		}
	})
}

// BenchmarkSchemaNameFunc benchmarks custom naming function execution.
func BenchmarkSchemaNameFunc(b *testing.B) {
	userType := reflect.TypeOf(benchUser{})
	responseType := reflect.TypeOf(benchResponse[benchUser]{})

	// Simple custom function
	simpleFn := func(ctx SchemaNameContext) string {
		return ctx.Type
	}

	// Complex custom function with conditionals
	complexFn := func(ctx SchemaNameContext) string {
		if ctx.IsAnonymous {
			return "AnonymousType"
		}
		if ctx.IsGeneric {
			return ctx.TypeBase + "Generic"
		}
		return toPascalCase(ctx.Package) + "_" + ctx.Type
	}

	b.Run("simple_func_non_generic", func(b *testing.B) {
		namer := newSchemaNamer()
		namer.fn = simpleFn
		for b.Loop() {
			_ = namer.name(userType)
		}
	})

	b.Run("simple_func_generic", func(b *testing.B) {
		namer := newSchemaNamer()
		namer.fn = simpleFn
		for b.Loop() {
			_ = namer.name(responseType)
		}
	})

	b.Run("complex_func_non_generic", func(b *testing.B) {
		namer := newSchemaNamer()
		namer.fn = complexFn
		for b.Loop() {
			_ = namer.name(userType)
		}
	})

	b.Run("complex_func_generic", func(b *testing.B) {
		namer := newSchemaNamer()
		namer.fn = complexFn
		for b.Loop() {
			_ = namer.name(responseType)
		}
	})
}

// BenchmarkFormatGenericSuffix benchmarks generic suffix formatting for each strategy.
func BenchmarkFormatGenericSuffix(b *testing.B) {
	params := []string{"User", "Order"}

	strategies := []struct {
		name   string
		config GenericNamingConfig
	}{
		{"underscore", GenericNamingConfig{Strategy: GenericNamingUnderscore, Separator: "_", ParamSeparator: "_"}},
		{"of", GenericNamingConfig{Strategy: GenericNamingOf}},
		{"for", GenericNamingConfig{Strategy: GenericNamingFor}},
		{"flat", GenericNamingConfig{Strategy: GenericNamingFlattened}},
		{"angle", GenericNamingConfig{Strategy: GenericNamingAngleBrackets}},
	}

	for _, s := range strategies {
		b.Run(s.name, func(b *testing.B) {
			namer := newSchemaNamer()
			namer.genericConfig = s.config
			for b.Loop() {
				_ = namer.formatGenericSuffix(params)
			}
		})
	}
}

// BenchmarkSanitizeGenericParams benchmarks generic parameter sanitization.
func BenchmarkSanitizeGenericParams(b *testing.B) {
	params := []string{"models.User", "api.Order", "internal.Config"}

	b.Run("exclude_package", func(b *testing.B) {
		namer := newSchemaNamer()
		namer.genericConfig.IncludePackage = false
		for b.Loop() {
			_ = namer.sanitizeGenericParams(params)
		}
	})

	b.Run("include_package", func(b *testing.B) {
		namer := newSchemaNamer()
		namer.genericConfig.IncludePackage = true
		for b.Loop() {
			_ = namer.sanitizeGenericParams(params)
		}
	})

	b.Run("apply_casing", func(b *testing.B) {
		namer := newSchemaNamer()
		namer.strategy = SchemaNamingPascalCase
		namer.genericConfig.ApplyBaseCasing = true
		for b.Loop() {
			_ = namer.sanitizeGenericParams(params)
		}
	})
}

// BenchmarkTemplateParsing benchmarks template parsing (one-time cost).
func BenchmarkTemplateParsing(b *testing.B) {
	templates := []struct {
		name string
		tmpl string
	}{
		{"simple", `{{.Type}}`},
		{"pascal", `{{pascal .Package}}{{pascal .Type}}`},
		{"complex", `{{if .IsGeneric}}Generic_{{end}}{{snake .Package}}_{{pascal .Type}}`},
		{"full", `{{.TypeBase}}{{if .IsGeneric}}Of{{range .GenericParamsSanitized}}{{pascal .}}{{end}}{{end}}`},
	}

	for _, tt := range templates {
		b.Run(tt.name, func(b *testing.B) {
			for b.Loop() {
				_, _ = parseSchemaNameTemplate(tt.tmpl)
			}
		})
	}
}

// BenchmarkSanitizePath benchmarks path sanitization.
func BenchmarkSanitizePath(b *testing.B) {
	paths := []struct {
		name string
		path string
	}{
		{"short", "models"},
		{"medium", "github.com/org/models"},
		{"long", "github.com/organization/project/internal/models/v2"},
	}

	for _, p := range paths {
		b.Run(p.name, func(b *testing.B) {
			for b.Loop() {
				_ = sanitizePath(p.path)
			}
		})
	}
}
