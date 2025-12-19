package parser

import (
	"encoding/json"

	"github.com/erraggy/oastools/parser/internal/jsonhelpers"
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

	// Build map with known fields
	m := map[string]any{
		"name": p.Name, // Required field, always include
		"in":   p.In,   // Required field, always include
	}
	jsonhelpers.SetIfNotEmpty(m, "$ref", p.Ref)
	jsonhelpers.SetIfNotEmpty(m, "description", p.Description)
	jsonhelpers.SetIfTrue(m, "required", p.Required)
	jsonhelpers.SetIfTrue(m, "deprecated", p.Deprecated)
	jsonhelpers.SetIfNotEmpty(m, "style", p.Style)
	jsonhelpers.SetIfNotNil(m, "explode", p.Explode)
	jsonhelpers.SetIfTrue(m, "allowReserved", p.AllowReserved)
	jsonhelpers.SetIfNotNil(m, "schema", p.Schema)
	jsonhelpers.SetIfNotNil(m, "example", p.Example)
	jsonhelpers.SetIfMapNotEmpty(m, "examples", p.Examples)
	jsonhelpers.SetIfMapNotEmpty(m, "content", p.Content)
	jsonhelpers.SetIfNotEmpty(m, "type", p.Type)
	jsonhelpers.SetIfNotEmpty(m, "format", p.Format)
	jsonhelpers.SetIfTrue(m, "allowEmptyValue", p.AllowEmptyValue)
	jsonhelpers.SetIfNotNil(m, "items", p.Items)
	jsonhelpers.SetIfNotEmpty(m, "collectionFormat", p.CollectionFormat)
	jsonhelpers.SetIfNotNil(m, "default", p.Default)
	jsonhelpers.SetIfNotNil(m, "maximum", p.Maximum)
	jsonhelpers.SetIfTrue(m, "exclusiveMaximum", p.ExclusiveMaximum)
	jsonhelpers.SetIfNotNil(m, "minimum", p.Minimum)
	jsonhelpers.SetIfTrue(m, "exclusiveMinimum", p.ExclusiveMinimum)
	jsonhelpers.SetIfNotNil(m, "maxLength", p.MaxLength)
	jsonhelpers.SetIfNotNil(m, "minLength", p.MinLength)
	jsonhelpers.SetIfNotEmpty(m, "pattern", p.Pattern)
	jsonhelpers.SetIfNotNil(m, "maxItems", p.MaxItems)
	jsonhelpers.SetIfNotNil(m, "minItems", p.MinItems)
	jsonhelpers.SetIfTrue(m, "uniqueItems", p.UniqueItems)
	jsonhelpers.SetIfNotNil(m, "enum", p.Enum)
	jsonhelpers.SetIfNotNil(m, "multipleOf", p.MultipleOf)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, p.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Parameter.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (p *Parameter) UnmarshalJSON(data []byte) error {
	type Alias Parameter
	if err := json.Unmarshal(data, (*Alias)(p)); err != nil {
		return err
	}
	p.Extra = jsonhelpers.ExtractExtensions(data)
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

	// Build map with known fields
	m := map[string]any{
		"type": i.Type, // Required field, always include
	}
	jsonhelpers.SetIfNotEmpty(m, "format", i.Format)
	jsonhelpers.SetIfNotNil(m, "items", i.Items)
	jsonhelpers.SetIfNotEmpty(m, "collectionFormat", i.CollectionFormat)
	jsonhelpers.SetIfNotNil(m, "default", i.Default)
	jsonhelpers.SetIfNotNil(m, "maximum", i.Maximum)
	jsonhelpers.SetIfTrue(m, "exclusiveMaximum", i.ExclusiveMaximum)
	jsonhelpers.SetIfNotNil(m, "minimum", i.Minimum)
	jsonhelpers.SetIfTrue(m, "exclusiveMinimum", i.ExclusiveMinimum)
	jsonhelpers.SetIfNotNil(m, "maxLength", i.MaxLength)
	jsonhelpers.SetIfNotNil(m, "minLength", i.MinLength)
	jsonhelpers.SetIfNotEmpty(m, "pattern", i.Pattern)
	jsonhelpers.SetIfNotNil(m, "maxItems", i.MaxItems)
	jsonhelpers.SetIfNotNil(m, "minItems", i.MinItems)
	jsonhelpers.SetIfTrue(m, "uniqueItems", i.UniqueItems)
	jsonhelpers.SetIfNotNil(m, "enum", i.Enum)
	jsonhelpers.SetIfNotNil(m, "multipleOf", i.MultipleOf)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, i.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Items.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (i *Items) UnmarshalJSON(data []byte) error {
	type Alias Items
	if err := json.Unmarshal(data, (*Alias)(i)); err != nil {
		return err
	}
	i.Extra = jsonhelpers.ExtractExtensions(data)
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

	// Build map with known fields
	m := map[string]any{
		"content": rb.Content, // Required field, always include
	}
	jsonhelpers.SetIfNotEmpty(m, "$ref", rb.Ref)
	jsonhelpers.SetIfNotEmpty(m, "description", rb.Description)
	jsonhelpers.SetIfTrue(m, "required", rb.Required)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, rb.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for RequestBody.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (rb *RequestBody) UnmarshalJSON(data []byte) error {
	type Alias RequestBody
	if err := json.Unmarshal(data, (*Alias)(rb)); err != nil {
		return err
	}
	rb.Extra = jsonhelpers.ExtractExtensions(data)
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

	// Build map with known fields
	m := make(map[string]any, 27+len(h.Extra))
	jsonhelpers.SetIfNotEmpty(m, "$ref", h.Ref)
	jsonhelpers.SetIfNotEmpty(m, "description", h.Description)
	jsonhelpers.SetIfTrue(m, "required", h.Required)
	jsonhelpers.SetIfTrue(m, "deprecated", h.Deprecated)
	jsonhelpers.SetIfNotEmpty(m, "style", h.Style)
	jsonhelpers.SetIfNotNil(m, "explode", h.Explode)
	jsonhelpers.SetIfNotNil(m, "schema", h.Schema)
	jsonhelpers.SetIfNotNil(m, "example", h.Example)
	jsonhelpers.SetIfMapNotEmpty(m, "examples", h.Examples)
	jsonhelpers.SetIfMapNotEmpty(m, "content", h.Content)
	jsonhelpers.SetIfNotEmpty(m, "type", h.Type)
	jsonhelpers.SetIfNotEmpty(m, "format", h.Format)
	jsonhelpers.SetIfNotNil(m, "items", h.Items)
	jsonhelpers.SetIfNotEmpty(m, "collectionFormat", h.CollectionFormat)
	jsonhelpers.SetIfNotNil(m, "default", h.Default)
	jsonhelpers.SetIfNotNil(m, "maximum", h.Maximum)
	jsonhelpers.SetIfTrue(m, "exclusiveMaximum", h.ExclusiveMaximum)
	jsonhelpers.SetIfNotNil(m, "minimum", h.Minimum)
	jsonhelpers.SetIfTrue(m, "exclusiveMinimum", h.ExclusiveMinimum)
	jsonhelpers.SetIfNotNil(m, "maxLength", h.MaxLength)
	jsonhelpers.SetIfNotNil(m, "minLength", h.MinLength)
	jsonhelpers.SetIfNotEmpty(m, "pattern", h.Pattern)
	jsonhelpers.SetIfNotNil(m, "maxItems", h.MaxItems)
	jsonhelpers.SetIfNotNil(m, "minItems", h.MinItems)
	jsonhelpers.SetIfTrue(m, "uniqueItems", h.UniqueItems)
	jsonhelpers.SetIfNotNil(m, "enum", h.Enum)
	jsonhelpers.SetIfNotNil(m, "multipleOf", h.MultipleOf)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, h.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Header.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (h *Header) UnmarshalJSON(data []byte) error {
	type Alias Header
	if err := json.Unmarshal(data, (*Alias)(h)); err != nil {
		return err
	}
	h.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}
