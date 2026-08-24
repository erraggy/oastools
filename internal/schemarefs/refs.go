// Package schemarefs walks the references a schema makes to component schemas,
// and answers which component names a document keeps distinct.
//
// It exists so joiner and builder read the same answer from the same walk. A
// second walk of its own would be a second list of subschema keywords to keep
// current, and one that missed a keyword would report a reference that is
// there as absent.
package schemarefs

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

// SchemaName returns the component schema a $ref names, or "" when the
// reference points elsewhere.
func SchemaName(ref string) string {
	if ref == "" {
		return ""
	}

	// OAS 3.x: #/components/schemas/Name
	if name, found := strings.CutPrefix(ref, "#/components/schemas/"); found {
		return name
	}

	// OAS 2.0: #/definitions/Name
	if name, found := strings.CutPrefix(ref, "#/definitions/"); found {
		return name
	}

	return ""
}

// EachRef calls visit for every reference to a component schema within schema,
// with the location inside schema where it sits. base names where schema
// itself sits, and is empty for a schema that is not nested in another.
func EachRef(schema *parser.Schema, base string, visit func(name, location string)) {
	eachRef(schema, base, visit)
}

func eachRef(schema *parser.Schema, location string, visit func(name, location string)) {
	if schema == nil {
		return
	}

	// Check direct $ref
	if schema.Ref != "" {
		if name := SchemaName(schema.Ref); name != "" {
			at := location
			if at == "" {
				at = "$ref"
			}
			visit(name, at)
		}
	}

	// Check properties
	for propName, propSchema := range schema.Properties {
		if propSchema != nil {
			propLoc := joinLocation(location, fmt.Sprintf("properties.%s", propName))
			eachRef(propSchema, propLoc, visit)
		}
	}

	// Check items
	for i, itemsSchema := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		eachRef(itemsSchema, joinLocation(location, "items"+schemautil.IndexSuffix(i)), visit)
	}

	// Check additionalProperties
	for i, addProps := range schemautil.SchemaOrBoolSchemas(schema.AdditionalProperties) {
		eachRef(addProps, joinLocation(location, "additionalProperties"+schemautil.IndexSuffix(i)), visit)
	}

	// Check composition keywords
	for i, s := range schema.AllOf {
		if s != nil {
			eachRef(s, joinLocation(location, fmt.Sprintf("allOf[%d]", i)), visit)
		}
	}
	for i, s := range schema.AnyOf {
		if s != nil {
			eachRef(s, joinLocation(location, fmt.Sprintf("anyOf[%d]", i)), visit)
		}
	}
	for i, s := range schema.OneOf {
		if s != nil {
			eachRef(s, joinLocation(location, fmt.Sprintf("oneOf[%d]", i)), visit)
		}
	}
	if schema.Not != nil {
		eachRef(schema.Not, joinLocation(location, "not"), visit)
	}

	// Check patternProperties
	for pattern, patternSchema := range schema.PatternProperties {
		if patternSchema != nil {
			eachRef(patternSchema, joinLocation(location, fmt.Sprintf("patternProperties[%s]", pattern)), visit)
		}
	}

	// Check prefixItems (JSON Schema 2020-12)
	for i, s := range schema.PrefixItems {
		if s != nil {
			eachRef(s, joinLocation(location, fmt.Sprintf("prefixItems[%d]", i)), visit)
		}
	}

	// Check additionalItems
	for i, addItems := range schemautil.SchemaOrBoolSchemas(schema.AdditionalItems) {
		eachRef(addItems, joinLocation(location, "additionalItems"+schemautil.IndexSuffix(i)), visit)
	}

	// Check contains
	if schema.Contains != nil {
		eachRef(schema.Contains, joinLocation(location, "contains"), visit)
	}

	// Check propertyNames
	if schema.PropertyNames != nil {
		eachRef(schema.PropertyNames, joinLocation(location, "propertyNames"), visit)
	}

	// Check dependentSchemas
	for depName, depSchema := range schema.DependentSchemas {
		if depSchema != nil {
			eachRef(depSchema, joinLocation(location, fmt.Sprintf("dependentSchemas.%s", depName)), visit)
		}
	}

	// Check conditional schemas (if/then/else)
	if schema.If != nil {
		eachRef(schema.If, joinLocation(location, "if"), visit)
	}
	if schema.Then != nil {
		eachRef(schema.Then, joinLocation(location, "then"), visit)
	}
	if schema.Else != nil {
		eachRef(schema.Else, joinLocation(location, "else"), visit)
	}

	// Check contentSchema
	if schema.ContentSchema != nil {
		eachRef(schema.ContentSchema, joinLocation(location, "contentSchema"), visit)
	}

	// Check $defs
	for defName, defSchema := range schema.Defs {
		if defSchema != nil {
			eachRef(defSchema, joinLocation(location, fmt.Sprintf("$defs.%s", defName)), visit)
		}
	}

	// Check unevaluatedProperties
	for i, unevProps := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedProperties) {
		eachRef(unevProps, joinLocation(location, "unevaluatedProperties"+schemautil.IndexSuffix(i)), visit)
	}

	// Check unevaluatedItems
	for i, unevItems := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedItems) {
		eachRef(unevItems, joinLocation(location, "unevaluatedItems"+schemautil.IndexSuffix(i)), visit)
	}
}

// joinLocation joins location path segments.
func joinLocation(base, segment string) string {
	if base == "" {
		return segment
	}
	return base + "." + segment
}

// Distinct records which schema trees reference each component
// schema name.
//
// Deduplication merges schemas that compare equal, and a name is not part of
// that comparison. So when two names describe the same shape, their names are
// the only thing left separating them, and a schema that references both is
// relying on exactly that:
//
//	Shipment:
//	  shippedFrom: {$ref: OriginAddress}
//	  shippedTo:   {$ref: DestinationAddress}
//
// Both addresses are a street and a city, so they compare equal and a merge
// rewrites one reference to the other's name. The result resolves and
// validates, and says a shipment's origin is its destination (#501).
//
// The unit is the whole tree, not one schema's own properties. Two references
// are held apart however deeply either is nested, and alternatives under
// oneOf are held apart as well, though only one of them is ever present. A
// tree naming two shapes is treating them as two things wherever they sit, and
// where the two readings differ, leaving the names alone keeps the document
// saying what its author wrote.
//
// joiner's Example_deduplicationDistinctNames shows both sides of this from a
// caller's point of view: the pair held apart, and a pair still consolidated.
type Distinct struct {
	// trees maps a component schema name to the trees that reference it.
	// Trees are numbered by the order they were walked in; the number is an
	// identity and means nothing else.
	trees map[string]map[string]struct{}
}

// split partitions a group of equivalent schema names so no part holds two
// names that one schema tree references. Each part is then free to collapse to
// a single name.
//
// OriginAddress, DestinationAddress and BillingAddress all describe the same
// address shape, so they compare equal and arrive as one group. Shipment
// references the first two, and nothing references BillingAddress:
//
//	{BillingAddress, DestinationAddress}  nothing holds these two apart
//	{OriginAddress}                       Shipment references it alongside
//	                                      DestinationAddress
//
// Names are taken in sorted order, which is why DestinationAddress is the one
// keeping BillingAddress company rather than OriginAddress: the parts must not
// depend on the order the deduplicator happened to hash the names in.
//
// A part remembers every tree its members are referenced by, not just its
// first member's, because a name can be free to join one member and still
// clash with another.
func (d *Distinct) Split(group []string) [][]string {
	var parts [][]string
	var partTrees []map[string]struct{}

	for _, name := range slices.Sorted(slices.Values(group)) {
		trees := d.trees[name]
		placed := false
		for i, held := range partTrees {
			if intersects(trees, held) {
				continue
			}
			parts[i] = append(parts[i], name)
			maps.Copy(held, trees)
			placed = true
			break
		}
		if !placed {
			parts = append(parts, []string{name})
			held := make(map[string]struct{}, len(trees))
			maps.Copy(held, trees)
			partTrees = append(partTrees, held)
		}
	}
	return parts
}

// intersects reports whether two tree sets share a member, scanning the smaller
// of the two.
func intersects(left, right map[string]struct{}) bool {
	if len(left) > len(right) {
		left, right = right, left
	}
	for tree := range left {
		if _, ok := right[tree]; ok {
			return true
		}
	}
	return false
}

// Collect records the component schema names that each
// schema tree in doc references.
//
// A tree is a schema with no schema above it, so the names it references are
// the union of all its subschemas'. That makes checking trees the same as
// checking every schema object, and it treats a tree an operation declares
// inline like a named component: deduplication rewrites references wherever
// they sit, so an unnamed parent loses the distinction just as a named one
// does.
//
// The per-tree walk is EachRef, which reads every keyword a subschema can
// hide under. Its reference locations go unused here.
func Collect(doc any) (*Distinct, error) {
	d := &Distinct{trees: make(map[string]map[string]struct{})}
	trees := 0
	err := walker.Walk(&parser.ParseResult{Document: doc},
		walker.WithSchemaHandler(func(_ *walker.WalkContext, schema *parser.Schema) walker.Action {
			// Every schema reaching this handler is a tree root, because
			// SkipChildren stops the walk from descending into subschemas.
			tree := strconv.Itoa(trees)
			EachRef(schema, "", func(name, _ string) {
				holders := d.trees[name]
				if holders == nil {
					holders = make(map[string]struct{}, 1)
					d.trees[name] = holders
				}
				holders[tree] = struct{}{}
			})
			trees++
			return walker.SkipChildren
		}))
	if err != nil {
		return nil, err
	}

	return d, nil
}
