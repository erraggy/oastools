package parser

import (
	"encoding/json"
)

// MarshalJSON implements custom JSON marshaling for Parameter.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (p *Parameter) MarshalJSON() ([]byte, error) {
	type Alias Parameter
	aux, err := json.Marshal((*Alias)(p))
	if err != nil {
		return nil, err
	}

	if len(p.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range p.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Parameter.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (p *Parameter) UnmarshalJSON(data []byte) error {
	type Alias Parameter
	aux := (*Alias)(p)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	knownFields := map[string]bool{
		"$ref": true, "name": true, "in": true, "description": true, "required": true, "deprecated": true,
		"style": true, "explode": true, "allowReserved": true, "schema": true, "example": true, "examples": true, "content": true,
		"type": true, "format": true, "allowEmptyValue": true, "items": true, "collectionFormat": true, "default": true,
		"maximum": true, "exclusiveMaximum": true, "minimum": true, "exclusiveMinimum": true,
		"maxLength": true, "minLength": true, "pattern": true, "maxItems": true, "minItems": true, "uniqueItems": true, "enum": true, "multipleOf": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		p.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Items.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (i *Items) MarshalJSON() ([]byte, error) {
	type Alias Items
	aux, err := json.Marshal((*Alias)(i))
	if err != nil {
		return nil, err
	}

	if len(i.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range i.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Items.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (i *Items) UnmarshalJSON(data []byte) error {
	type Alias Items
	aux := (*Alias)(i)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	knownFields := map[string]bool{
		"type": true, "format": true, "items": true, "collectionFormat": true, "default": true,
		"maximum": true, "exclusiveMaximum": true, "minimum": true, "exclusiveMinimum": true,
		"maxLength": true, "minLength": true, "pattern": true, "maxItems": true, "minItems": true, "uniqueItems": true, "enum": true, "multipleOf": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		i.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for RequestBody.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (rb *RequestBody) MarshalJSON() ([]byte, error) {
	type Alias RequestBody
	aux, err := json.Marshal((*Alias)(rb))
	if err != nil {
		return nil, err
	}

	if len(rb.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range rb.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for RequestBody.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (rb *RequestBody) UnmarshalJSON(data []byte) error {
	type Alias RequestBody
	aux := (*Alias)(rb)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	knownFields := map[string]bool{
		"$ref":        true,
		"description": true,
		"content":     true,
		"required":    true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		rb.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Header.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (h *Header) MarshalJSON() ([]byte, error) {
	type Alias Header
	aux, err := json.Marshal((*Alias)(h))
	if err != nil {
		return nil, err
	}

	if len(h.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range h.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Header.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (h *Header) UnmarshalJSON(data []byte) error {
	type Alias Header
	aux := (*Alias)(h)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	knownFields := map[string]bool{
		"$ref": true, "description": true, "required": true, "deprecated": true,
		"style": true, "explode": true, "schema": true, "example": true, "examples": true, "content": true,
		"type": true, "format": true, "items": true, "collectionFormat": true, "default": true,
		"maximum": true, "exclusiveMaximum": true, "minimum": true, "exclusiveMinimum": true,
		"maxLength": true, "minLength": true, "pattern": true, "maxItems": true, "minItems": true, "uniqueItems": true, "enum": true, "multipleOf": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		h.Extra = extra
	}

	return nil
}
