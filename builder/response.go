package builder

import (
	"github.com/erraggy/oastools/parser"
)

// responseConfig holds configuration for response building.
type responseConfig struct {
	description string
	contentType string
	example     any
	headers     map[string]*parser.Header
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
	}

	b.responses[name] = resp
	return b
}

// responseRefPrefix returns the appropriate $ref prefix for responses.
// OAS 2.0 uses "#/responses/" while OAS 3.x uses "#/components/responses/".
func (b *Builder) responseRefPrefix() string {
	if b.version == parser.OASVersion20 {
		return "#/responses/"
	}
	return "#/components/responses/"
}

// ResponseRef returns a reference to a named response.
// This method returns the version-appropriate ref path.
func (b *Builder) ResponseRef(name string) string {
	return b.responseRefPrefix() + name
}
