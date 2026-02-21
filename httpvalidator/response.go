package httpvalidator

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// ValidateResponseData validates response data without requiring an *http.Response.
// This is useful for middleware scenarios where you've captured response parts
// but don't have an *http.Response object.
//
// Parameters:
//   - req: The original HTTP request (to determine the operation)
//   - statusCode: The HTTP status code of the response
//   - headers: Response headers
//   - body: Response body bytes (can be nil for bodyless responses)
//
// Example middleware usage:
//
//	result, err := v.ValidateResponseData(req, rec.Code, rec.Header(), rec.Body.Bytes())
func (v *Validator) ValidateResponseData(req *http.Request, statusCode int, headers http.Header, body []byte) (*ResponseValidationResult, error) {
	// Snapshot mutable fields so concurrent mutations cannot cause
	// inconsistent behavior within a single validation call.
	flags := validationFlags{
		strictMode:      v.StrictMode,
		includeWarnings: v.IncludeWarnings,
	}

	result := newResponseResult()
	result.StatusCode = statusCode
	result.ContentType = headers.Get("Content-Type")

	// Find matching path and operation from the original request
	operation := v.findMatchedOperation(req, result)
	if operation == nil {
		return result, nil
	}

	// Validate response
	v.validateResponseParts(statusCode, headers, body, result.MatchedPath, operation, result, flags)

	return result, nil
}

// validateResponse validates an HTTP response against the operation spec.
func (v *Validator) validateResponse(resp *http.Response, matchedPath string, operation *parser.Operation, result *ResponseValidationResult, flags validationFlags) {
	// Read response body if present
	var body []byte
	if resp.Body != nil {
		maxSize := v.maxBodySizeOrDefault()
		var err error
		body, err = io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
		if err != nil {
			result.addError(
				"response",
				fmt.Sprintf("failed to read response body: %s", truncateForError(err.Error(), maxErrorValueLen)),
				SeverityError,
			)
			return
		}

		if int64(len(body)) > maxSize {
			result.addError("response",
				fmt.Sprintf("response body size %d bytes exceeds maximum %d bytes", len(body), maxSize),
				SeverityError)
			return
		}
	}

	v.validateResponseParts(resp.StatusCode, resp.Header, body, matchedPath, operation, result, flags)
}

// validateResponseParts is the shared implementation for response validation.
func (v *Validator) validateResponseParts(statusCode int, headers http.Header, body []byte, _ string, operation *parser.Operation, result *ResponseValidationResult, flags validationFlags) {
	// Find response definition for this status code
	responseDef := v.getResponseDefinition(operation, statusCode)
	if responseDef == nil {
		if flags.strictMode {
			result.addError(
				fmt.Sprintf("response.%d", statusCode),
				fmt.Sprintf("undocumented response status code: %d", statusCode),
				SeverityError,
			)
		} else if flags.includeWarnings {
			result.addWarning(
				fmt.Sprintf("response.%d", statusCode),
				fmt.Sprintf("response status code %d not documented", statusCode),
			)
		}
		return
	}

	// Validate response headers
	v.validateResponseHeaders(headers, responseDef, result)

	// Validate response body
	contentType := headers.Get("Content-Type")
	v.validateResponseBody(body, contentType, responseDef, result, flags)
}

// getResponseDefinition finds the response definition for a status code.
// It first tries exact match, then wildcard patterns (2XX, 4XX, 5XX), then default.
func (v *Validator) getResponseDefinition(operation *parser.Operation, statusCode int) *parser.Response {
	if operation.Responses == nil {
		return nil
	}

	responses := operation.Responses

	// 1. Try exact status code match in Codes map
	if responses.Codes != nil {
		statusStr := strconv.Itoa(statusCode)
		if resp, ok := responses.Codes[statusStr]; ok {
			return resp
		}

		// 2. Try wildcard patterns (2XX, 3XX, 4XX, 5XX)
		// OAS 3.x supports these patterns
		wildcards := []string{
			fmt.Sprintf("%dXX", statusCode/100), // 2XX, 4XX, 5XX
			fmt.Sprintf("%dx", statusCode/10),   // Less common: 20x, 40x
		}
		for _, pattern := range wildcards {
			// Try both uppercase and lowercase X
			if resp, ok := responses.Codes[pattern]; ok {
				return resp
			}
			if resp, ok := responses.Codes[strings.ToLower(pattern)]; ok {
				return resp
			}
		}
	}

	// 3. Try "default" response
	if responses.Default != nil {
		return responses.Default
	}

	return nil
}

// validateResponseHeaders validates response headers against the spec.
func (v *Validator) validateResponseHeaders(headers http.Header, responseDef *parser.Response, result *ResponseValidationResult) {
	if responseDef.Headers == nil {
		return
	}

	for headerName, headerDef := range responseDef.Headers {
		canonicalName := http.CanonicalHeaderKey(headerName)
		value := headers.Get(canonicalName)

		if value == "" {
			if headerDef.Required {
				result.addError(
					fmt.Sprintf("response.header.%s", headerName),
					fmt.Sprintf("required response header %q is missing", headerName),
					SeverityError,
				)
			}
			continue
		}

		// Validate against schema
		if headerDef.Schema != nil {
			deserializer := NewParamDeserializer()
			// Headers use simple style by default
			param := &parser.Parameter{
				Name:   headerName,
				In:     "header",
				Schema: headerDef.Schema,
			}
			deserializedValue := deserializer.DeserializeHeaderParam(value, param)

			errors := v.schemaValidator.Validate(deserializedValue, headerDef.Schema, fmt.Sprintf("response.header.%s", headerName))
			for _, err := range errors {
				result.addError(err.Path, err.Message, err.Severity)
			}
		}
	}
}

// validateResponseBody validates the response body against the spec.
func (v *Validator) validateResponseBody(body []byte, contentType string, responseDef *parser.Response, result *ResponseValidationResult, flags validationFlags) {
	// Get schema for this response
	var schema *parser.Schema

	if v.parsed.IsOAS3() {
		// OAS 3.x: Get schema from content map
		if responseDef.Content != nil && contentType != "" {
			mediaType, _, _ := mime.ParseMediaType(contentType)
			schema = v.getResponseSchema(responseDef, mediaType)
		}
	} else {
		// OAS 2.0: Schema is directly on response
		schema = responseDef.Schema
	}

	// If no schema defined, nothing to validate
	if schema == nil {
		return
	}

	// If body is empty but schema exists
	if len(body) == 0 {
		// Some schemas allow empty (e.g., nullable: true, or no required fields)
		// For now, just warn
		if flags.includeWarnings {
			result.addWarning("response.body", "response body is empty but schema is defined")
		}
		return
	}

	// Parse content type
	mediaType := contentType
	if contentType != "" {
		parsed, _, err := mime.ParseMediaType(contentType)
		if err == nil {
			mediaType = parsed
		}
	}

	// Validate based on media type
	switch {
	case strings.HasPrefix(mediaType, "application/json") || strings.HasSuffix(mediaType, "+json"):
		v.validateJSONResponseBody(body, schema, result)

	case strings.HasPrefix(mediaType, "text/"):
		// Text responses - validate as string if schema expects string
		if getSchemaType(schema) == "string" {
			errors := v.schemaValidator.Validate(string(body), schema, "response.body")
			for _, err := range errors {
				result.addError(err.Path, err.Message, err.Severity)
			}
		}

	default:
		if flags.includeWarnings {
			result.addWarning("response.body", fmt.Sprintf("cannot validate content type: %q", truncateForError(mediaType, maxErrorValueLen)))
		}
	}
}

// getResponseSchema returns the schema for the given media type from a response.
func (v *Validator) getResponseSchema(responseDef *parser.Response, mediaType string) *parser.Schema {
	if responseDef.Content == nil {
		return nil
	}

	// Try exact match first
	if content, ok := responseDef.Content[mediaType]; ok {
		return content.Schema
	}

	// Try wildcard matches
	for contentType, content := range responseDef.Content {
		if matchMediaType(contentType, mediaType) {
			return content.Schema
		}
	}

	return nil
}

// validateJSONResponseBody validates a JSON response body against a schema.
func (v *Validator) validateJSONResponseBody(body []byte, schema *parser.Schema, result *ResponseValidationResult) {
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		result.addError(
			"response.body",
			fmt.Sprintf("invalid JSON in response: %s", truncateForError(err.Error(), maxErrorValueLen)),
			SeverityError,
		)
		return
	}

	// Validate against schema
	errors := v.schemaValidator.Validate(data, schema, "response.body")
	for _, err := range errors {
		result.addError(err.Path, err.Message, err.Severity)
	}
}
