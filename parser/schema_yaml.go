package parser

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

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
