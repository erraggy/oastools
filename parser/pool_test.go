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
