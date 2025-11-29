package builder

import (
	"github.com/erraggy/oastools/parser"
)

// AddSecurityScheme adds a security scheme to components.securitySchemes.
func (b *Builder) AddSecurityScheme(name string, scheme *parser.SecurityScheme) *Builder {
	b.securitySchemes[name] = scheme
	return b
}

// AddAPIKeySecurityScheme adds an API key security scheme.
func (b *Builder) AddAPIKeySecurityScheme(name string, in string, keyName string, description string) *Builder {
	scheme := &parser.SecurityScheme{
		Type:        "apiKey",
		Name:        keyName,
		In:          in,
		Description: description,
	}
	return b.AddSecurityScheme(name, scheme)
}

// AddHTTPSecurityScheme adds an HTTP security scheme (Basic, Bearer, etc.).
func (b *Builder) AddHTTPSecurityScheme(name string, scheme string, bearerFormat string, description string) *Builder {
	ss := &parser.SecurityScheme{
		Type:         "http",
		Scheme:       scheme,
		BearerFormat: bearerFormat,
		Description:  description,
	}
	return b.AddSecurityScheme(name, ss)
}

// AddOAuth2SecurityScheme adds an OAuth2 security scheme.
func (b *Builder) AddOAuth2SecurityScheme(name string, flows *parser.OAuthFlows, description string) *Builder {
	scheme := &parser.SecurityScheme{
		Type:        "oauth2",
		Flows:       flows,
		Description: description,
	}
	return b.AddSecurityScheme(name, scheme)
}

// AddOpenIDConnectSecurityScheme adds an OpenID Connect security scheme.
func (b *Builder) AddOpenIDConnectSecurityScheme(name string, openIDConnectURL string, description string) *Builder {
	scheme := &parser.SecurityScheme{
		Type:             "openIdConnect",
		OpenIDConnectURL: openIDConnectURL,
		Description:      description,
	}
	return b.AddSecurityScheme(name, scheme)
}

// SetSecurity sets the global security requirements.
func (b *Builder) SetSecurity(requirements ...parser.SecurityRequirement) *Builder {
	b.security = requirements
	return b
}

// SecurityRequirement creates a security requirement for use with SetSecurity or WithSecurity.
func SecurityRequirement(schemeName string, scopes ...string) parser.SecurityRequirement {
	return parser.SecurityRequirement{
		schemeName: scopes,
	}
}

// AddTag adds a tag to the specification.
func (b *Builder) AddTag(name string, opts ...TagOption) *Builder {
	cfg := &tagConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	tag := &parser.Tag{
		Name:        name,
		Description: cfg.description,
	}

	if cfg.externalDocsURL != "" {
		tag.ExternalDocs = &parser.ExternalDocs{
			URL:         cfg.externalDocsURL,
			Description: cfg.externalDocsDesc,
		}
	}

	b.tags = append(b.tags, tag)
	return b
}
