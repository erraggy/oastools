package parser

import (
	"encoding/json"
)

// MarshalJSON implements custom JSON marshaling for Schema.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
//
//nolint:cyclop // Schema has 50+ fields per OpenAPI spec, complexity is inherent
func (s *Schema) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(s.Extra) == 0 {
		type Alias Schema
		return json.Marshal((*Alias)(s))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]any, 50+len(s.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if s.Ref != "" {
		m["$ref"] = s.Ref
	}
	if s.Schema != "" {
		m["$schema"] = s.Schema
	}
	if s.Title != "" {
		m["title"] = s.Title
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if s.Default != nil {
		m["default"] = s.Default
	}
	if len(s.Examples) > 0 {
		m["examples"] = s.Examples
	}
	if s.Type != nil {
		m["type"] = s.Type
	}
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}
	if s.Const != nil {
		m["const"] = s.Const
	}
	if s.MultipleOf != nil {
		m["multipleOf"] = s.MultipleOf
	}
	if s.Maximum != nil {
		m["maximum"] = s.Maximum
	}
	if s.ExclusiveMaximum != nil {
		m["exclusiveMaximum"] = s.ExclusiveMaximum
	}
	if s.Minimum != nil {
		m["minimum"] = s.Minimum
	}
	if s.ExclusiveMinimum != nil {
		m["exclusiveMinimum"] = s.ExclusiveMinimum
	}
	if s.MaxLength != nil {
		m["maxLength"] = s.MaxLength
	}
	if s.MinLength != nil {
		m["minLength"] = s.MinLength
	}
	if s.Pattern != "" {
		m["pattern"] = s.Pattern
	}
	if s.Items != nil {
		m["items"] = s.Items
	}
	if len(s.PrefixItems) > 0 {
		m["prefixItems"] = s.PrefixItems
	}
	if s.AdditionalItems != nil {
		m["additionalItems"] = s.AdditionalItems
	}
	if s.MaxItems != nil {
		m["maxItems"] = s.MaxItems
	}
	if s.MinItems != nil {
		m["minItems"] = s.MinItems
	}
	if s.UniqueItems {
		m["uniqueItems"] = s.UniqueItems
	}
	if s.Contains != nil {
		m["contains"] = s.Contains
	}
	if s.MaxContains != nil {
		m["maxContains"] = s.MaxContains
	}
	if s.MinContains != nil {
		m["minContains"] = s.MinContains
	}
	if len(s.Properties) > 0 {
		m["properties"] = s.Properties
	}
	if len(s.PatternProperties) > 0 {
		m["patternProperties"] = s.PatternProperties
	}
	if s.AdditionalProperties != nil {
		m["additionalProperties"] = s.AdditionalProperties
	}
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}
	if s.PropertyNames != nil {
		m["propertyNames"] = s.PropertyNames
	}
	if s.MaxProperties != nil {
		m["maxProperties"] = s.MaxProperties
	}
	if s.MinProperties != nil {
		m["minProperties"] = s.MinProperties
	}
	if len(s.DependentRequired) > 0 {
		m["dependentRequired"] = s.DependentRequired
	}
	if len(s.DependentSchemas) > 0 {
		m["dependentSchemas"] = s.DependentSchemas
	}
	if s.If != nil {
		m["if"] = s.If
	}
	if s.Then != nil {
		m["then"] = s.Then
	}
	if s.Else != nil {
		m["else"] = s.Else
	}
	if len(s.AllOf) > 0 {
		m["allOf"] = s.AllOf
	}
	if len(s.AnyOf) > 0 {
		m["anyOf"] = s.AnyOf
	}
	if len(s.OneOf) > 0 {
		m["oneOf"] = s.OneOf
	}
	if s.Not != nil {
		m["not"] = s.Not
	}
	if s.Nullable {
		m["nullable"] = s.Nullable
	}
	if s.Discriminator != nil {
		m["discriminator"] = s.Discriminator
	}
	if s.ReadOnly {
		m["readOnly"] = s.ReadOnly
	}
	if s.WriteOnly {
		m["writeOnly"] = s.WriteOnly
	}
	if s.XML != nil {
		m["xml"] = s.XML
	}
	if s.ExternalDocs != nil {
		m["externalDocs"] = s.ExternalDocs
	}
	if s.Example != nil {
		m["example"] = s.Example
	}
	if s.Deprecated {
		m["deprecated"] = s.Deprecated
	}
	if s.Format != "" {
		m["format"] = s.Format
	}
	if s.CollectionFormat != "" {
		m["collectionFormat"] = s.CollectionFormat
	}
	if s.ID != "" {
		m["$id"] = s.ID
	}
	if s.Anchor != "" {
		m["$anchor"] = s.Anchor
	}
	if s.DynamicRef != "" {
		m["$dynamicRef"] = s.DynamicRef
	}
	if s.DynamicAnchor != "" {
		m["$dynamicAnchor"] = s.DynamicAnchor
	}
	if len(s.Vocabulary) > 0 {
		m["$vocabulary"] = s.Vocabulary
	}
	if s.Comment != "" {
		m["$comment"] = s.Comment
	}
	if len(s.Defs) > 0 {
		m["$defs"] = s.Defs
	}

	// Add Extra fields (spec extensions must start with "x-")
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
		s.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Discriminator.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (d *Discriminator) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(d.Extra) == 0 {
		type Alias Discriminator
		return json.Marshal((*Alias)(d))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]any, 2+len(d.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	// PropertyName is required, always include
	m["propertyName"] = d.PropertyName
	if len(d.Mapping) > 0 {
		m["mapping"] = d.Mapping
	}

	// Add Extra fields (spec extensions must start with "x-")
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

// MarshalJSON implements custom JSON marshaling for XML.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (x *XML) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(x.Extra) == 0 {
		type Alias XML
		return json.Marshal((*Alias)(x))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]any, 5+len(x.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if x.Name != "" {
		m["name"] = x.Name
	}
	if x.Namespace != "" {
		m["namespace"] = x.Namespace
	}
	if x.Prefix != "" {
		m["prefix"] = x.Prefix
	}
	if x.Attribute {
		m["attribute"] = x.Attribute
	}
	if x.Wrapped {
		m["wrapped"] = x.Wrapped
	}

	// Add Extra fields (spec extensions must start with "x-")
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
		x.Extra = extra
	}

	return nil
}
