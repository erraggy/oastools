package pathutil

import "sync"

const (
	defaultPathCap = 8  // Most paths are <8 segments deep
	maxPathCap     = 64 // Don't pool excessively deep paths
)

var pathBuilderPool = sync.Pool{
	New: func() any {
		return &PathBuilder{
			segments: make([]string, 0, defaultPathCap),
		}
	},
}

// Get retrieves a PathBuilder from the pool, reset and ready to use.
func Get() *PathBuilder {
	p := pathBuilderPool.Get().(*PathBuilder)
	p.Reset()
	return p
}

// Put returns a PathBuilder to the pool if not oversized.
func Put(p *PathBuilder) {
	if p == nil || cap(p.segments) > maxPathCap {
		return // Let GC collect oversized builders
	}
	pathBuilderPool.Put(p)
}
