package converter

import "testing"

func TestConversionMapPool_Clear(t *testing.T) {
	m := getConversionMap()
	m["#/definitions/Pet"] = "#/components/schemas/Pet"
	putConversionMap(m)

	m2 := getConversionMap()
	if len(m2) != 0 {
		t.Errorf("expected empty map, got len=%d", len(m2))
	}
	putConversionMap(m2)
}

func BenchmarkConversionMap_WithPool(b *testing.B) {
	for b.Loop() {
		m := getConversionMap()
		for i := range 100 {
			m["key"+string(rune(i))] = "value"
		}
		putConversionMap(m)
	}
}

func BenchmarkConversionMap_WithoutPool(b *testing.B) {
	for b.Loop() {
		m := make(map[string]string, conversionMapCap)
		for i := range 100 {
			m["key"+string(rune(i))] = "value"
		}
	}
}
