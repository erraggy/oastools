package parser

import (
	"fmt"

	"go.yaml.in/yaml/v4"

	"github.com/erraggy/oastools/internal/httputil"
)

// Paths holds the relative paths to the individual endpoints
type Paths map[string]*PathItem

// PathItem describes the operations available on a single path
type PathItem struct {
	Ref         string       `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Summary     string       `yaml:"summary,omitempty" json:"summary,omitempty"`         // OAS 3.0+
	Description string       `yaml:"description,omitempty" json:"description,omitempty"` // OAS 3.0+
	Get         *Operation   `yaml:"get,omitempty" json:"get,omitempty"`
	Put         *Operation   `yaml:"put,omitempty" json:"put,omitempty"`
	Post        *Operation   `yaml:"post,omitempty" json:"post,omitempty"`
	Delete      *Operation   `yaml:"delete,omitempty" json:"delete,omitempty"`
	Options     *Operation   `yaml:"options,omitempty" json:"options,omitempty"`
	Head        *Operation   `yaml:"head,omitempty" json:"head,omitempty"`
	Patch       *Operation   `yaml:"patch,omitempty" json:"patch,omitempty"`
	Trace       *Operation   `yaml:"trace,omitempty" json:"trace,omitempty"`     // OAS 3.0+
	Query       *Operation   `yaml:"query,omitempty" json:"query,omitempty"`     // OAS 3.2+
	Servers     []*Server    `yaml:"servers,omitempty" json:"servers,omitempty"` // OAS 3.0+
	Parameters  []*Parameter `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Operation describes a single API operation on a path
type Operation struct {
	Tags         []string              `yaml:"tags,omitempty" json:"tags,omitempty"`
	Summary      string                `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description  string                `yaml:"description,omitempty" json:"description,omitempty"`
	ExternalDocs *ExternalDocs         `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`
	OperationID  string                `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Parameters   []*Parameter          `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody  *RequestBody          `yaml:"requestBody,omitempty" json:"requestBody,omitempty"` // OAS 3.0+
	Responses    *Responses            `yaml:"responses" json:"responses"`
	Callbacks    map[string]*Callback  `yaml:"callbacks,omitempty" json:"callbacks,omitempty"` // OAS 3.0+
	Deprecated   bool                  `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Security     []SecurityRequirement `yaml:"security,omitempty" json:"security,omitempty"`
	Servers      []*Server             `yaml:"servers,omitempty" json:"servers,omitempty"` // OAS 3.0+
	// OAS 2.0 specific
	Consumes []string `yaml:"consumes,omitempty" json:"consumes,omitempty"` // OAS 2.0
	Produces []string `yaml:"produces,omitempty" json:"produces,omitempty"` // OAS 2.0
	Schemes  []string `yaml:"schemes,omitempty" json:"schemes,omitempty"`   // OAS 2.0
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Responses is a container for the expected responses of an operation
type Responses struct {
	Default *Response            `yaml:"default,omitempty" json:"default,omitempty"`
	Codes   map[string]*Response `yaml:",inline" json:"-"` // Handled by custom marshaler
}

// UnmarshalYAML implements custom unmarshaling for Responses to validate status codes during parsing.
// This prevents invalid fields from being captured in the Codes map and provides clearer error messages.
func (r *Responses) UnmarshalYAML(unmarshal func(any) error) error {
	// First unmarshal into a raw map to inspect all fields
	var raw map[string]any
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
	Ref string `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	// Description uses omitempty because responses can be defined via $ref.
	// When a response uses $ref, this field should be empty in the referencing object
	// (the actual value is in the referenced response definition).
	Description string                `yaml:"description,omitempty" json:"description,omitempty"`
	Headers     map[string]*Header    `yaml:"headers,omitempty" json:"headers,omitempty"`
	Content     map[string]*MediaType `yaml:"content,omitempty" json:"content,omitempty"` // OAS 3.0+
	Links       map[string]*Link      `yaml:"links,omitempty" json:"links,omitempty"`     // OAS 3.0+
	// OAS 2.0 specific
	Schema   *Schema        `yaml:"schema,omitempty" json:"schema,omitempty"`     // OAS 2.0
	Examples map[string]any `yaml:"examples,omitempty" json:"examples,omitempty"` // OAS 2.0
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Callback is a map of expressions to path items (OAS 3.0+)
type Callback map[string]*PathItem

// Link represents a possible design-time link for a response (OAS 3.0+)
type Link struct {
	Ref          string         `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	OperationRef string         `yaml:"operationRef,omitempty" json:"operationRef,omitempty"`
	OperationID  string         `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Parameters   map[string]any `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody  any            `yaml:"requestBody,omitempty" json:"requestBody,omitempty"`
	Description  string         `yaml:"description,omitempty" json:"description,omitempty"`
	Server       *Server        `yaml:"server,omitempty" json:"server,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// MediaType provides schema and examples for the media type (OAS 3.0+)
type MediaType struct {
	Schema   *Schema              `yaml:"schema,omitempty" json:"schema,omitempty"`
	Example  any                  `yaml:"example,omitempty" json:"example,omitempty"`
	Examples map[string]*Example  `yaml:"examples,omitempty" json:"examples,omitempty"`
	Encoding map[string]*Encoding `yaml:"encoding,omitempty" json:"encoding,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Example represents an example object (OAS 3.0+)
type Example struct {
	Ref           string `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Summary       string `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description   string `yaml:"description,omitempty" json:"description,omitempty"`
	Value         any    `yaml:"value,omitempty" json:"value,omitempty"`
	ExternalValue string `yaml:"externalValue,omitempty" json:"externalValue,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Encoding defines encoding for a specific property (OAS 3.0+)
type Encoding struct {
	ContentType   string             `yaml:"contentType,omitempty" json:"contentType,omitempty"`
	Headers       map[string]*Header `yaml:"headers,omitempty" json:"headers,omitempty"`
	Style         string             `yaml:"style,omitempty" json:"style,omitempty"`
	Explode       *bool              `yaml:"explode,omitempty" json:"explode,omitempty"`
	AllowReserved bool               `yaml:"allowReserved,omitempty" json:"allowReserved,omitempty"`
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// yamlMarshalValue marshals a value to YAML bytes for re-parsing
func yamlMarshalValue(value any) ([]byte, error) {
	return yaml.Marshal(value)
}

// yamlUnmarshalValue unmarshals YAML bytes into a target
func yamlUnmarshalValue(data []byte, target any) error {
	return yaml.Unmarshal(data, target)
}
