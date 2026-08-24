package joiner

import (
	"maps"
	"slices"
	"strconv"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

// distinctSchemaNames records which schema trees reference each component
// schema name.
//
// A schema that references two names is distinguishing them, so semantic
// deduplication must not merge the two however alike their shapes are. In
//
//	Shipment:
//	  shippedFrom: {$ref: OriginAddress}
//	  shippedTo:   {$ref: DestinationAddress}
//
// both addresses have the same shape, and merging them would leave a
// shipment's origin as its destination (#501).
type distinctSchemaNames struct {
	// trees maps a component schema name to the trees that reference it.
	// Trees are numbered by the order they were walked in; the number is an
	// identity and means nothing else.
	trees map[string]map[string]struct{}
}

// split partitions a group of equivalent schema names so no part holds two
// names that one schema tree references. Each part is then free to collapse to
// a single name.
//
// Given a group of OriginAddress, DestinationAddress and BillingAddress, where
// Shipment references the first two and nothing references BillingAddress:
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
func (d *distinctSchemaNames) split(group []string) [][]string {
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

// collectDistinctSchemaNames records the component schema names that each
// schema tree in doc references.
//
// A tree is a schema with no schema above it, so the names it references are
// the union of all its subschemas'. That makes checking trees the same as
// checking every schema object, and it treats a tree an operation declares
// inline like a named component: deduplication rewrites references wherever
// they sit, so an unnamed parent loses the distinction just as a named one
// does.
//
// The per-tree walk is RefGraph.recordSchemaRefs, which already reads every
// keyword a subschema can hide under. Its reference locations go unused here,
// but a second walk of its own would be a second list of keywords to keep
// current, and one that missed a keyword would merge names a document
// distinguishes under it.
func collectDistinctSchemaNames(doc any) (*distinctSchemaNames, error) {
	g := newRefGraph()
	trees := 0
	err := walker.Walk(&parser.ParseResult{Document: doc},
		walker.WithSchemaHandler(func(_ *walker.WalkContext, schema *parser.Schema) walker.Action {
			// Every schema reaching this handler is a tree root, because
			// SkipChildren stops the walk from descending into subschemas.
			g.recordSchemaRefs(strconv.Itoa(trees), schema, "")
			trees++
			return walker.SkipChildren
		}))
	if err != nil {
		return nil, err
	}

	d := &distinctSchemaNames{trees: make(map[string]map[string]struct{}, len(g.schemaRefs))}
	for name, refs := range g.schemaRefs {
		holders := make(map[string]struct{}, len(refs))
		for _, ref := range refs {
			holders[ref.FromSchema] = struct{}{}
		}
		d.trees[name] = holders
	}
	return d, nil
}
