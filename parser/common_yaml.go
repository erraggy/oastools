package parser

import "go.yaml.in/yaml/v4"

// UnmarshalYAML implements custom YAML unmarshaling for ServerVariable.
//
// The alias type sheds this method so default struct decoding (including the
// inline Extra map) applies. The node is then inspected to record whether the
// default key was explicitly present, allowing the validator to distinguish
// an absent default from a default of "" (which is a valid default in OAS 3.x).
func (sv *ServerVariable) UnmarshalYAML(node *yaml.Node) error {
	type Alias ServerVariable
	var alias Alias
	if err := node.Decode(&alias); err != nil {
		return err
	}
	*sv = ServerVariable(alias)

	sv.DefaultSet = hasYAMLKey(node, "default")
	return nil
}

// hasYAMLKey reports whether key is present in node's mapping, following anchors
// and merge keys.
func hasYAMLKey(node *yaml.Node, key string) bool {
	mapping := unwrapYAMLNode(node)
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		k := mapping.Content[i]
		if k.Value == key {
			return true
		}
		if k.Value == "<<" {
			target := unwrapYAMLNode(mapping.Content[i+1])
			if target != nil {
				if target.Kind == yaml.MappingNode && hasYAMLKey(target, key) {
					return true
				} else if target.Kind == yaml.SequenceNode {
					for _, elem := range target.Content {
						if hasYAMLKey(elem, key) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
