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
