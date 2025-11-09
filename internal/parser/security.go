package parser

// SecurityRequirement lists the required security schemes to execute an operation
// Maps security scheme names to scopes (if applicable)
type SecurityRequirement map[string][]string

// SecurityScheme defines a security scheme that can be used by the operations
type SecurityScheme struct {
	Ref         string `yaml:"$ref,omitempty"`
	Type        string `yaml:"type"` // "apiKey", "http", "oauth2", "openIdConnect" (OAS 3.0+), "basic", "apiKey", "oauth2" (OAS 2.0)
	Description string `yaml:"description,omitempty"`

	// Type: apiKey (OAS 2.0+, 3.0+)
	Name string `yaml:"name,omitempty"` // Header, query, or cookie parameter name
	In   string `yaml:"in,omitempty"`   // "query", "header", "cookie" (OAS 3.0+)

	// Type: http (OAS 3.0+)
	Scheme       string `yaml:"scheme,omitempty"`       // e.g., "basic", "bearer"
	BearerFormat string `yaml:"bearerFormat,omitempty"` // e.g., "JWT"

	// Type: oauth2
	Flows *OAuthFlows `yaml:"flows,omitempty"` // OAS 3.0+

	// Type: oauth2 (OAS 2.0)
	Flow             string            `yaml:"flow,omitempty"`             // "implicit", "password", "application", "accessCode"
	AuthorizationURL string            `yaml:"authorizationUrl,omitempty"` // OAS 2.0
	TokenURL         string            `yaml:"tokenUrl,omitempty"`         // OAS 2.0
	Scopes           map[string]string `yaml:"scopes,omitempty"`           // OAS 2.0

	// Type: openIdConnect (OAS 3.0+)
	OpenIDConnectURL string `yaml:"openIdConnectUrl,omitempty"`

	Extra map[string]interface{} `yaml:",inline"`
}

// OAuthFlows allows configuration of the supported OAuth Flows (OAS 3.0+)
type OAuthFlows struct {
	Implicit          *OAuthFlow             `yaml:"implicit,omitempty"`
	Password          *OAuthFlow             `yaml:"password,omitempty"`
	ClientCredentials *OAuthFlow             `yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow             `yaml:"authorizationCode,omitempty"`
	Extra             map[string]interface{} `yaml:",inline"`
}

// OAuthFlow represents configuration for a single OAuth flow (OAS 3.0+)
type OAuthFlow struct {
	AuthorizationURL string                 `yaml:"authorizationUrl,omitempty"`
	TokenURL         string                 `yaml:"tokenUrl,omitempty"`
	RefreshURL       string                 `yaml:"refreshUrl,omitempty"`
	Scopes           map[string]string      `yaml:"scopes"`
	Extra            map[string]interface{} `yaml:",inline"`
}
