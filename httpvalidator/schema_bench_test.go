package httpvalidator

import "testing"

func BenchmarkMatchPattern(b *testing.B) {
	sv := NewSchemaValidator()
	patterns := []string{
		`^[a-zA-Z]+$`, `^\d{3}-\d{2}-\d{4}$`, `^[a-f0-9]+$`,
		`^\w+@\w+\.\w+$`, `^https?://`, `^\d+\.\d+\.\d+$`,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pattern := patterns[i%len(patterns)]
		_, _ = sv.matchPattern(pattern, "test-value-123")
	}
}
