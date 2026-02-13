package generator

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateBufferPool_TieredSizes(t *testing.T) {
	small := getTemplateBuffer(5)
	assert.GreaterOrEqual(t, small.Cap(), smallBufferSize)
	putTemplateBuffer(small, 5)

	medium := getTemplateBuffer(25)
	assert.GreaterOrEqual(t, medium.Cap(), mediumBufferSize)
	putTemplateBuffer(medium, 25)

	large := getTemplateBuffer(100)
	assert.GreaterOrEqual(t, large.Cap(), largeBufferSize)
	putTemplateBuffer(large, 100)
}

func BenchmarkTemplateBuffer_WithPool(b *testing.B) {
	for b.Loop() {
		buf := getTemplateBuffer(25)
		buf.WriteString("package main\n\nfunc main() {}\n")
		putTemplateBuffer(buf, 25)
	}
}

func BenchmarkTemplateBuffer_WithoutPool(b *testing.B) {
	for b.Loop() {
		buf := bytes.NewBuffer(make([]byte, 0, mediumBufferSize))
		buf.WriteString("package main\n\nfunc main() {}\n")
		// No return to pool - buffer is discarded
	}
}
