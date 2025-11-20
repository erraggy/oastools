package parser

// Info provides metadata about the API
// Common across all OAS versions (2.0, 3.0, 3.1, 3.2)
type Info struct {
	Title          string   `yaml:"title" json:"title"`
	Description    string   `yaml:"description,omitempty" json:"description,omitempty"`
	TermsOfService string   `yaml:"termsOfService,omitempty" json:"termsOfService,omitempty"`
	Contact        *Contact `yaml:"contact,omitempty" json:"contact,omitempty"`
	License        *License `yaml:"license,omitempty" json:"license,omitempty"`
	Version        string   `yaml:"version" json:"version"`
	// OAS 3.1+ additions
	Summary string `yaml:"summary,omitempty" json:"summary,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	// and any other fields not explicitly defined in the struct
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Contact information for the exposed API
type Contact struct {
	Name  string `yaml:"name,omitempty" json:"name,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// License information for the exposed API
type License struct {
	Name       string `yaml:"name" json:"name"`
	URL        string `yaml:"url,omitempty" json:"url,omitempty"`
	Identifier string `yaml:"identifier,omitempty" json:"identifier,omitempty"` // OAS 3.1+
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// ExternalDocs allows referencing external documentation
type ExternalDocs struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	URL         string `yaml:"url" json:"url"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Tag adds metadata to a single tag used by operations
type Tag struct {
	Name         string        `yaml:"name" json:"name"`
	Description  string        `yaml:"description,omitempty" json:"description,omitempty"`
	ExternalDocs *ExternalDocs `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Server represents a Server object (OAS 3.0+)
type Server struct {
	URL         string                    `yaml:"url" json:"url"`
	Description string                    `yaml:"description,omitempty" json:"description,omitempty"`
	Variables   map[string]ServerVariable `yaml:"variables,omitempty" json:"variables,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// ServerVariable represents a Server Variable object (OAS 3.0+)
type ServerVariable struct {
	Enum        []string `yaml:"enum,omitempty" json:"enum,omitempty"`
	Default     string   `yaml:"default" json:"default"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Reference represents a JSON Reference ($ref)
type Reference struct {
	Ref         string `yaml:"$ref" json:"$ref"`
	Summary     string `yaml:"summary,omitempty" json:"summary,omitempty"`         // OAS 3.1+
	Description string `yaml:"description,omitempty" json:"description,omitempty"` // OAS 3.1+
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}
