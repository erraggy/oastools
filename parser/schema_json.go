package parser

import (
	"encoding/json"
)

// MarshalJSON implements custom JSON marshaling for Schema.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (s *Schema) MarshalJSON() ([]byte, error) {
	type Alias Schema
	aux, err := json.Marshal((*Alias)(s))
	if err != nil {
		return nil, err
	}

	if len(s.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range s.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Schema.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (s *Schema) UnmarshalJSON(data []byte) error {
	type Alias Schema
	aux := (*Alias)(s)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	knownFields := map[string]bool{
		"$ref": true, "$schema": true, "title": true, "description": true, "default": true,
		"examples": true, "type": true, "enum": true, "const": true,
		"multipleOf": true, "maximum": true, "exclusiveMaximum": true, "minimum": true, "exclusiveMinimum": true,
		"maxLength": true, "minLength": true, "pattern": true,
		"items": true, "prefixItems": true, "additionalItems": true, "maxItems": true, "minItems": true,
		"uniqueItems": true, "contains": true, "maxContains": true, "minContains": true,
		"properties": true, "patternProperties": true, "additionalProperties": true, "required": true,
		"propertyNames": true, "maxProperties": true, "minProperties": true, "dependentRequired": true, "dependentSchemas": true,
		"if": true, "then": true, "else": true,
		"allOf": true, "anyOf": true, "oneOf": true, "not": true,
		"nullable": true, "discriminator": true, "readOnly": true, "writeOnly": true, "xml": true,
		"externalDocs": true, "example": true, "deprecated": true, "format": true, "collectionFormat": true,
		"$id": true, "$anchor": true, "$dynamicRef": true, "$dynamicAnchor": true, "$vocabulary": true, "$comment": true, "$defs": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		s.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Discriminator.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (d *Discriminator) MarshalJSON() ([]byte, error) {
	type Alias Discriminator
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

// UnmarshalJSON implements custom JSON unmarshaling for Discriminator.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (d *Discriminator) UnmarshalJSON(data []byte) error {
	type Alias Discriminator
	aux := (*Alias)(d)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	knownFields := map[string]bool{
		"propertyName": true,
		"mapping":      true,
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

// MarshalJSON implements custom JSON marshaling for XML.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (x *XML) MarshalJSON() ([]byte, error) {
	type Alias XML
	aux, err := json.Marshal((*Alias)(x))
	if err != nil {
		return nil, err
	}

	if len(x.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range x.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for XML.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (x *XML) UnmarshalJSON(data []byte) error {
	type Alias XML
	aux := (*Alias)(x)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	knownFields := map[string]bool{
		"name":      true,
		"namespace": true,
		"prefix":    true,
		"attribute": true,
		"wrapped":   true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		x.Extra = extra
	}

	return nil
}
