package differ

import "sync"

// Pool capacity (corpus: median=12, p95=13)
const changeSliceCap = 16

var changeSlicePool = sync.Pool{
	New: func() any {
		s := make([]Change, 0, changeSliceCap)
		return &s
	},
}

func getChangeSlice() *[]Change {
	s := changeSlicePool.Get().(*[]Change)
	*s = (*s)[:0]
	return s
}

func putChangeSlice(s *[]Change) {
	if s == nil || cap(*s) > 128 {
		return
	}
	changeSlicePool.Put(s)
}
