package parser

import (
	"encoding/json"
	"maps"

	"github.com/erraggy/oastools/parser/internal/jsonhelpers"
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
		return marshalToJSON((*Alias)(d))
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
	maps.Copy(m, d.Extra)

	return marshalToJSON(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for OAS2Document.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (d *OAS2Document) UnmarshalJSON(data []byte) error {
	type Alias OAS2Document
	if err := json.Unmarshal(data, (*Alias)(d)); err != nil {
		return err
	}
	d.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}
