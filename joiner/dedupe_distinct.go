package joiner

import (
	"maps"
	"slices"

	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

// distinctSchemaNames records which schema trees reference each component
// schema name, so semantic deduplication can hold apart two names that one
// tree references. A schema that references both is distinguishing them:
// whatever their shapes have in common, the document says a value of one is
// not a value of the other (#501).
type distinctSchemaNames struct {
	// trees maps a component schema name to the trees that reference it.
	trees map[string]map[int]struct{}
}

// split partitions a group of equivalent schema names so that no part holds
// two names one schema tree references.
//
// Greedy over the sorted group: a name joins the first part no tree references
// it alongside, so the partition does not depend on the order the deduplicator
// hashed the names in. Each part carries the union of its members' trees,
// which is what a name has to miss to join it: intersecting the union once is
// the same answer as comparing the name against every member, and costs the
// references the name actually has rather than the size of the part.
func (d *distinctSchemaNames) split(group []string) [][]string {
	sorted := slices.Sorted(slices.Values(group))

	var parts [][]string
	var partTrees []map[int]struct{}

	for _, name := range sorted {
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
			held := make(map[int]struct{}, len(trees))
			maps.Copy(held, trees)
			partTrees = append(partTrees, held)
		}
	}
	return parts
}

// intersects reports whether two tree sets share a member, scanning the smaller
// of the two.
func intersects(left, right map[int]struct{}) bool {
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

// collectDistinctSchemaNames records the component schema names each schema
// tree in doc references.
//
// A tree is a schema with no schema above it, so the names it references are
// the union of every subschema's. Checking trees is therefore the same as
// checking every schema object, and it counts a tree an operation declares
// inline the same as a named component: deduplication rewrites references
// wherever they sit, so an unnamed parent loses the distinction just as a
// named one does.
func collectDistinctSchemaNames(doc any) (*distinctSchemaNames, error) {
	d := &distinctSchemaNames{trees: make(map[string]map[int]struct{})}
	tree := 0

	err := walker.Walk(&parser.ParseResult{Document: doc},
		walker.WithSchemaHandler(func(_ *walker.WalkContext, schema *parser.Schema) walker.Action {
			// Every schema reaching this handler is a tree root, because
			// SkipChildren stops the walk from descending into subschemas.
			d.record(tree, schema)
			tree++
			return walker.SkipChildren
		}))
	if err != nil {
		return nil, err
	}
	return d, nil
}

// record notes every component schema name referenced anywhere under schema as
// referenced by tree.
func (d *distinctSchemaNames) record(tree int, schema *parser.Schema) {
	if schema == nil {
		return
	}

	if name := extractSchemaNameFromRef(schema.Ref); name != "" {
		trees := d.trees[name]
		if trees == nil {
			trees = make(map[int]struct{}, 1)
			d.trees[name] = trees
		}
		trees[tree] = struct{}{}
	}

	// Every keyword a subschema can sit under. RefGraph.recordSchemaRefs walks
	// the same set; this one is kept separate because it needs no location
	// strings, and building them for references that are only counted was the
	// larger cost of the two.
	for _, sub := range schema.Properties {
		d.record(tree, sub)
	}
	for _, sub := range schema.PatternProperties {
		d.record(tree, sub)
	}
	for _, sub := range schema.DependentSchemas {
		d.record(tree, sub)
	}
	for _, sub := range schema.Defs {
		d.record(tree, sub)
	}
	for _, sub := range schema.AllOf {
		d.record(tree, sub)
	}
	for _, sub := range schema.AnyOf {
		d.record(tree, sub)
	}
	for _, sub := range schema.OneOf {
		d.record(tree, sub)
	}
	for _, sub := range schema.PrefixItems {
		d.record(tree, sub)
	}
	for _, sub := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		d.record(tree, sub)
	}
	for _, sub := range schemautil.SchemaOrBoolSchemas(schema.AdditionalProperties) {
		d.record(tree, sub)
	}
	for _, sub := range schemautil.SchemaOrBoolSchemas(schema.AdditionalItems) {
		d.record(tree, sub)
	}
	for _, sub := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedProperties) {
		d.record(tree, sub)
	}
	for _, sub := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedItems) {
		d.record(tree, sub)
	}
	d.record(tree, schema.Not)
	d.record(tree, schema.Contains)
	d.record(tree, schema.PropertyNames)
	d.record(tree, schema.If)
	d.record(tree, schema.Then)
	d.record(tree, schema.Else)
	d.record(tree, schema.ContentSchema)
}
