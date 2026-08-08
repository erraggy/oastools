package joiner

import (
	"maps"

	"github.com/erraggy/oastools/parser"
)

// renameScope records schema renames per source document.
//
// A rename resolves a collision between two documents, so it only speaks for
// the documents that were written against the renamed name. Renaming an
// incoming schema (rename-right, or a namespace prefix) concerns that document
// alone; renaming a schema already in the joined document (rename-left)
// concerns the documents merged before it. Applying every rename to the whole
// merged document repoints references that were already correct: see #478.
type renameScope struct {
	// version selects the $ref spelling: "#/definitions/" or "#/components/schemas/".
	version parser.OASVersion
	// byDoc is indexed by source document position and maps a name as spelled
	// in that document to the name it ends up under in the joined document.
	// Entries stay nil until that document has a rename, which is the common
	// case: most joins resolve no collision by renaming.
	byDoc []map[string]string
}

// newRenameScope returns a scope sized for docCount source documents.
func newRenameScope(docCount int, version parser.OASVersion) *renameScope {
	return &renameScope{version: version, byDoc: make([]map[string]string, docCount)}
}

// renamesFor returns the rename map for a document, creating it on first use.
func (s *renameScope) renamesFor(docIndex int) map[string]string {
	if s.byDoc[docIndex] == nil {
		s.byDoc[docIndex] = make(map[string]string)
	}
	return s.byDoc[docIndex]
}

// registerRight records that a schema arriving with document docIndex was stored
// under newName.
//
// sourceName is the name the schema carries in its own document, which is what
// that document's references spell. Keying on it means a schema renamed twice
// (a namespace prefix, then a collision) resolves to its final name in one step
// instead of leaving a reference stranded at the intermediate name.
func (s *renameScope) registerRight(docIndex int, sourceName, newName string) {
	if s == nil || docIndex < 0 || docIndex >= len(s.byDoc) {
		return
	}
	s.renamesFor(docIndex)[sourceName] = newName
}

// registerLeft records that the schema already stored under oldName was moved to
// newName to make room for a schema from document docIndex.
//
// Every document merged before docIndex spelled oldName meaning the schema being
// moved, so each of them follows it. A document that already maps some name onto
// oldName is redirected rather than given a second entry, and one that already
// maps oldName somewhere else is not talking about the schema being moved and is
// left alone.
func (s *renameScope) registerLeft(docIndex int, oldName, newName string) {
	if s == nil {
		return
	}
	for e := 0; e < docIndex && e < len(s.byDoc); e++ {
		for src, dst := range s.byDoc[e] {
			if dst == oldName {
				s.byDoc[e][src] = newName
			}
		}
		if _, taken := s.byDoc[e][oldName]; taken {
			continue
		}
		s.renamesFor(e)[oldName] = newName
	}
}

// empty reports whether no document has a rename recorded.
func (s *renameScope) empty() bool {
	if s == nil {
		return true
	}
	for _, m := range s.byDoc {
		if len(m) > 0 {
			return false
		}
	}
	return true
}

// applyOAS2 rewrites the joined document's references, one source document's
// renames at a time. sources are the documents that were merged, in order.
//
// The entries claimed here must cover everything SchemaRewriter traverses at the
// top level, otherwise an unclaimed entry belongs to no document and is never
// rewritten: keep this in step with rewriteOAS2Document.
func (s *renameScope) applyOAS2(joined *parser.OAS2Document, sources []*parser.OAS2Document) error {
	if s.empty() {
		return nil
	}
	owner := make(map[any]int)
	for docIndex, src := range sources {
		claimEntries(owner, docIndex, src.Definitions)
		claimEntries(owner, docIndex, src.Parameters)
		claimEntries(owner, docIndex, src.Responses)
		claimEntries(owner, docIndex, src.Paths)
	}
	return s.rewrite(joined, owner)
}

// applyOAS3 rewrites the joined document's references, one source document's
// renames at a time. sources are the documents that were merged, in order.
//
// The entries claimed here must cover everything SchemaRewriter traverses at the
// top level, otherwise an unclaimed entry belongs to no document and is never
// rewritten: keep this in step with rewriteOAS3Document.
func (s *renameScope) applyOAS3(joined *parser.OAS3Document, sources []*parser.OAS3Document) error {
	if s.empty() {
		return nil
	}
	owner := make(map[any]int)
	for docIndex, src := range sources {
		if src.Components != nil {
			claimEntries(owner, docIndex, src.Components.Schemas)
			claimEntries(owner, docIndex, src.Components.Parameters)
			claimEntries(owner, docIndex, src.Components.Responses)
			claimEntries(owner, docIndex, src.Components.RequestBodies)
			claimEntries(owner, docIndex, src.Components.Headers)
			claimEntries(owner, docIndex, src.Components.Callbacks)
			claimEntries(owner, docIndex, src.Components.PathItems)
		}
		claimEntries(owner, docIndex, src.Paths)
		claimEntries(owner, docIndex, src.Webhooks)
	}
	return s.rewrite(joined, owner)
}

// rewrite runs one restricted pass over the joined document per source document
// that had a rename recorded. Each pass skips the top-level entries another
// document contributed, so it descends into a subtree only when that subtree's
// own document is being rewritten.
func (s *renameScope) rewrite(joined any, owner map[any]int) error {
	for docIndex, renames := range s.byDoc {
		if len(renames) == 0 {
			continue
		}
		rewriter := NewSchemaRewriter()
		for oldName, newName := range renames {
			rewriter.RegisterRename(oldName, newName, s.version)
		}
		rewriter.restrictTo(func(entry any) bool {
			contributor, known := owner[entry]
			return known && contributor == docIndex
		})
		if err := rewriter.RewriteDocument(joined); err != nil {
			return err
		}
	}
	return s.rewriteUnowned(joined, owner)
}

// rewriteUnowned rewrites the top-level entries that came from no source document.
//
// A collision handler returning ResolutionCustom supplies a value the joiner
// never received from a document (see applySchemaResolution and
// applyPathResolution). It belongs to no document's namespace, so there is
// nothing to scope its references to and every rename applies, which is the
// document-wide treatment every entry had before renames became scoped. Skipping
// these would leave a handler's references pointing at names the join no longer
// has.
func (s *renameScope) rewriteUnowned(joined any, owner map[any]int) error {
	merged := make(map[string]string)
	for _, renames := range s.byDoc {
		// Documents are visited in merge order, so when two of them renamed the
		// same name to different targets the later one wins. The name is genuinely
		// ambiguous for a value that belongs to no document, and merge order is the
		// only ordering the join has to break the tie with.
		maps.Copy(merged, renames)
	}
	rewriter := NewSchemaRewriter()
	for oldName, newName := range merged {
		rewriter.RegisterRename(oldName, newName, s.version)
	}
	rewriter.restrictTo(func(entry any) bool {
		_, known := owner[entry]
		return !known
	})
	return rewriter.RewriteDocument(joined)
}

// rewriteDedupeAliases repoints every reference to a deduplicated name at the
// canonical name that replaced it.
//
// Unlike a collision rename, this is document-wide: deduplication only
// consolidates schemas it found equivalent, so a reference to an alias means the
// canonical schema no matter which document wrote it.
func rewriteDedupeAliases(joined any, aliases map[string]string, version parser.OASVersion) error {
	rewriter := NewSchemaRewriter()
	for alias, canonical := range aliases {
		rewriter.RegisterRename(alias, canonical, version)
	}
	return rewriter.RewriteDocument(joined)
}

// claimEntries records the source document that contributed each entry of a
// top-level container. The joiner merges by pointer, so the joined document
// holds these very values and pointer identity is what links them back.
//
// The first document to contribute a value keeps it. That only matters when the
// caller passes the same parsed document twice, in which case the two positions
// share every pointer and there is no contribution to tell apart.
func claimEntries[T comparable](owner map[any]int, docIndex int, entries map[string]T) {
	var zero T
	for _, entry := range entries {
		if entry == zero {
			continue
		}
		if _, claimed := owner[entry]; !claimed {
			owner[entry] = docIndex
		}
	}
}
