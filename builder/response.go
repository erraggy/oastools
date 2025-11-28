package builder

import (
	"github.com/erraggy/oastools/parser"
)

// AddResponse adds a reusable response to components.responses.
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

// ResponseRef returns a reference to a named response in components.
func ResponseRef(name string) string {
	return "#/components/responses/" + name
}
