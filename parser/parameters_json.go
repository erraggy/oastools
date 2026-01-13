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
		return marshalToJSON((*Alias)(p))
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
	jsonhelpers.SetIfTrue(m, "allowEmptyValue", p.AllowEmptyValue)
	jsonhelpers.SetOAS2PrimitiveFields(m, jsonhelpers.OAS2PrimitiveFields{
		Type: p.Type, Format: p.Format, Items: p.Items,
		CollectionFormat: p.CollectionFormat, Default: p.Default,
	})
	jsonhelpers.SetSchemaConstraints(m, jsonhelpers.SchemaConstraints{
		Maximum: p.Maximum, ExclusiveMaximum: p.ExclusiveMaximum,
		Minimum: p.Minimum, ExclusiveMinimum: p.ExclusiveMinimum,
		MaxLength: p.MaxLength, MinLength: p.MinLength, Pattern: p.Pattern,
		MaxItems: p.MaxItems, MinItems: p.MinItems, UniqueItems: p.UniqueItems,
		Enum: p.Enum, MultipleOf: p.MultipleOf,
	})

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
		return marshalToJSON((*Alias)(i))
	}

	// Build map with known fields
	m := map[string]any{
		"type": i.Type, // Required field, always include
	}
	jsonhelpers.SetOAS2PrimitiveFields(m, jsonhelpers.OAS2PrimitiveFields{
		Format: i.Format, Items: i.Items,
		CollectionFormat: i.CollectionFormat, Default: i.Default,
	})
	jsonhelpers.SetSchemaConstraints(m, jsonhelpers.SchemaConstraints{
		Maximum: i.Maximum, ExclusiveMaximum: i.ExclusiveMaximum,
		Minimum: i.Minimum, ExclusiveMinimum: i.ExclusiveMinimum,
		MaxLength: i.MaxLength, MinLength: i.MinLength, Pattern: i.Pattern,
		MaxItems: i.MaxItems, MinItems: i.MinItems, UniqueItems: i.UniqueItems,
		Enum: i.Enum, MultipleOf: i.MultipleOf,
	})

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
		return marshalToJSON((*Alias)(rb))
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
		return marshalToJSON((*Alias)(h))
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
	jsonhelpers.SetOAS2PrimitiveFields(m, jsonhelpers.OAS2PrimitiveFields{
		Type: h.Type, Format: h.Format, Items: h.Items,
		CollectionFormat: h.CollectionFormat, Default: h.Default,
	})
	jsonhelpers.SetSchemaConstraints(m, jsonhelpers.SchemaConstraints{
		Maximum: h.Maximum, ExclusiveMaximum: h.ExclusiveMaximum,
		Minimum: h.Minimum, ExclusiveMinimum: h.ExclusiveMinimum,
		MaxLength: h.MaxLength, MinLength: h.MinLength, Pattern: h.Pattern,
		MaxItems: h.MaxItems, MinItems: h.MinItems, UniqueItems: h.UniqueItems,
		Enum: h.Enum, MultipleOf: h.MultipleOf,
	})

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
