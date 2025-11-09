package parser

// Info provides metadata about the API
// Common across all OAS versions (2.0, 3.0, 3.1, 3.2)
type Info struct {
	Title          string   `yaml:"title"`
	Description    string   `yaml:"description,omitempty"`
	TermsOfService string   `yaml:"termsOfService,omitempty"`
	Contact        *Contact `yaml:"contact,omitempty"`
	License        *License `yaml:"license,omitempty"`
	Version        string   `yaml:"version"`
	// OAS 3.1+ additions
	Summary string `yaml:"summary,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	// and any other fields not explicitly defined in the struct
	Extra map[string]interface{} `yaml:",inline"`
}

// Contact information for the exposed API
type Contact struct {
	Name  string `yaml:"name,omitempty"`
	URL   string `yaml:"url,omitempty"`
	Email string `yaml:"email,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// License information for the exposed API
type License struct {
	Name       string `yaml:"name"`
	URL        string `yaml:"url,omitempty"`
	Identifier string `yaml:"identifier,omitempty"` // OAS 3.1+
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// ExternalDocs allows referencing external documentation
type ExternalDocs struct {
	Description string `yaml:"description,omitempty"`
	URL         string `yaml:"url"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// Tag adds metadata to a single tag used by operations
type Tag struct {
	Name         string        `yaml:"name"`
	Description  string        `yaml:"description,omitempty"`
	ExternalDocs *ExternalDocs `yaml:"externalDocs,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// Server represents a Server object (OAS 3.0+)
type Server struct {
	URL         string                    `yaml:"url"`
	Description string                    `yaml:"description,omitempty"`
	Variables   map[string]ServerVariable `yaml:"variables,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// ServerVariable represents a Server Variable object (OAS 3.0+)
type ServerVariable struct {
	Enum        []string `yaml:"enum,omitempty"`
	Default     string   `yaml:"default"`
	Description string   `yaml:"description,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// Reference represents a JSON Reference ($ref)
type Reference struct {
	Ref         string `yaml:"$ref"`
	Summary     string `yaml:"summary,omitempty"`     // OAS 3.1+
	Description string `yaml:"description,omitempty"` // OAS 3.1+
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}
