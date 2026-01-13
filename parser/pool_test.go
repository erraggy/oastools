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

// DeepCopy work pool tests

func TestGetDeepCopyWork(t *testing.T) {
	s := getDeepCopyWork()
	if s == nil {
		t.Fatal("getDeepCopyWork returned nil")
	}
	if len(*s) != 0 {
		t.Errorf("slice should be empty, got len=%d", len(*s))
	}
	if cap(*s) < deepCopyWorkCap {
		t.Errorf("slice capacity should be at least %d, got %d", deepCopyWorkCap, cap(*s))
	}
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
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putDeepCopyWork(s2)
}

func TestPutDeepCopyWork_Oversized(t *testing.T) {
	// Create an oversized slice
	large := make([]any, 0, deepCopyWorkMaxCap+1)
	putDeepCopyWork(&large)

	// Get a new slice - it should have the standard capacity
	s := getDeepCopyWork()
	if cap(*s) > deepCopyWorkMaxCap {
		t.Errorf("pool returned oversized slice with cap=%d", cap(*s))
	}
	putDeepCopyWork(s)
}

func TestDeepCopyWorkPoolConstants(t *testing.T) {
	// Verify constants are reasonable
	if deepCopyWorkCap != 16 {
		t.Errorf("deepCopyWorkCap should be 16, got %d", deepCopyWorkCap)
	}
	if deepCopyWorkMaxCap != 256 {
		t.Errorf("deepCopyWorkMaxCap should be 256, got %d", deepCopyWorkMaxCap)
	}
	if deepCopyWorkCap >= deepCopyWorkMaxCap {
		t.Error("initial capacity should be less than max capacity")
	}
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
			if string(got) != tt.want {
				t.Errorf("marshalToJSON() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestMarshalToJSON_NoTrailingNewline(t *testing.T) {
	// json.Encoder adds a trailing newline, but marshalToJSON should strip it
	got, err := marshalToJSON(map[string]string{"test": "value"})
	if err != nil {
		t.Fatalf("marshalToJSON() error = %v", err)
	}
	if len(got) > 0 && got[len(got)-1] == '\n' {
		t.Error("marshalToJSON() output should not have trailing newline")
	}
	// Compare with json.Marshal behavior
	want := `{"test":"value"}`
	if string(got) != want {
		t.Errorf("marshalToJSON() = %q, want %q", string(got), want)
	}
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
