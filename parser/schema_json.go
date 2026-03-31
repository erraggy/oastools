package parser

import (
	"encoding/json"
	"fmt"

	"github.com/erraggy/oastools/parser/internal/jsonhelpers"
)

// MarshalJSON implements custom JSON marshaling for Schema.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (s *Schema) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(s.Extra) == 0 {
		type Alias Schema
		return marshalToJSON((*Alias)(s))
	}

	// Build map with known fields
	m := make(map[string]any, 50+len(s.Extra))

	// Add known fields (using helpers to omit zero values)
	jsonhelpers.SetIfNotEmpty(m, "$ref", s.Ref)
	jsonhelpers.SetIfNotEmpty(m, "$schema", s.Schema)
	jsonhelpers.SetIfNotEmpty(m, "title", s.Title)
	jsonhelpers.SetIfNotEmpty(m, "description", s.Description)
	jsonhelpers.SetIfNotNil(m, "default", s.Default)
	jsonhelpers.SetIfSliceNotEmpty(m, "examples", s.Examples)
	jsonhelpers.SetIfNotNil(m, "type", s.Type)
	jsonhelpers.SetIfNotNil(m, "enum", s.Enum)
	jsonhelpers.SetIfNotNil(m, "const", s.Const)
	jsonhelpers.SetIfNotNil(m, "multipleOf", s.MultipleOf)
	jsonhelpers.SetIfNotNil(m, "maximum", s.Maximum)
	jsonhelpers.SetIfNotNil(m, "exclusiveMaximum", s.ExclusiveMaximum)
	jsonhelpers.SetIfNotNil(m, "minimum", s.Minimum)
	jsonhelpers.SetIfNotNil(m, "exclusiveMinimum", s.ExclusiveMinimum)
	jsonhelpers.SetIfNotNil(m, "maxLength", s.MaxLength)
	jsonhelpers.SetIfNotNil(m, "minLength", s.MinLength)
	jsonhelpers.SetIfNotEmpty(m, "pattern", s.Pattern)
	jsonhelpers.SetIfNotNil(m, "items", s.Items)
	jsonhelpers.SetIfNotNil(m, "prefixItems", s.PrefixItems)
	jsonhelpers.SetIfNotNil(m, "additionalItems", s.AdditionalItems)
	jsonhelpers.SetIfNotNil(m, "maxItems", s.MaxItems)
	jsonhelpers.SetIfNotNil(m, "minItems", s.MinItems)
	jsonhelpers.SetIfTrue(m, "uniqueItems", s.UniqueItems)
	jsonhelpers.SetIfNotNil(m, "contains", s.Contains)
	jsonhelpers.SetIfNotNil(m, "maxContains", s.MaxContains)
	jsonhelpers.SetIfNotNil(m, "minContains", s.MinContains)
	jsonhelpers.SetIfNotNil(m, "properties", s.Properties)
	jsonhelpers.SetIfNotNil(m, "patternProperties", s.PatternProperties)
	jsonhelpers.SetIfNotNil(m, "additionalProperties", s.AdditionalProperties)
	jsonhelpers.SetIfNotNil(m, "required", s.Required)
	jsonhelpers.SetIfNotNil(m, "propertyNames", s.PropertyNames)
	jsonhelpers.SetIfNotNil(m, "maxProperties", s.MaxProperties)
	jsonhelpers.SetIfNotNil(m, "minProperties", s.MinProperties)
	jsonhelpers.SetIfNotNil(m, "dependentRequired", s.DependentRequired)
	jsonhelpers.SetIfNotNil(m, "dependentSchemas", s.DependentSchemas)
	jsonhelpers.SetIfNotNil(m, "if", s.If)
	jsonhelpers.SetIfNotNil(m, "then", s.Then)
	jsonhelpers.SetIfNotNil(m, "else", s.Else)
	jsonhelpers.SetIfNotNil(m, "allOf", s.AllOf)
	jsonhelpers.SetIfNotNil(m, "anyOf", s.AnyOf)
	jsonhelpers.SetIfNotNil(m, "oneOf", s.OneOf)
	jsonhelpers.SetIfNotNil(m, "not", s.Not)
	jsonhelpers.SetIfTrue(m, "nullable", s.Nullable)
	jsonhelpers.SetIfNotNil(m, "discriminator", s.Discriminator)
	jsonhelpers.SetIfTrue(m, "readOnly", s.ReadOnly)
	jsonhelpers.SetIfTrue(m, "writeOnly", s.WriteOnly)
	jsonhelpers.SetIfNotNil(m, "xml", s.XML)
	jsonhelpers.SetIfNotNil(m, "externalDocs", s.ExternalDocs)
	jsonhelpers.SetIfNotNil(m, "example", s.Example)
	jsonhelpers.SetIfTrue(m, "deprecated", s.Deprecated)
	jsonhelpers.SetIfNotEmpty(m, "format", s.Format)
	jsonhelpers.SetIfNotEmpty(m, "collectionFormat", s.CollectionFormat)
	jsonhelpers.SetIfNotEmpty(m, "$id", s.ID)
	jsonhelpers.SetIfNotEmpty(m, "$anchor", s.Anchor)
	jsonhelpers.SetIfNotEmpty(m, "$dynamicRef", s.DynamicRef)
	jsonhelpers.SetIfNotEmpty(m, "$dynamicAnchor", s.DynamicAnchor)
	jsonhelpers.SetIfNotNil(m, "$vocabulary", s.Vocabulary)
	jsonhelpers.SetIfNotEmpty(m, "$comment", s.Comment)
	jsonhelpers.SetIfNotNil(m, "$defs", s.Defs)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, s.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Schema.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (s *Schema) UnmarshalJSON(data []byte) error {
	type Alias Schema
	if err := json.Unmarshal(data, (*Alias)(s)); err != nil {
		return err
	}
	s.Extra = jsonhelpers.ExtractExtensions(data)
	// The Alias trick bypasses custom unmarshalers, causing encoding/json to decode
	// any-typed fields (Items, AdditionalProperties, etc.) as map[string]any instead
	// of *Schema. Promote them back so downstream type assertions work correctly.
	var err error
	if s.Items, err = promoteSchemaOrBool(s.Items); err != nil {
		return err
	}
	if s.AdditionalProperties, err = promoteSchemaOrBool(s.AdditionalProperties); err != nil {
		return err
	}
	if s.AdditionalItems, err = promoteSchemaOrBool(s.AdditionalItems); err != nil {
		return err
	}
	if s.UnevaluatedItems, err = promoteSchemaOrBool(s.UnevaluatedItems); err != nil {
		return err
	}
	if s.UnevaluatedProperties, err = promoteSchemaOrBool(s.UnevaluatedProperties); err != nil {
		return err
	}
	return nil
}

// promoteSchemaOrBool converts a map[string]any value (produced by the standard
// JSON decoder for any-typed schema fields) into a *Schema. Bool values and
// already-typed *Schema values pass through unchanged. Returns an error if the
// map cannot be round-tripped through JSON into a *Schema, so callers get a
// clear parse error rather than a silent type-assertion panic downstream.
func promoteSchemaOrBool(v any) (any, error) {
	switch val := v.(type) {
	case nil, bool, *Schema:
		return v, nil
	case map[string]any:
		data, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("parser: schema field promotion: %w", err)
		}
		s := &Schema{}
		if err := json.Unmarshal(data, s); err != nil {
			return nil, fmt.Errorf("parser: schema field promotion: %w", err)
		}
		return s, nil
	}
	return v, nil
}

// MarshalJSON implements custom JSON marshaling for Discriminator.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (d *Discriminator) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(d.Extra) == 0 {
		type Alias Discriminator
		return marshalToJSON((*Alias)(d))
	}

	// Build map with known fields
	m := map[string]any{
		"propertyName": d.PropertyName, // Required field, always include
	}
	jsonhelpers.SetIfNotNil(m, "mapping", d.Mapping)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, d.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Discriminator.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (d *Discriminator) UnmarshalJSON(data []byte) error {
	type Alias Discriminator
	if err := json.Unmarshal(data, (*Alias)(d)); err != nil {
		return err
	}
	d.Extra = jsonhelpers.ExtractExtensions(data)
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
		return marshalToJSON((*Alias)(x))
	}

	// Build map with known fields
	m := make(map[string]any, 5+len(x.Extra))
	jsonhelpers.SetIfNotEmpty(m, "name", x.Name)
	jsonhelpers.SetIfNotEmpty(m, "namespace", x.Namespace)
	jsonhelpers.SetIfNotEmpty(m, "prefix", x.Prefix)
	jsonhelpers.SetIfTrue(m, "attribute", x.Attribute)
	jsonhelpers.SetIfTrue(m, "wrapped", x.Wrapped)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, x.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for XML.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (x *XML) UnmarshalJSON(data []byte) error {
	type Alias XML
	if err := json.Unmarshal(data, (*Alias)(x)); err != nil {
		return err
	}
	x.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}
