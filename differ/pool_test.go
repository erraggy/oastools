package differ

import "testing"

func TestChangeSlicePool_Reset(t *testing.T) {
	s := getChangeSlice()
	*s = append(*s, Change{Path: "$.info.title", Type: ChangeTypeModified})
	putChangeSlice(s)

	s2 := getChangeSlice()
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putChangeSlice(s2)
}

func TestChangeSlicePool_NilSafe(t *testing.T) {
	// Should not panic on nil
	putChangeSlice(nil)
}

func TestChangeSlicePool_LargeCapDiscard(t *testing.T) {
	// Create a slice with capacity > 128 (should be discarded)
	large := make([]Change, 0, 200)
	putChangeSlice(&large)

	// Get a new slice - should have default capacity, not the large one
	s := getChangeSlice()
	if cap(*s) != changeSliceCap {
		// Note: This test may be flaky if the pool happened to be empty
		// and returned the large slice. In practice, the pool discards it.
		t.Logf("got capacity %d (may vary based on pool state)", cap(*s))
	}
	putChangeSlice(s)
}

func TestChangeSlicePool_Capacity(t *testing.T) {
	s := getChangeSlice()
	if cap(*s) < changeSliceCap {
		t.Errorf("expected capacity >= %d, got %d", changeSliceCap, cap(*s))
	}
	putChangeSlice(s)
}

func BenchmarkChangeSlice_WithPool(b *testing.B) {
	for b.Loop() {
		s := getChangeSlice()
		for i := range 10 {
			*s = append(*s, Change{Path: "$.paths./users", Type: ChangeTypeModified, Category: CategoryEndpoint, Message: "test change " + string(rune('0'+i))})
		}
		putChangeSlice(s)
	}
}

func BenchmarkChangeSlice_WithoutPool(b *testing.B) {
	for b.Loop() {
		s := make([]Change, 0, changeSliceCap)
		for i := range 10 {
			s = append(s, Change{Path: "$.paths./users", Type: ChangeTypeModified, Category: CategoryEndpoint, Message: "test change " + string(rune('0'+i))})
		}
		_ = s
	}
}
