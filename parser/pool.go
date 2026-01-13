package parser

import (
	"bytes"
	"sync"
)

// Pool size limits (corpus-validated)
const (
	marshalBufferInitialSize = 4096    // 4KB - covers most fields
	marshalBufferMaxSize     = 1 << 20 // 1MB - prevent memory leaks
)

var marshalBufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, marshalBufferInitialSize))
	},
}

// getMarshalBuffer retrieves a buffer from the pool and resets it.
func getMarshalBuffer() *bytes.Buffer {
	buf := marshalBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// putMarshalBuffer returns a buffer to the pool if not oversized.
func putMarshalBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	if buf.Cap() > marshalBufferMaxSize {
		return // Let GC collect oversized buffers
	}
	marshalBufferPool.Put(buf)
}
