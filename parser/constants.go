package parser

// Parameter location constants (used in Parameter.In field)
const (
	// ParamInQuery indicates the parameter is passed in the query string
	ParamInQuery = "query"
	// ParamInHeader indicates the parameter is passed in a request header
	ParamInHeader = "header"
	// ParamInPath indicates the parameter is part of the URL path
	ParamInPath = "path"
	// ParamInCookie indicates the parameter is passed as a cookie (OAS 3.0+)
	ParamInCookie = "cookie"
	// ParamInFormData indicates the parameter is passed as form data (OAS 2.0 only)
	ParamInFormData = "formData"
	// ParamInBody indicates the parameter is in the request body (OAS 2.0 only)
	ParamInBody = "body"
)

// JSON field key constants used across the parser package.
const (
	jsonKeyTitle       = "title"
	jsonKeyVersion     = "version"
	jsonKeyName        = "name"
	jsonKeyDefault     = "default"
	jsonKeyRef         = "$ref"
	jsonKeyType        = "type"
	jsonKeyDescription = "description"
)

// Reference type constants (used in Ref.RefType field).
const (
	refTypeFile = "file"
	refTypeHTTP = "http"
)

// MIME type constants used for content-type detection.
const (
	mimeApplicationJSON = "application/json"
	mimeApplicationYAML = "application/yaml"
)

// OAS version string constants.
const (
	oas300 = "3.0.0"
	oas301 = "3.0.1"
	oas303 = "3.0.3"
	oas310 = "3.1.0"
	oas320 = "3.2.0"
)
