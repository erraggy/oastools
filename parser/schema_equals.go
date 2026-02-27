package parser

import (
	"maps"

	"github.com/erraggy/oastools/internal/equalutil"
)

// This file contains the Equals method for Schema equality comparison.
// The comparison is optimized for early exit by checking cheaper fields first.

// schemaPair represents a pair of schema pointers for cycle detection.
type schemaPair struct {
	a, b *Schema
}

// Equals compares two Schemas for structural equality.
// Returns true if both schemas have identical content.
// This method handles cyclic schema references safely.
func (s *Schema) Equals(other *Schema) bool {
	return s.equalsWithVisited(other, make(map[schemaPair]bool))
}

// equalsWithVisited compares two Schemas with cycle detection.
// The visited map tracks schema pairs currently being compared to detect cycles.
//
//nolint:cyclop // High complexity is inherent - Schema has many fields that must be compared
func (s *Schema) equalsWithVisited(other *Schema, visited map[schemaPair]bool) bool {
	if s == nil && other == nil {
		return true
	}
	if s == nil || other == nil {
		return false
	}

	// Cycle detection: if we're already comparing this pair, assume equal
	// to break infinite recursion. This is correct because if we've reached
	// this point, all fields compared so far have matched.
	pair := schemaPair{s, other}
	if visited[pair] {
		return true
	}
	visited[pair] = true

	// Group 1: Boolean fields (cheapest comparisons first)
	if s.ReadOnly != other.ReadOnly {
		return false
	}
	if s.WriteOnly != other.WriteOnly {
		return false
	}
	if s.Deprecated != other.Deprecated {
		return false
	}
	if s.Nullable != other.Nullable {
		return false
	}
	if s.UniqueItems != other.UniqueItems {
		return false
	}

	// Group 2: String fields
	if s.Ref != other.Ref {
		return false
	}
	if s.Schema != other.Schema {
		return false
	}
	if s.Title != other.Title {
		return false
	}
	if s.Description != other.Description {
		return false
	}
	if s.Pattern != other.Pattern {
		return false
	}
	if s.Format != other.Format {
		return false
	}
	if s.ContentEncoding != other.ContentEncoding {
		return false
	}
	if s.ContentMediaType != other.ContentMediaType {
		return false
	}
	if s.CollectionFormat != other.CollectionFormat {
		return false
	}
	if s.ID != other.ID {
		return false
	}
	if s.Anchor != other.Anchor {
		return false
	}
	if s.DynamicRef != other.DynamicRef {
		return false
	}
	if s.DynamicAnchor != other.DynamicAnchor {
		return false
	}
	if s.Comment != other.Comment {
		return false
	}

	// Group 3: Polymorphic type/bounds
	if !equalSchemaType(s.Type, other.Type) {
		return false
	}
	if !equalBoolOrNumber(s.ExclusiveMinimum, other.ExclusiveMinimum) {
		return false
	}
	if !equalBoolOrNumber(s.ExclusiveMaximum, other.ExclusiveMaximum) {
		return false
	}

	// Group 4: Pointer fields
	if !equalutil.EqualPtr(s.Maximum, other.Maximum) {
		return false
	}
	if !equalutil.EqualPtr(s.Minimum, other.Minimum) {
		return false
	}
	if !equalutil.EqualPtr(s.MultipleOf, other.MultipleOf) {
		return false
	}
	if !equalutil.EqualPtr(s.MaxLength, other.MaxLength) {
		return false
	}
	if !equalutil.EqualPtr(s.MinLength, other.MinLength) {
		return false
	}
	if !equalutil.EqualPtr(s.MaxItems, other.MaxItems) {
		return false
	}
	if !equalutil.EqualPtr(s.MinItems, other.MinItems) {
		return false
	}
	if !equalutil.EqualPtr(s.MaxProperties, other.MaxProperties) {
		return false
	}
	if !equalutil.EqualPtr(s.MinProperties, other.MinProperties) {
		return false
	}
	if !equalutil.EqualPtr(s.MaxContains, other.MaxContains) {
		return false
	}
	if !equalutil.EqualPtr(s.MinContains, other.MinContains) {
		return false
	}

	// Group 5: String slices
	if !equalStringSlice(s.Required, other.Required) {
		return false
	}

	// Group 6: Any slices
	if !equalAnySlice(s.Enum, other.Enum) {
		return false
	}
	if !equalAnySlice(s.Examples, other.Examples) {
		return false
	}

	// Group 7: Any fields
	if !equalJSONValue(s.Default, other.Default) {
		return false
	}
	if !equalJSONValue(s.Example, other.Example) {
		return false
	}
	if !equalJSONValue(s.Const, other.Const) {
		return false
	}

	// Group 8: Polymorphic schema fields (schema or bool) with cycle detection
	if !equalSchemaOrBoolWithVisited(s.Items, other.Items, visited) {
		return false
	}
	if !equalSchemaOrBoolWithVisited(s.AdditionalItems, other.AdditionalItems, visited) {
		return false
	}
	if !equalSchemaOrBoolWithVisited(s.AdditionalProperties, other.AdditionalProperties, visited) {
		return false
	}
	if !equalSchemaOrBoolWithVisited(s.UnevaluatedItems, other.UnevaluatedItems, visited) {
		return false
	}
	if !equalSchemaOrBoolWithVisited(s.UnevaluatedProperties, other.UnevaluatedProperties, visited) {
		return false
	}

	// Group 9: Nested *Schema pointers (recursive with cycle detection)
	if !s.Contains.equalsWithVisited(other.Contains, visited) {
		return false
	}
	if !s.Not.equalsWithVisited(other.Not, visited) {
		return false
	}
	if !s.If.equalsWithVisited(other.If, visited) {
		return false
	}
	if !s.Then.equalsWithVisited(other.Then, visited) {
		return false
	}
	if !s.Else.equalsWithVisited(other.Else, visited) {
		return false
	}
	if !s.PropertyNames.equalsWithVisited(other.PropertyNames, visited) {
		return false
	}
	if !s.ContentSchema.equalsWithVisited(other.ContentSchema, visited) {
		return false
	}

	// Group 10: Schema slices (with cycle detection)
	if !equalSchemaSliceWithVisited(s.AllOf, other.AllOf, visited) {
		return false
	}
	if !equalSchemaSliceWithVisited(s.OneOf, other.OneOf, visited) {
		return false
	}
	if !equalSchemaSliceWithVisited(s.AnyOf, other.AnyOf, visited) {
		return false
	}
	if !equalSchemaSliceWithVisited(s.PrefixItems, other.PrefixItems, visited) {
		return false
	}

	// Group 11: Schema maps (with cycle detection)
	if !equalSchemaMapWithVisited(s.Properties, other.Properties, visited) {
		return false
	}
	if !equalSchemaMapWithVisited(s.PatternProperties, other.PatternProperties, visited) {
		return false
	}
	if !equalSchemaMapWithVisited(s.DependentSchemas, other.DependentSchemas, visited) {
		return false
	}
	if !equalSchemaMapWithVisited(s.Defs, other.Defs, visited) {
		return false
	}

	// Group 12: Other maps
	if !equalMapStringStringSlice(s.DependentRequired, other.DependentRequired) {
		return false
	}
	if !equalMapStringBool(s.Vocabulary, other.Vocabulary) {
		return false
	}

	// Group 13: Struct pointers
	if !equalDiscriminator(s.Discriminator, other.Discriminator) {
		return false
	}
	if !equalXML(s.XML, other.XML) {
		return false
	}
	if !equalExternalDocs(s.ExternalDocs, other.ExternalDocs) {
		return false
	}

	// Group 14: Extensions
	if !equalMapStringAny(s.Extra, other.Extra) {
		return false
	}

	return true
}

// equalSchemaSliceWithVisited compares two []*Schema slices for equality with cycle detection.
// Order-sensitive comparison. Nil and empty slices are considered equal.
func equalSchemaSliceWithVisited(a, b []*Schema, visited map[schemaPair]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].equalsWithVisited(b[i], visited) {
			return false
		}
	}
	return true
}

// equalSchemaMap compares two map[string]*Schema maps for equality.
// Nil and empty maps are considered equal.
func equalSchemaMap(a, b map[string]*Schema) bool {
	return equalSchemaMapWithVisited(a, b, make(map[schemaPair]bool))
}

// equalSchemaMapWithVisited compares two map[string]*Schema maps for equality with cycle detection.
// Nil and empty maps are considered equal.
func equalSchemaMapWithVisited(a, b map[string]*Schema, visited map[schemaPair]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !va.equalsWithVisited(vb, visited) {
			return false
		}
	}
	return true
}

// equalDiscriminator compares two *Discriminator for equality.
func equalDiscriminator(a, b *Discriminator) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.PropertyName != b.PropertyName {
		return false
	}
	if !equalMapStringString(a.Mapping, b.Mapping) {
		return false
	}
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}
	return true
}

// equalMapStringString compares two map[string]string maps for equality.
// Nil and empty maps are considered equal.
func equalMapStringString(a, b map[string]string) bool {
	return maps.Equal(a, b)
}

// equalXML compares two *XML for equality.
func equalXML(a, b *XML) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.Namespace != b.Namespace {
		return false
	}
	if a.Prefix != b.Prefix {
		return false
	}
	if a.Attribute != b.Attribute {
		return false
	}
	if a.Wrapped != b.Wrapped {
		return false
	}
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}
	return true
}

// equalExternalDocs compares two *ExternalDocs for equality.
func equalExternalDocs(a, b *ExternalDocs) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.URL != b.URL {
		return false
	}
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}
	return true
}

// equalSchemaOrBoolWithVisited handles polymorphic fields (*Schema or bool) with cycle detection.
// Used for Items, AdditionalItems, AdditionalProperties, UnevaluatedItems, UnevaluatedProperties.
func equalSchemaOrBoolWithVisited(a, b any, visited map[schemaPair]bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch ta := a.(type) {
	case bool:
		tb, ok := b.(bool)
		if !ok {
			return false
		}
		return ta == tb
	case *Schema:
		tb, ok := b.(*Schema)
		if !ok {
			return false
		}
		return ta.equalsWithVisited(tb, visited)
	default:
		// Unknown type, fall back to reflect.DeepEqual
		return equalSchemaOrBool(a, b)
	}
}
