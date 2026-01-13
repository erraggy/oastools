package parser

import (
	"bytes"
	"encoding/json"
	"sync"
)

// Pool size limits (corpus-validated)
const (
	marshalBufferInitialSize = 4096    // 4KB - covers most fields
	marshalBufferMaxSize     = 1 << 20 // 1MB - prevent memory leaks
)

// Slice pool capacities (corpus-validated)
const (
	parameterSliceCap = 4 // p75=2, p90=8
	serverSliceCap    = 2 // max=2 across corpus
	stringSliceCap    = 2 // p99=1 (tags)
)

// DeepCopy pool capacities (corpus: P99=3, max=9 schema depth)
const (
	deepCopyWorkCap    = 16
	deepCopyWorkMaxCap = 256
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

// parameterSlicePool provides reusable slices for Parameter pointers.
var parameterSlicePool = sync.Pool{
	New: func() any {
		s := make([]*Parameter, 0, parameterSliceCap)
		return &s
	},
}

// getParameterSlice retrieves a Parameter slice from the pool and resets it.
func getParameterSlice() *[]*Parameter {
	s := parameterSlicePool.Get().(*[]*Parameter)
	*s = (*s)[:0]
	return s
}

// putParameterSlice returns a Parameter slice to the pool if not oversized.
func putParameterSlice(s *[]*Parameter) {
	if s == nil || cap(*s) > 64 {
		return
	}
	parameterSlicePool.Put(s)
}

// serverSlicePool provides reusable slices for Server pointers.
var serverSlicePool = sync.Pool{
	New: func() any {
		s := make([]*Server, 0, serverSliceCap)
		return &s
	},
}

// getServerSlice retrieves a Server slice from the pool and resets it.
func getServerSlice() *[]*Server {
	s := serverSlicePool.Get().(*[]*Server)
	*s = (*s)[:0]
	return s
}

// putServerSlice returns a Server slice to the pool if not oversized.
func putServerSlice(s *[]*Server) {
	if s == nil || cap(*s) > 16 {
		return
	}
	serverSlicePool.Put(s)
}

// stringSlicePool provides reusable slices for strings (e.g., tags).
var stringSlicePool = sync.Pool{
	New: func() any {
		s := make([]string, 0, stringSliceCap)
		return &s
	},
}

// getStringSlice retrieves a string slice from the pool and resets it.
func getStringSlice() *[]string {
	s := stringSlicePool.Get().(*[]string)
	*s = (*s)[:0]
	return s
}

// putStringSlice returns a string slice to the pool if not oversized.
func putStringSlice(s *[]string) {
	if s == nil || cap(*s) > 32 {
		return
	}
	stringSlicePool.Put(s)
}

// deepCopyWorkPool provides reusable work slices for DeepCopy traversal.
var deepCopyWorkPool = sync.Pool{
	New: func() any {
		s := make([]any, 0, deepCopyWorkCap)
		return &s
	},
}

// getDeepCopyWork retrieves a work slice from the pool and resets it.
func getDeepCopyWork() *[]any {
	s := deepCopyWorkPool.Get().(*[]any)
	*s = (*s)[:0]
	return s
}

// putDeepCopyWork returns a work slice to the pool if not oversized.
func putDeepCopyWork(s *[]any) {
	if s == nil || cap(*s) > deepCopyWorkMaxCap {
		return
	}
	deepCopyWorkPool.Put(s)
}
