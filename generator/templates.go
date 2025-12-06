// Package generator provides code generation from OpenAPI specifications.
//
// TEMPLATE REFACTORING STRATEGY:
//
// This package is in the process of refactoring from string-based code generation
// (using bytes.Buffer.WriteString()) to Go text/template-based generation.
//
// CURRENT STATE:
// - Infrastructure for templates is in place (templates.go, template_data.go)
// - Template files are embedded (.tmpl files in templates/ directory)
// - Existing generation functions (oas3_generator.go, oas2_generator.go) remain unchanged
//
// NEXT STEPS:
// 1. Create builder functions that convert parser types to template data structures
// 2. Refactor generation methods to build template data instead of using WriteString
// 3. Execute templates to generate output
// 4. Verify output is byte-for-byte identical to original
// 5. Remove old WriteString-based code
//
// DESIGN PRINCIPLE:
// Templates should handle OUTPUT FORMATTING ONLY. All complex logic (type resolution,
// ref handling, conditionals) happens in Go code that builds the template data.
// This keeps templates simple, readable, and maintainable.
//
// EXAMPLE:
// Instead of:
//   buf.WriteString(fmt.Sprintf("type %s struct {\n", typeName))
// We do:
//   data := buildStructData(typeName, schema, generator)
//   executeTemplate("struct.go.tmpl", data)
package generator

import (
	"bytes"
	"embed"
	"go/format"
	"strconv"
	"strings"
	"text/template"
)

//go:embed templates/*/*.tmpl
var templateFS embed.FS

var templates *template.Template

func init() {
	var err error
	templates, err = template.New("").
		Funcs(templateFuncs).
		ParseFS(templateFS, "templates/*/*.tmpl")
	if err != nil {
		panic(err)
	}
}

// templateFuncs provides custom functions for templates
var templateFuncs = template.FuncMap{
	// String manipulation
	"quote":   strconv.Quote,
	"join":    strings.Join,
	"upper":   strings.ToUpper,
	"lower":   strings.ToLower,
	"hasSuffix": strings.HasSuffix,
	"hasPrefix": strings.HasPrefix,

	// Custom helpers
	"zeroValue":   zeroValue,
	"cleanDesc":   cleanDescription,
	"toTypeName":  toTypeName,
	"toFieldName": toFieldName,
	"toParamName": toParamName,
}

// executeTemplate executes a template by name and returns the formatted bytes
func executeTemplate(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, err
	}

	// Format the output using go/format
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// If formatting fails, return unformatted but don't fail the generation
		return buf.Bytes(), nil
	}
	return formatted, nil
}
