// callbacks.go holds the Callback type and the splitting and merging that keeps
// the two Go fields carrying a `callbacks` object behaving as the one field the
// specification defines. An entry is either a Callback Object or a Reference
// Object; [Callback] explains why they cannot share a field.
//
// Each of the three decode paths splits the object for itself: YAML through the
// node tree, JSON through the raw message, and decodeFromMap through the generic
// map that $ref resolution produces. They agree on two things a test pins:
//
//   - a `$ref` key selects the reference form, whatever else sits beside it
//   - the Callbacks map is non-nil exactly when the document carried a
//     `callbacks` key, so presence survives decoding on every path
//
// https://spec.openapis.org/oas/v3.2.0.html#callback-object

package parser

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"go.yaml.in/yaml/v4"

	"github.com/erraggy/oastools/parser/internal/jsonhelpers"
)

// Callback is a map of runtime expressions to path items (OAS 3.0+).
//
// A `callbacks` entry may be written either as a Callback Object or as a
// Reference Object, and the specification tells them apart by the presence of a
// `$ref` key: https://spec.openapis.org/oas/v3.2.0.html#callback-object
//
// The two forms cannot share one Go field. A Callback Object is an open map
// keyed by user-authored runtime expressions, so a named map type is the only
// faithful model, and a named map type has no field to hold `$ref`. The
// reference form therefore lands in a parallel CallbackRefs map beside the
// Callbacks map on both [Operation] and [Components]; the two are merged back
// into a single `callbacks` object on serialization.
//
// A name belongs to one map or the other. Every decode path enforces that by
// construction, and marshaling a value that breaks it is an error rather than a
// silent choice between the two.
type Callback map[string]*PathItem

const yamlKeyCallbacks = "callbacks"

// UnmarshalYAML implements custom YAML unmarshaling for Operation.
//
// The reference-form entries are lifted out of the `callbacks` mapping before
// the alias type decodes the rest, because Callback is a map of path items and
// a `$ref` string is not one.
func (o *Operation) UnmarshalYAML(node *yaml.Node) error {
	type Alias Operation
	node, refs, err := splitYAMLCallbackRefs(node)
	if err != nil {
		return err
	}
	var alias Alias
	if err := node.Decode(&alias); err != nil {
		return err
	}
	*o = Operation(alias)
	o.CallbackRefs = refs
	return nil
}

// MarshalYAML implements custom YAML marshaling for Operation, merging
// CallbackRefs back into the single `callbacks` object.
func (o *Operation) MarshalYAML() (any, error) {
	type Alias Operation
	return marshalYAMLWithCallbackRefs((*Alias)(o), o.Callbacks, o.CallbackRefs)
}

// UnmarshalYAML implements custom YAML unmarshaling for Components, splitting
// the reference-form `callbacks` entries out the way [Operation.UnmarshalYAML]
// does.
func (c *Components) UnmarshalYAML(node *yaml.Node) error {
	type Alias Components
	node, refs, err := splitYAMLCallbackRefs(node)
	if err != nil {
		return err
	}
	var alias Alias
	if err := node.Decode(&alias); err != nil {
		return err
	}
	*c = Components(alias)
	c.CallbackRefs = refs
	return nil
}

// MarshalYAML implements custom YAML marshaling for Components, merging
// CallbackRefs back into the single `callbacks` object.
func (c *Components) MarshalYAML() (any, error) {
	type Alias Components
	return marshalYAMLWithCallbackRefs((*Alias)(c), c.Callbacks, c.CallbackRefs)
}

// splitYAMLCallbackRefs returns the given mapping with the reference-form
// entries removed from its `callbacks` value, plus those entries decoded. The
// returned node is what the caller decodes into its struct.
//
// The node is returned unchanged, and no copy is made, whenever there is nothing
// to split: a non-mapping node, no `callbacks` key, or no reference among its
// entries. That is every document that does not use the form, so the common case
// costs one scan of the callbacks mapping and no allocation.
func splitYAMLCallbackRefs(node *yaml.Node) (*yaml.Node, map[string]*Reference, error) {
	mapping := unwrapYAMLNode(node)
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		// Leave the type error to the caller's decode, which names the field.
		return node, nil, nil
	}
	callbacks := unwrapYAMLNode(childValueNode(mapping, yamlKeyCallbacks))
	if callbacks == nil || callbacks.Kind != yaml.MappingNode {
		return node, nil, nil
	}

	var refs map[string]*Reference
	var kept []*yaml.Node
	for i := 0; i+1 < len(callbacks.Content); i += 2 {
		name, value := callbacks.Content[i], callbacks.Content[i+1]
		entry := unwrapYAMLNode(value)
		if entry == nil || entry.Kind != yaml.MappingNode || childValueNode(entry, jsonKeyRef) == nil {
			// Only worth collecting once something has been taken out; until
			// then the original content is still the answer.
			if refs != nil {
				kept = append(kept, name, value)
			}
			continue
		}
		ref := new(Reference)
		if err := entry.Decode(ref); err != nil {
			return nil, nil, fmt.Errorf("callback %q: %w", name.Value, err)
		}
		if refs == nil {
			refs = make(map[string]*Reference)
			kept = slices.Clone(callbacks.Content[:i])
		}
		refs[name.Value] = ref
	}
	if refs == nil {
		return node, nil, nil
	}

	// The caller decodes the node returned here, so the reference entries have to
	// be gone from it: handing one to Callback, a map of path items, is the
	// failure this split exists to prevent.
	//
	// The `callbacks` key itself stays, holding whatever was not a reference, so
	// a document that carried one decodes to a non-nil Callbacks map on this path
	// as it does on the other two.
	remaining := *callbacks
	remaining.Content = kept
	return replaceChildValue(mapping, yamlKeyCallbacks, &remaining), refs, nil
}

// replaceChildValue returns a copy of mapping with the value under key replaced.
// The copy is what leaves the decoder's own node untouched, since it may hand
// the same one to another Unmarshaler.
func replaceChildValue(mapping *yaml.Node, key string, replacement *yaml.Node) *yaml.Node {
	edited := *mapping
	edited.Content = slices.Clone(mapping.Content)
	for i := 0; i+1 < len(edited.Content); i += 2 {
		if edited.Content[i].Value == key {
			edited.Content[i+1] = replacement
			break
		}
	}
	return &edited
}

// marshalYAMLWithCallbackRefs encodes an alias of a callback-carrying type and
// folds the reference-form entries back into its `callbacks` mapping.
func marshalYAMLWithCallbackRefs[T any](alias *T, callbacks map[string]*Callback, refs map[string]*Reference) (any, error) {
	// Nothing to fold in, so the alias goes back untouched. This is the common
	// case, and the one whose output must not change.
	if len(refs) == 0 {
		return alias, nil
	}
	if err := checkCallbackNameConflicts(callbacks, refs); err != nil {
		return nil, err
	}

	var node yaml.Node
	if err := node.Encode(alias); err != nil {
		return nil, err
	}
	entries := childValueNode(&node, yamlKeyCallbacks)
	if entries == nil {
		// Every entry is a reference, so the alias emitted no `callbacks` key.
		entries = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: yamlKeyCallbacks},
			entries)
	}
	for _, name := range slices.Sorted(maps.Keys(refs)) {
		var value yaml.Node
		if err := value.Encode(refs[name]); err != nil {
			return nil, err
		}
		entries.Content = append(entries.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name},
			&value)
	}
	return &node, nil
}

// splitJSONCallbackRefs returns the given JSON object with the reference-form
// entries removed from its `callbacks` member, plus those entries decoded.
//
// Like the YAML split, the input is returned unchanged when there is nothing to
// split, so only a document that actually uses the form pays for the extra
// decode and re-encode.
func splitJSONCallbackRefs(data []byte) ([]byte, map[string]*Reference, error) {
	// A shape this cannot split is left alone rather than reported: the caller's
	// decode runs next and names the field the value belongs to, which is a
	// better error than anything available here.
	object, ok := jsonObjectMembers(data)
	if !ok {
		return data, nil, nil
	}
	raw, ok := object[yamlKeyCallbacks]
	if !ok {
		return data, nil, nil
	}
	entries, ok := jsonObjectMembers(raw)
	if !ok {
		return data, nil, nil
	}

	var refs map[string]*Reference
	for name, entry := range entries {
		if !jsonObjectHasKey(entry, jsonKeyRef) {
			continue
		}
		ref := new(Reference)
		if err := json.Unmarshal(entry, ref); err != nil {
			return nil, nil, fmt.Errorf("callback %q: %w", name, err)
		}
		if refs == nil {
			refs = make(map[string]*Reference)
		}
		refs[name] = ref
		delete(entries, name)
	}
	if refs == nil {
		return data, nil, nil
	}

	// The `callbacks` member stays even when nothing is left under it, so
	// Callbacks decodes non-nil here as it does on the YAML and map paths.
	remaining, err := json.Marshal(entries)
	if err != nil {
		return nil, nil, err
	}
	object[yamlKeyCallbacks] = remaining
	rest, err := json.Marshal(object)
	if err != nil {
		return nil, nil, err
	}
	return rest, refs, nil
}

// unmarshalJSONWithCallbackRefs decodes data into alias, returning the
// reference-form callbacks and the extensions for the caller to store.
//
// Extensions come from the original bytes rather than from the split remainder:
// the split rewrites the object only to lift out reference-form callbacks, but
// re-encoding a map loses nothing an x- key needs, so reading them from data
// keeps the two independent of each other.
func unmarshalJSONWithCallbackRefs[A any](data []byte, alias *A) (map[string]*Reference, map[string]any, error) {
	rest, refs, err := splitJSONCallbackRefs(data)
	if err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(rest, alias); err != nil {
		return nil, nil, err
	}
	return refs, jsonhelpers.ExtractExtensions(data), nil
}

// jsonObjectMembers decodes a JSON object into its raw members, reporting false
// when the value is not an object.
func jsonObjectMembers(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		return nil, false
	}
	return members, true
}

// jsonObjectHasKey reports whether raw is a JSON object carrying key.
func jsonObjectHasKey(raw json.RawMessage, key string) bool {
	members, ok := jsonObjectMembers(raw)
	if !ok {
		return false
	}
	_, ok = members[key]
	return ok
}

// mergedCallbacks renders the two Go fields as the single `callbacks` object the
// specification defines, or nil when both are empty.
func mergedCallbacks(callbacks map[string]*Callback, refs map[string]*Reference) (map[string]any, error) {
	if len(callbacks) == 0 && len(refs) == 0 {
		return nil, nil
	}
	if err := checkCallbackNameConflicts(callbacks, refs); err != nil {
		return nil, err
	}
	merged := make(map[string]any, len(callbacks)+len(refs))
	for name, callback := range callbacks {
		merged[name] = callback
	}
	for name, ref := range refs {
		merged[name] = ref
	}
	return merged, nil
}

// checkCallbackNameConflicts rejects a name held by both maps. Decoding cannot
// produce one, since each entry is classified once; a value assembled in Go can,
// and serializing it would have to pick a form, which is a choice no caller
// asked for.
func checkCallbackNameConflicts(callbacks map[string]*Callback, refs map[string]*Reference) error {
	for _, name := range slices.Sorted(maps.Keys(refs)) {
		if _, ok := callbacks[name]; ok {
			return fmt.Errorf("callback %q is present as both a Callback Object and a Reference Object: "+
				"a callbacks entry is one or the other", name)
		}
	}
	return nil
}
