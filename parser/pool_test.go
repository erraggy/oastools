package parser

import (
	"bytes"
	"testing"
)

func TestGetMarshalBuffer(t *testing.T) {
	buf := getMarshalBuffer()
	if buf == nil {
		t.Fatal("getMarshalBuffer returned nil")
	}
	if buf.Len() != 0 {
		t.Errorf("buffer should be empty, got len=%d", buf.Len())
	}
	if buf.Cap() < marshalBufferInitialSize {
		t.Errorf("buffer capacity should be at least %d, got %d", marshalBufferInitialSize, buf.Cap())
	}
	putMarshalBuffer(buf)
}

func TestPutMarshalBuffer_Nil(t *testing.T) {
	// Should not panic
	putMarshalBuffer(nil)
}

func TestPutMarshalBuffer_Oversized(t *testing.T) {
	// Create an oversized buffer
	oversized := bytes.NewBuffer(make([]byte, 0, marshalBufferMaxSize+1))
	oversized.Write(make([]byte, marshalBufferMaxSize+1))

	// This should not panic and should not return the buffer to the pool
	putMarshalBuffer(oversized)

	// Get a new buffer - it should have initial size, not the oversized capacity
	buf := getMarshalBuffer()
	if buf.Cap() > marshalBufferMaxSize {
		t.Errorf("pool returned oversized buffer with cap=%d", buf.Cap())
	}
	putMarshalBuffer(buf)
}

func TestMarshalBufferPool_Reuse(t *testing.T) {
	// Get a buffer and write to it
	buf1 := getMarshalBuffer()
	buf1.WriteString("test data")
	putMarshalBuffer(buf1)

	// Get another buffer - it should be reset
	buf2 := getMarshalBuffer()
	if buf2.Len() != 0 {
		t.Errorf("reused buffer should be reset, got len=%d", buf2.Len())
	}
	putMarshalBuffer(buf2)
}

func TestMarshalBufferPool_Constants(t *testing.T) {
	// Verify constants are reasonable
	if marshalBufferInitialSize != 4096 {
		t.Errorf("marshalBufferInitialSize should be 4096, got %d", marshalBufferInitialSize)
	}
	if marshalBufferMaxSize != 1<<20 {
		t.Errorf("marshalBufferMaxSize should be 1MB, got %d", marshalBufferMaxSize)
	}
	if marshalBufferInitialSize >= marshalBufferMaxSize {
		t.Error("initial size should be less than max size")
	}
}

func BenchmarkMarshalBufferPool(b *testing.B) {
	for b.Loop() {
		buf := getMarshalBuffer()
		buf.WriteString(`{"openapi":"3.0.0","info":{"title":"Test","version":"1.0"}}`)
		putMarshalBuffer(buf)
	}
}

func BenchmarkMarshalBufferNoPool(b *testing.B) {
	for b.Loop() {
		buf := bytes.NewBuffer(make([]byte, 0, marshalBufferInitialSize))
		buf.WriteString(`{"openapi":"3.0.0","info":{"title":"Test","version":"1.0"}}`)
	}
}

func BenchmarkMarshalBufferPool_LargePayload(b *testing.B) {
	// Simulate a larger payload typical of OAS documents
	payload := make([]byte, 64*1024) // 64KB
	for i := range payload {
		payload[i] = 'x'
	}

	for b.Loop() {
		buf := getMarshalBuffer()
		buf.Write(payload)
		putMarshalBuffer(buf)
	}
}

// Parameter slice pool tests

func TestGetParameterSlice(t *testing.T) {
	s := getParameterSlice()
	if s == nil {
		t.Fatal("getParameterSlice returned nil")
	}
	if len(*s) != 0 {
		t.Errorf("slice should be empty, got len=%d", len(*s))
	}
	if cap(*s) < parameterSliceCap {
		t.Errorf("slice capacity should be at least %d, got %d", parameterSliceCap, cap(*s))
	}
	putParameterSlice(s)
}

func TestPutParameterSlice_Nil(t *testing.T) {
	// Should not panic
	putParameterSlice(nil)
}

func TestParameterSlicePool_Reset(t *testing.T) {
	s := getParameterSlice()
	*s = append(*s, &Parameter{Name: "test"})
	putParameterSlice(s)

	s2 := getParameterSlice()
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putParameterSlice(s2)
}

func TestPutParameterSlice_Oversized(t *testing.T) {
	// Create an oversized slice
	large := make([]*Parameter, 0, 100)
	putParameterSlice(&large)

	// Get a new slice - it should have the standard capacity
	s := getParameterSlice()
	if cap(*s) > 64 {
		t.Errorf("pool returned oversized slice with cap=%d", cap(*s))
	}
	putParameterSlice(s)
}

func BenchmarkParameterSlice_WithPool(b *testing.B) {
	for b.Loop() {
		s := getParameterSlice()
		*s = append(*s, &Parameter{Name: "id"})
		*s = append(*s, &Parameter{Name: "limit"})
		putParameterSlice(s)
	}
}

func BenchmarkParameterSlice_WithoutPool(b *testing.B) {
	for b.Loop() {
		s := make([]*Parameter, 0, parameterSliceCap)
		s = append(s, &Parameter{Name: "id"})
		s = append(s, &Parameter{Name: "limit"})
		_ = s
	}
}

// Server slice pool tests

func TestGetServerSlice(t *testing.T) {
	s := getServerSlice()
	if s == nil {
		t.Fatal("getServerSlice returned nil")
	}
	if len(*s) != 0 {
		t.Errorf("slice should be empty, got len=%d", len(*s))
	}
	if cap(*s) < serverSliceCap {
		t.Errorf("slice capacity should be at least %d, got %d", serverSliceCap, cap(*s))
	}
	putServerSlice(s)
}

func TestPutServerSlice_Nil(t *testing.T) {
	// Should not panic
	putServerSlice(nil)
}

func TestServerSlicePool_Reset(t *testing.T) {
	s := getServerSlice()
	*s = append(*s, &Server{URL: "https://api.example.com"})
	putServerSlice(s)

	s2 := getServerSlice()
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putServerSlice(s2)
}

func TestPutServerSlice_Oversized(t *testing.T) {
	// Create an oversized slice
	large := make([]*Server, 0, 32)
	putServerSlice(&large)

	// Get a new slice - it should have the standard capacity
	s := getServerSlice()
	if cap(*s) > 16 {
		t.Errorf("pool returned oversized slice with cap=%d", cap(*s))
	}
	putServerSlice(s)
}

func BenchmarkServerSlice_WithPool(b *testing.B) {
	for b.Loop() {
		s := getServerSlice()
		*s = append(*s, &Server{URL: "https://api.example.com"})
		*s = append(*s, &Server{URL: "https://staging.example.com"})
		putServerSlice(s)
	}
}

func BenchmarkServerSlice_WithoutPool(b *testing.B) {
	for b.Loop() {
		s := make([]*Server, 0, serverSliceCap)
		s = append(s, &Server{URL: "https://api.example.com"})
		s = append(s, &Server{URL: "https://staging.example.com"})
		_ = s
	}
}

// String slice pool tests

func TestGetStringSlice(t *testing.T) {
	s := getStringSlice()
	if s == nil {
		t.Fatal("getStringSlice returned nil")
	}
	if len(*s) != 0 {
		t.Errorf("slice should be empty, got len=%d", len(*s))
	}
	if cap(*s) < stringSliceCap {
		t.Errorf("slice capacity should be at least %d, got %d", stringSliceCap, cap(*s))
	}
	putStringSlice(s)
}

func TestPutStringSlice_Nil(t *testing.T) {
	// Should not panic
	putStringSlice(nil)
}

func TestStringSlicePool_Reset(t *testing.T) {
	s := getStringSlice()
	*s = append(*s, "users")
	putStringSlice(s)

	s2 := getStringSlice()
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putStringSlice(s2)
}

func TestPutStringSlice_Oversized(t *testing.T) {
	// Create an oversized slice
	large := make([]string, 0, 64)
	putStringSlice(&large)

	// Get a new slice - it should have the standard capacity
	s := getStringSlice()
	if cap(*s) > 32 {
		t.Errorf("pool returned oversized slice with cap=%d", cap(*s))
	}
	putStringSlice(s)
}

func BenchmarkStringSlice_WithPool(b *testing.B) {
	for b.Loop() {
		s := getStringSlice()
		*s = append(*s, "users")
		*s = append(*s, "orders")
		putStringSlice(s)
	}
}

func BenchmarkStringSlice_WithoutPool(b *testing.B) {
	for b.Loop() {
		s := make([]string, 0, stringSliceCap)
		s = append(s, "users")
		s = append(s, "orders")
		_ = s
	}
}

// Slice pool constants tests

func TestSlicePoolConstants(t *testing.T) {
	// Verify constants are reasonable
	if parameterSliceCap != 4 {
		t.Errorf("parameterSliceCap should be 4, got %d", parameterSliceCap)
	}
	if serverSliceCap != 2 {
		t.Errorf("serverSliceCap should be 2, got %d", serverSliceCap)
	}
	if stringSliceCap != 2 {
		t.Errorf("stringSliceCap should be 2, got %d", stringSliceCap)
	}
}
