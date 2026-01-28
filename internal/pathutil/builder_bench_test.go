// internal/pathutil/builder_bench_test.go
package pathutil

import (
	"fmt"
	"testing"
)

func BenchmarkPathBuilder_DeepPath(b *testing.B) {
	b.Run("PathBuilder", func(b *testing.B) {
		for b.Loop() {
			p := Get()
			p.Push("components")
			p.Push("schemas")
			p.Push("Pet")
			p.Push("properties")
			p.Push("tags")
			p.Push("items")
			p.Push("properties")
			p.Push("name")
			_ = p.String()
			Put(p)
		}
	})

	b.Run("FmtSprintf", func(b *testing.B) {
		for b.Loop() {
			path := "components"
			path = fmt.Sprintf("%s.%s", path, "schemas")
			path = fmt.Sprintf("%s.%s", path, "Pet")
			path = fmt.Sprintf("%s.%s", path, "properties")
			path = fmt.Sprintf("%s.%s", path, "tags")
			path = fmt.Sprintf("%s.%s", path, "items")
			path = fmt.Sprintf("%s.%s", path, "properties")
			path = fmt.Sprintf("%s.%s", path, "name")
			_ = path
		}
	})
}

func BenchmarkPathBuilder_NoStringCall(b *testing.B) {
	b.Run("PathBuilder_NoString", func(b *testing.B) {
		for b.Loop() {
			p := Get()
			for j := 0; j < 8; j++ {
				p.Push("segment")
			}
			for j := 0; j < 8; j++ {
				p.Pop()
			}
			Put(p)
		}
	})

	b.Run("FmtSprintf_Equivalent", func(b *testing.B) {
		for b.Loop() {
			path := ""
			for j := 0; j < 8; j++ {
				if path == "" {
					path = "segment"
				} else {
					path = fmt.Sprintf("%s.%s", path, "segment")
				}
			}
			_ = path
		}
	})
}

func BenchmarkRefBuilders(b *testing.B) {
	b.Run("SchemaRef", func(b *testing.B) {
		for b.Loop() {
			_ = SchemaRef("MySchema")
		}
	})

	b.Run("FmtSprintf", func(b *testing.B) {
		for b.Loop() {
			_ = fmt.Sprintf("#/components/schemas/%s", "MySchema")
		}
	})
}

func BenchmarkPathBuilder_WithIndex(b *testing.B) {
	b.Run("PathBuilder", func(b *testing.B) {
		for b.Loop() {
			p := Get()
			p.Push("allOf")
			p.PushIndex(0)
			p.Push("properties")
			p.Push("name")
			_ = p.String()
			Put(p)
		}
	})

	b.Run("FmtSprintf", func(b *testing.B) {
		for b.Loop() {
			path := "allOf"
			path = fmt.Sprintf("%s[%d]", path, 0)
			path = fmt.Sprintf("%s.%s", path, "properties")
			path = fmt.Sprintf("%s.%s", path, "name")
			_ = path
		}
	})
}
