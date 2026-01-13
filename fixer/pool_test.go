package fixer

import "testing"

func TestFixSlicePool_Reset(t *testing.T) {
	s := getFixSlice()
	*s = append(*s, Fix{Type: FixTypeMissingPathParameter, Path: "$.paths./users/{id}"})
	putFixSlice(s)

	s2 := getFixSlice()
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putFixSlice(s2)
}

func BenchmarkFixSlice_WithPool(b *testing.B) {
	for b.Loop() {
		s := getFixSlice()
		*s = append(*s, Fix{Type: FixTypeMissingPathParameter})
		putFixSlice(s)
	}
}
