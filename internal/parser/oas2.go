package parser

// OAS2Document represents an OpenAPI Specification 2.0 (Swagger) document
// Reference: https://spec.openapis.org/oas/v2.0.html
type OAS2Document struct {
	Swagger             string                     `yaml:"swagger"` // Required: "2.0"
	Info                *Info                      `yaml:"info"`    // Required
	Host                string                     `yaml:"host,omitempty"`
	BasePath            string                     `yaml:"basePath,omitempty"`
	Schemes             []string                   `yaml:"schemes,omitempty"` // e.g., ["http", "https"]
	Consumes            []string                   `yaml:"consumes,omitempty"`
	Produces            []string                   `yaml:"produces,omitempty"`
	Paths               Paths                      `yaml:"paths"` // Required
	Definitions         map[string]*Schema         `yaml:"definitions,omitempty"`
	Parameters          map[string]*Parameter      `yaml:"parameters,omitempty"`
	Responses           map[string]*Response       `yaml:"responses,omitempty"`
	SecurityDefinitions map[string]*SecurityScheme `yaml:"securityDefinitions,omitempty"`
	Security            []SecurityRequirement      `yaml:"security,omitempty"`
	Tags                []*Tag                     `yaml:"tags,omitempty"`
	ExternalDocs        *ExternalDocs              `yaml:"externalDocs,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra      map[string]interface{} `yaml:",inline"`
	OASVersion OASVersion             `yaml:"-"`
}
