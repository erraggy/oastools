package fixer

import "sync"

// Pool capacity (corpus: p95=3, median=0)
// Note: Most specs have 0 fixes, so this pool has low hit rate.
// Kept small to minimize memory overhead.
const fixSliceCap = 4

var fixSlicePool = sync.Pool{
	New: func() any {
		s := make([]Fix, 0, fixSliceCap)
		return &s
	},
}

func getFixSlice() *[]Fix {
	s := fixSlicePool.Get().(*[]Fix)
	*s = (*s)[:0]
	return s
}

func putFixSlice(s *[]Fix) {
	if s == nil || cap(*s) > 32 {
		return
	}
	fixSlicePool.Put(s)
}
