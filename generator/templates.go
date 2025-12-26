package generator

import (
	"bytes"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/erraggy/oastools/internal/httputil"
)

//go:embed templates/*.tmpl templates/*/*.tmpl
var templateFS embed.FS

var (
	templates     *template.Template
	templatesOnce sync.Once
	templatesErr  error
)

// getTemplates returns the parsed templates, initializing them lazily on first call.
// This avoids panicking in init() and allows errors to be propagated to callers.
func getTemplates() (*template.Template, error) {
	templatesOnce.Do(func() {
		templates, templatesErr = template.New("").
			Funcs(templateFuncs).
			ParseFS(templateFS, "templates/*.tmpl", "templates/*/*.tmpl")
		if templatesErr != nil {
			templatesErr = fmt.Errorf("generator: failed to parse templates: %w", templatesErr)
		}
	})
	return templates, templatesErr
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

// executeTemplate executes a template by name and returns the formatted bytes.
// The second return value indicates whether formatting succeeded (true) or failed (false).
// When formatting fails, unformatted but valid Go code is returned.
func executeTemplate(name string, data any) ([]byte, bool, error) {
	tmpl, err := getTemplates()
	if err != nil {
		return nil, false, err
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, false, fmt.Errorf("generator: failed to execute template %s: %w", name, err)
	}

	// Format the output and fix imports using goimports-equivalent processing
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		// If formatting fails, return unformatted but don't fail the generation.
		// Return false to indicate formatting failed so callers can warn users.
		//nolint:nilerr // intentional: generation succeeds, only formatting failed
		return buf.Bytes(), false, nil
	}
	return formatted, true, nil
}
