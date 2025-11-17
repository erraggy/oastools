package parser

import (
	"encoding/json"
)

// MarshalJSON implements custom JSON marshaling for OAS3Document.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (d *OAS3Document) MarshalJSON() ([]byte, error) {
	type Alias OAS3Document
	aux, err := json.Marshal((*Alias)(d))
	if err != nil {
		return nil, err
	}

	if len(d.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

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

	knownFields := map[string]bool{
		"openapi":           true,
		"info":              true,
		"servers":           true,
		"paths":             true,
		"webhooks":          true,
		"components":        true,
		"security":          true,
		"tags":              true,
		"externalDocs":      true,
		"jsonSchemaDialect": true,
	}

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

// MarshalJSON implements custom JSON marshaling for Components.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (c *Components) MarshalJSON() ([]byte, error) {
	type Alias Components
	aux, err := json.Marshal((*Alias)(c))
	if err != nil {
		return nil, err
	}

	if len(c.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

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

	knownFields := map[string]bool{
		"schemas":         true,
		"responses":       true,
		"parameters":      true,
		"examples":        true,
		"requestBodies":   true,
		"headers":         true,
		"securitySchemes": true,
		"links":           true,
		"callbacks":       true,
		"pathItems":       true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		c.Extra = extra
	}

	return nil
}
