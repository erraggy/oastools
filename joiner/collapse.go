package joiner

import (
	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// This file is the second half of StrategyDeduplicateOrRename. The merge loop
// renames every colliding schema; everything here runs afterwards, in the one
// window where both of these hold:
//
//   - Every document's renames are known. rename-left moves an earlier
//     document's schema, so an earlier document's mapping can still change
//     while a later document is merged.
//   - No rename has been applied. A schema headed for a collapse can still be
//     dropped rather than rewritten and copied.
//
// Leaving that window in either direction is what makes the decision either
// unsound or expensive. See issue #487.

// collapseDeferredRenames settles every collision the strategy put off: it
// drops the renames that turn out not to be needed, then reports the outcome
// of each collision.
func (j *Joiner) collapseDeferredRenames(schemas map[string]*parser.Schema, owner map[any]int, result *JoinResult) {
	if len(result.deferred) == 0 {
		return
	}
	j.reportDeferredRenames(result, j.collapseEquivalent(schemas, owner, result))
}

// collapseEquivalent removes every schema that another schema in the joined
// document can stand in for, and returns what each removed name collapsed into.
func (j *Joiner) collapseEquivalent(schemas map[string]*parser.Schema, owner map[any]int, result *JoinResult) map[string]string {
	// owner is the map the rewrite reads entries through, so every schema is
	// compared under the renames that will actually be applied to it.
	views := newViewIndex(result.scope, owner, schemas)
	generated := result.generated
	// Always deep, whatever EquivalenceMode says. A shallow comparison reads
	// neither $ref nor nested properties, so it cannot decide whether two
	// schemas are interchangeable. EquivalenceDocs still applies.
	opts := j.buildCompareOptions(EquivalenceModeDeep)

	aliases := make(map[string]string)
	for _, group := range collapseGroups(result.deferred, schemas, views) {
		for _, class := range partitionEquivalent(group, schemas, views, opts) {
			if len(class) < 2 {
				continue
			}
			canonical := canonicalName(class, generated)
			for _, name := range class {
				if name == canonical {
					continue
				}
				aliases[name] = canonical
				// Withdraw the rename before the entry goes. Left in place, it
				// rewrites references to a name the collapse just removed.
				result.scope.redirect(views.docOf(name), name, canonical)
				delete(schemas, name)
			}
		}
	}
	return aliases
}

// reportDeferredRenames writes the warning and the collision event for each
// deferred collision, now that the collapse has decided its outcome.
func (j *Joiner) reportDeferredRenames(result *JoinResult, aliases map[string]string) {
	for _, deferred := range result.deferred {
		// The collision was deduplicated if its two names ended up in the same
		// class, whichever of the two is the one that survived.
		if canonicalOf(aliases, deferred.name) == canonicalOf(aliases, deferred.newName) {
			result.AddWarning(NewSchemaDedupWarning(
				deferred.name, deferred.label, deferred.rightSource, deferred.line, deferred.column))
			j.recordCollisionEvent(result, deferred.name, deferred.leftSource, deferred.rightSource,
				StrategyDeduplicateOrRename, resolutionDeduplicated, "")
			continue
		}
		result.AddWarning(NewSchemaRenamedWarning(
			deferred.name, deferred.newName, deferred.label, deferred.rightSource, deferred.line, deferred.column, false))
		j.recordCollisionEvent(result, deferred.name, deferred.leftSource, deferred.rightSource,
			StrategyDeduplicateOrRename, resolutionRenamed, deferred.newName)
	}
	result.deferred = nil
}

// canonicalOf returns the name something collapsed into, or the name itself.
func canonicalOf(aliases map[string]string, name string) string {
	// Classes are disjoint and a canonical name is never itself an alias, so a
	// single lookup is the whole chain.
	if canonical, ok := aliases[name]; ok {
		return canonical
	}
	return name
}

// viewIndex says, for each schema in the joined document, which document
// contributed it and how that document's names read once renames are applied.
type viewIndex struct {
	// docs maps a name in the joined document to the position of the document
	// that contributed it, or -1 when no document did.
	docs map[string]int
	// views holds one view per document, built once.
	views map[int]*refView
	// merged is the view for entries no document contributed, which a collision
	// handler produces with ResolutionCustom. It matches how rewriteUnowned
	// rewrites them.
	merged *refView
}

func newViewIndex(scope *renameScope, owner map[any]int, schemas map[string]*parser.Schema) *viewIndex {
	index := &viewIndex{
		docs:   make(map[string]int, len(schemas)),
		views:  make(map[int]*refView),
		merged: scope.mergedView(),
	}
	for name, schema := range schemas {
		// Ownership is by pointer, not by name. A schema is read in the
		// namespace of the document that wrote it, which is not necessarily the
		// one whose renames are recorded against the name it now sits at.
		docIndex, known := owner[schema]
		if !known {
			docIndex = -1
		}
		index.docs[name] = docIndex
		if _, built := index.views[docIndex]; !built && docIndex >= 0 {
			index.views[docIndex] = scope.view(docIndex)
		}
	}
	return index
}

// docOf returns the position of the document that contributed name, or -1.
func (i *viewIndex) docOf(name string) int {
	docIndex, known := i.docs[name]
	if !known {
		return -1
	}
	return docIndex
}

// view returns the view a schema's references are read through.
func (i *viewIndex) view(name string) *refView {
	if docIndex := i.docOf(name); docIndex >= 0 {
		return i.views[docIndex]
	}
	return i.merged
}

// collapseGroups returns the sets of names a collapse may consider, one per
// connected set of deferred collisions.
func collapseGroups(deferred []deferredRename, schemas map[string]*parser.Schema, views *viewIndex) [][]string {
	parent := make(map[string]string)
	var members []string

	var find func(string) string
	find = func(name string) string {
		if parent[name] == name {
			return name
		}
		parent[name] = find(parent[name])
		return parent[name]
	}
	canCompare := func(name string) bool {
		// An entry no document contributed has no namespace to be read in, so
		// no view can say what its references resolve to.
		_, present := schemas[name]
		return present && views.docOf(name) >= 0
	}
	note := func(name string) {
		if _, seen := parent[name]; seen {
			return
		}
		parent[name] = name
		members = append(members, name)
	}

	// Only names that collided are candidates. Two schemas that merely happen to
	// have the same shape are left alone: consolidating those is what
	// SemanticDeduplication is for, and doing it from a collision strategy would
	// move names nothing collided on.
	for _, entry := range deferred {
		// Skipping the pair costs only this collision. The two names still group
		// through whatever other collisions they took part in.
		if !canCompare(entry.name) || !canCompare(entry.newName) {
			continue
		}
		note(entry.name)
		note(entry.newName)
		// Union, not a pair: repeated collisions on one name leave a chain, so
		// three documents spelling Pet leave (Pet, Pet_1) and (Pet, Pet_2),
		// which is one group of three rather than two groups of two.
		parent[find(entry.name)] = find(entry.newName)
	}

	// Walking members rather than parent keeps the result in collision order,
	// which the merge loop produced in document order. Ranging a map here would
	// make the grouping, and so the canonical names, vary between runs.
	groups := make(map[string][]string)
	var order []string
	for _, name := range members {
		root := find(name)
		if len(groups[root]) == 0 {
			order = append(order, root)
		}
		groups[root] = append(groups[root], name)
	}

	result := make([][]string, 0, len(order))
	for _, root := range order {
		result = append(result, groups[root])
	}
	return result
}

// partitionEquivalent splits a group into classes of interchangeable schemas,
// in the order their first member appears in the group.
func partitionEquivalent(
	group []string,
	schemas map[string]*parser.Schema,
	views *viewIndex,
	opts CompareOptions,
) [][]string {
	hasher := schemautil.NewSchemaHasher()
	var classes [][]string
	byHash := make(map[uint64][]int)

	for _, name := range group {
		schema := schemas[name]
		// The hash is a pre-filter, bounding the comparison when a group's
		// members mostly differ. It reads reference spellings rather than what
		// they resolve to, so two schemas that reach the same target by
		// different spellings are never compared and their rename stands.
		// Getting there takes a collision resolved outside this strategy, and it
		// costs a rename that could have been withdrawn, never a wrong merge.
		hash := hasher.Hash(schema)
		placed := false
		for _, index := range byHash[hash] {
			representative := classes[index][0]
			// Each side is read through its own document's view, so the two are
			// equivalent exactly when they would still be equivalent after the
			// rewrite. Comparing what the documents literally say instead merges
			// schemas whose references diverge further down: see refView.
			if compareSchemas(schemas[representative], schema, opts,
				views.view(representative), views.view(name)).Equivalent {
				classes[index] = append(classes[index], name)
				placed = true
				break
			}
		}
		if !placed {
			byHash[hash] = append(byHash[hash], len(classes))
			classes = append(classes, []string{name})
		}
	}

	return classes
}

// canonicalName picks the name a collapsed class keeps.
func canonicalName(class []string, generated map[string]bool) string {
	canonical := class[0]
	for _, name := range class[1:] {
		if outranks(name, canonical, generated) {
			canonical = name
		}
	}
	return canonical
}

// outranksGenerated adapts outranks for schemautil.SchemaDeduplicator, so
// semantic deduplication settles a group of equivalent names the way a collapse
// would rather than by sorting alone (#498). It returns nil when no name was
// generated, leaving the deduplicator's own alphabetical tiebreak in place.
func outranksGenerated(generated map[string]bool) schemautil.OutranksFunc {
	if len(generated) == 0 {
		return nil
	}
	return func(name, candidate string) bool {
		return outranks(name, candidate, generated)
	}
}

// outranks reports whether name should survive a collapse in place of canonical.
func outranks(name, canonical string, generated map[string]bool) bool {
	// A name no rename generated wins, so the name the documents wrote survives
	// rather than an alias that happens to sort first: a rename template of
	// Api_{{.Name}} puts Api_Common ahead of Common alphabetically, and Common
	// is what every document spelled.
	if generated[name] != generated[canonical] {
		return !generated[name]
	}
	// Ties go alphabetically, the way schemautil.SchemaDeduplicator breaks them.
	return name < canonical
}
