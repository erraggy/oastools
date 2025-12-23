package generator

import (
	"bytes"
	"embed"
	"strconv"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl templates/*/*.tmpl
var templateFS embed.FS

var templates *template.Template

func init() {
	var err error
	templates, err = template.New("").
		Funcs(templateFuncs).
		ParseFS(templateFS, "templates/*.tmpl", "templates/*/*.tmpl")
	if err != nil {
		panic(err)
	}
}

// templateFuncs provides custom functions for templates
var templateFuncs = template.FuncMap{
	// String manipulation
	"quote":     strconv.Quote,
	"join":      strings.Join,
	"upper":     strings.ToUpper,
	"lower":     strings.ToLower,
	"hasSuffix": strings.HasSuffix,
	"hasPrefix": strings.HasPrefix,

	// Custom helpers
	"zeroValue":       zeroValue,
	"cleanDesc":       cleanDescription,
	"toTypeName":      toTypeName,
	"toFieldName":     toFieldName,
	"toParamName":     toParamName,
	"trimPointer":     trimPointer,
	"methodToChiFunc": methodToChiFunc,
}

// trimPointer removes the leading * from a pointer type string
func trimPointer(s string) string {
	return strings.TrimPrefix(s, "*")
}

// methodToChiFunc converts an HTTP method to its chi router function name.
// Example: "GET" -> "Get", "POST" -> "Post"
func methodToChiFunc(method string) string {
	// Chi methods are title-cased (Get, Post, Put, Delete, etc.)
	method = strings.ToUpper(method)
	if len(method) == 0 {
		return ""
	}
	return strings.ToUpper(method[:1]) + strings.ToLower(method[1:])
}

// executeTemplate executes a template by name and returns the formatted bytes
func executeTemplate(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, err
	}

	// Format the output and fix imports using goimports-equivalent processing
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		// If formatting fails, return unformatted but don't fail the generation
		// nolint:nilerr // intentional: formatting is optional, unformatted code is acceptable
		return buf.Bytes(), nil
	}
	return formatted, nil
}
