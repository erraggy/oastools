package generator

import (
	"bytes"
	"sync"
)

// Tiered buffer sizes (corpus-validated)
const (
	smallBufferSize  = 8 * 1024  // 8KB for <10 ops
	mediumBufferSize = 32 * 1024 // 32KB for 10-50 ops
	largeBufferSize  = 64 * 1024 // 64KB for 50+ ops
)

var smallBufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, smallBufferSize))
	},
}

var mediumBufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, mediumBufferSize))
	},
}

var largeBufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, largeBufferSize))
	},
}

// getTemplateBuffer returns a buffer sized for the operation count.
func getTemplateBuffer(opCount int) *bytes.Buffer {
	var buf *bytes.Buffer
	switch {
	case opCount < 10:
		buf = smallBufferPool.Get().(*bytes.Buffer)
	case opCount < 50:
		buf = mediumBufferPool.Get().(*bytes.Buffer)
	default:
		buf = largeBufferPool.Get().(*bytes.Buffer)
	}
	buf.Reset()
	return buf
}

// putTemplateBuffer returns a buffer to the appropriate pool.
func putTemplateBuffer(buf *bytes.Buffer, opCount int) {
	if buf == nil {
		return
	}
	// Don't pool oversized buffers
	if buf.Cap() > 1<<20 {
		return
	}
	switch {
	case opCount < 10:
		smallBufferPool.Put(buf)
	case opCount < 50:
		mediumBufferPool.Put(buf)
	default:
		largeBufferPool.Put(buf)
	}
}
