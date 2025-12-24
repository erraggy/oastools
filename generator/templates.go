package generator

import (
	"bytes"
	"embed"
	"strconv"
	"strings"
	"text/template"

	"github.com/erraggy/oastools/internal/httputil"
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
// Example: "GET" -> "Get", "POST" -> "Post".
// For non-standard methods like "QUERY" (OAS 3.2+), returns an empty string
// to signal the template to use chi's generic Method() function.
func methodToChiFunc(method string) string {
	if len(method) == 0 {
		return ""
	}

	// Map standard HTTP methods to their chi router counterparts.
	// Uses httputil constants for consistency with the rest of the codebase.
	// For non-standard methods (like QUERY), return empty string to signal
	// the template to use chi.Method() instead.
	switch {
	case strings.EqualFold(method, httputil.MethodGet):
		return "Get"
	case strings.EqualFold(method, httputil.MethodPost):
		return "Post"
	case strings.EqualFold(method, httputil.MethodPut):
		return "Put"
	case strings.EqualFold(method, httputil.MethodDelete):
		return "Delete"
	case strings.EqualFold(method, httputil.MethodPatch):
		return "Patch"
	case strings.EqualFold(method, httputil.MethodHead):
		return "Head"
	case strings.EqualFold(method, httputil.MethodOptions):
		return "Options"
	case strings.EqualFold(method, httputil.MethodConnect):
		return "Connect"
	case strings.EqualFold(method, httputil.MethodTrace):
		return "Trace"
	default:
		// Non-standard method (like QUERY) - return empty to signal template
		return ""
	}
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
