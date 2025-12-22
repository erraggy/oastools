package httpvalidator

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// Validator validates HTTP requests and responses against an OpenAPI specification.
// It supports both OAS 2.0 (Swagger) and OAS 3.x specifications.
//
// Create a Validator using the New function:
//
//	parsed, _ := parser.ParseWithOptions(parser.WithFilePath("openapi.yaml"))
//	v, err := httpvalidator.New(parsed)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	result, err := v.ValidateRequest(req)
//	if !result.Valid {
//	    // Handle validation errors
//	}
type Validator struct {
	// parsed holds the parsed OpenAPI specification
	parsed *parser.ParseResult

	// pathMatcherSet handles path template matching
	pathMatcherSet *PathMatcherSet

	// schemaValidator handles JSON Schema validation of data
	schemaValidator *SchemaValidator

	// IncludeWarnings determines whether to include best practice warnings
	// in validation results. Default is true.
	IncludeWarnings bool

	// StrictMode enables stricter validation behavior:
	// - Rejects requests with unknown query parameters
	// - Rejects requests with unknown headers
	// - Rejects responses with undocumented status codes
	StrictMode bool
}

// New creates a new HTTP Validator from a parsed OpenAPI specification.
// The validator pre-compiles path matchers for efficient matching.
//
// Returns an error if the parsed result is nil or contains invalid path templates.
func New(parsed *parser.ParseResult) (*Validator, error) {
	if parsed == nil {
		return nil, fmt.Errorf("httpvalidator: parsed result cannot be nil")
	}

	v := &Validator{
		parsed:          parsed,
		schemaValidator: NewSchemaValidator(),
		IncludeWarnings: true,
		StrictMode:      false,
	}

	// Pre-compile all path matchers
	if err := v.initPathMatchers(); err != nil {
		return nil, err
	}

	return v, nil
}

// initPathMatchers pre-compiles regex patterns for all paths in the spec.
func (v *Validator) initPathMatchers() error {
	paths := v.getPaths()
	if len(paths) == 0 {
		// Empty spec is valid but will match no requests
		v.pathMatcherSet = &PathMatcherSet{}
		return nil
	}

	templates := make([]string, 0, len(paths))
	for template := range paths {
		templates = append(templates, template)
	}

	matcherSet, err := NewPathMatcherSet(templates)
	if err != nil {
		return fmt.Errorf("httpvalidator: %w", err)
	}

	v.pathMatcherSet = matcherSet
	return nil
}

// getPaths returns the paths map from the parsed specification.
func (v *Validator) getPaths() map[string]*parser.PathItem {
	if v.parsed.IsOAS3() {
		if doc, ok := v.parsed.OAS3Document(); ok && doc.Paths != nil {
			return doc.Paths
		}
	} else if v.parsed.IsOAS2() {
		if doc, ok := v.parsed.OAS2Document(); ok && doc.Paths != nil {
			return doc.Paths
		}
	}
	return nil
}

// getPathItem returns the PathItem for the given path template.
func (v *Validator) getPathItem(pathTemplate string) *parser.PathItem {
	paths := v.getPaths()
	if paths == nil {
		return nil
	}
	return paths[pathTemplate]
}

// getOperation returns the Operation for the given path and HTTP method.
func (v *Validator) getOperation(pathTemplate, method string) *parser.Operation {
	pathItem := v.getPathItem(pathTemplate)
	if pathItem == nil {
		return nil
	}

	switch strings.ToUpper(method) {
	case http.MethodGet:
		return pathItem.Get
	case http.MethodPost:
		return pathItem.Post
	case http.MethodPut:
		return pathItem.Put
	case http.MethodDelete:
		return pathItem.Delete
	case http.MethodPatch:
		return pathItem.Patch
	case http.MethodHead:
		return pathItem.Head
	case http.MethodOptions:
		return pathItem.Options
	case http.MethodTrace:
		return pathItem.Trace
	default:
		return nil
	}
}

// matchPath finds the matching path template for the given request path.
func (v *Validator) matchPath(requestPath string) (template string, params map[string]string, found bool) {
	if v.pathMatcherSet == nil {
		return "", nil, false
	}
	return v.pathMatcherSet.Match(requestPath)
}

// getParameters returns all parameters for an operation, including path-level parameters.
// Operation-level parameters override path-level parameters with the same name and location.
func (v *Validator) getParameters(pathTemplate string, operation *parser.Operation) []*parser.Parameter {
	pathItem := v.getPathItem(pathTemplate)
	if pathItem == nil {
		return operation.Parameters
	}

	// Merge path-level and operation-level parameters
	// Operation parameters override path parameters with same name+in
	paramMap := make(map[string]*parser.Parameter)

	// Add path-level parameters first
	for _, p := range pathItem.Parameters {
		if p != nil {
			key := p.In + ":" + p.Name
			paramMap[key] = p
		}
	}

	// Override with operation-level parameters
	if operation != nil {
		for _, p := range operation.Parameters {
			if p != nil {
				key := p.In + ":" + p.Name
				paramMap[key] = p
			}
		}
	}

	// Convert back to slice
	result := make([]*parser.Parameter, 0, len(paramMap))
	for _, p := range paramMap {
		result = append(result, p)
	}

	return result
}

// getParametersByLocation returns parameters filtered by location (path, query, header, cookie).
func (v *Validator) getParametersByLocation(pathTemplate string, operation *parser.Operation, location string) []*parser.Parameter {
	all := v.getParameters(pathTemplate, operation)
	result := make([]*parser.Parameter, 0)

	for _, p := range all {
		if p != nil && p.In == location {
			result = append(result, p)
		}
	}

	return result
}

// ValidateRequest validates an HTTP request against the OpenAPI specification.
// It checks path parameters, query parameters, headers, cookies, and request body.
//
// Returns a RequestValidationResult containing validation errors and extracted parameters.
// The error return is reserved for internal errors (e.g., body reading failures),
// not validation errors which are captured in the result.
func (v *Validator) ValidateRequest(req *http.Request) (*RequestValidationResult, error) {
	result := newRequestResult()

	// 1. Find matching path
	matchedPath, pathParams, found := v.matchPath(req.URL.Path)
	if !found {
		result.addError(req.URL.Path, fmt.Sprintf("no matching path found for %s", req.URL.Path), SeverityError)
		return result, nil
	}
	result.MatchedPath = matchedPath
	result.MatchedMethod = req.Method

	// 2. Get operation for method
	operation := v.getOperation(matchedPath, req.Method)
	if operation == nil {
		result.addError(
			fmt.Sprintf("%s.%s", matchedPath, strings.ToLower(req.Method)),
			fmt.Sprintf("method %s not allowed for path %s", req.Method, matchedPath),
			SeverityError,
		)
		return result, nil
	}

	// 3. Validate path parameters
	v.validatePathParams(pathParams, matchedPath, operation, result)

	// 4. Validate query parameters
	v.validateQueryParams(req, matchedPath, operation, result)

	// 5. Validate header parameters
	v.validateHeaderParams(req, matchedPath, operation, result)

	// 6. Validate cookie parameters
	v.validateCookieParams(req, matchedPath, operation, result)

	// 7. Validate request body
	v.validateRequestBody(req, matchedPath, operation, result)

	return result, nil
}

// ValidateResponse validates an HTTP response against the OpenAPI specification.
// It checks the status code, response headers, and response body.
//
// The original request is needed to determine which operation's response to validate against.
//
// Returns a ResponseValidationResult containing validation errors.
// The error return is reserved for internal errors (e.g., body reading failures),
// not validation errors which are captured in the result.
func (v *Validator) ValidateResponse(req *http.Request, resp *http.Response) (*ResponseValidationResult, error) {
	result := newResponseResult()
	result.StatusCode = resp.StatusCode
	result.ContentType = resp.Header.Get("Content-Type")

	// Find matching path and operation from the original request
	matchedPath, _, found := v.matchPath(req.URL.Path)
	if !found {
		result.addError(req.URL.Path, "no matching path found for request", SeverityError)
		return result, nil
	}
	result.MatchedPath = matchedPath
	result.MatchedMethod = req.Method

	operation := v.getOperation(matchedPath, req.Method)
	if operation == nil {
		result.addError(
			fmt.Sprintf("%s.%s", matchedPath, strings.ToLower(req.Method)),
			fmt.Sprintf("no operation found for %s %s", req.Method, matchedPath),
			SeverityError,
		)
		return result, nil
	}

	// Validate response
	v.validateResponse(resp, matchedPath, operation, result)

	return result, nil
}

// IsOAS3 returns true if the specification is OAS 3.x.
func (v *Validator) IsOAS3() bool {
	return v.parsed.IsOAS3()
}

// IsOAS2 returns true if the specification is OAS 2.0.
func (v *Validator) IsOAS2() bool {
	return v.parsed.IsOAS2()
}
