package builder

import (
	"github.com/erraggy/oastools/parser"
)

// AddResponse adds a reusable response to components.responses (OAS 3.x)
// or responses (OAS 2.0).
func (b *Builder) AddResponse(name string, description string, responseType any, opts ...ResponseOption) *Builder {
	rCfg := &responseConfig{
		description: description,
	}
	for _, opt := range opts {
		opt(rCfg)
	}

	schema := b.generateSchema(responseType)

	resp := &parser.Response{
		Description: rCfg.description,
		Headers:     rCfg.headers,
		Content: map[string]*parser.MediaType{
			"application/json": {
				Schema:  schema,
				Example: rCfg.example,
			},
		},
	}

	b.responses[name] = resp
	return b
}

// AddResponseWithContentType adds a reusable response with a specific content type.
func (b *Builder) AddResponseWithContentType(name string, description string, contentType string, responseType any, opts ...ResponseOption) *Builder {
	rCfg := &responseConfig{
		description: description,
	}
	for _, opt := range opts {
		opt(rCfg)
	}

	schema := b.generateSchema(responseType)

	resp := &parser.Response{
		Description: rCfg.description,
		Headers:     rCfg.headers,
		Content: map[string]*parser.MediaType{
			contentType: {
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
