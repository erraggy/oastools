package builder

import (
	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/parser"
)

// responseConfig holds configuration for response building.
type responseConfig struct {
	description string
	contentType string
	example     any
	headers     map[string]*parser.Header
	extensions  map[string]any
}

// ResponseOption configures a response.
type ResponseOption func(*responseConfig)

// WithResponseDescription sets the response description.
func WithResponseDescription(desc string) ResponseOption {
	return func(cfg *responseConfig) {
		cfg.description = desc
	}
}

// WithResponseContentType sets the content type for the response.
// Defaults to "application/json" if not specified.
func WithResponseContentType(contentType string) ResponseOption {
	return func(cfg *responseConfig) {
		cfg.contentType = contentType
	}
}

// WithResponseExample sets the response example.
func WithResponseExample(example any) ResponseOption {
	return func(cfg *responseConfig) {
		cfg.example = example
	}
}

// WithResponseHeader adds a header to the response.
func WithResponseHeader(name string, header *parser.Header) ResponseOption {
	return func(cfg *responseConfig) {
		if cfg.headers == nil {
			cfg.headers = make(map[string]*parser.Header)
		}
		cfg.headers[name] = header
	}
}

// WithResponseExtension adds a vendor extension (x-* field) to the response.
// The key must start with "x-" as per the OpenAPI specification.
// Extensions are preserved in both OAS 2.0 and OAS 3.x output.
//
// Example:
//
//	builder.WithResponse(200, User{},
//	    builder.WithResponseExtension("x-cache-ttl", 3600),
//	)
func WithResponseExtension(key string, value any) ResponseOption {
	return func(cfg *responseConfig) {
		if cfg.extensions == nil {
			cfg.extensions = make(map[string]any)
		}
		cfg.extensions[key] = value
	}
}

// AddResponse adds a reusable response to components.responses (OAS 3.x)
// or responses (OAS 2.0).
// Use WithResponseContentType to specify a content type other than "application/json".
func (b *Builder) AddResponse(name string, description string, responseType any, opts ...ResponseOption) *Builder {
	rCfg := &responseConfig{
		description: description,
		contentType: "application/json", // Default content type
	}
	for _, opt := range opts {
		opt(rCfg)
	}

	schema := b.generateSchema(responseType)

	resp := &parser.Response{
		Description: rCfg.description,
		Headers:     rCfg.headers,
		Content: map[string]*parser.MediaType{
			rCfg.contentType: {
				Schema:  schema,
				Example: rCfg.example,
			},
		},
		Extra: rCfg.extensions,
	}

	b.responses[name] = resp
	return b
}

// ResponseRef returns a reference to a named response.
// This method returns the version-appropriate ref path.
func (b *Builder) ResponseRef(name string) string {
	return pathutil.ResponseRef(name, b.version == parser.OASVersion20)
}

// buildResponsesFromMap converts a map of status codes to responses into
// a parser.Responses object. The "default" key is treated specially and
// assigned to the Default field.
func buildResponsesFromMap(responseMap map[string]*parser.Response) *parser.Responses {
	if len(responseMap) == 0 {
		return nil
	}
	responses := &parser.Responses{
		Codes: make(map[string]*parser.Response),
	}
	for code, resp := range responseMap {
		if code == "default" {
			responses.Default = resp
		} else {
			responses.Codes[code] = resp
		}
	}
	return responses
}
