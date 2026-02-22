package doctest

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeepDiveOptionTables verifies that every exported With* function in each
// package appears in its deep_dive.md option table, and vice versa.
func TestDeepDiveOptionTables(t *testing.T) {
	// Resolve the repo root from this test file's location.
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller(0) failed to retrieve file path")
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	packages := []struct {
		name string
		dir  string // relative to repo root
	}{
		{"parser", "parser"},
		{"validator", "validator"},
		{"fixer", "fixer"},
		{"converter", "converter"},
		{"joiner", "joiner"},
		{"differ", "differ"},
		{"overlay", "overlay"},
		{"httpvalidator", "httpvalidator"},
		{"generator", "generator"},
		{"builder", "builder"},
		{"walker", "walker"},
	}

	// Known exceptions: With* functions intentionally not documented in deep_dive.md.
	// Each entry maps package name -> set of function names to skip for source->doc checks.
	sourceExceptions := map[string]map[string]bool{
		// WithLogger is an internal debugging option, not part of the public API docs.
		"parser": {"WithLogger": true},

		// Programmatic overlay variants: file-path versions are documented instead.
		// WithSourceMap and WithUserAgent are utility plumbing options.
		"converter": {
			"WithPreConversionOverlay":  true,
			"WithPostConversionOverlay": true,
			"WithSourceMap":             true,
			"WithUserAgent":             true,
		},

		// Many advanced configuration options not yet in the option table.
		// Programmatic overlay variants (file-path versions are documented).
		"joiner": {
			"WithDefaultStrategy":        true,
			"WithDeduplicateTags":        true,
			"WithMergeArrays":            true,
			"WithRenameTemplate":         true,
			"WithNamespacePrefix":        true,
			"WithAlwaysApplyPrefix":      true,
			"WithEquivalenceMode":        true,
			"WithOperationContext":       true,
			"WithPrimaryOperationPolicy": true,
			"WithPreJoinOverlay":         true,
			"WithPostJoinOverlay":        true,
			"WithSpecOverlay":            true,
			"WithSpecOverlayFile":        true,
		},

		// The differ deep_dive.md documents options in code examples rather than
		// a formal option table. All With* functions appear in examples.
		"differ": {
			"WithSourceFilePath": true,
			"WithTargetFilePath": true,
			"WithSourceParsed":   true,
			"WithTargetParsed":   true,
			"WithMode":           true,
			"WithIncludeInfo":    true,
			"WithBreakingRules":  true,
			"WithUserAgent":      true,
		},

		// Generator has long-form aliases (WithGenerate*) and utility options.
		"generator": {
			// Long-form aliases: WithSecurity -> WithGenerateSecurity, etc.
			"WithGenerateSecurity":        true,
			"WithGenerateOAuth2Flows":     true,
			"WithGenerateCredentialMgmt":  true,
			"WithGenerateSecurityEnforce": true,
			"WithGenerateOIDCDiscovery":   true,
			"WithGenerateReadme":          true,
			// Utility and less common options.
			"WithPointers":             true,
			"WithValidation":           true,
			"WithStrictMode":           true,
			"WithIncludeInfo":          true,
			"WithUserAgent":            true,
			"WithSourceMap":            true,
			"WithMaxTypesPerFile":      true,
			"WithMaxOperationsPerFile": true,
			"WithSplitByPathPrefix":    true,
		},

		// Builder has many specialized option types (ParamOption, TagOption,
		// ServerOption, etc.) with fine-grained options documented in Go examples
		// rather than the summary option table.
		"builder": {
			// Tag options
			"WithTagDescription":  true,
			"WithTagExternalDocs": true,
			// Server options
			"WithServerDescription":         true,
			"WithServerVariable":            true,
			"WithServerVariableEnum":        true,
			"WithServerVariableDescription": true,
			// Response detail options
			"WithResponseContentType": true,
			"WithResponseExample":     true,
			"WithResponseHeader":      true,
			"WithResponseRawSchema":   true,
			"WithResponseRef":         true,
			"WithDefaultResponse":     true,
			// Request body detail options
			"WithRequestBodyRawSchema": true,
			"WithRequestDescription":   true,
			"WithRequestExample":       true,
			"WithRequired":             true,
			// Operation detail options
			"WithHandler":      true,
			"WithHandlerFunc":  true,
			"WithNoSecurity":   true,
			"WithParameter":    true,
			"WithParameterRef": true,
			"WithFileParam":    true,
			"WithFormParam":    true,
			// Parameter validation options
			"WithParamDefault":          true,
			"WithParamDeprecated":       true,
			"WithParamEnum":             true,
			"WithParamExample":          true,
			"WithParamExclusiveMaximum": true,
			"WithParamExclusiveMinimum": true,
			"WithParamMaximum":          true,
			"WithParamMaxItems":         true,
			"WithParamMaxLength":        true,
			"WithParamMinimum":          true,
			"WithParamMinItems":         true,
			"WithParamMinLength":        true,
			"WithParamMultipleOf":       true,
			"WithParamPattern":          true,
			"WithParamUniqueItems":      true,
			// Builder-level configuration options
			"WithGenericNaming":          true,
			"WithGenericNamingConfig":    true,
			"WithGenericSeparator":       true,
			"WithGenericParamSeparator":  true,
			"WithGenericIncludePackage":  true,
			"WithGenericApplyBaseCasing": true,
			"WithSchemaFieldProcessor":   true,
			"WithoutValidation":          true,
		},

		// Walker documents handler options in prose/examples and post handlers
		// in a table. Many individual handler registrations are not in a
		// consolidated option table.
		"walker": {
			"WithDocumentHandler":       true,
			"WithOAS2DocumentHandler":   true,
			"WithOAS3DocumentHandler":   true,
			"WithInfoHandler":           true,
			"WithServerHandler":         true,
			"WithTagHandler":            true,
			"WithPathHandler":           true,
			"WithPathItemHandler":       true,
			"WithParameterHandler":      true,
			"WithRequestBodyHandler":    true,
			"WithResponseHandler":       true,
			"WithSecuritySchemeHandler": true,
			"WithHeaderHandler":         true,
			"WithMediaTypeHandler":      true,
			"WithLinkHandler":           true,
			"WithCallbackHandler":       true,
			"WithExampleHandler":        true,
			"WithExternalDocsHandler":   true,
			"WithFilePath":              true,
			"WithParsed":                true,
			"WithContext":               true,
		},
	}

	// Known exceptions: With* names that appear in docs but don't correspond
	// to an actual exported function in the package source.
	docExceptions := map[string]map[string]bool{
		// WithMessage appears in joiner docs as a helper variant pattern,
		// not an actual exported function.
		"joiner": {"WithMessage": true},

		// The overlay deep_dive.md references converter options
		// (WithPreConversionOverlayFile, WithPostConversionOverlayFile)
		// that belong to the converter package, not overlay.
		"overlay": {
			"WithPreConversionOverlayFile":  true,
			"WithPostConversionOverlayFile": true,
		},

		// WithErrorHandler is referenced in generator docs but is actually
		// a builder.ServerBuilderOption, not a generator.Option.
		"generator": {"WithErrorHandler": true},
	}

	for _, pkg := range packages {
		t.Run(pkg.name, func(t *testing.T) {
			pkgDir := filepath.Join(repoRoot, pkg.dir)
			deepDivePath := filepath.Join(pkgDir, "deep_dive.md")

			// Extract With* functions from Go source.
			sourceOpts := extractWithFunctions(t, pkgDir)
			if len(sourceOpts) == 0 {
				t.Skipf("no With* functions found in %s", pkg.dir)
			}

			// Extract With* names from deep_dive.md.
			docOpts := extractDocOptions(t, deepDivePath)

			srcExc := sourceExceptions[pkg.name]
			docExc := docExceptions[pkg.name]

			// Check: every source With* function must appear in the doc.
			for _, fn := range sourceOpts {
				if srcExc[fn] {
					continue
				}
				assert.True(t, docOpts[fn], "function %s() exists in %s/ source but is not referenced in deep_dive.md", fn, pkg.name)
			}

			// Check: every documented With* must exist in source.
			sourceSet := make(map[string]bool, len(sourceOpts))
			for _, fn := range sourceOpts {
				sourceSet[fn] = true
			}
			for fn := range docOpts {
				if docExc[fn] {
					continue
				}
				assert.True(t, sourceSet[fn], "deep_dive.md references %s() but no such function exists in %s/ source", fn, pkg.name)
			}

			// Check: sourceExceptions entries are not stale.
			for fn := range srcExc {
				assert.True(t, sourceSet[fn], "sourceExceptions lists %s for %s/ but no such function exists in source (stale exception?)", fn, pkg.name)
			}

			// Check: docExceptions entries are not stale.
			for fn := range docExc {
				assert.True(t, docOpts[fn], "docExceptions lists %s for %s/ but no such reference exists in deep_dive.md (stale exception?)", fn, pkg.name)
			}
		})
	}
}

// extractWithFunctions uses go/ast to find all exported With* functions
// (not methods) in the given package directory, excluding test files.
func extractWithFunctions(t *testing.T, dir string) []string {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := goparser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	require.NoError(t, err, "parsing package dir %s", dir)

	var funcs []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil {
					continue
				}
				if fn.Name.IsExported() && strings.HasPrefix(fn.Name.Name, "With") {
					funcs = append(funcs, fn.Name.Name)
				}
			}
		}
	}
	return funcs
}

// extractDocOptions parses a deep_dive.md file and extracts With* function
// names from backtick-wrapped references (option tables, code mentions, etc.).
// Matches patterns like: `WithFoo(...)` or `WithFoo` in backticks.
func extractDocOptions(t *testing.T, path string) map[string]bool {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading %s", path)

	// Match backtick-wrapped With* names: `WithSomething(` or `WithSomething`
	// NOTE: Uses With[a-zA-Z] (not With[A-Z]) to also match Without* patterns.
	// This intentionally does NOT match With* in fenced code blocks (```go ... ```).
	// Functions documented only in code examples must be added to sourceExceptions.
	re := regexp.MustCompile("`(With[a-zA-Z][a-zA-Z0-9]*)(?:\\(|`)")

	result := make(map[string]bool)
	for _, match := range re.FindAllStringSubmatch(string(data), -1) {
		if len(match) > 1 {
			result[match[1]] = true
		}
	}
	return result
}
