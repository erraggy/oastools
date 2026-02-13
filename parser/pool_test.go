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

// Parameter slice pool tests

func TestGetParameterSlice(t *testing.T) {
	s := getParameterSlice()
	require.NotNil(t, s)
	assert.Empty(t, *s)
	assert.GreaterOrEqual(t, cap(*s), parameterSliceCap, "slice capacity should be at least %d", parameterSliceCap)
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
	assert.Empty(t, *s2)
	putParameterSlice(s2)
}

func TestPutParameterSlice_Oversized(t *testing.T) {
	// Create an oversized slice
	large := make([]*Parameter, 0, 100)
	putParameterSlice(&large)

	// Get a new slice - it should have the standard capacity
	s := getParameterSlice()
	assert.LessOrEqual(t, cap(*s), 64, "pool returned oversized slice")
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
	require.NotNil(t, s)
	assert.Empty(t, *s)
	assert.GreaterOrEqual(t, cap(*s), serverSliceCap, "slice capacity should be at least %d", serverSliceCap)
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
	assert.Empty(t, *s2)
	putServerSlice(s2)
}

func TestPutServerSlice_Oversized(t *testing.T) {
	// Create an oversized slice
	large := make([]*Server, 0, 32)
	putServerSlice(&large)

	// Get a new slice - it should have the standard capacity
	s := getServerSlice()
	assert.LessOrEqual(t, cap(*s), 16, "pool returned oversized slice")
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
	require.NotNil(t, s)
	assert.Empty(t, *s)
	assert.GreaterOrEqual(t, cap(*s), stringSliceCap, "slice capacity should be at least %d", stringSliceCap)
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
	assert.Empty(t, *s2)
	putStringSlice(s2)
}

func TestPutStringSlice_Oversized(t *testing.T) {
	// Create an oversized slice
	large := make([]string, 0, 64)
	putStringSlice(&large)

	// Get a new slice - it should have the standard capacity
	s := getStringSlice()
	assert.LessOrEqual(t, cap(*s), 32, "pool returned oversized slice")
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
	assert.Equal(t, 4, parameterSliceCap)
	assert.Equal(t, 2, serverSliceCap)
	assert.Equal(t, 2, stringSliceCap)
}

// DeepCopy work pool tests

func TestGetDeepCopyWork(t *testing.T) {
	s := getDeepCopyWork()
	require.NotNil(t, s)
	assert.Empty(t, *s)
	assert.GreaterOrEqual(t, cap(*s), deepCopyWorkCap, "slice capacity should be at least %d", deepCopyWorkCap)
	putDeepCopyWork(s)
}

func TestPutDeepCopyWork_Nil(t *testing.T) {
	// Should not panic
	putDeepCopyWork(nil)
}

func TestDeepCopyWorkPool_Reset(t *testing.T) {
	s := getDeepCopyWork()
	*s = append(*s, "item1", "item2")
	putDeepCopyWork(s)

	s2 := getDeepCopyWork()
	assert.Empty(t, *s2)
	putDeepCopyWork(s2)
}

func TestPutDeepCopyWork_Oversized(t *testing.T) {
	// Create an oversized slice
	large := make([]any, 0, deepCopyWorkMaxCap+1)
	putDeepCopyWork(&large)

	// Get a new slice - it should have the standard capacity
	s := getDeepCopyWork()
	assert.LessOrEqual(t, cap(*s), deepCopyWorkMaxCap, "pool returned oversized slice")
	putDeepCopyWork(s)
}

func TestDeepCopyWorkPoolConstants(t *testing.T) {
	// Verify constants are reasonable
	assert.Equal(t, 16, deepCopyWorkCap)
	assert.Equal(t, 256, deepCopyWorkMaxCap)
	assert.Less(t, deepCopyWorkCap, deepCopyWorkMaxCap, "initial capacity should be less than max capacity")
}

func BenchmarkDeepCopyWork_WithPool(b *testing.B) {
	for b.Loop() {
		s := getDeepCopyWork()
		for i := range 10 {
			*s = append(*s, i)
		}
		putDeepCopyWork(s)
	}
}

func BenchmarkDeepCopyWork_WithoutPool(b *testing.B) {
	for b.Loop() {
		s := make([]any, 0, deepCopyWorkCap)
		for i := range 10 {
			s = append(s, i)
		}
		_ = s
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
