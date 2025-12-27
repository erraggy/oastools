package httpvalidator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// validatePathParams validates extracted path parameters against the spec.
func (v *Validator) validatePathParams(pathParams map[string]string, pathTemplate string, operation *parser.Operation, result *RequestValidationResult) {
	paramDefs := v.getParametersByLocation(pathTemplate, operation, "path")
	deserializer := NewParamDeserializer()

	// Build a map of defined parameters
	defined := make(map[string]*parser.Parameter)
	for _, param := range paramDefs {
		defined[param.Name] = param
	}

	// Check each extracted path parameter
	for name, rawValue := range pathParams {
		param, ok := defined[name]
		if !ok {
			// This shouldn't happen if spec and path matcher are in sync
			result.addWarning(
				fmt.Sprintf("path.%s", name),
				fmt.Sprintf("path parameter %q not defined in specification", name),
			)
			result.PathParams[name] = rawValue
			continue
		}

		// Deserialize according to style
		value := deserializer.DeserializePathParam(rawValue, param)
		result.PathParams[name] = value

		// Validate against schema
		if param.Schema != nil {
			errors := v.schemaValidator.Validate(value, param.Schema, fmt.Sprintf("path.%s", name))
			for _, err := range errors {
				result.addError(err.Path, err.Message, err.Severity)
			}
		}
	}

	// Check for required path parameters that are missing
	// (Path params are always required per OAS spec, but check anyway)
	for name, param := range defined {
		if _, found := pathParams[name]; !found {
			if param.Required {
				result.addError(
					fmt.Sprintf("path.%s", name),
					fmt.Sprintf("required path parameter %q is missing", name),
					SeverityError,
				)
			}
		}
	}
}

// validateQueryParams validates query parameters against the spec.
func (v *Validator) validateQueryParams(req *http.Request, pathTemplate string, operation *parser.Operation, result *RequestValidationResult) {
	paramDefs := v.getParametersByLocation(pathTemplate, operation, "query")
	deserializer := NewParamDeserializer()
	queryValues := req.URL.Query()

	// Build a map of defined parameters
	defined := make(map[string]*parser.Parameter)
	for _, param := range paramDefs {
		defined[param.Name] = param
	}

	// Track which query params we've processed
	processed := make(map[string]bool)

	// Validate each defined query parameter
	for name, param := range defined {
		values, present := queryValues[name]

		if !present {
			// Check for deepObject style parameters
			if param.Style == "deepObject" && param.Schema != nil {
				deepValues := deserializer.DeserializeQueryParamsDeepObject(queryValues, name, param.Schema)
				if len(deepValues) > 0 {
					result.QueryParams[name] = deepValues
					// Mark all related keys as processed
					prefix := name + "["
					for key := range queryValues {
						if strings.HasPrefix(key, prefix) {
							processed[key] = true
						}
					}
					// Validate the deserialized object
					errors := v.schemaValidator.Validate(deepValues, param.Schema, fmt.Sprintf("query.%s", name))
					for _, err := range errors {
						result.addError(err.Path, err.Message, err.Severity)
					}
					continue
				}
			}

			if param.Required {
				result.addError(
					fmt.Sprintf("query.%s", name),
					fmt.Sprintf("required query parameter %q is missing", name),
					SeverityError,
				)
			}
			continue
		}

		processed[name] = true

		// Check for empty value if not allowed
		if len(values) == 1 && values[0] == "" && !param.AllowEmptyValue {
			if v.IncludeWarnings {
				result.addWarning(
					fmt.Sprintf("query.%s", name),
					fmt.Sprintf("query parameter %q has empty value", name),
				)
			}
		}

		// Deserialize according to style
		value := deserializer.DeserializeQueryParam(values, param)
		result.QueryParams[name] = value

		// Validate against schema
		if param.Schema != nil {
			errors := v.schemaValidator.Validate(value, param.Schema, fmt.Sprintf("query.%s", name))
			for _, err := range errors {
				result.addError(err.Path, err.Message, err.Severity)
			}
		}
	}

	// In strict mode, reject unknown query parameters
	if v.StrictMode {
		for key := range queryValues {
			if !processed[key] {
				result.addError(
					fmt.Sprintf("query.%s", key),
					fmt.Sprintf("unknown query parameter %q", key),
					SeverityError,
				)
			}
		}
	}
}

// validateHeaderParams validates header parameters against the spec.
func (v *Validator) validateHeaderParams(req *http.Request, pathTemplate string, operation *parser.Operation, result *RequestValidationResult) {
	paramDefs := v.getParametersByLocation(pathTemplate, operation, "header")
	deserializer := NewParamDeserializer()

	// Build a map of defined parameters (case-insensitive)
	defined := make(map[string]*parser.Parameter)
	for _, param := range paramDefs {
		defined[strings.ToLower(param.Name)] = param
	}

	// Track which headers we've processed
	processed := make(map[string]bool)

	// Validate each defined header parameter
	for lowerName, param := range defined {
		// HTTP headers are case-insensitive, so check with canonical form
		canonicalName := http.CanonicalHeaderKey(param.Name)
		value := req.Header.Get(canonicalName)

		if value == "" {
			// Check if header is present but empty vs not present at all
			_, present := req.Header[canonicalName]
			if !present && param.Required {
				result.addError(
					fmt.Sprintf("header.%s", param.Name),
					fmt.Sprintf("required header parameter %q is missing", param.Name),
					SeverityError,
				)
			}
			continue
		}

		processed[lowerName] = true

		// Deserialize according to style
		deserializedValue := deserializer.DeserializeHeaderParam(value, param)
		result.HeaderParams[param.Name] = deserializedValue

		// Validate against schema using redacting validator to prevent
		// credential leakage in error messages (headers may contain Authorization, etc.)
		if param.Schema != nil {
			errors := v.sensitiveSchemaValidator.Validate(deserializedValue, param.Schema, fmt.Sprintf("header.%s", param.Name))
			for _, err := range errors {
				result.addError(err.Path, err.Message, err.Severity)
			}
		}
	}

	// In strict mode, reject unknown header parameters (excluding standard headers)
	if v.StrictMode {
		standardHeaders := map[string]bool{
			"accept": true, "accept-charset": true, "accept-encoding": true,
			"accept-language": true, "authorization": true, "cache-control": true,
			"connection": true, "content-length": true, "content-type": true,
			"cookie": true, "host": true, "origin": true, "referer": true,
			"user-agent": true, "x-forwarded-for": true, "x-forwarded-host": true,
			"x-forwarded-proto": true, "x-real-ip": true, "x-request-id": true,
		}

		for headerName := range req.Header {
			lowerName := strings.ToLower(headerName)
			if !processed[lowerName] && !standardHeaders[lowerName] && !strings.HasPrefix(lowerName, "sec-") {
				result.addError(
					fmt.Sprintf("header.%s", headerName),
					fmt.Sprintf("unknown header parameter %q", headerName),
					SeverityError,
				)
			}
		}
	}
}

// validateCookieParams validates cookie parameters against the spec.
func (v *Validator) validateCookieParams(req *http.Request, pathTemplate string, operation *parser.Operation, result *RequestValidationResult) {
	paramDefs := v.getParametersByLocation(pathTemplate, operation, "cookie")
	deserializer := NewParamDeserializer()

	// Build a map of defined parameters
	defined := make(map[string]*parser.Parameter)
	for _, param := range paramDefs {
		defined[param.Name] = param
	}

	// Track which cookies we've processed
	processed := make(map[string]bool)

	// Validate each defined cookie parameter
	for name, param := range defined {
		cookie, err := req.Cookie(name)

		if errors.Is(err, http.ErrNoCookie) {
			if param.Required {
				result.addError(
					fmt.Sprintf("cookie.%s", name),
					fmt.Sprintf("required cookie parameter %q is missing", name),
					SeverityError,
				)
			}
			continue
		}

		processed[name] = true

		// Deserialize according to style
		value := deserializer.DeserializeCookieParam(cookie.Value, param)
		result.CookieParams[name] = value

		// Validate against schema using redacting validator to prevent
		// credential leakage in error messages (cookies may contain session tokens, etc.)
		if param.Schema != nil {
			errs := v.sensitiveSchemaValidator.Validate(value, param.Schema, fmt.Sprintf("cookie.%s", name))
			for _, err := range errs {
				result.addError(err.Path, err.Message, err.Severity)
			}
		}
	}

	// In strict mode, reject unknown cookies
	if v.StrictMode {
		for _, cookie := range req.Cookies() {
			if !processed[cookie.Name] {
				result.addError(
					fmt.Sprintf("cookie.%s", cookie.Name),
					fmt.Sprintf("unknown cookie parameter %q", cookie.Name),
					SeverityError,
				)
			}
		}
	}
}

// validateRequestBody validates the request body against the spec.
func (v *Validator) validateRequestBody(req *http.Request, pathTemplate string, operation *parser.Operation, result *RequestValidationResult) {
	// Get request body definition
	var requestBody *parser.RequestBody
	var bodySchema *parser.Schema
	var bodyRequired bool

	if v.parsed.IsOAS3() {
		requestBody = operation.RequestBody
		if requestBody != nil {
			bodyRequired = requestBody.Required
		}
	} else {
		// OAS 2.0: Find body parameter
		for _, param := range v.getParameters(pathTemplate, operation) {
			if param.In == "body" {
				bodySchema = param.Schema
				bodyRequired = param.Required
				break
			}
		}
	}

	// Check if body is present
	if req.Body == nil || req.ContentLength == 0 {
		if bodyRequired {
			result.addError(
				"requestBody",
				"request body is required but missing",
				SeverityError,
			)
		}
		return
	}

	// Get and validate Content-Type
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		if v.IncludeWarnings {
			result.addWarning("requestBody", "Content-Type header is missing")
		}
		return
	}

	// Parse media type
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		result.addError(
			"requestBody",
			fmt.Sprintf("invalid Content-Type header: %s", contentType),
			SeverityError,
		)
		return
	}

	// Get schema for this content type
	if v.parsed.IsOAS3() && requestBody != nil {
		bodySchema = v.getRequestBodySchema(requestBody, mediaType)
		if bodySchema == nil && v.StrictMode {
			result.addError(
				"requestBody",
				fmt.Sprintf("unsupported Content-Type: %s", mediaType),
				SeverityError,
			)
			return
		}
	}

	// If no schema, we can't validate the body
	if bodySchema == nil {
		return
	}

	// Read and parse body based on content type
	body, readErr := io.ReadAll(req.Body)
	if readErr != nil {
		result.addError(
			"requestBody",
			fmt.Sprintf("failed to read request body: %v", readErr),
			SeverityError,
		)
		return
	}

	// Validate based on media type
	switch {
	case strings.HasPrefix(mediaType, "application/json") || strings.HasSuffix(mediaType, "+json"):
		v.validateJSONBody(body, bodySchema, result)

	case mediaType == "application/x-www-form-urlencoded":
		v.validateFormBody(body, bodySchema, pathTemplate, operation, result)

	case strings.HasPrefix(mediaType, "multipart/form-data"):
		// Multipart forms require special handling
		if v.IncludeWarnings {
			result.addWarning("requestBody", "multipart/form-data validation is limited")
		}

	case strings.HasPrefix(mediaType, "text/"):
		// Text body - validate as string
		if v.IncludeWarnings && len(body) == 0 {
			result.addWarning("requestBody", "request body is empty")
		}

	default:
		if v.IncludeWarnings {
			result.addWarning("requestBody", fmt.Sprintf("cannot validate content type: %s", mediaType))
		}
	}
}

// getRequestBodySchema returns the schema for the given media type from a request body.
func (v *Validator) getRequestBodySchema(requestBody *parser.RequestBody, mediaType string) *parser.Schema {
	if requestBody == nil || requestBody.Content == nil {
		return nil
	}

	// Try exact match first
	if content, ok := requestBody.Content[mediaType]; ok {
		return content.Schema
	}

	// Try wildcard matches
	for contentType, content := range requestBody.Content {
		if matchMediaType(contentType, mediaType) {
			return content.Schema
		}
	}

	return nil
}

// matchMediaType checks if a pattern matches a media type.
// Supports wildcards like "application/*" and "*/*".
func matchMediaType(pattern, mediaType string) bool {
	if pattern == "*/*" {
		return true
	}

	if strings.HasSuffix(pattern, "/*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(mediaType, prefix)
	}

	return pattern == mediaType
}

// validateJSONBody validates a JSON request body against a schema.
func (v *Validator) validateJSONBody(body []byte, schema *parser.Schema, result *RequestValidationResult) {
	const path = "requestBody"
	if len(body) == 0 {
		result.addError(path, "request body is empty", SeverityError)
		return
	}

	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		result.addError(
			path,
			fmt.Sprintf("invalid JSON: %v", err),
			SeverityError,
		)
		return
	}

	// Validate against schema
	errors := v.schemaValidator.Validate(data, schema, path)
	for _, err := range errors {
		result.addError(err.Path, err.Message, err.Severity)
	}
}

// validateFormBody validates a form-urlencoded request body.
func (v *Validator) validateFormBody(body []byte, schema *parser.Schema, pathTemplate string, operation *parser.Operation, result *RequestValidationResult) {
	// For OAS 2.0, formData parameters define the form fields
	// For OAS 3.x, the schema properties define the form fields

	if v.parsed.IsOAS2() {
		// OAS 2.0: Validate against formData parameters
		v.validateFormDataParams(body, pathTemplate, operation, result)
		return
	}

	// OAS 3.x: Parse form data and validate against schema
	formData := make(map[string]any)
	pairs := strings.Split(string(body), "&")
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		key := parts[0]
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}
		formData[key] = value
	}

	errors := v.schemaValidator.Validate(formData, schema, "requestBody")
	for _, err := range errors {
		result.addError(err.Path, err.Message, err.Severity)
	}
}

// validateFormDataParams validates OAS 2.0 formData parameters.
func (v *Validator) validateFormDataParams(body []byte, pathTemplate string, operation *parser.Operation, result *RequestValidationResult) {
	// Parse form data
	formValues := make(map[string][]string)
	pairs := strings.Split(string(body), "&")
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		key := parts[0]
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}
		formValues[key] = append(formValues[key], value)
	}

	// Get formData parameters
	paramDefs := v.getParametersByLocation(pathTemplate, operation, "formData")
	defined := make(map[string]*parser.Parameter)
	for _, param := range paramDefs {
		defined[param.Name] = param
	}

	// Validate each defined parameter
	for name, param := range defined {
		values, present := formValues[name]
		if !present || len(values) == 0 {
			if param.Required {
				result.addError(
					fmt.Sprintf("requestBody.%s", name),
					fmt.Sprintf("required form field %q is missing", name),
					SeverityError,
				)
			}
			continue
		}

		// For formData, we typically have single values
		value := values[0]
		if param.Schema != nil {
			errors := v.schemaValidator.Validate(value, param.Schema, fmt.Sprintf("requestBody.%s", name))
			for _, err := range errors {
				result.addError(err.Path, err.Message, err.Severity)
			}
		}
	}
}
