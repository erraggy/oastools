package parser

import (
	"encoding/json"
)

// MarshalJSON implements custom JSON marshaling for Parameter.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (p *Parameter) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(p.Extra) == 0 {
		type Alias Parameter
		return json.Marshal((*Alias)(p))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 30+len(p.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if p.Ref != "" {
		m["$ref"] = p.Ref
	}
	// Name and In are required, always include
	m["name"] = p.Name
	m["in"] = p.In
	if p.Description != "" {
		m["description"] = p.Description
	}
	if p.Required {
		m["required"] = p.Required
	}
	if p.Deprecated {
		m["deprecated"] = p.Deprecated
	}
	if p.Style != "" {
		m["style"] = p.Style
	}
	if p.Explode != nil {
		m["explode"] = p.Explode
	}
	if p.AllowReserved {
		m["allowReserved"] = p.AllowReserved
	}
	if p.Schema != nil {
		m["schema"] = p.Schema
	}
	if p.Example != nil {
		m["example"] = p.Example
	}
	if len(p.Examples) > 0 {
		m["examples"] = p.Examples
	}
	if len(p.Content) > 0 {
		m["content"] = p.Content
	}
	if p.Type != "" {
		m["type"] = p.Type
	}
	if p.Format != "" {
		m["format"] = p.Format
	}
	if p.AllowEmptyValue {
		m["allowEmptyValue"] = p.AllowEmptyValue
	}
	if p.Items != nil {
		m["items"] = p.Items
	}
	if p.CollectionFormat != "" {
		m["collectionFormat"] = p.CollectionFormat
	}
	if p.Default != nil {
		m["default"] = p.Default
	}
	if p.Maximum != nil {
		m["maximum"] = p.Maximum
	}
	if p.ExclusiveMaximum {
		m["exclusiveMaximum"] = p.ExclusiveMaximum
	}
	if p.Minimum != nil {
		m["minimum"] = p.Minimum
	}
	if p.ExclusiveMinimum {
		m["exclusiveMinimum"] = p.ExclusiveMinimum
	}
	if p.MaxLength != nil {
		m["maxLength"] = p.MaxLength
	}
	if p.MinLength != nil {
		m["minLength"] = p.MinLength
	}
	if p.Pattern != "" {
		m["pattern"] = p.Pattern
	}
	if p.MaxItems != nil {
		m["maxItems"] = p.MaxItems
	}
	if p.MinItems != nil {
		m["minItems"] = p.MinItems
	}
	if p.UniqueItems {
		m["uniqueItems"] = p.UniqueItems
	}
	if len(p.Enum) > 0 {
		m["enum"] = p.Enum
	}
	if p.MultipleOf != nil {
		m["multipleOf"] = p.MultipleOf
	}

	// Add Extra fields (spec extensions must start with "x-")
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

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
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
	// Fast path: no Extra fields, use standard marshaling
	if len(i.Extra) == 0 {
		type Alias Items
		return json.Marshal((*Alias)(i))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 18+len(i.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	// Type is required, always include
	m["type"] = i.Type
	if i.Format != "" {
		m["format"] = i.Format
	}
	if i.Items != nil {
		m["items"] = i.Items
	}
	if i.CollectionFormat != "" {
		m["collectionFormat"] = i.CollectionFormat
	}
	if i.Default != nil {
		m["default"] = i.Default
	}
	if i.Maximum != nil {
		m["maximum"] = i.Maximum
	}
	if i.ExclusiveMaximum {
		m["exclusiveMaximum"] = i.ExclusiveMaximum
	}
	if i.Minimum != nil {
		m["minimum"] = i.Minimum
	}
	if i.ExclusiveMinimum {
		m["exclusiveMinimum"] = i.ExclusiveMinimum
	}
	if i.MaxLength != nil {
		m["maxLength"] = i.MaxLength
	}
	if i.MinLength != nil {
		m["minLength"] = i.MinLength
	}
	if i.Pattern != "" {
		m["pattern"] = i.Pattern
	}
	if i.MaxItems != nil {
		m["maxItems"] = i.MaxItems
	}
	if i.MinItems != nil {
		m["minItems"] = i.MinItems
	}
	if i.UniqueItems {
		m["uniqueItems"] = i.UniqueItems
	}
	if len(i.Enum) > 0 {
		m["enum"] = i.Enum
	}
	if i.MultipleOf != nil {
		m["multipleOf"] = i.MultipleOf
	}

	// Add Extra fields (spec extensions must start with "x-")
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

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
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
	// Fast path: no Extra fields, use standard marshaling
	if len(rb.Extra) == 0 {
		type Alias RequestBody
		return json.Marshal((*Alias)(rb))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 4+len(rb.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if rb.Ref != "" {
		m["$ref"] = rb.Ref
	}
	if rb.Description != "" {
		m["description"] = rb.Description
	}
	// Content is required, always include
	m["content"] = rb.Content
	if rb.Required {
		m["required"] = rb.Required
	}

	// Add Extra fields (spec extensions must start with "x-")
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

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
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
	// Fast path: no Extra fields, use standard marshaling
	if len(h.Extra) == 0 {
		type Alias Header
		return json.Marshal((*Alias)(h))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 27+len(h.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if h.Ref != "" {
		m["$ref"] = h.Ref
	}
	if h.Description != "" {
		m["description"] = h.Description
	}
	if h.Required {
		m["required"] = h.Required
	}
	if h.Deprecated {
		m["deprecated"] = h.Deprecated
	}
	if h.Style != "" {
		m["style"] = h.Style
	}
	if h.Explode != nil {
		m["explode"] = h.Explode
	}
	if h.Schema != nil {
		m["schema"] = h.Schema
	}
	if h.Example != nil {
		m["example"] = h.Example
	}
	if len(h.Examples) > 0 {
		m["examples"] = h.Examples
	}
	if len(h.Content) > 0 {
		m["content"] = h.Content
	}
	if h.Type != "" {
		m["type"] = h.Type
	}
	if h.Format != "" {
		m["format"] = h.Format
	}
	if h.Items != nil {
		m["items"] = h.Items
	}
	if h.CollectionFormat != "" {
		m["collectionFormat"] = h.CollectionFormat
	}
	if h.Default != nil {
		m["default"] = h.Default
	}
	if h.Maximum != nil {
		m["maximum"] = h.Maximum
	}
	if h.ExclusiveMaximum {
		m["exclusiveMaximum"] = h.ExclusiveMaximum
	}
	if h.Minimum != nil {
		m["minimum"] = h.Minimum
	}
	if h.ExclusiveMinimum {
		m["exclusiveMinimum"] = h.ExclusiveMinimum
	}
	if h.MaxLength != nil {
		m["maxLength"] = h.MaxLength
	}
	if h.MinLength != nil {
		m["minLength"] = h.MinLength
	}
	if h.Pattern != "" {
		m["pattern"] = h.Pattern
	}
	if h.MaxItems != nil {
		m["maxItems"] = h.MaxItems
	}
	if h.MinItems != nil {
		m["minItems"] = h.MinItems
	}
	if h.UniqueItems {
		m["uniqueItems"] = h.UniqueItems
	}
	if len(h.Enum) > 0 {
		m["enum"] = h.Enum
	}
	if h.MultipleOf != nil {
		m["multipleOf"] = h.MultipleOf
	}

	// Add Extra fields (spec extensions must start with "x-")
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

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		h.Extra = extra
	}

	return nil
}
