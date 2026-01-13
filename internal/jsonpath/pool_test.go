package jsonpath

import "testing"

func TestSegmentSlicePool_Reset(t *testing.T) {
	s := getSegmentSlice()
	*s = append(*s, RootSegment{}, ChildSegment{Key: "paths"})
	putSegmentSlice(s)

	s2 := getSegmentSlice()
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putSegmentSlice(s2)
}

func TestSegmentSlicePool_NilSafe(t *testing.T) {
	// Should not panic on nil
	putSegmentSlice(nil)
}

func TestSegmentSlicePool_OversizedNotReturned(t *testing.T) {
	// Create an oversized slice
	s := getSegmentSlice()
	// Grow beyond the 32-cap threshold
	for i := range 40 {
		*s = append(*s, ChildSegment{Key: string(rune('a' + i%26))})
	}
	// This should not be returned to pool due to size
	putSegmentSlice(s)

	// Get a new slice - should have default capacity
	s2 := getSegmentSlice()
	if cap(*s2) > 32 {
		t.Errorf("expected capacity <= 32, got %d", cap(*s2))
	}
	putSegmentSlice(s2)
}

func TestResultSlicePool_Reset(t *testing.T) {
	s := getResultSlice()
	*s = append(*s, "value1", 42, true)
	putResultSlice(s)

	s2 := getResultSlice()
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putResultSlice(s2)
}

func TestResultSlicePool_NilSafe(t *testing.T) {
	// Should not panic on nil
	putResultSlice(nil)
}

func TestResultSlicePool_OversizedNotReturned(t *testing.T) {
	// Create an oversized slice
	s := getResultSlice()
	// Grow beyond the 128-cap threshold
	for i := range 150 {
		*s = append(*s, i)
	}
	// This should not be returned to pool due to size
	putResultSlice(s)

	// Get a new slice - should have default capacity
	s2 := getResultSlice()
	if cap(*s2) > 128 {
		t.Errorf("expected capacity <= 128, got %d", cap(*s2))
	}
	putResultSlice(s2)
}

func BenchmarkSegmentSlice_WithPool(b *testing.B) {
	for b.Loop() {
		s := getSegmentSlice()
		*s = append(*s, RootSegment{})
		*s = append(*s, ChildSegment{Key: "paths"})
		*s = append(*s, WildcardSegment{})
		putSegmentSlice(s)
	}
}

func BenchmarkSegmentSlice_WithoutPool(b *testing.B) {
	for b.Loop() {
		s := make([]Segment, 0, segmentSliceCap)
		s = append(s, RootSegment{})
		s = append(s, ChildSegment{Key: "paths"})
		s = append(s, WildcardSegment{})
		_ = s
	}
}

func BenchmarkResultSlice_WithPool(b *testing.B) {
	for b.Loop() {
		s := getResultSlice()
		*s = append(*s, "value1", "value2", "value3")
		putResultSlice(s)
	}
}

func BenchmarkResultSlice_WithoutPool(b *testing.B) {
	for b.Loop() {
		s := make([]any, 0, resultSliceCap)
		s = append(s, "value1", "value2", "value3")
		_ = s
	}
}
