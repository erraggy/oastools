package parser

import (
	"encoding/json"
)

// MarshalJSON implements custom JSON marshaling for OAS2Document.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline". The Extra map is merged after marshaling
// the base struct to ensure specification extensions appear at the root level.
func (d *OAS2Document) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(d.Extra) == 0 {
		type Alias OAS2Document
		return json.Marshal((*Alias)(d))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]any, 15+len(d.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	// Swagger and Info are required, always include
	m["swagger"] = d.Swagger
	m["info"] = d.Info
	if d.Host != "" {
		m["host"] = d.Host
	}
	if d.BasePath != "" {
		m["basePath"] = d.BasePath
	}
	if len(d.Schemes) > 0 {
		m["schemes"] = d.Schemes
	}
	if len(d.Consumes) > 0 {
		m["consumes"] = d.Consumes
	}
	if len(d.Produces) > 0 {
		m["produces"] = d.Produces
	}
	// Paths is required, always include
	m["paths"] = d.Paths
	if len(d.Definitions) > 0 {
		m["definitions"] = d.Definitions
	}
	if len(d.Parameters) > 0 {
		m["parameters"] = d.Parameters
	}
	if len(d.Responses) > 0 {
		m["responses"] = d.Responses
	}
	if len(d.SecurityDefinitions) > 0 {
		m["securityDefinitions"] = d.SecurityDefinitions
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

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range d.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for OAS2Document.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
// It unmarshals known fields using the standard decoder, then identifies and stores
// any fields not defined in the OAS 2.0 specification in the Extra map.
func (d *OAS2Document) UnmarshalJSON(data []byte) error {
	type Alias OAS2Document
	aux := (*Alias)(d)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]any)
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
