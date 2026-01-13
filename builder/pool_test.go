package builder

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestSchemaMapPool_Clear(t *testing.T) {
	m := getSchemaMap()
	m["Pet"] = &parser.Schema{Type: "object"}
	putSchemaMap(m)

	m2 := getSchemaMap()
	if len(m2) != 0 {
		t.Errorf("expected empty map, got len=%d", len(m2))
	}
	putSchemaMap(m2)
}

func TestPathMapPool_Clear(t *testing.T) {
	m := getPathMap()
	m["/pets"] = &parser.PathItem{}
	putPathMap(m)

	m2 := getPathMap()
	if len(m2) != 0 {
		t.Errorf("expected empty map, got len=%d", len(m2))
	}
	putPathMap(m2)
}

func TestSchemaMapPool_NilSafe(t *testing.T) {
	// Should not panic when putting nil
	putSchemaMap(nil)
}

func TestPathMapPool_NilSafe(t *testing.T) {
	// Should not panic when putting nil
	putPathMap(nil)
}

func TestSchemaMapPool_OversizedNotReturned(t *testing.T) {
	// Create an oversized map
	m := make(map[string]*parser.Schema, 200)
	for i := range 150 {
		m[string(rune('A'+i%26))+string(rune('0'+i/26))] = &parser.Schema{}
	}

	// Put should not return it to pool (len > 128)
	putSchemaMap(m)

	// Get should return a fresh map, not the oversized one
	m2 := getSchemaMap()
	if len(m2) != 0 {
		t.Errorf("expected empty map from pool, got len=%d", len(m2))
	}
	putSchemaMap(m2)
}

func TestPathMapPool_OversizedNotReturned(t *testing.T) {
	// Create an oversized map
	m := make(map[string]*parser.PathItem, 100)
	for i := range 70 {
		m["/path"+string(rune('0'+i))] = &parser.PathItem{}
	}

	// Put should not return it to pool (len > 64)
	putPathMap(m)

	// Get should return a fresh map, not the oversized one
	m2 := getPathMap()
	if len(m2) != 0 {
		t.Errorf("expected empty map from pool, got len=%d", len(m2))
	}
	putPathMap(m2)
}

func BenchmarkSchemaMap_WithPool(b *testing.B) {
	for b.Loop() {
		m := getSchemaMap()
		m["Pet"] = &parser.Schema{}
		m["User"] = &parser.Schema{}
		putSchemaMap(m)
	}
}

func BenchmarkSchemaMap_WithoutPool(b *testing.B) {
	for b.Loop() {
		m := make(map[string]*parser.Schema, schemaMapCap)
		m["Pet"] = &parser.Schema{}
		m["User"] = &parser.Schema{}
		_ = m
	}
}

func BenchmarkPathMap_WithPool(b *testing.B) {
	for b.Loop() {
		m := getPathMap()
		m["/pets"] = &parser.PathItem{}
		m["/users"] = &parser.PathItem{}
		putPathMap(m)
	}
}

func BenchmarkPathMap_WithoutPool(b *testing.B) {
	for b.Loop() {
		m := make(map[string]*parser.PathItem, pathMapCap)
		m["/pets"] = &parser.PathItem{}
		m["/users"] = &parser.PathItem{}
		_ = m
	}
}
