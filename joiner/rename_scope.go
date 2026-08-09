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

// redirect points whatever mapped onto from at to, because the schema under
// from was collapsed into the one under to.
func (s *renameScope) redirect(docIndex int, from, to string) {
	if s == nil || docIndex < 0 || docIndex >= len(s.byDoc) || from == to {
		return
	}
	// uniqueSchemaName gives every rename a target no other entry holds, so at
	// most one source name maps onto from, and the first match ends the search.
	for source, destination := range s.byDoc[docIndex] {
		if destination != from {
			continue
		}
		if source == to {
			// The document's own spelling survived the collapse. Dropping the
			// entry is what makes the saving: with nothing left to rewrite, the
			// rewrite skips this document and copies none of its schemas.
			delete(s.byDoc[docIndex], source)
		} else {
			s.byDoc[docIndex][source] = to
		}
		return
	}
	// No rename produced from, so the document spells it as it stands and needs
	// a new entry to follow the collapse.
	s.renamesFor(docIndex)[from] = to
}

// generatedNames returns the names that exist only because a rename produced
// them, which is what tells a generated alias apart from a name a document
// wrote when a collapse has to choose between them (#487).
func (s *renameScope) generatedNames() map[string]bool {
	if s == nil {
		return nil
	}
	// Every rename target qualifies: uniqueSchemaName picks a name no other
	// entry holds, so no document can have spelled it.
	generated := make(map[string]bool)
	for _, renames := range s.byDoc {
		for _, destination := range renames {
			generated[destination] = true
		}
	}
	return generated
}

// view returns the reference view for one source document: how the names that
// document spells read once its renames are applied. It is nil when the
// document has no renames, which refView treats as the identity.
func (s *renameScope) view(docIndex int) *refView {
	if s == nil || docIndex < 0 || docIndex >= len(s.byDoc) {
		return nil
	}
	return newRefView(s.byDoc[docIndex], s.version)
}

// mergedView returns the view for entries no source document contributed,
// matching how rewriteUnowned rewrites them.
func (s *renameScope) mergedView() *refView {
	if s == nil {
		return nil
	}
	return newRefView(s.mergedRenames(), s.version)
}

// mergedRenames returns every document's renames in one map, merge order
// breaking the tie when two documents renamed the same name.
//
// Entries no document contributed have no namespace of their own to be read in,
// so every rename applies to them. mergedView and rewriteUnowned both read this
// so the comparison and the rewrite cannot disagree about which.
func (s *renameScope) mergedRenames() map[string]string {
	merged := make(map[string]string)
	for _, renames := range s.byDoc {
		maps.Copy(merged, renames)
	}
	return merged
}

// newRefView builds the view for one set of renames.
func newRefView(renames map[string]string, version parser.OASVersion) *refView {
	if len(renames) == 0 {
		return nil
	}
	view := &refView{
		refs: make(map[string]string, len(renames)),
		// Cloned, not shared. The collapse redirects renames once it has
		// finished comparing, and a view that followed those edits would judge
		// later groups against a mapping earlier groups never saw.
		names: maps.Clone(renames),
	}
	// Both spellings, so a $ref and a discriminator entry naming the same schema
	// resolve through the same view.
	for oldName, newName := range renames {
		view.refs[schemaRefPath(oldName, version)] = schemaRefPath(newName, version)
	}
	return view
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
func (s *renameScope) applyOAS2(joined *parser.OAS2Document, owner map[any]int) (map[any]bool, error) {
	if s.empty() {
		return nil, nil
	}
	return s.rewrite(joined, owner)
}

// ownersOAS2 records which document contributed each top-level entry.
//
// Keep the claimed containers in step with rewriteOAS2Document: an unclaimed
// entry counts as belonging to no document.
func ownersOAS2(sources []*parser.OAS2Document) map[any]int {
	owner := make(map[any]int)
	for docIndex, src := range sources {
		claimEntries(owner, docIndex, src.Definitions)
		claimEntries(owner, docIndex, src.Parameters)
		claimEntries(owner, docIndex, src.Responses)
		claimEntries(owner, docIndex, src.Paths)
	}
	return owner
}

// applyOAS3 rewrites the joined document's references, one source document's
// renames at a time. sources are the merged documents, in order.
//
// Keep the claimed containers in step with rewriteOAS3Document: an unclaimed
// entry counts as belonging to no document.
func (s *renameScope) applyOAS3(joined *parser.OAS3Document, owner map[any]int) (map[any]bool, error) {
	if s.empty() {
		return nil, nil
	}
	return s.rewrite(joined, owner)
}

// ownersOAS3 records which document contributed each top-level entry.
//
// Keep the claimed containers in step with rewriteOAS3Document: an unclaimed
// entry counts as belonging to no document.
func ownersOAS3(sources []*parser.OAS3Document) map[any]int {
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
			claimEntries(owner, docIndex, src.Components.MediaTypes)
		}
		claimEntries(owner, docIndex, src.Paths)
		claimEntries(owner, docIndex, src.Webhooks)
	}
	return owner
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
	rewriter := NewSchemaRewriter()
	for oldName, newName := range s.mergedRenames() {
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
