package parser

// OAS3Document represents an OpenAPI Specification 3.x document
// Supports OAS 3.0.x, 3.1.x, and 3.2.x
// References:
// - OAS 3.0.0: https://spec.openapis.org/oas/v3.0.0.html
// - OAS 3.1.0: https://spec.openapis.org/oas/v3.1.0.html
// - OAS 3.2.0: https://spec.openapis.org/oas/v3.2.0.html
type OAS3Document struct {
	OpenAPI      string                `yaml:"openapi" json:"openapi"` // Required: "3.0.x", "3.1.x", or "3.2.x"
	Info         *Info                 `yaml:"info" json:"info"`       // Required
	Servers      []*Server             `yaml:"servers,omitempty" json:"servers,omitempty"`
	Paths        Paths                 `yaml:"paths,omitempty" json:"paths,omitempty"`       // Required in 3.0, optional in 3.1+
	Webhooks     map[string]*PathItem  `yaml:"webhooks,omitempty" json:"webhooks,omitempty"` // OAS 3.1+
	Components   *Components           `yaml:"components,omitempty" json:"components,omitempty"`
	Security     []SecurityRequirement `yaml:"security,omitempty" json:"security,omitempty"`
	Tags         []*Tag                `yaml:"tags,omitempty" json:"tags,omitempty"`
	ExternalDocs *ExternalDocs         `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
	OASVersion   OASVersion            `yaml:"-" json:"-"`

	// OAS 3.1+ additions
	JSONSchemaDialect string `yaml:"jsonSchemaDialect,omitempty" json:"jsonSchemaDialect,omitempty"` // OAS 3.1+

	// OAS 3.2+ additions
	Self string `yaml:"$self,omitempty" json:"$self,omitempty"` // OAS 3.2+ - Document identity/base URI

	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Components holds reusable objects for different aspects of the OAS (OAS 3.0+)
type Components struct {
	Schemas         map[string]*Schema         `yaml:"schemas,omitempty" json:"schemas,omitempty"`
	Responses       map[string]*Response       `yaml:"responses,omitempty" json:"responses,omitempty"`
	Parameters      map[string]*Parameter      `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Examples        map[string]*Example        `yaml:"examples,omitempty" json:"examples,omitempty"`
	RequestBodies   map[string]*RequestBody    `yaml:"requestBodies,omitempty" json:"requestBodies,omitempty"`
	Headers         map[string]*Header         `yaml:"headers,omitempty" json:"headers,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `yaml:"securitySchemes,omitempty" json:"securitySchemes,omitempty"`
	Links           map[string]*Link           `yaml:"links,omitempty" json:"links,omitempty"`
	Callbacks       map[string]*Callback       `yaml:"callbacks,omitempty" json:"callbacks,omitempty"`

	// OAS 3.1+ additions
	PathItems map[string]*PathItem `yaml:"pathItems,omitempty" json:"pathItems,omitempty"` // OAS 3.1+

	// OAS 3.2+ additions
	MediaTypes map[string]*MediaType `yaml:"mediaTypes,omitempty" json:"mediaTypes,omitempty"` // OAS 3.2+

	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}
