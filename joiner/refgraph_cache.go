package joiner

// refGraphs hands out a source document's reference graph by position, building
// each at most once.
//
// A nil *refGraphs yields nil graphs, which is what the rename context reads
// when WithOperationContext is off.
type refGraphs struct {
	build func(docIndex int) *RefGraph
	cache map[int]*RefGraph
}

// newRefGraphs returns a cache over build, or nil when enabled is false so the
// graphs are never built.
func newRefGraphs(enabled bool, build func(docIndex int) *RefGraph) *refGraphs {
	if !enabled {
		return nil
	}
	return &refGraphs{build: build, cache: make(map[int]*RefGraph)}
}

// forDoc returns the graph of the document at docIndex.
func (g *refGraphs) forDoc(docIndex int) *RefGraph {
	if g == nil || docIndex < 0 {
		return nil
	}
	if cached, ok := g.cache[docIndex]; ok {
		return cached
	}
	graph := g.build(docIndex)
	g.cache[docIndex] = graph
	return graph
}
