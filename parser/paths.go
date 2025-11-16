package parser

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/erraggy/oastools/internal/httputil"
)

// Paths holds the relative paths to the individual endpoints
type Paths map[string]*PathItem

// PathItem describes the operations available on a single path
type PathItem struct {
	Ref         string       `yaml:"$ref,omitempty"`
	Summary     string       `yaml:"summary,omitempty"`     // OAS 3.0+
	Description string       `yaml:"description,omitempty"` // OAS 3.0+
	Get         *Operation   `yaml:"get,omitempty"`
	Put         *Operation   `yaml:"put,omitempty"`
	Post        *Operation   `yaml:"post,omitempty"`
	Delete      *Operation   `yaml:"delete,omitempty"`
	Options     *Operation   `yaml:"options,omitempty"`
	Head        *Operation   `yaml:"head,omitempty"`
	Patch       *Operation   `yaml:"patch,omitempty"`
	Trace       *Operation   `yaml:"trace,omitempty"`   // OAS 3.0+
	Servers     []*Server    `yaml:"servers,omitempty"` // OAS 3.0+
	Parameters  []*Parameter `yaml:"parameters,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// Operation describes a single API operation on a path
type Operation struct {
	Tags         []string              `yaml:"tags,omitempty"`
	Summary      string                `yaml:"summary,omitempty"`
	Description  string                `yaml:"description,omitempty"`
	ExternalDocs *ExternalDocs         `yaml:"externalDocs,omitempty"`
	OperationID  string                `yaml:"operationId,omitempty"`
	Parameters   []*Parameter          `yaml:"parameters,omitempty"`
	RequestBody  *RequestBody          `yaml:"requestBody,omitempty"` // OAS 3.0+
	Responses    *Responses            `yaml:"responses"`
	Callbacks    map[string]*Callback  `yaml:"callbacks,omitempty"` // OAS 3.0+
	Deprecated   bool                  `yaml:"deprecated,omitempty"`
	Security     []SecurityRequirement `yaml:"security,omitempty"`
	Servers      []*Server             `yaml:"servers,omitempty"` // OAS 3.0+
	// OAS 2.0 specific
	Consumes []string `yaml:"consumes,omitempty"` // OAS 2.0
	Produces []string `yaml:"produces,omitempty"` // OAS 2.0
	Schemes  []string `yaml:"schemes,omitempty"`  // OAS 2.0
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// Responses is a container for the expected responses of an operation
type Responses struct {
	Default *Response            `yaml:"default,omitempty"`
	Codes   map[string]*Response `yaml:",inline"`
}

// UnmarshalYAML implements custom unmarshaling for Responses to validate status codes during parsing.
// This prevents invalid fields from being captured in the Codes map and provides clearer error messages.
func (r *Responses) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First unmarshal into a raw map to inspect all fields
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Initialize the Codes map
	r.Codes = make(map[string]*Response)

	// Process each field
	for key, value := range raw {
		if key == "default" {
			// Handle the default response
			valueBytes, err := yamlMarshalValue(value)
			if err != nil {
				return fmt.Errorf("failed to marshal default response: %w", err)
			}
			var defaultResp Response
			if err := yamlUnmarshalValue(valueBytes, &defaultResp); err != nil {
				return fmt.Errorf("failed to unmarshal default response: %w", err)
			}
			r.Default = &defaultResp
		} else {
			// All other fields should be valid status codes or extension fields
			if !httputil.ValidateStatusCode(key) {
				return fmt.Errorf("invalid status code '%s' in responses: must be a valid HTTP status code (e.g., \"200\", \"404\"), wildcard pattern (e.g., \"2XX\"), or extension field (e.g., \"x-custom\")", key)
			}
			valueBytes, err := yamlMarshalValue(value)
			if err != nil {
				return fmt.Errorf("failed to marshal response for status code %s: %w", key, err)
			}
			var resp Response
			if err := yamlUnmarshalValue(valueBytes, &resp); err != nil {
				return fmt.Errorf("failed to unmarshal response for status code %s: %w", key, err)
			}
			r.Codes[key] = &resp
		}
	}

	return nil
}

// Response describes a single response from an API Operation
type Response struct {
	Ref         string                `yaml:"$ref,omitempty"`
	Description string                `yaml:"description"`
	Headers     map[string]*Header    `yaml:"headers,omitempty"`
	Content     map[string]*MediaType `yaml:"content,omitempty"` // OAS 3.0+
	Links       map[string]*Link      `yaml:"links,omitempty"`   // OAS 3.0+
	// OAS 2.0 specific
	Schema   *Schema                `yaml:"schema,omitempty"`   // OAS 2.0
	Examples map[string]interface{} `yaml:"examples,omitempty"` // OAS 2.0
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// Callback is a map of expressions to path items (OAS 3.0+)
type Callback map[string]*PathItem

// Link represents a possible design-time link for a response (OAS 3.0+)
type Link struct {
	Ref          string                 `yaml:"$ref,omitempty"`
	OperationRef string                 `yaml:"operationRef,omitempty"`
	OperationID  string                 `yaml:"operationId,omitempty"`
	Parameters   map[string]interface{} `yaml:"parameters,omitempty"`
	RequestBody  interface{}            `yaml:"requestBody,omitempty"`
	Description  string                 `yaml:"description,omitempty"`
	Server       *Server                `yaml:"server,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// MediaType provides schema and examples for the media type (OAS 3.0+)
type MediaType struct {
	Schema   *Schema              `yaml:"schema,omitempty"`
	Example  interface{}          `yaml:"example,omitempty"`
	Examples map[string]*Example  `yaml:"examples,omitempty"`
	Encoding map[string]*Encoding `yaml:"encoding,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// Example represents an example object (OAS 3.0+)
type Example struct {
	Ref           string      `yaml:"$ref,omitempty"`
	Summary       string      `yaml:"summary,omitempty"`
	Description   string      `yaml:"description,omitempty"`
	Value         interface{} `yaml:"value,omitempty"`
	ExternalValue string      `yaml:"externalValue,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// Encoding defines encoding for a specific property (OAS 3.0+)
type Encoding struct {
	ContentType   string             `yaml:"contentType,omitempty"`
	Headers       map[string]*Header `yaml:"headers,omitempty"`
	Style         string             `yaml:"style,omitempty"`
	Explode       *bool              `yaml:"explode,omitempty"`
	AllowReserved bool               `yaml:"allowReserved,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline"`
}

// yamlMarshalValue marshals a value to YAML bytes for re-parsing
func yamlMarshalValue(value interface{}) ([]byte, error) {
	return yaml.Marshal(value)
}

// yamlUnmarshalValue unmarshals YAML bytes into a target
func yamlUnmarshalValue(data []byte, target interface{}) error {
	return yaml.Unmarshal(data, target)
}
