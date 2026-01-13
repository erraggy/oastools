package jsonpath

import "sync"

// Pool capacities (corpus-validated: 3-4 tokens typical)
const (
	segmentSliceCap = 8
	resultSliceCap  = 32
)

var segmentSlicePool = sync.Pool{
	New: func() any {
		s := make([]Segment, 0, segmentSliceCap)
		return &s
	},
}

func getSegmentSlice() *[]Segment {
	s := segmentSlicePool.Get().(*[]Segment)
	*s = (*s)[:0]
	return s
}

func putSegmentSlice(s *[]Segment) {
	if s == nil || cap(*s) > 32 {
		return
	}
	segmentSlicePool.Put(s)
}

var resultSlicePool = sync.Pool{
	New: func() any {
		s := make([]any, 0, resultSliceCap)
		return &s
	},
}

func getResultSlice() *[]any {
	s := resultSlicePool.Get().(*[]any)
	*s = (*s)[:0]
	return s
}

func putResultSlice(s *[]any) {
	if s == nil || cap(*s) > 128 {
		return
	}
	resultSlicePool.Put(s)
}
