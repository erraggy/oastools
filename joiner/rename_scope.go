package joiner

import (
	"maps"

	"github.com/erraggy/oastools/parser"
)

// renameScope records schema renames per source document, so a rename rewrites
// only the references that arrived with the documents it concerns (#478).
type renameScope struct {
	// version selects the $ref spelling: "#/definitions/" or "#/components/schemas/".
	version parser.OASVersion
	// byDoc maps, per source document position, a name as spelled in that
	// document to the name it ends up under. Entries are nil until first use.
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

// registerRight records that a schema from document docIndex was stored under
// newName. sourceName is its name in that document, which is what that
// document's references spell, so a prefix followed by a collision still
// resolves in one step.
func (s *renameScope) registerRight(docIndex int, sourceName, newName string) {
	if s == nil || docIndex < 0 || docIndex >= len(s.byDoc) {
		return
	}
	s.renamesFor(docIndex)[sourceName] = newName
}

// registerLeft records that the schema under oldName moved to newName to make
// room for one from document docIndex. Only earlier documents referenced it, so
// only they follow: one already mapping a name onto oldName is redirected, and
// one already mapping oldName elsewhere means a different schema.
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
// renames at a time. sources are the merged documents, in order.
//
// Keep the claimed containers in step with rewriteOAS2Document: an unclaimed
// entry counts as belonging to no document.
func (s *renameScope) applyOAS2(joined *parser.OAS2Document, sources []*parser.OAS2Document) (map[any]bool, error) {
	if s.empty() {
		return nil, nil
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
// renames at a time. sources are the merged documents, in order.
//
// Keep the claimed containers in step with rewriteOAS3Document: an unclaimed
// entry counts as belonging to no document.
func (s *renameScope) applyOAS3(joined *parser.OAS3Document, sources []*parser.OAS3Document) (map[any]bool, error) {
	if s.empty() {
		return nil, nil
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

// rewrite runs one pass per source document that recorded a rename, each
// restricted to the entries that document contributed, then one for the rest.
func (s *renameScope) rewrite(joined any, owner map[any]int) (map[any]bool, error) {
	copied := make(map[any]bool)
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
		rewriter.copyOnWrite(copied, reown(owner))
		if err := rewriter.RewriteDocument(joined); err != nil {
			return nil, err
		}
	}
	return copied, s.rewriteUnowned(joined, owner, copied)
}

// rewriteUnowned rewrites entries that came from no source document, which a
// collision handler produces with ResolutionCustom. There is no document whose
// namespace to read them in, so every rename applies.
func (s *renameScope) rewriteUnowned(joined any, owner map[any]int, copied map[any]bool) error {
	merged := make(map[string]string)
	for _, renames := range s.byDoc {
		// Merge order breaks the tie when two documents renamed the same name.
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
	rewriter.copyOnWrite(copied, reown(owner))
	return rewriter.RewriteDocument(joined)
}

// reown keeps the ownership map following an entry that copy-on-write replaced,
// so a later pass still reads the copy as belonging to the same document.
func reown(owner map[any]int) func(old, replacement any) {
	return func(old, replacement any) {
		if contributor, known := owner[old]; known {
			owner[replacement] = contributor
		}
	}
}

// rewriteDedupeAliases repoints references from deduplicated names to canonical
// ones. Document-wide, unlike a collision rename: the schemas were found
// equivalent, so the reference means the same thing whoever wrote it.
func rewriteDedupeAliases(joined any, aliases map[string]string, version parser.OASVersion, copied map[any]bool) error {
	rewriter := NewSchemaRewriter()
	for alias, canonical := range aliases {
		rewriter.RegisterRename(alias, canonical, version)
	}
	if copied == nil {
		copied = make(map[any]bool)
	}
	rewriter.copyOnWrite(copied, nil)
	return rewriter.RewriteDocument(joined)
}

// claimEntries records which document contributed each entry of a top-level
// container. The joiner merges by pointer, so pointer identity links a joined
// entry back to its document. First contributor wins. The *T value type keeps
// non-pointer entries out, which would be matched on content instead.
func claimEntries[T any](owner map[any]int, docIndex int, entries map[string]*T) {
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if _, claimed := owner[entry]; !claimed {
			owner[entry] = docIndex
		}
	}
}
