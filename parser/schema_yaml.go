package parser

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

// UnmarshalYAML implements custom YAML unmarshaling for Schema.
//
// The alias type sheds this method so default struct decoding (including the
// inline Extra map) applies, then the any-typed schema-or-bool fields are
// promoted to *Schema. Without the promotion the decoder leaves a nested
// mapping as map[string]any, and every consumer that type-asserts to *Schema
// silently skips the subtree: validation, $ref rewriting, and fixes inside
// `items:` would all no-op.
//
// Schema.UnmarshalJSON does the same job for JSON, and decodeFromMap for the
// ResolveRefs path. All three must agree on the decoded types.
func (s *Schema) UnmarshalYAML(node *yaml.Node) error {
	// A schema may be a bare boolean in JSON Schema 2020-12, which OAS 3.1+
	// adopts — `true` accepts anything, `false` accepts nothing. Decoding a
	// scalar into the struct alias below fails, so catch it first and record
	// the spelling. See Schema.BoolForm.
	if b, ok := boolSchemaNode(node); ok {
		*s = Schema{BoolForm: &b}
		return nil
	}

	type Alias Schema
	var alias Alias
	if err := node.Decode(&alias); err != nil {
		return err
	}
	*s = Schema(alias)

	// Promotion reads the mapping's keys, so the wrappers have to come off
	// first. A nil result only means there is nothing to look inside.
	node = unwrapSchemaNode(node)

	var err error
	if s.Items, err = promoteYAMLSchemaOrBool(s.Items, node, "items"); err != nil {
		return err
	}
	if s.AdditionalProperties, err = promoteYAMLSchemaOrBool(s.AdditionalProperties, node, "additionalProperties"); err != nil {
		return err
	}
	if s.AdditionalItems, err = promoteYAMLSchemaOrBool(s.AdditionalItems, node, "additionalItems"); err != nil {
		return err
	}
	if s.UnevaluatedItems, err = promoteYAMLSchemaOrBool(s.UnevaluatedItems, node, "unevaluatedItems"); err != nil {
		return err
	}
	if s.UnevaluatedProperties, err = promoteYAMLSchemaOrBool(s.UnevaluatedProperties, node, "unevaluatedProperties"); err != nil {
		return err
	}
	return nil
}

// promoteYAMLSchemaOrBool converts the generic value the YAML decoder leaves in
// an any-typed schema-or-bool field into a *Schema, or a []*Schema for the OAS
// 2.0 tuple form. Bools and absent values pass through untouched.
//
// parent is the mapping the field was decoded from and key its spec name, so the
// field's own node can be decoded rather than the generic map. That routes the
// nested schema back through this method, giving it the same decoding — and the
// same errors — as a top-level one, and keeps the inline Extra map that the
// map-based decoder narrows to x-* keys.
func promoteYAMLSchemaOrBool(v any, parent *yaml.Node, key string) (any, error) {
	switch v.(type) {
	case map[string]any, []any:
		// Generic representation left behind by the decoder — needs promoting.
	default:
		// bool, nil, or already a *Schema: nothing to do. Checked before
		// scanning parent so the common absent-field case stays free.
		return v, nil
	}

	node := unwrapSchemaNode(childValueNode(parent, key))

	switch {
	case node == nil:
		// The value arrived through a merge key (`<<`): the decoder resolved the
		// merge, but the merged pairs live in the anchored node, so there is
		// nothing under parent to decode. Promoting the map the decoder produced
		// costs only the non-x- unknown keys of Extra.
		return decodeSchemaOrBool(v), nil

	case node.Kind == yaml.MappingNode:
		schema := new(Schema)
		if err := node.Decode(schema); err != nil {
			return nil, err
		}
		return schema, nil

	case node.Kind == yaml.SequenceNode:
		// OAS 2.0 tuple validation: items may be a sequence of schemas.
		var schemas []*Schema
		if err := node.Decode(&schemas); err != nil {
			return nil, err
		}
		return schemas, nil

	default:
		return v, nil
	}
}

// childValueNode returns the value node stored under key in a mapping node, or
// nil when the mapping has no such key.
func childValueNode(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// unwrapSchemaNode strips the wrappers that stand between a node and the
// mapping it represents: a DocumentNode when the Schema is the root of the
// decoded document — yaml.Unmarshal hands an Unmarshaler the document node
// rather than its content — and an AliasNode when the schema was written as
// `*anchor`. Returns nil when no node is left to unwrap.
// boolSchemaNode reports whether a node is the bare-boolean schema form, and
// its value. Anchors are followed first so `schema: *alwaysValid` is classified
// by what it points at rather than by the alias node.
//
// Only a genuine `!!bool` counts. A quoted "true" is a string scalar and is not
// a boolean schema, so tag is checked rather than the raw value.
func boolSchemaNode(node *yaml.Node) (bool, bool) {
	node = unwrapSchemaNode(node)
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!bool" {
		return false, false
	}
	var b bool
	if err := node.Decode(&b); err != nil {
		return false, false
	}
	return b, true
}

func unwrapSchemaNode(node *yaml.Node) *yaml.Node {
	for node != nil {
		switch node.Kind {
		case yaml.DocumentNode:
			if len(node.Content) != 1 {
				return node
			}
			node = node.Content[0]
		case yaml.AliasNode:
			node = node.Alias
		default:
			return node
		}
	}
	return nil
}

// UnmarshalYAML implements custom YAML unmarshaling for Discriminator.
//
// Both dialects are accepted: the OAS 2.0 string-only form (`discriminator:
// petType`) decodes into PropertyName with StringForm set to true, and the
// OAS 3.0+ mapping form decodes normally. Rejecting the form that is wrong
// for the document's version is the validator's job, since the parser cannot
// see the version from here.
func (d *Discriminator) UnmarshalYAML(node *yaml.Node) error {
	// Follow anchors so `discriminator: *petTypeAnchor` is classified by the
	// kind of the node it points at rather than always failing the scalar test.
	for node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}

	switch node.Kind {
	case yaml.ScalarNode:
		// OAS 2.0: a scalar naming the property.
		var name string
		if err := node.Decode(&name); err != nil {
			return err
		}
		d.PropertyName = name
		d.StringForm = true
		return nil

	case yaml.MappingNode:
		// OAS 3.0+: the alias type sheds this method so default struct
		// decoding (including the inline Extra map) applies.
		type Alias Discriminator
		var alias Alias
		if err := node.Decode(&alias); err != nil {
			return err
		}
		*d = Discriminator(alias)
		return nil

	default:
		// Naming the offending kind is better than yaml's default message,
		// which would report the unexported Alias type rather than Discriminator.
		// yaml prefixes its own "line N:", so this message must not add one.
		return fmt.Errorf("'discriminator' must be a string (OAS 2.0) "+
			"or a mapping (OAS 3.0+), but got: %s", yamlKindName(node.Kind))
	}
}

// yamlKindName renders a yaml.Kind as its constant name for error messages.
// yaml.Kind is a bare integer constant with no String method, so formatting it
// directly yields an unhelpful number.
func yamlKindName(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "DocumentNode"
	case yaml.SequenceNode:
		return "SequenceNode"
	case yaml.MappingNode:
		return "MappingNode"
	case yaml.ScalarNode:
		return "ScalarNode"
	case yaml.AliasNode:
		return "AliasNode"
	default:
		return fmt.Sprintf("unknown yaml.Kind(%d)", k)
	}
}

// MarshalYAML implements custom YAML marshaling for Schema.
//
// When BoolForm is set the bare-boolean form is emitted, so `MySchema: true`
// is not silently rewritten into the empty object `MySchema: {}` — which is a
// different schema, and one that constrains nothing rather than the `false`
// case that permits nothing. Every other field is dropped, since a boolean
// schema has no keywords.
func (s *Schema) MarshalYAML() (any, error) {
	if b, ok := s.IsBool(); ok {
		return b, nil
	}
	type Alias Schema
	return (*Alias)(s), nil
}

// MarshalYAML implements custom YAML marshaling for Discriminator.
//
// When StringForm is set the OAS 2.0 bare-string form is emitted, so a 2.0
// document is not silently rewritten into the OAS 3.x object form. Mapping and
// Extra have no representation in that form and are dropped.
func (d *Discriminator) MarshalYAML() (any, error) {
	if d.StringForm {
		return d.PropertyName, nil
	}
	type Alias Discriminator
	return (*Alias)(d), nil
}
