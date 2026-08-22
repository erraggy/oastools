package schemautil

import (
	"fmt"
	"hash"
	"hash/fnv"
	"reflect"
	"sort"
	"strconv"

	"github.com/erraggy/oastools/internal/maputil"
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

	// The bare-boolean form has no keywords, so it hashes on its value alone.
	// `true` and `false` are opposite schemas and must not share a bucket with
	// each other or with an object schema, or deduplication groups them and the
	// deep comparison never gets to reject the merge.
	if b, ok := schema.IsBool(); ok {
		h.writeBoolSchema(hasher, b)
		return
	}

	// Hash $ref if present. JSON Schema 2020-12 allows keywords alongside $ref, so
	// this records the reference and keeps going: returning here made
	// {$ref: X, default: 1} and {$ref: X, default: 2} hash alike.
	if schema.Ref != "" {
		h.writeLabeled(hasher, "$ref", schema.Ref)
	}

	// Type (handle both string and []any for OAS 3.1+)
	h.hashType(hasher, schema.Type)

	// Format
	h.writeString(hasher, "format:")
	h.writeString(hasher, schema.Format)

	// Pattern
	h.writeString(hasher, "pattern:")
	h.writeString(hasher, schema.Pattern)

	h.hashEnumConstRequired(hasher, schema)

	// Properties (sorted by key for deterministic hashing)
	if len(schema.Properties) > 0 {
		h.writeString(hasher, "properties:")
		keys := maputil.SortedKeys(schema.Properties)
		for _, k := range keys {
			h.writeString(hasher, k)
			h.hashSchema(hasher, schema.Properties[k])
		}
	}

	// PatternProperties (sorted by key)
	if len(schema.PatternProperties) > 0 {
		h.writeString(hasher, "patternProperties:")
		keys := maputil.SortedKeys(schema.PatternProperties)
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
	// Every string here is length-framed. Unframed, adjacent values run together:
	// mapping {"ab": "c"} hashed the same as {"a": "bc"}, and a mapping value could
	// forge the defaultMapping label that follows it.
	if schema.Discriminator != nil {
		h.writeString(hasher, "discriminator:")
		h.writeLabeled(hasher, "propertyName", schema.Discriminator.PropertyName)
		if len(schema.Discriminator.Mapping) > 0 {
			keys := maputil.SortedKeys(schema.Discriminator.Mapping)
			for _, k := range keys {
				h.writeLabeled(hasher, "map", k)
				h.writeLabeled(hasher, "to", schema.Discriminator.Mapping[k])
			}
		}
		// OAS 3.2+. Written only when set, so a schema without it hashes as it
		// always has, while two schemas with different fallbacks stop colliding.
		// https://spec.openapis.org/oas/v3.2.0.html#discriminator-default-mapping
		if schema.Discriminator.DefaultMapping != "" {
			h.writeLabeled(hasher, "defaultMapping", schema.Discriminator.DefaultMapping)
		}
	}

	// XML is structural, not descriptive: it decides the element name, namespace,
	// and node kind a value serializes to, unlike the title and description this
	// type's doc comment excludes.
	// https://spec.openapis.org/oas/v3.2.0.html#xml-object
	//
	// Omitting it put schemas that serialize to different XML in one bucket, where
	// deduplication merged them. Fields are labeled so no value can be mistaken for
	// its neighbor's; a schema with no `xml` hashes as it always has.
	if schema.XML != nil {
		h.writeString(hasher, "xml:")
		// Length-framed via writeLabeled: writeString appends raw bytes, so an
		// unframed name "anamespace:b" would hash the same as name "a" with
		// namespace "b".
		if schema.XML.Name != "" {
			h.writeLabeled(hasher, "name", schema.XML.Name)
		}
		if schema.XML.Namespace != "" {
			h.writeLabeled(hasher, "namespace", schema.XML.Namespace)
		}
		if schema.XML.Prefix != "" {
			h.writeLabeled(hasher, "prefix", schema.XML.Prefix)
		}
		if schema.XML.Attribute {
			h.writeString(hasher, "attribute:true")
		}
		if schema.XML.Wrapped {
			h.writeString(hasher, "wrapped:true")
		}
		// Supersedes attribute and wrapped, so it must separate schemas at least as
		// strongly as they do.
		if schema.XML.NodeType != "" {
			h.writeLabeled(hasher, "nodeType", schema.XML.NodeType)
		}
	}

	// JSON Schema identity and dialect, value semantics, and the content keywords
	h.hashIdentity(hasher, schema)
	h.hashValueSemantics(hasher, schema)
	h.hashContentKeywords(hasher, schema)

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
		keys := maputil.SortedKeys(schema.DependentRequired)
		for _, k := range keys {
			h.writeLabeled(hasher, "k", k)
			deps := make([]string, len(schema.DependentRequired[k]))
			copy(deps, schema.DependentRequired[k])
			sort.Strings(deps)
			for _, d := range deps {
				h.writeLabeled(hasher, "d", d)
			}
		}
	}

	// DependentSchemas
	if len(schema.DependentSchemas) > 0 {
		h.writeString(hasher, "dependentSchemas:")
		keys := maputil.SortedKeys(schema.DependentSchemas)
		for _, k := range keys {
			h.writeString(hasher, k)
			h.hashSchema(hasher, schema.DependentSchemas[k])
		}
	}

	// Defs
	if len(schema.Defs) > 0 {
		h.writeString(hasher, "$defs:")
		keys := maputil.SortedKeys(schema.Defs)
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
	case []*parser.Schema:
		// The tuple form, which OAS 2.0 allows. The index is hashed too, so
		// [A, B] and [B, A] do not collide.
		h.writeString(hasher, "tuple:")
		for i, s := range val {
			h.writeString(hasher, strconv.Itoa(i)+":")
			h.hashSchema(hasher, s)
		}
	case bool:
		// Same encoding hashSchema uses for Schema.BoolForm. A boolean schema
		// has two representations — a raw bool here, a *Schema with BoolForm
		// set in a *Schema-typed position — and they mean the same thing, so
		// they must hash the same. Writing a bare "true" here put `items: true`
		// and `items: NewBoolSchema(true)` in different buckets.
		h.writeBoolSchema(hasher, val)
	}
}

// writeBoolSchema writes the bare-boolean schema form. Shared by hashSchema and
// hashSchemaOrBool so the two representations cannot drift apart.
func (h *SchemaHasher) writeBoolSchema(hasher hash.Hash64, v bool) {
	h.writeString(hasher, "boolschema:")
	h.writeString(hasher, strconv.FormatBool(v))
}

// hashIdentity hashes the JSON Schema identity and dialect keywords. They decide
// which schema a $ref or $dynamicRef resolves to and which vocabulary validates
// it, so two schemas differing here are not interchangeable however alike their
// constraints look.
func (h *SchemaHasher) hashIdentity(hasher hash.Hash64, schema *parser.Schema) {
	if schema.Schema != "" {
		h.writeLabeled(hasher, "$schema", schema.Schema)
	}
	if schema.ID != "" {
		h.writeLabeled(hasher, "$id", schema.ID)
	}
	if schema.Anchor != "" {
		h.writeLabeled(hasher, "$anchor", schema.Anchor)
	}
	if schema.DynamicRef != "" {
		h.writeLabeled(hasher, "$dynamicRef", schema.DynamicRef)
	}
	if schema.DynamicAnchor != "" {
		h.writeLabeled(hasher, "$dynamicAnchor", schema.DynamicAnchor)
	}
	if len(schema.Vocabulary) == 0 {
		return
	}
	h.writeString(hasher, "$vocabulary:")
	keys := maputil.SortedKeys(schema.Vocabulary)
	for _, k := range keys {
		h.writeLabeled(hasher, "k", k)
		h.writeLabeled(hasher, "v", strconv.FormatBool(schema.Vocabulary[k]))
	}
}

// hashValueSemantics hashes the keywords that decide what a value is when the
// payload does not say: the default, and the OAS 2.0 array serialization.
//
// default is an annotation in JSON Schema terms, but two schemas that default
// differently generate different code and fill payloads differently, so
// consolidating them is not safe. collectionFormat decides a wire format outright.
// hashEnumConstRequired hashes the three value-set keywords. Extracted from
// hashSchema purely to keep that function under the complexity limit; the write
// order is unchanged, so hashes computed before and after the extraction match.
func (h *SchemaHasher) hashEnumConstRequired(hasher hash.Hash64, schema *parser.Schema) {
	// Enum (order matters). Length-framed: unframed, ["ab"] and ["a", "b"] both
	// hash as "enum:ab".
	if len(schema.Enum) > 0 {
		h.writeString(hasher, "enum:")
		for _, v := range schema.Enum {
			h.writeLabeled(hasher, "v", fmt.Sprintf("%v", v))
		}
	}

	// Const
	if schema.Const != nil {
		h.writeString(hasher, "const:")
		h.writeString(hasher, fmt.Sprintf("%v", schema.Const))
	}

	// Required (sort for order-independent comparison). Length-framed for the same
	// reason as Enum.
	if len(schema.Required) > 0 {
		h.writeString(hasher, "required:")
		sorted := make([]string, len(schema.Required))
		copy(sorted, schema.Required)
		sort.Strings(sorted)
		for _, r := range sorted {
			h.writeLabeled(hasher, "r", r)
		}
	}
}

func (h *SchemaHasher) hashValueSemantics(hasher hash.Hash64, schema *parser.Schema) {
	if schema.Default != nil {
		h.writeLabeled(hasher, "default", fmt.Sprintf("%v", schema.Default))
	}
	if schema.CollectionFormat != "" {
		h.writeLabeled(hasher, "collectionFormat", schema.CollectionFormat)
	}
}

// hashContentKeywords hashes the JSON Schema 2020-12 unevaluated and content
// keywords, all of which participate in validation.
func (h *SchemaHasher) hashContentKeywords(hasher hash.Hash64, schema *parser.Schema) {
	if schema.UnevaluatedProperties != nil {
		h.writeString(hasher, "unevaluatedProperties:")
		h.hashSchemaOrBool(hasher, schema.UnevaluatedProperties)
	}
	if schema.UnevaluatedItems != nil {
		h.writeString(hasher, "unevaluatedItems:")
		h.hashSchemaOrBool(hasher, schema.UnevaluatedItems)
	}
	if schema.ContentEncoding != "" {
		h.writeLabeled(hasher, "contentEncoding", schema.ContentEncoding)
	}
	if schema.ContentMediaType != "" {
		h.writeLabeled(hasher, "contentMediaType", schema.ContentMediaType)
	}
	if schema.ContentSchema != nil {
		h.writeString(hasher, "contentSchema:")
		h.hashSchema(hasher, schema.ContentSchema)
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

// writeLabeled writes a labeled value framed by its byte length, so no value can
// be mistaken for its neighbor whatever it contains.
//
// A delimiter cannot do this: any sentinel byte can also appear inside a value,
// so a NUL terminator is defeated by a name holding a NUL. The length pins where
// the value ends, which makes the encoding injective.
func (h *SchemaHasher) writeLabeled(hasher hash.Hash64, label, value string) {
	h.writeString(hasher, label)
	h.writeString(hasher, ":")
	h.writeString(hasher, strconv.Itoa(len(value)))
	h.writeString(hasher, ":")
	h.writeString(hasher, value)
}
