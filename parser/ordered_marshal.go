package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"

	"go.yaml.in/yaml/v4"
)

// MarshalOrderedJSON marshals the parsed document to JSON with fields
// in the same order as the original source document.
//
// This method requires PreserveOrder to be enabled during parsing.
// If PreserveOrder was not enabled, it falls back to standard JSON marshaling
// which sorts map keys alphabetically.
//
// The ordered output is useful for:
//   - Hash-based caching where roundtrip identity matters
//   - Minimizing diffs when editing and re-serializing specs
//   - Maintaining human-friendly key ordering
//
// Example:
//
//	p := parser.New()
//	p.PreserveOrder = true
//	result, _ := p.Parse("api.yaml")
//	orderedJSON, _ := result.MarshalOrderedJSON()
func (pr *ParseResult) MarshalOrderedJSON() ([]byte, error) {
	if pr.sourceNode == nil {
		// Fall back to standard marshaling if order not preserved
		return json.Marshal(pr.Document)
	}

	var buf bytes.Buffer
	if err := marshalNodeAsJSON(&buf, pr.sourceNode, pr.Data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalOrderedJSONIndent marshals the parsed document to indented JSON
// with fields in the same order as the original source document.
//
// This method requires PreserveOrder to be enabled during parsing.
// If PreserveOrder was not enabled, it falls back to standard JSON marshaling.
func (pr *ParseResult) MarshalOrderedJSONIndent(prefix, indent string) ([]byte, error) {
	data, err := pr.MarshalOrderedJSON()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, data, prefix, indent); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalOrderedYAML marshals the parsed document to YAML with fields
// in the same order as the original source document.
//
// This method requires PreserveOrder to be enabled during parsing.
// If PreserveOrder was not enabled, it falls back to standard YAML marshaling
// which sorts map keys alphabetically.
//
// Example:
//
//	p := parser.New()
//	p.PreserveOrder = true
//	result, _ := p.Parse("api.yaml")
//	orderedYAML, _ := result.MarshalOrderedYAML()
func (pr *ParseResult) MarshalOrderedYAML() ([]byte, error) {
	if pr.sourceNode == nil {
		// Fall back to standard marshaling if order not preserved
		return yaml.Marshal(pr.Document)
	}

	// Build an ordered node from typed data using source order
	orderedNode, err := buildOrderedNode(pr.sourceNode, pr.Data)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(orderedNode)
}

// HasPreservedOrder returns true if this ParseResult has preserved
// the original field ordering from the source document.
// This is true when PreserveOrder was enabled during parsing.
func (pr *ParseResult) HasPreservedOrder() bool {
	return pr.sourceNode != nil
}

// marshalNodeAsJSON writes a yaml.Node to a buffer as JSON, using the
// typed data values but preserving the key order from the node.
func marshalNodeAsJSON(buf *bytes.Buffer, node *yaml.Node, data any) error {
	if node == nil {
		return writeJSON(buf, data)
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			return marshalNodeAsJSON(buf, node.Content[0], data)
		}
		return writeJSON(buf, data)

	case yaml.MappingNode:
		dataMap, ok := data.(map[string]any)
		if !ok {
			// Data doesn't match node structure, fall back to typed data
			return writeJSON(buf, data)
		}

		buf.WriteByte('{')

		// Extract key order from node and merge with data keys
		sourceKeys := extractKeyOrder(node)
		dataKeys := make([]string, 0, len(dataMap))
		for k := range dataMap {
			dataKeys = append(dataKeys, k)
		}
		keyOrder := mergeKeyOrder(sourceKeys, dataKeys)

		// Build index for O(1) child node lookup
		idx := buildNodeIndex(node)

		first := true
		for _, key := range keyOrder {
			val, exists := dataMap[key]
			if !exists {
				continue // Key was in source but removed from data
			}

			if !first {
				buf.WriteByte(',')
			}
			first = false

			// Write key
			keyJSON, err := json.Marshal(key)
			if err != nil {
				return err
			}
			buf.Write(keyJSON)
			buf.WriteByte(':')

			// Lookup child node using index (O(1) instead of O(n))
			childNode := idx[key]
			if err := marshalNodeAsJSON(buf, childNode, val); err != nil {
				return err
			}
		}

		buf.WriteByte('}')
		return nil

	case yaml.SequenceNode:
		dataSlice, ok := data.([]any)
		if !ok {
			return writeJSON(buf, data)
		}

		buf.WriteByte('[')
		for i, item := range dataSlice {
			if i > 0 {
				buf.WriteByte(',')
			}
			var childNode *yaml.Node
			if i < len(node.Content) {
				childNode = node.Content[i]
			}
			if err := marshalNodeAsJSON(buf, childNode, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil

	default:
		// Handles ScalarNode, AliasNode, and any unknown node kinds
		return writeJSON(buf, data)
	}
}

// buildOrderedNode creates a yaml.Node tree with content ordered according
// to sourceNode but with values from data.
func buildOrderedNode(sourceNode *yaml.Node, data any) (*yaml.Node, error) {
	if sourceNode == nil {
		return valueToNode(data)
	}

	switch sourceNode.Kind {
	case yaml.DocumentNode:
		if len(sourceNode.Content) > 0 {
			child, err := buildOrderedNode(sourceNode.Content[0], data)
			if err != nil {
				return nil, err
			}
			return &yaml.Node{
				Kind:    yaml.DocumentNode,
				Content: []*yaml.Node{child},
			}, nil
		}
		return valueToNode(data)

	case yaml.MappingNode:
		dataMap, ok := data.(map[string]any)
		if !ok {
			return valueToNode(data)
		}

		result := &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: make([]*yaml.Node, 0),
		}

		// Extract key order from source and merge with data keys
		sourceKeys := extractKeyOrder(sourceNode)
		dataKeys := make([]string, 0, len(dataMap))
		for k := range dataMap {
			dataKeys = append(dataKeys, k)
		}
		keyOrder := mergeKeyOrder(sourceKeys, dataKeys)

		// Build index for O(1) child node lookup
		idx := buildNodeIndex(sourceNode)

		for _, key := range keyOrder {
			val, exists := dataMap[key]
			if !exists {
				continue
			}

			valNode, err := buildOrderedNode(idx[key], val)
			if err != nil {
				return nil, err
			}
			result.Content = append(result.Content, scalarNode("!!str", key), valNode)
		}

		return result, nil

	case yaml.SequenceNode:
		dataSlice, ok := data.([]any)
		if !ok {
			return valueToNode(data)
		}

		result := &yaml.Node{
			Kind:    yaml.SequenceNode,
			Content: make([]*yaml.Node, 0, len(dataSlice)),
		}

		for i, item := range dataSlice {
			var childSourceNode *yaml.Node
			if i < len(sourceNode.Content) {
				childSourceNode = sourceNode.Content[i]
			}
			itemNode, err := buildOrderedNode(childSourceNode, item)
			if err != nil {
				return nil, err
			}
			result.Content = append(result.Content, itemNode)
		}

		return result, nil

	default:
		return valueToNode(data)
	}
}

// extractKeyOrder returns the keys from a MappingNode in their original order.
func extractKeyOrder(node *yaml.Node) []string {
	if node.Kind != yaml.MappingNode {
		return nil
	}

	keys := make([]string, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) && node.Content[i].Kind == yaml.ScalarNode {
			keys = append(keys, node.Content[i].Value)
		}
	}
	return keys
}

// nodeIndex provides O(1) lookup for child nodes in a MappingNode.
type nodeIndex map[string]*yaml.Node

// buildNodeIndex creates an index from key names to value nodes for O(1) lookup.
// This replaces the O(n) linear search in findChildNode for better performance
// when processing large OAS documents.
func buildNodeIndex(node *yaml.Node) nodeIndex {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	idx := make(nodeIndex, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) && node.Content[i].Kind == yaml.ScalarNode {
			idx[node.Content[i].Value] = node.Content[i+1]
		}
	}
	return idx
}

// mergeKeyOrder returns keys in source order, with any extra keys from data appended (sorted for determinism).
// This deduplicates the key ordering logic used by marshalNodeAsJSON and buildOrderedNode.
func mergeKeyOrder(sourceKeys, dataKeys []string) []string {
	seenKeys := make(map[string]bool, len(sourceKeys))
	for _, k := range sourceKeys {
		seenKeys[k] = true
	}

	var extraKeys []string
	for _, k := range dataKeys {
		if !seenKeys[k] {
			extraKeys = append(extraKeys, k)
		}
	}
	slices.Sort(extraKeys)

	return append(sourceKeys, extraKeys...)
}

// writeJSON marshals a value to JSON and writes it to the buffer.
func writeJSON(buf *bytes.Buffer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	buf.Write(data)
	return nil
}

// scalarNode creates a yaml.Node for a scalar value.
func scalarNode(tag, value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
}

// valueToNode converts a Go value to a yaml.Node.
func valueToNode(v any) (*yaml.Node, error) {
	if v == nil {
		return scalarNode("!!null", "null"), nil
	}

	switch val := v.(type) {
	case bool:
		return scalarNode("!!bool", strconv.FormatBool(val)), nil
	case int:
		return scalarNode("!!int", strconv.Itoa(val)), nil
	case int64:
		return scalarNode("!!int", strconv.FormatInt(val, 10)), nil
	case float64:
		return scalarNode("!!float", strconv.FormatFloat(val, 'f', -1, 64)), nil
	case string:
		return scalarNode("!!str", val), nil
	case []any:
		node := &yaml.Node{
			Kind:    yaml.SequenceNode,
			Content: make([]*yaml.Node, 0, len(val)),
		}
		for _, item := range val {
			child, err := valueToNode(item)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, child)
		}
		return node, nil
	case map[string]any:
		// Guard against integer overflow: len(val)*2 could overflow for very large maps
		mapLen := len(val)
		if mapLen > math.MaxInt/2 {
			return nil, fmt.Errorf("map size %d exceeds safe conversion limit", mapLen)
		}
		// Use intermediate variable so static analysis can track the overflow guard
		capacity := mapLen * 2
		node := &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: make([]*yaml.Node, 0, capacity),
		}
		// Sort keys for determinism when no source order
		keys := make([]string, 0, mapLen)
		for k := range val {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			valNode, err := valueToNode(val[k])
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, scalarNode("!!str", k), valNode)
		}
		return node, nil
	default:
		// For unknown types, marshal to JSON then parse
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %T to yaml.Node: %w", v, err)
		}
		var result any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		return valueToNode(result)
	}
}
