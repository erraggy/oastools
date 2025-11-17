package parser

// OAS2Document represents an OpenAPI Specification 2.0 (Swagger) document
// Reference: https://spec.openapis.org/oas/v2.0.html
type OAS2Document struct {
	Swagger             string                     `yaml:"swagger" json:"swagger"` // Required: "2.0"
	Info                *Info                      `yaml:"info" json:"info"`       // Required
	Host                string                     `yaml:"host,omitempty" json:"host,omitempty"`
	BasePath            string                     `yaml:"basePath,omitempty" json:"basePath,omitempty"`
	Schemes             []string                   `yaml:"schemes,omitempty" json:"schemes,omitempty"` // e.g., ["http", "https"]
	Consumes            []string                   `yaml:"consumes,omitempty" json:"consumes,omitempty"`
	Produces            []string                   `yaml:"produces,omitempty" json:"produces,omitempty"`
	Paths               Paths                      `yaml:"paths" json:"paths"` // Required
	Definitions         map[string]*Schema         `yaml:"definitions,omitempty" json:"definitions,omitempty"`
	Parameters          map[string]*Parameter      `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Responses           map[string]*Response       `yaml:"responses,omitempty" json:"responses,omitempty"`
	SecurityDefinitions map[string]*SecurityScheme `yaml:"securityDefinitions,omitempty" json:"securityDefinitions,omitempty"`
	Security            []SecurityRequirement      `yaml:"security,omitempty" json:"security,omitempty"`
	Tags                []*Tag                     `yaml:"tags,omitempty" json:"tags,omitempty"`
	ExternalDocs        *ExternalDocs              `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra      map[string]interface{} `yaml:",inline" json:"-"`
	OASVersion OASVersion             `yaml:"-" json:"-"`
}
