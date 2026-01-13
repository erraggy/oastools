package generator

import "testing"

func TestTemplateBufferPool_TieredSizes(t *testing.T) {
	small := getTemplateBuffer(5)
	if small.Cap() < smallBufferSize {
		t.Errorf("small buffer cap %d < %d", small.Cap(), smallBufferSize)
	}
	putTemplateBuffer(small, 5)

	medium := getTemplateBuffer(25)
	if medium.Cap() < mediumBufferSize {
		t.Errorf("medium buffer cap %d < %d", medium.Cap(), mediumBufferSize)
	}
	putTemplateBuffer(medium, 25)

	large := getTemplateBuffer(100)
	if large.Cap() < largeBufferSize {
		t.Errorf("large buffer cap %d < %d", large.Cap(), largeBufferSize)
	}
	putTemplateBuffer(large, 100)
}

func BenchmarkTemplateBuffer_WithPool(b *testing.B) {
	for b.Loop() {
		buf := getTemplateBuffer(25)
		buf.WriteString("package main\n\nfunc main() {}\n")
		putTemplateBuffer(buf, 25)
	}
}
