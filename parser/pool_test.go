package parser

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMarshalBuffer(t *testing.T) {
	buf := getMarshalBuffer()
	require.NotNil(t, buf)
	assert.Equal(t, 0, buf.Len(), "buffer should be empty")
	assert.GreaterOrEqual(t, buf.Cap(), marshalBufferInitialSize, "buffer capacity should be at least %d", marshalBufferInitialSize)
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
	assert.LessOrEqual(t, buf.Cap(), marshalBufferMaxSize, "pool returned oversized buffer")
	putMarshalBuffer(buf)
}

func TestMarshalBufferPool_Reuse(t *testing.T) {
	// Get a buffer and write to it
	buf1 := getMarshalBuffer()
	buf1.WriteString("test data")
	putMarshalBuffer(buf1)

	// Get another buffer - it should be reset
	buf2 := getMarshalBuffer()
	assert.Equal(t, 0, buf2.Len(), "reused buffer should be reset")
	putMarshalBuffer(buf2)
}

func TestMarshalBufferPool_Constants(t *testing.T) {
	// Verify constants are reasonable
	assert.Equal(t, 4096, marshalBufferInitialSize)
	assert.Equal(t, 1<<20, marshalBufferMaxSize)
	assert.Less(t, marshalBufferInitialSize, marshalBufferMaxSize, "initial size should be less than max size")
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

// marshalToJSON tests

func TestMarshalToJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{
			name:  "simple object",
			input: map[string]string{"key": "value"},
			want:  `{"key":"value"}`,
		},
		{
			name:  "nested object",
			input: map[string]any{"outer": map[string]string{"inner": "value"}},
			want:  `{"outer":{"inner":"value"}}`,
		},
		{
			name:  "array",
			input: []int{1, 2, 3},
			want:  `[1,2,3]`,
		},
		{
			name:  "string",
			input: "hello",
			want:  `"hello"`,
		},
		{
			name:  "number",
			input: 42,
			want:  `42`,
		},
		{
			name:  "boolean",
			input: true,
			want:  `true`,
		},
		{
			name:  "null",
			input: nil,
			want:  `null`,
		},
		{
			name:    "unmarshalable channel",
			input:   make(chan int),
			wantErr: true,
		},
		{
			name:    "unmarshalable function",
			input:   func() {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalToJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("marshalToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestMarshalToJSON_NoTrailingNewline(t *testing.T) {
	// json.Encoder adds a trailing newline, but marshalToJSON should strip it
	got, err := marshalToJSON(map[string]string{"test": "value"})
	require.NoError(t, err)
	if len(got) > 0 {
		assert.NotEqual(t, byte('\n'), got[len(got)-1], "marshalToJSON() output should not have trailing newline")
	}
	// Compare with json.Marshal behavior
	assert.Equal(t, `{"test":"value"}`, string(got))
}

func BenchmarkMarshalToJSON(b *testing.B) {
	input := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]string{
			"title":   "Test API",
			"version": "1.0.0",
		},
	}
	for b.Loop() {
		_, err := marshalToJSON(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
