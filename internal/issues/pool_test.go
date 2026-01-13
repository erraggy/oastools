package issues

import "testing"

func TestFormatPath(t *testing.T) {
	tests := []struct {
		segments []string
		want     string
	}{
		{nil, ""},
		{[]string{"paths"}, "paths"},
		{[]string{"paths", "/users", "get"}, "paths./users.get"},
	}

	for _, tt := range tests {
		got := FormatPath(tt.segments...)
		if got != tt.want {
			t.Errorf("FormatPath(%v) = %q, want %q", tt.segments, got, tt.want)
		}
	}
}

func BenchmarkFormatPath_WithPool(b *testing.B) {
	segments := []string{"paths", "/users/{id}", "get", "parameters", "0"}
	for b.Loop() {
		_ = FormatPath(segments...)
	}
}

func BenchmarkFormatPath_WithoutPool(b *testing.B) {
	segments := []string{"paths", "/users/{id}", "get", "parameters", "0"}
	for b.Loop() {
		result := ""
		for i, s := range segments {
			if i > 0 {
				result += "."
			}
			result += s
		}
		_ = result
	}
}
