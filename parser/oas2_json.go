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
	// Create an alias type to avoid recursion
	type Alias OAS2Document

	// Marshal the base struct
	aux, err := json.Marshal((*Alias)(d))
	if err != nil {
		return nil, err
	}

	// If there's no Extra, return as-is
	if len(d.Extra) == 0 {
		return aux, nil
	}

	// Unmarshal into a map
	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	// Merge Extra fields into the map
	for k, v := range d.Extra {
		m[k] = v
	}

	// Marshal the final result
	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for OAS2Document.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
// It unmarshals known fields using the standard decoder, then identifies and stores
// any fields not defined in the OAS 2.0 specification in the Extra map.
func (d *OAS2Document) UnmarshalJSON(data []byte) error {
	// Create an alias type to avoid recursion
	type Alias OAS2Document
	aux := (*Alias)(d)

	// Unmarshal known fields
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Unmarshal into a map to find unknown fields
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Known field names from OAS 2.0 spec
	knownFields := map[string]bool{
		"swagger":             true,
		"info":                true,
		"host":                true,
		"basePath":            true,
		"schemes":             true,
		"consumes":            true,
		"produces":            true,
		"paths":               true,
		"definitions":         true,
		"parameters":          true,
		"responses":           true,
		"securityDefinitions": true,
		"security":            true,
		"tags":                true,
		"externalDocs":        true,
	}

	// Collect unknown fields
	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		d.Extra = extra
	}

	return nil
}
