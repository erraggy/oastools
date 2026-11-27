package parser

import (
	"bytes"
	"encoding/json"
	"sync"
)

// Marshal buffer size limits (corpus-validated)
const (
	marshalBufferInitialSize = 4096    // 4KB - covers most fields
	marshalBufferMaxSize     = 1 << 20 // 1MB - prevent memory leaks
)

// marshalBufferPool provides reusable buffers for marshalToJSON.
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

// marshalToJSON marshals a value to JSON using pooled buffers.
// This replaces json.Marshal calls to reduce allocations.
// Note: json.Encoder.Encode adds a trailing newline which we strip.
func marshalToJSON(v any) ([]byte, error) {
	buf := getMarshalBuffer()
	defer putMarshalBuffer(buf)

	enc := json.NewEncoder(buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// Strip trailing newline added by Encoder (json.Marshal doesn't add one)
	n := buf.Len()
	if n > 0 && buf.Bytes()[n-1] == '\n' {
		n--
	}
	result := make([]byte, n)
	copy(result, buf.Bytes())
	return result, nil
}
