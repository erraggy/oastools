# sync.Pool Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement 12 sync.Pool optimizations across oastools to reduce GC pressure and improve performance by 15-30%.

**Architecture:** Each package gets a `pool.go` file with package-private pools, get/put wrappers with reset-on-get and size guards. All capacities are corpus-validated.

**Tech Stack:** Go 1.24+, sync.Pool, bytes.Buffer, strings.Builder

---

## Phase 1: High-Priority Low-Risk Pools (1-3)

### Task 1: Marshal Buffer Pool

**Files:**
- Create: `parser/pool.go`
- Create: `parser/pool_test.go`
- Modify: `parser/ordered_marshal.go:32-43` (MarshalOrderedJSON)
- Modify: `parser/ordered_marshal.go:50-61` (MarshalOrderedJSONIndent)

**Step 1.1: Write the pool implementation**

Create `parser/pool.go`:

```go
package parser

import (
	"bytes"
	"sync"
)

// Pool size limits (corpus-validated)
const (
	marshalBufferInitialSize = 4096      // 4KB - covers most fields
	marshalBufferMaxSize     = 1 << 20   // 1MB - prevent memory leaks
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
```

**Step 1.2: Write the benchmark test**

Create `parser/pool_test.go`:

```go
package parser

import (
	"bytes"
	"testing"
)

func BenchmarkMarshalBuffer_WithPool(b *testing.B) {
	for b.Loop() {
		buf := getMarshalBuffer()
		buf.WriteString("test data for benchmarking pool performance")
		putMarshalBuffer(buf)
	}
}

func BenchmarkMarshalBuffer_WithoutPool(b *testing.B) {
	for b.Loop() {
		buf := bytes.NewBuffer(make([]byte, 0, marshalBufferInitialSize))
		buf.WriteString("test data for benchmarking pool performance")
		// No return to pool - simulates current behavior
	}
}

func TestMarshalBufferPool_Reset(t *testing.T) {
	buf := getMarshalBuffer()
	buf.WriteString("some content")
	putMarshalBuffer(buf)

	buf2 := getMarshalBuffer()
	if buf2.Len() != 0 {
		t.Errorf("expected empty buffer, got len=%d", buf2.Len())
	}
	putMarshalBuffer(buf2)
}

func TestMarshalBufferPool_OversizedNotPooled(t *testing.T) {
	buf := getMarshalBuffer()
	// Write more than maxSize
	large := make([]byte, marshalBufferMaxSize+1)
	buf.Write(large)

	// Should not panic, just not pool it
	putMarshalBuffer(buf)
}
```

**Step 1.3: Run tests to verify they pass**

```bash
go test -v ./parser -run TestMarshalBuffer
go test -bench=BenchmarkMarshalBuffer ./parser -benchmem
```

Expected: Tests pass, benchmarks show allocation difference.

**Step 1.4: Integrate pool into ordered_marshal.go**

Modify `parser/ordered_marshal.go` - update `MarshalOrderedJSON`:

```go
// MarshalOrderedJSON marshals the parsed document to JSON with fields
// in the same order as the original source document.
func (pr *ParseResult) MarshalOrderedJSON() ([]byte, error) {
	if pr.sourceNode == nil {
		return json.Marshal(pr.Document)
	}

	buf := getMarshalBuffer()
	if err := marshalNodeAsJSON(buf, pr.sourceNode, pr.Data); err != nil {
		putMarshalBuffer(buf)
		return nil, err
	}
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	putMarshalBuffer(buf)
	return result, nil
}
```

Update `MarshalOrderedJSONIndent`:

```go
func (pr *ParseResult) MarshalOrderedJSONIndent(prefix, indent string) ([]byte, error) {
	data, err := pr.MarshalOrderedJSON()
	if err != nil {
		return nil, err
	}

	buf := getMarshalBuffer()
	if err := json.Indent(buf, data, prefix, indent); err != nil {
		putMarshalBuffer(buf)
		return nil, err
	}
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	putMarshalBuffer(buf)
	return result, nil
}
```

**Step 1.5: Run full test suite and benchmarks**

```bash
go test ./parser -race
go test -bench=BenchmarkMarshalJSON ./parser -benchmem
```

Expected: All tests pass, race detector clean.

**Step 1.6: Commit**

```bash
git add parser/pool.go parser/pool_test.go parser/ordered_marshal.go
git commit -m "feat(parser): add sync.Pool for marshal buffers

Reduces allocations by pooling bytes.Buffer instances used during
JSON/YAML marshaling. Initial capacity 4KB, max pooled 1MB.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 2: HTTP Validator Context Pool

**Files:**
- Create: `httpvalidator/pool.go`
- Create: `httpvalidator/pool_test.go`
- Modify: `httpvalidator/result.go` (add reset method)
- Modify: `httpvalidator/request.go` (use pool in ValidateRequest)

**Step 2.1: Read result.go to understand current structures**

```bash
# Agent should read httpvalidator/result.go first
```

**Step 2.2: Write pool implementation**

Create `httpvalidator/pool.go`:

```go
package httpvalidator

import "sync"

// Pool capacities (corpus-validated)
const (
	requestResultErrorsCap   = 8
	requestResultWarningsCap = 4
)

var requestResultPool = sync.Pool{
	New: func() any {
		return &RequestValidationResult{
			Errors:   make([]ValidationIssue, 0, requestResultErrorsCap),
			Warnings: make([]ValidationIssue, 0, requestResultWarningsCap),
			Params:   make(map[string]any),
		}
	},
}

// getRequestResult retrieves a result from the pool and resets it.
func getRequestResult() *RequestValidationResult {
	r := requestResultPool.Get().(*RequestValidationResult)
	r.reset()
	return r
}

// putRequestResult returns a result to the pool.
func putRequestResult(r *RequestValidationResult) {
	if r == nil {
		return
	}
	requestResultPool.Put(r)
}

var responseResultPool = sync.Pool{
	New: func() any {
		return &ResponseValidationResult{
			Errors:   make([]ValidationIssue, 0, requestResultErrorsCap),
			Warnings: make([]ValidationIssue, 0, requestResultWarningsCap),
		}
	},
}

func getResponseResult() *ResponseValidationResult {
	r := responseResultPool.Get().(*ResponseValidationResult)
	r.reset()
	return r
}

func putResponseResult(r *ResponseValidationResult) {
	if r == nil {
		return
	}
	responseResultPool.Put(r)
}
```

**Step 2.3: Add reset methods to result types**

Add to `httpvalidator/result.go`:

```go
// reset clears the result for reuse from pool.
func (r *RequestValidationResult) reset() {
	r.Valid = true
	r.MatchedPath = ""
	r.MatchedMethod = ""
	r.Errors = r.Errors[:0]
	r.Warnings = r.Warnings[:0]
	clear(r.Params)
}

// reset clears the result for reuse from pool.
func (r *ResponseValidationResult) reset() {
	r.Valid = true
	r.MatchedPath = ""
	r.MatchedMethod = ""
	r.StatusCode = 0
	r.ContentType = ""
	r.Errors = r.Errors[:0]
	r.Warnings = r.Warnings[:0]
}
```

**Step 2.4: Write tests**

Create `httpvalidator/pool_test.go`:

```go
package httpvalidator

import "testing"

func BenchmarkRequestResult_WithPool(b *testing.B) {
	for b.Loop() {
		r := getRequestResult()
		r.MatchedPath = "/users/{id}"
		r.addError("/path", "test error", SeverityError)
		putRequestResult(r)
	}
}

func BenchmarkRequestResult_WithoutPool(b *testing.B) {
	for b.Loop() {
		r := newRequestResult()
		r.MatchedPath = "/users/{id}"
		r.addError("/path", "test error", SeverityError)
	}
}

func TestRequestResultPool_Reset(t *testing.T) {
	r := getRequestResult()
	r.MatchedPath = "/test"
	r.addError("/path", "error", SeverityError)
	r.Params["key"] = "value"
	putRequestResult(r)

	r2 := getRequestResult()
	if r2.MatchedPath != "" {
		t.Error("expected empty MatchedPath after reset")
	}
	if len(r2.Errors) != 0 {
		t.Error("expected empty Errors after reset")
	}
	if len(r2.Params) != 0 {
		t.Error("expected empty Params after reset")
	}
	putRequestResult(r2)
}
```

**Step 2.5: Run tests**

```bash
go test -v ./httpvalidator -run TestRequestResultPool
go test -bench=BenchmarkRequestResult ./httpvalidator -benchmem -race
```

**Step 2.6: Update validator to use pool (optional - depends on API design)**

Note: Since ValidateRequest returns the result to callers, we cannot automatically return it to the pool. The pool is useful for internal temporary results. Consider adding `ValidateRequestPooled` or document that callers should call `Release()`.

**Step 2.7: Commit**

```bash
git add httpvalidator/pool.go httpvalidator/pool_test.go httpvalidator/result.go
git commit -m "feat(httpvalidator): add sync.Pool for validation results

Adds pooling infrastructure for RequestValidationResult and
ResponseValidationResult. Reduces allocations for high-throughput
validation scenarios.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 3: String Builder Pool

**Files:**
- Create: `internal/issues/pool.go`
- Create: `internal/issues/pool_test.go`
- Modify: `internal/issues/issue.go` (use pool for path formatting)

**Step 3.1: Read issue.go structure**

```bash
# Agent reads internal/issues/issue.go
```

**Step 3.2: Write pool implementation**

Create `internal/issues/pool.go`:

```go
package issues

import (
	"strings"
	"sync"
)

var stringBuilderPool = sync.Pool{
	New: func() any {
		return new(strings.Builder)
	},
}

// getStringBuilder retrieves a builder from the pool and resets it.
func getStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// putStringBuilder returns a builder to the pool.
func putStringBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}
	stringBuilderPool.Put(sb)
}

// FormatPath efficiently formats a JSON path from segments.
func FormatPath(segments ...string) string {
	if len(segments) == 0 {
		return ""
	}
	if len(segments) == 1 {
		return segments[0]
	}

	sb := getStringBuilder()
	for i, seg := range segments {
		if i > 0 {
			sb.WriteByte('.')
		}
		sb.WriteString(seg)
	}
	result := sb.String()
	putStringBuilder(sb)
	return result
}
```

**Step 3.3: Write tests**

Create `internal/issues/pool_test.go`:

```go
package issues

import "testing"

func TestFormatPath(t *testing.T) {
	tests := []struct {
		segments []string
		want     string
	}{
		{nil, ""},
		{[]string{"paths"}, "paths"},
		{[]string{"paths", "/users", "get"}, "paths./users.get"},
	}

	for _, tt := range tests {
		got := FormatPath(tt.segments...)
		if got != tt.want {
			t.Errorf("FormatPath(%v) = %q, want %q", tt.segments, got, tt.want)
		}
	}
}

func BenchmarkFormatPath_WithPool(b *testing.B) {
	segments := []string{"paths", "/users/{id}", "get", "parameters", "0"}
	for b.Loop() {
		_ = FormatPath(segments...)
	}
}

func BenchmarkFormatPath_WithoutPool(b *testing.B) {
	segments := []string{"paths", "/users/{id}", "get", "parameters", "0"}
	for b.Loop() {
		result := ""
		for i, s := range segments {
			if i > 0 {
				result += "."
			}
			result += s
		}
		_ = result
	}
}
```

**Step 3.4: Run tests**

```bash
go test -v ./internal/issues -run TestFormatPath
go test -bench=BenchmarkFormatPath ./internal/issues -benchmem
```

**Step 3.5: Commit**

```bash
git add internal/issues/pool.go internal/issues/pool_test.go
git commit -m "feat(issues): add sync.Pool for string builders

Adds FormatPath helper using pooled strings.Builder for efficient
path construction in validation/conversion messages.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Phase 2: High-Priority Medium-Risk Pools (4-8)

### Task 4: Slice Pre-allocation Pools

**Files:**
- Modify: `parser/pool.go` (add slice pools)
- Modify: `parser/pool_test.go` (add slice tests)

**Step 4.1: Add slice pools to parser/pool.go**

Add to existing `parser/pool.go`:

```go
// Slice pool capacities (corpus-validated)
const (
	parameterSliceCap = 4  // p75=2, p90=8
	responseMapCap    = 4  // p95=4
	serverSliceCap    = 2  // max=2 across corpus
	tagSliceCap       = 2  // p99=1
)

var parameterSlicePool = sync.Pool{
	New: func() any {
		s := make([]*Parameter, 0, parameterSliceCap)
		return &s
	},
}

func getParameterSlice() *[]*Parameter {
	s := parameterSlicePool.Get().(*[]*Parameter)
	*s = (*s)[:0]
	return s
}

func putParameterSlice(s *[]*Parameter) {
	if s == nil || cap(*s) > 64 {
		return
	}
	parameterSlicePool.Put(s)
}

var serverSlicePool = sync.Pool{
	New: func() any {
		s := make([]*Server, 0, serverSliceCap)
		return &s
	},
}

func getServerSlice() *[]*Server {
	s := serverSlicePool.Get().(*[]*Server)
	*s = (*s)[:0]
	return s
}

func putServerSlice(s *[]*Server) {
	if s == nil || cap(*s) > 16 {
		return
	}
	serverSlicePool.Put(s)
}

var stringSlicePool = sync.Pool{
	New: func() any {
		s := make([]string, 0, tagSliceCap)
		return &s
	},
}

func getStringSlice() *[]string {
	s := stringSlicePool.Get().(*[]string)
	*s = (*s)[:0]
	return s
}

func putStringSlice(s *[]string) {
	if s == nil || cap(*s) > 32 {
		return
	}
	stringSlicePool.Put(s)
}
```

**Step 4.2: Write tests**

Add to `parser/pool_test.go`:

```go
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
```

**Step 4.3: Run tests**

```bash
go test -v ./parser -run TestParameterSlice
go test -bench=BenchmarkParameterSlice ./parser -benchmem -race
```

**Step 4.4: Commit**

```bash
git add parser/pool.go parser/pool_test.go
git commit -m "feat(parser): add sync.Pool for common slice types

Adds pools for Parameter, Server, and string slices with
corpus-validated capacities. Reduces slice allocations during parsing.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Walker Context Pool

**Files:**
- Create: `walker/pool.go`
- Create: `walker/pool_test.go`
- Modify: `walker/walker.go` (use pool for WalkContext)

**Step 5.1: Read walker.go to understand WalkContext**

```bash
# Agent reads walker/walker.go to find WalkContext definition
```

**Step 5.2: Write pool implementation**

Create `walker/pool.go`:

```go
package walker

import "sync"

// Pool capacities (corpus-validated: P99=14, max=14)
const (
	pathCapacity     = 16
	ancestorCapacity = 16
)

var walkContextPool = sync.Pool{
	New: func() any {
		return &WalkContext{
			pathParts: make([]string, 0, pathCapacity),
			ancestors: make([]any, 0, ancestorCapacity),
		}
	},
}

func getWalkContext() *WalkContext {
	ctx := walkContextPool.Get().(*WalkContext)
	ctx.reset()
	return ctx
}

func putWalkContext(ctx *WalkContext) {
	if ctx == nil {
		return
	}
	// Don't pool oversized contexts
	if cap(ctx.pathParts) > 64 || cap(ctx.ancestors) > 64 {
		return
	}
	walkContextPool.Put(ctx)
}

// reset clears the context for reuse.
func (wc *WalkContext) reset() {
	wc.pathParts = wc.pathParts[:0]
	wc.ancestors = wc.ancestors[:0]
	wc.JSONPath = ""
	wc.PathTemplate = ""
	wc.Method = ""
	wc.StatusCode = ""
	wc.Name = ""
}
```

**Step 5.3: Write tests**

Create `walker/pool_test.go`:

```go
package walker

import "testing"

func TestWalkContextPool_Reset(t *testing.T) {
	ctx := getWalkContext()
	ctx.JSONPath = "$.paths./users"
	ctx.pathParts = append(ctx.pathParts, "paths", "/users")
	putWalkContext(ctx)

	ctx2 := getWalkContext()
	if ctx2.JSONPath != "" {
		t.Error("expected empty JSONPath after reset")
	}
	if len(ctx2.pathParts) != 0 {
		t.Error("expected empty pathParts after reset")
	}
	putWalkContext(ctx2)
}

func BenchmarkWalkContext_WithPool(b *testing.B) {
	for b.Loop() {
		ctx := getWalkContext()
		ctx.JSONPath = "$.paths./users.get"
		ctx.pathParts = append(ctx.pathParts, "paths", "/users", "get")
		putWalkContext(ctx)
	}
}

func BenchmarkWalkContext_WithoutPool(b *testing.B) {
	for b.Loop() {
		ctx := &WalkContext{
			pathParts: make([]string, 0, pathCapacity),
			ancestors: make([]any, 0, ancestorCapacity),
		}
		ctx.JSONPath = "$.paths./users.get"
		ctx.pathParts = append(ctx.pathParts, "paths", "/users", "get")
	}
}
```

**Step 5.4: Run tests**

```bash
go test -v ./walker -run TestWalkContextPool
go test -bench=BenchmarkWalkContext ./walker -benchmem -race
```

**Step 5.5: Commit**

```bash
git add walker/pool.go walker/pool_test.go
git commit -m "feat(walker): add sync.Pool for WalkContext

Pools WalkContext with pre-allocated path and ancestor slices.
Capacity 16 based on corpus analysis (max depth 14).

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Conversion Map Pool

**Files:**
- Create: `converter/pool.go`
- Create: `converter/pool_test.go`
- Modify: `converter/oas2_to_oas3.go` (use pool for ref tracking)

**Step 6.1: Write pool implementation**

Create `converter/pool.go`:

```go
package converter

import "sync"

// Pool capacity (corpus-validated: P75=6,319 refs)
const (
	conversionMapCap    = 8192
	conversionMapMaxCap = 16384
)

var conversionMapPool = sync.Pool{
	New: func() any {
		return make(map[string]string, conversionMapCap)
	},
}

func getConversionMap() map[string]string {
	m := conversionMapPool.Get().(map[string]string)
	clear(m)
	return m
}

func putConversionMap(m map[string]string) {
	if m == nil || len(m) > conversionMapMaxCap {
		return
	}
	conversionMapPool.Put(m)
}
```

**Step 6.2: Write tests**

Create `converter/pool_test.go`:

```go
package converter

import "testing"

func TestConversionMapPool_Clear(t *testing.T) {
	m := getConversionMap()
	m["#/definitions/Pet"] = "#/components/schemas/Pet"
	putConversionMap(m)

	m2 := getConversionMap()
	if len(m2) != 0 {
		t.Errorf("expected empty map, got len=%d", len(m2))
	}
	putConversionMap(m2)
}

func BenchmarkConversionMap_WithPool(b *testing.B) {
	for b.Loop() {
		m := getConversionMap()
		for i := 0; i < 100; i++ {
			m["key"+string(rune(i))] = "value"
		}
		putConversionMap(m)
	}
}

func BenchmarkConversionMap_WithoutPool(b *testing.B) {
	for b.Loop() {
		m := make(map[string]string, conversionMapCap)
		for i := 0; i < 100; i++ {
			m["key"+string(rune(i))] = "value"
		}
	}
}
```

**Step 6.3: Run tests**

```bash
go test -v ./converter -run TestConversionMapPool
go test -bench=BenchmarkConversionMap ./converter -benchmem -race
```

**Step 6.4: Commit**

```bash
git add converter/pool.go converter/pool_test.go
git commit -m "feat(converter): add sync.Pool for conversion maps

Pools string maps used for $ref tracking during OAS2<->OAS3 conversion.
Capacity 8192 based on corpus P75 of 6,319 refs.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 7: JSONPath Expression Pool

**Files:**
- Create: `internal/jsonpath/pool.go`
- Create: `internal/jsonpath/pool_test.go`

**Step 7.1: Write pool implementation**

Create `internal/jsonpath/pool.go`:

```go
package jsonpath

import "sync"

// Pool capacities (corpus-validated: 3-4 tokens typical)
const (
	segmentSliceCap = 8
	resultSliceCap  = 32
)

var segmentSlicePool = sync.Pool{
	New: func() any {
		s := make([]Segment, 0, segmentSliceCap)
		return &s
	},
}

func getSegmentSlice() *[]Segment {
	s := segmentSlicePool.Get().(*[]Segment)
	*s = (*s)[:0]
	return s
}

func putSegmentSlice(s *[]Segment) {
	if s == nil || cap(*s) > 32 {
		return
	}
	segmentSlicePool.Put(s)
}

var resultSlicePool = sync.Pool{
	New: func() any {
		s := make([]any, 0, resultSliceCap)
		return &s
	},
}

func getResultSlice() *[]any {
	s := resultSlicePool.Get().(*[]any)
	*s = (*s)[:0]
	return s
}

func putResultSlice(s *[]any) {
	if s == nil || cap(*s) > 128 {
		return
	}
	resultSlicePool.Put(s)
}
```

**Step 7.2: Write tests**

Create `internal/jsonpath/pool_test.go`:

```go
package jsonpath

import "testing"

func TestSegmentSlicePool_Reset(t *testing.T) {
	s := getSegmentSlice()
	*s = append(*s, RootSegment{}, ChildSegment{Key: "paths"})
	putSegmentSlice(s)

	s2 := getSegmentSlice()
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putSegmentSlice(s2)
}

func BenchmarkSegmentSlice_WithPool(b *testing.B) {
	for b.Loop() {
		s := getSegmentSlice()
		*s = append(*s, RootSegment{})
		*s = append(*s, ChildSegment{Key: "paths"})
		*s = append(*s, WildcardSegment{})
		putSegmentSlice(s)
	}
}
```

**Step 7.3: Run tests**

```bash
go test -v ./internal/jsonpath -run TestSegmentSlice
go test -bench=BenchmarkSegmentSlice ./internal/jsonpath -benchmem
```

**Step 7.4: Commit**

```bash
git add internal/jsonpath/pool.go internal/jsonpath/pool_test.go
git commit -m "feat(jsonpath): add sync.Pool for expression segments

Pools segment and result slices for JSONPath evaluation.
Capacity 8 segments based on typical overlay patterns.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 8: DeepCopy Work Pool

**Files:**
- Modify: `parser/pool.go` (add deepcopy work pool)
- Modify: `parser/pool_test.go` (add tests)

**Step 8.1: Add deepcopy pool to parser/pool.go**

Add to existing `parser/pool.go`:

```go
// DeepCopy pool capacities (corpus: P99=3, max=9 schema depth)
const (
	deepCopyWorkCap    = 16
	deepCopyWorkMaxCap = 256
)

var deepCopyWorkPool = sync.Pool{
	New: func() any {
		s := make([]any, 0, deepCopyWorkCap)
		return &s
	},
}

func getDeepCopyWork() *[]any {
	s := deepCopyWorkPool.Get().(*[]any)
	*s = (*s)[:0]
	return s
}

func putDeepCopyWork(s *[]any) {
	if s == nil || cap(*s) > deepCopyWorkMaxCap {
		return
	}
	deepCopyWorkPool.Put(s)
}
```

**Step 8.2: Write tests**

Add to `parser/pool_test.go`:

```go
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

func BenchmarkDeepCopyWork_WithPool(b *testing.B) {
	for b.Loop() {
		s := getDeepCopyWork()
		for i := 0; i < 10; i++ {
			*s = append(*s, i)
		}
		putDeepCopyWork(s)
	}
}
```

**Step 8.3: Run tests**

```bash
go test -v ./parser -run TestDeepCopyWork
go test -bench=BenchmarkDeepCopyWork ./parser -benchmem
```

**Step 8.4: Commit**

```bash
git add parser/pool.go parser/pool_test.go
git commit -m "feat(parser): add sync.Pool for DeepCopy work slices

Pools work slices used during recursive DeepCopy traversal.
Capacity 16 based on corpus max schema depth of 9.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Phase 3: Medium-Impact Pools (9-12)

### Task 9: Differ Change Slice Pool

**Files:**
- Create: `differ/pool.go`
- Create: `differ/pool_test.go`

**Step 9.1: Write pool implementation**

Create `differ/pool.go`:

```go
package differ

import "sync"

// Pool capacity (corpus: median=12, p95=13)
const changeSliceCap = 16

var changeSlicePool = sync.Pool{
	New: func() any {
		s := make([]Change, 0, changeSliceCap)
		return &s
	},
}

func getChangeSlice() *[]Change {
	s := changeSlicePool.Get().(*[]Change)
	*s = (*s)[:0]
	return s
}

func putChangeSlice(s *[]Change) {
	if s == nil || cap(*s) > 128 {
		return
	}
	changeSlicePool.Put(s)
}
```

**Step 9.2: Write tests**

Create `differ/pool_test.go`:

```go
package differ

import "testing"

func TestChangeSlicePool_Reset(t *testing.T) {
	s := getChangeSlice()
	*s = append(*s, Change{Path: "$.info.title", Type: ChangeTypeModified})
	putChangeSlice(s)

	s2 := getChangeSlice()
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putChangeSlice(s2)
}

func BenchmarkChangeSlice_WithPool(b *testing.B) {
	for b.Loop() {
		s := getChangeSlice()
		for i := 0; i < 10; i++ {
			*s = append(*s, Change{Path: "$.paths./users"})
		}
		putChangeSlice(s)
	}
}
```

**Step 9.3: Run tests and commit**

```bash
go test -v ./differ -run TestChangeSlice
go test -bench=BenchmarkChangeSlice ./differ -benchmem
git add differ/pool.go differ/pool_test.go
git commit -m "feat(differ): add sync.Pool for change slices

Pools Change slices with capacity 16 based on corpus median of 12.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 10: Generator Template Buffer Pool

**Files:**
- Create: `generator/pool.go`
- Create: `generator/pool_test.go`

**Step 10.1: Write tiered pool implementation**

Create `generator/pool.go`:

```go
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
```

**Step 10.2: Write tests**

Create `generator/pool_test.go`:

```go
package generator

import "testing"

func TestTemplateBufferPool_TieredSizes(t *testing.T) {
	small := getTemplateBuffer(5)
	if small.Cap() < smallBufferSize {
		t.Errorf("small buffer cap %d < %d", small.Cap(), smallBufferSize)
	}
	putTemplateBuffer(small, 5)

	medium := getTemplateBuffer(25)
	if medium.Cap() < mediumBufferSize {
		t.Errorf("medium buffer cap %d < %d", medium.Cap(), mediumBufferSize)
	}
	putTemplateBuffer(medium, 25)

	large := getTemplateBuffer(100)
	if large.Cap() < largeBufferSize {
		t.Errorf("large buffer cap %d < %d", large.Cap(), largeBufferSize)
	}
	putTemplateBuffer(large, 100)
}

func BenchmarkTemplateBuffer_WithPool(b *testing.B) {
	for b.Loop() {
		buf := getTemplateBuffer(25)
		buf.WriteString("package main\n\nfunc main() {}\n")
		putTemplateBuffer(buf, 25)
	}
}
```

**Step 10.3: Run tests and commit**

```bash
go test -v ./generator -run TestTemplateBuffer
go test -bench=BenchmarkTemplateBuffer ./generator -benchmem
git add generator/pool.go generator/pool_test.go
git commit -m "feat(generator): add tiered sync.Pool for template buffers

Pools template output buffers with tiered sizing (8KB/32KB/64KB)
based on operation count for optimal memory usage.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 11: Builder Component Map Pool

**Files:**
- Create: `builder/pool.go`
- Create: `builder/pool_test.go`

**Step 11.1: Write pool implementation**

Create `builder/pool.go`:

```go
package builder

import (
	"sync"

	"github.com/erraggy/oastools/parser"
)

const (
	schemaMapCap    = 8
	pathMapCap      = 4
	operationSliceCap = 8
)

var schemaMapPool = sync.Pool{
	New: func() any {
		return make(map[string]*parser.Schema, schemaMapCap)
	},
}

func getSchemaMap() map[string]*parser.Schema {
	m := schemaMapPool.Get().(map[string]*parser.Schema)
	clear(m)
	return m
}

func putSchemaMap(m map[string]*parser.Schema) {
	if m == nil || len(m) > 128 {
		return
	}
	schemaMapPool.Put(m)
}

var pathMapPool = sync.Pool{
	New: func() any {
		return make(map[string]*parser.PathItem, pathMapCap)
	},
}

func getPathMap() map[string]*parser.PathItem {
	m := pathMapPool.Get().(map[string]*parser.PathItem)
	clear(m)
	return m
}

func putPathMap(m map[string]*parser.PathItem) {
	if m == nil || len(m) > 64 {
		return
	}
	pathMapPool.Put(m)
}
```

**Step 11.2: Write tests**

Create `builder/pool_test.go`:

```go
package builder

import "testing"

func TestSchemaMapPool_Clear(t *testing.T) {
	m := getSchemaMap()
	m["Pet"] = &parser.Schema{Type: "object"}
	putSchemaMap(m)

	m2 := getSchemaMap()
	if len(m2) != 0 {
		t.Errorf("expected empty map, got len=%d", len(m2))
	}
	putSchemaMap(m2)
}

func BenchmarkSchemaMap_WithPool(b *testing.B) {
	for b.Loop() {
		m := getSchemaMap()
		m["Pet"] = &parser.Schema{}
		m["User"] = &parser.Schema{}
		putSchemaMap(m)
	}
}
```

**Step 11.3: Run tests and commit**

```bash
go test -v ./builder -run TestSchemaMap
go test -bench=BenchmarkSchemaMap ./builder -benchmem
git add builder/pool.go builder/pool_test.go
git commit -m "feat(builder): add sync.Pool for component maps

Pools schema and path maps for programmatic spec construction.
Small capacities (8, 4) suitable for incremental building.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 12: Fixer Issue Collection Pool

**Files:**
- Create: `fixer/pool.go`
- Create: `fixer/pool_test.go`

**Step 12.1: Write pool implementation with lazy init note**

Create `fixer/pool.go`:

```go
package fixer

import "sync"

// Pool capacity (corpus: p95=3, median=0)
// Note: Most specs have 0 fixes, so this pool has low hit rate.
// Kept small to minimize memory overhead.
const fixSliceCap = 4

var fixSlicePool = sync.Pool{
	New: func() any {
		s := make([]Fix, 0, fixSliceCap)
		return &s
	},
}

func getFixSlice() *[]Fix {
	s := fixSlicePool.Get().(*[]Fix)
	*s = (*s)[:0]
	return s
}

func putFixSlice(s *[]Fix) {
	if s == nil || cap(*s) > 32 {
		return
	}
	fixSlicePool.Put(s)
}
```

**Step 12.2: Write tests**

Create `fixer/pool_test.go`:

```go
package fixer

import "testing"

func TestFixSlicePool_Reset(t *testing.T) {
	s := getFixSlice()
	*s = append(*s, Fix{Type: FixTypeMissingPathParameter, Path: "$.paths./users/{id}"})
	putFixSlice(s)

	s2 := getFixSlice()
	if len(*s2) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(*s2))
	}
	putFixSlice(s2)
}

func BenchmarkFixSlice_WithPool(b *testing.B) {
	for b.Loop() {
		s := getFixSlice()
		*s = append(*s, Fix{Type: FixTypeMissingPathParameter})
		putFixSlice(s)
	}
}
```

**Step 12.3: Run tests and commit**

```bash
go test -v ./fixer -run TestFixSlice
go test -bench=BenchmarkFixSlice ./fixer -benchmem
git add fixer/pool.go fixer/pool_test.go
git commit -m "feat(fixer): add sync.Pool for fix slices

Pools Fix slices with small capacity (4) since corpus shows
median 0 fixes. Low-overhead pool for edge cases.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Phase 4: Verification

### Task 13: Run Full Test Suite

**Step 13.1: Run all tests with race detector**

```bash
go test -race ./...
```

Expected: All tests pass, no race conditions.

**Step 13.2: Run benchmarks to measure improvement**

```bash
go test -bench=. ./parser ./httpvalidator ./walker ./converter ./differ \
    ./generator ./builder ./fixer ./internal/... -benchmem
```

**Step 13.3: Run corpus integration tests**

```bash
go test -v ./parser -run TestCorpus
go test -v ./converter -run TestCorpus
```

---

### Task 14: Final Commit

**Step 14.1: Verify diagnostics clean**

```bash
# Use gopls to check for any issues
```

**Step 14.2: Run make check**

```bash
make check
```

Expected: All checks pass.

---

## Summary

| Task | Package | Pool Type | Capacity | Risk |
|------|---------|-----------|----------|------|
| 1 | parser | Marshal buffer | 4KB/1MB | Low |
| 2 | httpvalidator | Request context | 8/4 | Low |
| 3 | internal/issues | String builder | - | Low |
| 4 | parser | Slice pools | 4/4/2/2 | Medium |
| 5 | walker | Walk context | 16/16 | Low |
| 6 | converter | Conversion map | 8192 | Low |
| 7 | internal/jsonpath | Segments/results | 8/32 | Low |
| 8 | parser | DeepCopy work | 16 | Medium |
| 9 | differ | Change slice | 16 | Low |
| 10 | generator | Template buffer | tiered | Low |
| 11 | builder | Component maps | 8/4 | Low |
| 12 | fixer | Fix slice | 4 | Low |

**Total: 20 new files, 16 modified files**
