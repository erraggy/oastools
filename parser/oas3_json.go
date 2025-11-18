package parser

import (
	"encoding/json"
)

// MarshalJSON implements custom JSON marshaling for OAS3Document.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (d *OAS3Document) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(d.Extra) == 0 {
		type Alias OAS3Document
		return json.Marshal((*Alias)(d))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 10+len(d.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	// OpenAPI and Info are required, always include
	m["openapi"] = d.OpenAPI
	m["info"] = d.Info
	if len(d.Servers) > 0 {
		m["servers"] = d.Servers
	}
	if len(d.Paths) > 0 {
		m["paths"] = d.Paths
	}
	if len(d.Webhooks) > 0 {
		m["webhooks"] = d.Webhooks
	}
	if d.Components != nil {
		m["components"] = d.Components
	}
	if len(d.Security) > 0 {
		m["security"] = d.Security
	}
	if len(d.Tags) > 0 {
		m["tags"] = d.Tags
	}
	if d.ExternalDocs != nil {
		m["externalDocs"] = d.ExternalDocs
	}
	if d.JSONSchemaDialect != "" {
		m["jsonSchemaDialect"] = d.JSONSchemaDialect
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range d.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for OAS3Document.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (d *OAS3Document) UnmarshalJSON(data []byte) error {
	type Alias OAS3Document
	aux := (*Alias)(d)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		d.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Components.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (c *Components) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(c.Extra) == 0 {
		type Alias Components
		return json.Marshal((*Alias)(c))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 10+len(c.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if len(c.Schemas) > 0 {
		m["schemas"] = c.Schemas
	}
	if len(c.Responses) > 0 {
		m["responses"] = c.Responses
	}
	if len(c.Parameters) > 0 {
		m["parameters"] = c.Parameters
	}
	if len(c.Examples) > 0 {
		m["examples"] = c.Examples
	}
	if len(c.RequestBodies) > 0 {
		m["requestBodies"] = c.RequestBodies
	}
	if len(c.Headers) > 0 {
		m["headers"] = c.Headers
	}
	if len(c.SecuritySchemes) > 0 {
		m["securitySchemes"] = c.SecuritySchemes
	}
	if len(c.Links) > 0 {
		m["links"] = c.Links
	}
	if len(c.Callbacks) > 0 {
		m["callbacks"] = c.Callbacks
	}
	if len(c.PathItems) > 0 {
		m["pathItems"] = c.PathItems
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range c.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Components.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (c *Components) UnmarshalJSON(data []byte) error {
	type Alias Components
	aux := (*Alias)(c)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		c.Extra = extra
	}

	return nil
}
