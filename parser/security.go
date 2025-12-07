package parser

// SecurityRequirement lists the required security schemes to execute an operation
// Maps security scheme names to scopes (if applicable)
type SecurityRequirement map[string][]string

// SecurityScheme defines a security scheme that can be used by the operations
type SecurityScheme struct {
	Ref string `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	// Type uses omitempty because security schemes can be defined via $ref.
	// When a security scheme uses $ref, this field should be empty in the referencing object
	// (the actual value is in the referenced security scheme definition).
	Type        string `yaml:"type,omitempty" json:"type,omitempty"` // "apiKey", "http", "oauth2", "openIdConnect" (OAS 3.0+), "basic", "apiKey", "oauth2" (OAS 2.0)
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Type: apiKey (OAS 2.0+, 3.0+)
	Name string `yaml:"name,omitempty" json:"name,omitempty"` // Header, query, or cookie parameter name
	In   string `yaml:"in,omitempty" json:"in,omitempty"`     // "query", "header", "cookie" (OAS 3.0+)

	// Type: http (OAS 3.0+)
	Scheme       string `yaml:"scheme,omitempty" json:"scheme,omitempty"`             // e.g., "basic", "bearer"
	BearerFormat string `yaml:"bearerFormat,omitempty" json:"bearerFormat,omitempty"` // e.g., "JWT"

	// Type: oauth2
	Flows *OAuthFlows `yaml:"flows,omitempty" json:"flows,omitempty"` // OAS 3.0+

	// Type: oauth2 (OAS 2.0)
	Flow             string            `yaml:"flow,omitempty" json:"flow,omitempty"`                         // "implicit", "password", "application", "accessCode"
	AuthorizationURL string            `yaml:"authorizationUrl,omitempty" json:"authorizationUrl,omitempty"` // OAS 2.0
	TokenURL         string            `yaml:"tokenUrl,omitempty" json:"tokenUrl,omitempty"`                 // OAS 2.0
	Scopes           map[string]string `yaml:"scopes,omitempty" json:"scopes,omitempty"`                     // OAS 2.0

	// Type: openIdConnect (OAS 3.0+)
	OpenIDConnectURL string `yaml:"openIdConnectUrl,omitempty" json:"openIdConnectUrl,omitempty"`

	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// OAuthFlows allows configuration of the supported OAuth Flows (OAS 3.0+)
type OAuthFlows struct {
	Implicit          *OAuthFlow     `yaml:"implicit,omitempty" json:"implicit,omitempty"`
	Password          *OAuthFlow     `yaml:"password,omitempty" json:"password,omitempty"`
	ClientCredentials *OAuthFlow     `yaml:"clientCredentials,omitempty" json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow     `yaml:"authorizationCode,omitempty" json:"authorizationCode,omitempty"`
	Extra             map[string]any `yaml:",inline" json:"-"`
}

// OAuthFlow represents configuration for a single OAuth flow (OAS 3.0+)
type OAuthFlow struct {
	AuthorizationURL string            `yaml:"authorizationUrl,omitempty" json:"authorizationUrl,omitempty"`
	TokenURL         string            `yaml:"tokenUrl,omitempty" json:"tokenUrl,omitempty"`
	RefreshURL       string            `yaml:"refreshUrl,omitempty" json:"refreshUrl,omitempty"`
	Scopes           map[string]string `yaml:"scopes" json:"scopes"`
	Extra            map[string]any    `yaml:",inline" json:"-"`
}
