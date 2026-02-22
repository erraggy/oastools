package schemautil

import (
	"fmt"
	"hash"
	"hash/fnv"
	"reflect"
	"sort"
	"strconv"

	"github.com/erraggy/oastools/parser"
)

// SchemaHasher computes structural hashes for schemas.
// Structural hashes ignore metadata fields (title, description, example, deprecated)
// and focus on fields that affect the schema's semantic meaning.
type SchemaHasher struct {
	visited map[uintptr]bool
}

// NewSchemaHasher creates a new SchemaHasher.
func NewSchemaHasher() *SchemaHasher {
	return &SchemaHasher{
		visited: make(map[uintptr]bool),
	}
}

// Hash computes a structural hash for a schema.
// Schemas with identical structural properties will have the same hash.
// Note: Hash collisions are possible; use deep comparison to verify equivalence.
func (h *SchemaHasher) Hash(schema *parser.Schema) uint64 {
	clear(h.visited) // Reset visited map without reallocating
	hasher := fnv.New64a()
	h.hashSchema(hasher, schema)
	return hasher.Sum64()
}

// GroupByHash groups schemas by their structural hash.
// Returns a map from hash value to list of schema names with that hash.
func (h *SchemaHasher) GroupByHash(schemas map[string]*parser.Schema) map[uint64][]string {
	groups := make(map[uint64][]string)
	for name, schema := range schemas {
		hashVal := h.Hash(schema)
		groups[hashVal] = append(groups[hashVal], name)
	}
	return groups
}

// hashSchema recursively hashes a schema's structural properties.
func (h *SchemaHasher) hashSchema(hasher hash.Hash64, schema *parser.Schema) {
	if schema == nil {
		h.writeString(hasher, "nil")
		return
	}

	// Check for circular reference
	ptr := reflect.ValueOf(schema).Pointer()
	if h.visited[ptr] {
		h.writeString(hasher, "circular")
		return
	}
	h.visited[ptr] = true
	defer func() { h.visited[ptr] = false }()

	// Hash $ref if present (schema is just a reference)
	if schema.Ref != "" {
		h.writeString(hasher, "$ref:")
		h.writeString(hasher, schema.Ref)
		return
	}

	// Type (handle both string and []any for OAS 3.1+)
	h.hashType(hasher, schema.Type)

	// Format
	h.writeString(hasher, "format:")
	h.writeString(hasher, schema.Format)

	// Pattern
	h.writeString(hasher, "pattern:")
	h.writeString(hasher, schema.Pattern)

	// Enum (order matters)
	if len(schema.Enum) > 0 {
		h.writeString(hasher, "enum:")
		for _, v := range schema.Enum {
			h.writeString(hasher, fmt.Sprintf("%v", v))
		}
	}

	// Const
	if schema.Const != nil {
		h.writeString(hasher, "const:")
		h.writeString(hasher, fmt.Sprintf("%v", schema.Const))
	}

	// Required (sort for order-independent comparison)
	if len(schema.Required) > 0 {
		h.writeString(hasher, "required:")
		sorted := make([]string, len(schema.Required))
		copy(sorted, schema.Required)
		sort.Strings(sorted)
		for _, r := range sorted {
			h.writeString(hasher, r)
		}
	}

	// Properties (sorted by key for deterministic hashing)
	if len(schema.Properties) > 0 {
		h.writeString(hasher, "properties:")
		keys := make([]string, 0, len(schema.Properties))
		for k := range schema.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h.writeString(hasher, k)
			h.hashSchema(hasher, schema.Properties[k])
		}
	}

	// PatternProperties (sorted by key)
	if len(schema.PatternProperties) > 0 {
		h.writeString(hasher, "patternProperties:")
		keys := make([]string, 0, len(schema.PatternProperties))
		for k := range schema.PatternProperties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h.writeString(hasher, k)
			h.hashSchema(hasher, schema.PatternProperties[k])
		}
	}

	// AdditionalProperties (can be *Schema or bool)
	if schema.AdditionalProperties != nil {
		h.writeString(hasher, "additionalProperties:")
		h.hashSchemaOrBool(hasher, schema.AdditionalProperties)
	}

	// Items (can be *Schema or bool in OAS 3.1+)
	if schema.Items != nil {
		h.writeString(hasher, "items:")
		h.hashSchemaOrBool(hasher, schema.Items)
	}

	// PrefixItems
	if len(schema.PrefixItems) > 0 {
		h.writeString(hasher, "prefixItems:")
		for _, item := range schema.PrefixItems {
			h.hashSchema(hasher, item)
		}
	}

	// AdditionalItems
	if schema.AdditionalItems != nil {
		h.writeString(hasher, "additionalItems:")
		h.hashSchemaOrBool(hasher, schema.AdditionalItems)
	}

	// Numeric constraints
	h.hashNumericConstraints(hasher, schema)

	// String constraints
	h.hashStringConstraints(hasher, schema)

	// Array constraints
	h.hashArrayConstraints(hasher, schema)

	// Object constraints
	h.hashObjectConstraints(hasher, schema)

	// Composition (allOf, anyOf, oneOf, not)
	h.hashComposition(hasher, schema)

	// Conditionals (if/then/else)
	if schema.If != nil {
		h.writeString(hasher, "if:")
		h.hashSchema(hasher, schema.If)
	}
	if schema.Then != nil {
		h.writeString(hasher, "then:")
		h.hashSchema(hasher, schema.Then)
	}
	if schema.Else != nil {
		h.writeString(hasher, "else:")
		h.hashSchema(hasher, schema.Else)
	}

	// Nullable (OAS 3.0)
	if schema.Nullable {
		h.writeString(hasher, "nullable:true")
	}

	// ReadOnly/WriteOnly
	if schema.ReadOnly {
		h.writeString(hasher, "readOnly:true")
	}
	if schema.WriteOnly {
		h.writeString(hasher, "writeOnly:true")
	}

	// Discriminator
	if schema.Discriminator != nil {
		h.writeString(hasher, "discriminator:")
		h.writeString(hasher, schema.Discriminator.PropertyName)
		if len(schema.Discriminator.Mapping) > 0 {
			keys := make([]string, 0, len(schema.Discriminator.Mapping))
			for k := range schema.Discriminator.Mapping {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				h.writeString(hasher, k)
				h.writeString(hasher, schema.Discriminator.Mapping[k])
			}
		}
	}

	// Contains
	if schema.Contains != nil {
		h.writeString(hasher, "contains:")
		h.hashSchema(hasher, schema.Contains)
	}

	// PropertyNames
	if schema.PropertyNames != nil {
		h.writeString(hasher, "propertyNames:")
		h.hashSchema(hasher, schema.PropertyNames)
	}

	// DependentRequired
	if len(schema.DependentRequired) > 0 {
		h.writeString(hasher, "dependentRequired:")
		keys := make([]string, 0, len(schema.DependentRequired))
		for k := range schema.DependentRequired {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h.writeString(hasher, k)
			deps := make([]string, len(schema.DependentRequired[k]))
			copy(deps, schema.DependentRequired[k])
			sort.Strings(deps)
			for _, d := range deps {
				h.writeString(hasher, d)
			}
		}
	}

	// DependentSchemas
	if len(schema.DependentSchemas) > 0 {
		h.writeString(hasher, "dependentSchemas:")
		keys := make([]string, 0, len(schema.DependentSchemas))
		for k := range schema.DependentSchemas {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h.writeString(hasher, k)
			h.hashSchema(hasher, schema.DependentSchemas[k])
		}
	}

	// Defs
	if len(schema.Defs) > 0 {
		h.writeString(hasher, "$defs:")
		keys := make([]string, 0, len(schema.Defs))
		for k := range schema.Defs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h.writeString(hasher, k)
			h.hashSchema(hasher, schema.Defs[k])
		}
	}
}

// hashType handles both string and []any type values.
func (h *SchemaHasher) hashType(hasher hash.Hash64, t any) {
	h.writeString(hasher, "type:")
	switch v := t.(type) {
	case string:
		h.writeString(hasher, v)
	case []any:
		// Sort for consistent hashing
		types := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				types = append(types, s)
			}
		}
		sort.Strings(types)
		for _, s := range types {
			h.writeString(hasher, s)
		}
	case []string:
		// Sort for consistent hashing
		sorted := make([]string, len(v))
		copy(sorted, v)
		sort.Strings(sorted)
		for _, s := range sorted {
			h.writeString(hasher, s)
		}
	}
}

// hashSchemaOrBool handles fields that can be *Schema or bool.
func (h *SchemaHasher) hashSchemaOrBool(hasher hash.Hash64, v any) {
	switch val := v.(type) {
	case *parser.Schema:
		h.hashSchema(hasher, val)
	case bool:
		if val {
			h.writeString(hasher, "true")
		} else {
			h.writeString(hasher, "false")
		}
	}
}

// hashNumericConstraints hashes numeric validation fields.
func (h *SchemaHasher) hashNumericConstraints(hasher hash.Hash64, schema *parser.Schema) {
	if schema.Minimum != nil {
		h.writeString(hasher, "minimum:"+strconv.FormatFloat(*schema.Minimum, 'g', -1, 64))
	}
	if schema.Maximum != nil {
		h.writeString(hasher, "maximum:"+strconv.FormatFloat(*schema.Maximum, 'g', -1, 64))
	}
	if schema.ExclusiveMinimum != nil {
		h.writeString(hasher, fmt.Sprintf("exclusiveMinimum:%v", schema.ExclusiveMinimum))
	}
	if schema.ExclusiveMaximum != nil {
		h.writeString(hasher, fmt.Sprintf("exclusiveMaximum:%v", schema.ExclusiveMaximum))
	}
	if schema.MultipleOf != nil {
		h.writeString(hasher, "multipleOf:"+strconv.FormatFloat(*schema.MultipleOf, 'g', -1, 64))
	}
}

// hashStringConstraints hashes string validation fields.
func (h *SchemaHasher) hashStringConstraints(hasher hash.Hash64, schema *parser.Schema) {
	if schema.MinLength != nil {
		h.writeString(hasher, "minLength:"+strconv.Itoa(*schema.MinLength))
	}
	if schema.MaxLength != nil {
		h.writeString(hasher, "maxLength:"+strconv.Itoa(*schema.MaxLength))
	}
}

// hashArrayConstraints hashes array validation fields.
func (h *SchemaHasher) hashArrayConstraints(hasher hash.Hash64, schema *parser.Schema) {
	if schema.MinItems != nil {
		h.writeString(hasher, "minItems:"+strconv.Itoa(*schema.MinItems))
	}
	if schema.MaxItems != nil {
		h.writeString(hasher, "maxItems:"+strconv.Itoa(*schema.MaxItems))
	}
	if schema.UniqueItems {
		h.writeString(hasher, "uniqueItems:true")
	}
	if schema.MinContains != nil {
		h.writeString(hasher, "minContains:"+strconv.Itoa(*schema.MinContains))
	}
	if schema.MaxContains != nil {
		h.writeString(hasher, "maxContains:"+strconv.Itoa(*schema.MaxContains))
	}
}

// hashObjectConstraints hashes object validation fields.
func (h *SchemaHasher) hashObjectConstraints(hasher hash.Hash64, schema *parser.Schema) {
	if schema.MinProperties != nil {
		h.writeString(hasher, "minProperties:"+strconv.Itoa(*schema.MinProperties))
	}
	if schema.MaxProperties != nil {
		h.writeString(hasher, "maxProperties:"+strconv.Itoa(*schema.MaxProperties))
	}
}

// hashComposition hashes schema composition fields.
func (h *SchemaHasher) hashComposition(hasher hash.Hash64, schema *parser.Schema) {
	if len(schema.AllOf) > 0 {
		h.writeString(hasher, "allOf:")
		for _, s := range schema.AllOf {
			h.hashSchema(hasher, s)
		}
	}
	if len(schema.AnyOf) > 0 {
		h.writeString(hasher, "anyOf:")
		for _, s := range schema.AnyOf {
			h.hashSchema(hasher, s)
		}
	}
	if len(schema.OneOf) > 0 {
		h.writeString(hasher, "oneOf:")
		for _, s := range schema.OneOf {
			h.hashSchema(hasher, s)
		}
	}
	if schema.Not != nil {
		h.writeString(hasher, "not:")
		h.hashSchema(hasher, schema.Not)
	}
}

// writeString writes a string to the hash.
func (h *SchemaHasher) writeString(hasher hash.Hash64, s string) {
	_, _ = hasher.Write([]byte(s))
}
