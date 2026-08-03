// yaml_nodes.go holds helpers for reading a decoded YAML tree directly, used by
// the types whose shape the struct tags cannot express on their own:
//
//   - a schema-or-bool field, which has to be promoted after decoding
//   - a `callbacks` entry, which has to be classified before decoding
//
// Both need the node the decoder saw rather than the value it produced, which is
// what these return.

package parser

import "go.yaml.in/yaml/v4"

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

// unwrapYAMLNode strips the wrappers that stand between a node and the mapping
// or scalar it represents: a DocumentNode when the value is the root of the
// decoded document (yaml.Unmarshal hands an Unmarshaler the document node rather
// than its content), and an AliasNode when the value was written as `*anchor`.
// Returns nil when no node is left to unwrap.
func unwrapYAMLNode(node *yaml.Node) *yaml.Node {
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
