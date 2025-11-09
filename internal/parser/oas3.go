package parser

// OAS3Document represents an OpenAPI Specification 3.x document
// Supports OAS 3.0.x, 3.1.x, and 3.2.x
// References:
// - OAS 3.0.0: https://spec.openapis.org/oas/v3.0.0.html
// - OAS 3.1.0: https://spec.openapis.org/oas/v3.1.0.html
// - OAS 3.2.0: https://spec.openapis.org/oas/v3.2.0.html
type OAS3Document struct {
	OpenAPI      string                `yaml:"openapi"` // Required: "3.0.x", "3.1.x", or "3.2.x"
	Info         *Info                 `yaml:"info"`    // Required
	Servers      []*Server             `yaml:"servers,omitempty"`
	Paths        Paths                 `yaml:"paths,omitempty"`    // Required in 3.0, optional in 3.1+
	Webhooks     map[string]*PathItem  `yaml:"webhooks,omitempty"` // OAS 3.1+
	Components   *Components           `yaml:"components,omitempty"`
	Security     []SecurityRequirement `yaml:"security,omitempty"`
	Tags         []*Tag                `yaml:"tags,omitempty"`
	ExternalDocs *ExternalDocs         `yaml:"externalDocs,omitempty"`
	OASVersion   OASVersion            `yaml:"-"`

	// OAS 3.1+ additions
	JSONSchemaDialect string `yaml:"jsonSchemaDialect,omitempty"` // OAS 3.1+

	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// Components holds reusable objects for different aspects of the OAS (OAS 3.0+)
type Components struct {
	Schemas         map[string]*Schema         `yaml:"schemas,omitempty"`
	Responses       map[string]*Response       `yaml:"responses,omitempty"`
	Parameters      map[string]*Parameter      `yaml:"parameters,omitempty"`
	Examples        map[string]*Example        `yaml:"examples,omitempty"`
	RequestBodies   map[string]*RequestBody    `yaml:"requestBodies,omitempty"`
	Headers         map[string]*Header         `yaml:"headers,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `yaml:"securitySchemes,omitempty"`
	Links           map[string]*Link           `yaml:"links,omitempty"`
	Callbacks       map[string]*Callback       `yaml:"callbacks,omitempty"`

	// OAS 3.1+ additions
	PathItems map[string]*PathItem `yaml:"pathItems,omitempty"` // OAS 3.1+

	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}
