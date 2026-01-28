# PathUtil String Optimization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create `internal/pathutil` package with `PathBuilder` type and ref builders to eliminate `fmt.Sprintf` allocations in hot paths.

**Architecture:** Lazy path building via push/pop stack semantics. String only materialized when `String()` called. `sync.Pool` for `PathBuilder` reuse. Const-prefix concatenation for ref builders.

**Tech Stack:** Go 1.24+, `sync.Pool`, `strings.Builder`, `strconv`

**Worktree:** `/Users/robbie/code/oastools/.worktrees/perf-pathutil`

**Design Doc:** `docs/plans/2026-01-28-pathutil-string-optimization-design.md` (in main repo)

---

## Task 1: Create PathBuilder Core Type

**Files:**
- Create: `internal/pathutil/builder.go`
- Create: `internal/pathutil/builder_test.go`

**Step 1: Create package and write failing test for basic Push/String**

```go
// internal/pathutil/builder_test.go
package pathutil

import "testing"

func TestPathBuilder_Basic(t *testing.T) {
	p := &PathBuilder{}
	p.Push("properties")
	p.Push("name")

	got := p.String()
	want := "properties.name"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/pathutil -run TestPathBuilder_Basic -v`
Expected: FAIL - package or type not found

**Step 3: Write minimal PathBuilder implementation**

```go
// internal/pathutil/builder.go
package pathutil

import "strings"

// PathBuilder provides efficient incremental path construction.
// Uses push/pop semantics to avoid allocations during traversal.
// The full string is only materialized when String() is called.
type PathBuilder struct {
	segments []string
	length   int // Pre-calculated length for String() allocation
}

// Push adds a segment to the path.
func (p *PathBuilder) Push(segment string) {
	p.segments = append(p.segments, segment)
	if len(p.segments) > 1 {
		p.length++ // For dot separator
	}
	p.length += len(segment)
}

// String materializes the full path. Only call when the path is needed.
func (p *PathBuilder) String() string {
	if len(p.segments) == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(p.length)
	b.WriteString(p.segments[0])
	for _, seg := range p.segments[1:] {
		if len(seg) > 0 && seg[0] == '[' {
			b.WriteString(seg)
		} else {
			b.WriteByte('.')
			b.WriteString(seg)
		}
	}
	return b.String()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/pathutil -run TestPathBuilder_Basic -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/pathutil/builder.go internal/pathutil/builder_test.go
git commit -m "feat(pathutil): add PathBuilder with Push and String"
```

---

## Task 2: Add PushIndex and Pop Methods

**Files:**
- Modify: `internal/pathutil/builder.go`
- Modify: `internal/pathutil/builder_test.go`

**Step 1: Write failing tests for PushIndex and Pop**

```go
// Add to internal/pathutil/builder_test.go

func TestPathBuilder_WithIndex(t *testing.T) {
	p := &PathBuilder{}
	p.Push("allOf")
	p.PushIndex(0)
	p.Push("properties")

	got := p.String()
	want := "allOf[0].properties"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPathBuilder_PushPop(t *testing.T) {
	p := &PathBuilder{}
	p.Push("a")
	p.Push("b")
	p.Pop()
	p.Push("c")

	got := p.String()
	want := "a.c"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPathBuilder_Empty(t *testing.T) {
	p := &PathBuilder{}
	got := p.String()
	if got != "" {
		t.Errorf("String() on empty = %q, want empty", got)
	}
}

func TestPathBuilder_PopEmpty(t *testing.T) {
	p := &PathBuilder{}
	p.Pop() // Should not panic
	got := p.String()
	if got != "" {
		t.Errorf("String() after Pop on empty = %q, want empty", got)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/pathutil -v`
Expected: FAIL - PushIndex and Pop not defined

**Step 3: Implement PushIndex and Pop**

```go
// Add to internal/pathutil/builder.go after Push method

import "strconv"

// PushIndex adds an array index segment: "[0]", "[1]", etc.
func (p *PathBuilder) PushIndex(i int) {
	seg := "[" + strconv.Itoa(i) + "]"
	p.segments = append(p.segments, seg)
	p.length += len(seg) // No dot separator for brackets
}

// Pop removes the last segment.
func (p *PathBuilder) Pop() {
	if len(p.segments) == 0 {
		return
	}
	last := p.segments[len(p.segments)-1]
	p.segments = p.segments[:len(p.segments)-1]
	p.length -= len(last)
	// Remove dot separator if this wasn't the first segment and wasn't a bracket
	if len(p.segments) > 0 && (len(last) == 0 || last[0] != '[') {
		p.length--
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/pathutil -v`
Expected: PASS (all 4 tests)

**Step 5: Commit**

```bash
git add internal/pathutil/builder.go internal/pathutil/builder_test.go
git commit -m "feat(pathutil): add PushIndex and Pop methods"
```

---

## Task 3: Add Reset Method and sync.Pool

**Files:**
- Create: `internal/pathutil/pool.go`
- Modify: `internal/pathutil/builder.go`
- Modify: `internal/pathutil/builder_test.go`

**Step 1: Write failing tests for Reset and pool functions**

```go
// Add to internal/pathutil/builder_test.go

func TestPathBuilder_Reset(t *testing.T) {
	p := &PathBuilder{}
	p.Push("a")
	p.Push("b")
	p.Reset()

	got := p.String()
	if got != "" {
		t.Errorf("String() after Reset = %q, want empty", got)
	}

	// Should be reusable after reset
	p.Push("c")
	got = p.String()
	if got != "c" {
		t.Errorf("String() after Reset+Push = %q, want %q", got, "c")
	}
}

func TestPool_GetPut(t *testing.T) {
	p := Get()
	if p == nil {
		t.Fatal("Get() returned nil")
	}

	p.Push("test")
	Put(p)

	// Get another - may or may not be same instance
	p2 := Get()
	if p2 == nil {
		t.Fatal("Get() returned nil after Put")
	}
	// After Get, should be reset
	if p2.String() != "" {
		t.Errorf("Get() returned non-empty PathBuilder: %q", p2.String())
	}
	Put(p2)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/pathutil -v`
Expected: FAIL - Reset, Get, Put not defined

**Step 3: Implement Reset in builder.go**

```go
// Add to internal/pathutil/builder.go

// Reset clears the builder for reuse.
func (p *PathBuilder) Reset() {
	p.segments = p.segments[:0]
	p.length = 0
}
```

**Step 4: Create pool.go with Get/Put**

```go
// internal/pathutil/pool.go
package pathutil

import "sync"

const (
	defaultPathCap = 8  // Most paths are <8 segments deep
	maxPathCap     = 64 // Don't pool excessively deep paths
)

var pathBuilderPool = sync.Pool{
	New: func() any {
		return &PathBuilder{
			segments: make([]string, 0, defaultPathCap),
		}
	},
}

// Get retrieves a PathBuilder from the pool, reset and ready to use.
func Get() *PathBuilder {
	p := pathBuilderPool.Get().(*PathBuilder)
	p.Reset()
	return p
}

// Put returns a PathBuilder to the pool if not oversized.
func Put(p *PathBuilder) {
	if p == nil || cap(p.segments) > maxPathCap {
		return // Let GC collect oversized builders
	}
	pathBuilderPool.Put(p)
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/pathutil -v`
Expected: PASS (all 6 tests)

**Step 6: Commit**

```bash
git add internal/pathutil/builder.go internal/pathutil/pool.go internal/pathutil/builder_test.go
git commit -m "feat(pathutil): add Reset method and sync.Pool"
```

---

## Task 4: Add Reference String Builders

**Files:**
- Create: `internal/pathutil/refs.go`
- Modify: `internal/pathutil/builder_test.go`

**Step 1: Write failing tests for ref builders**

```go
// Add to internal/pathutil/builder_test.go

func TestSchemaRef(t *testing.T) {
	got := SchemaRef("Pet")
	want := "#/components/schemas/Pet"
	if got != want {
		t.Errorf("SchemaRef(Pet) = %q, want %q", got, want)
	}
}

func TestDefinitionRef(t *testing.T) {
	got := DefinitionRef("Pet")
	want := "#/definitions/Pet"
	if got != want {
		t.Errorf("DefinitionRef(Pet) = %q, want %q", got, want)
	}
}

func TestParameterRef(t *testing.T) {
	tests := []struct {
		name    string
		version int // 2 for OAS2, 3 for OAS3
		want    string
	}{
		{"limitParam", 2, "#/parameters/limitParam"},
		{"limitParam", 3, "#/components/parameters/limitParam"},
	}
	for _, tt := range tests {
		got := ParameterRef(tt.name, tt.version == 2)
		if got != tt.want {
			t.Errorf("ParameterRef(%q, oas2=%v) = %q, want %q", tt.name, tt.version == 2, got, tt.want)
		}
	}
}

func TestResponseRef(t *testing.T) {
	tests := []struct {
		name    string
		version int
		want    string
	}{
		{"NotFound", 2, "#/responses/NotFound"},
		{"NotFound", 3, "#/components/responses/NotFound"},
	}
	for _, tt := range tests {
		got := ResponseRef(tt.name, tt.version == 2)
		if got != tt.want {
			t.Errorf("ResponseRef(%q, oas2=%v) = %q, want %q", tt.name, tt.version == 2, got, tt.want)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/pathutil -v`
Expected: FAIL - SchemaRef, DefinitionRef, etc. not defined

**Step 3: Implement refs.go**

```go
// internal/pathutil/refs.go
package pathutil

// OAS 2.0 reference prefixes
const (
	RefPrefixDefinitions         = "#/definitions/"
	RefPrefixParameters          = "#/parameters/"
	RefPrefixResponses           = "#/responses/"
	RefPrefixSecurityDefinitions = "#/securityDefinitions/"
)

// OAS 3.x reference prefixes
const (
	RefPrefixSchemas         = "#/components/schemas/"
	RefPrefixParameters3     = "#/components/parameters/"
	RefPrefixResponses3      = "#/components/responses/"
	RefPrefixExamples        = "#/components/examples/"
	RefPrefixRequestBodies   = "#/components/requestBodies/"
	RefPrefixHeaders         = "#/components/headers/"
	RefPrefixSecuritySchemes = "#/components/securitySchemes/"
	RefPrefixLinks           = "#/components/links/"
	RefPrefixCallbacks       = "#/components/callbacks/"
	RefPrefixPathItems       = "#/components/pathItems/"
)

// SchemaRef builds "#/components/schemas/{name}" (OAS 3.x)
func SchemaRef(name string) string {
	return RefPrefixSchemas + name
}

// DefinitionRef builds "#/definitions/{name}" (OAS 2.0)
func DefinitionRef(name string) string {
	return RefPrefixDefinitions + name
}

// ParameterRef builds the appropriate parameter ref.
// If oas2 is true, returns "#/parameters/{name}", otherwise "#/components/parameters/{name}".
func ParameterRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixParameters + name
	}
	return RefPrefixParameters3 + name
}

// ResponseRef builds the appropriate response ref.
// If oas2 is true, returns "#/responses/{name}", otherwise "#/components/responses/{name}".
func ResponseRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixResponses + name
	}
	return RefPrefixResponses3 + name
}

// SecuritySchemeRef builds the appropriate security scheme ref.
// If oas2 is true, returns "#/securityDefinitions/{name}", otherwise "#/components/securitySchemes/{name}".
func SecuritySchemeRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixSecurityDefinitions + name
	}
	return RefPrefixSecuritySchemes + name
}

// HeaderRef builds "#/components/headers/{name}" (OAS 3.x only).
func HeaderRef(name string) string {
	return RefPrefixHeaders + name
}

// RequestBodyRef builds "#/components/requestBodies/{name}" (OAS 3.x only).
func RequestBodyRef(name string) string {
	return RefPrefixRequestBodies + name
}

// ExampleRef builds "#/components/examples/{name}" (OAS 3.x only).
func ExampleRef(name string) string {
	return RefPrefixExamples + name
}

// LinkRef builds "#/components/links/{name}" (OAS 3.x only).
func LinkRef(name string) string {
	return RefPrefixLinks + name
}

// CallbackRef builds "#/components/callbacks/{name}" (OAS 3.x only).
func CallbackRef(name string) string {
	return RefPrefixCallbacks + name
}

// PathItemRef builds "#/components/pathItems/{name}" (OAS 3.1+ only).
func PathItemRef(name string) string {
	return RefPrefixPathItems + name
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/pathutil -v`
Expected: PASS (all 10 tests)

**Step 5: Commit**

```bash
git add internal/pathutil/refs.go internal/pathutil/builder_test.go
git commit -m "feat(pathutil): add reference string builders"
```

---

## Task 5: Add Benchmarks

**Files:**
- Create: `internal/pathutil/builder_bench_test.go`

**Step 1: Write benchmarks comparing PathBuilder vs fmt.Sprintf**

```go
// internal/pathutil/builder_bench_test.go
package pathutil

import (
	"fmt"
	"testing"
)

func BenchmarkPathBuilder_DeepPath(b *testing.B) {
	b.Run("PathBuilder", func(b *testing.B) {
		for b.Loop() {
			p := Get()
			p.Push("components")
			p.Push("schemas")
			p.Push("Pet")
			p.Push("properties")
			p.Push("tags")
			p.Push("items")
			p.Push("properties")
			p.Push("name")
			_ = p.String()
			Put(p)
		}
	})

	b.Run("FmtSprintf", func(b *testing.B) {
		for b.Loop() {
			path := "components"
			path = fmt.Sprintf("%s.%s", path, "schemas")
			path = fmt.Sprintf("%s.%s", path, "Pet")
			path = fmt.Sprintf("%s.%s", path, "properties")
			path = fmt.Sprintf("%s.%s", path, "tags")
			path = fmt.Sprintf("%s.%s", path, "items")
			path = fmt.Sprintf("%s.%s", path, "properties")
			path = fmt.Sprintf("%s.%s", path, "name")
			_ = path
		}
	})
}

func BenchmarkPathBuilder_NoStringCall(b *testing.B) {
	b.Run("PathBuilder_NoString", func(b *testing.B) {
		for b.Loop() {
			p := Get()
			for j := 0; j < 8; j++ {
				p.Push("segment")
			}
			for j := 0; j < 8; j++ {
				p.Pop()
			}
			Put(p)
		}
	})

	b.Run("FmtSprintf_Equivalent", func(b *testing.B) {
		for b.Loop() {
			path := ""
			for j := 0; j < 8; j++ {
				if path == "" {
					path = "segment"
				} else {
					path = fmt.Sprintf("%s.%s", path, "segment")
				}
			}
			_ = path
		}
	})
}

func BenchmarkRefBuilders(b *testing.B) {
	b.Run("SchemaRef", func(b *testing.B) {
		for b.Loop() {
			_ = SchemaRef("MySchema")
		}
	})

	b.Run("FmtSprintf", func(b *testing.B) {
		for b.Loop() {
			_ = fmt.Sprintf("#/components/schemas/%s", "MySchema")
		}
	})
}

func BenchmarkPathBuilder_WithIndex(b *testing.B) {
	b.Run("PathBuilder", func(b *testing.B) {
		for b.Loop() {
			p := Get()
			p.Push("allOf")
			p.PushIndex(0)
			p.Push("properties")
			p.Push("name")
			_ = p.String()
			Put(p)
		}
	})

	b.Run("FmtSprintf", func(b *testing.B) {
		for b.Loop() {
			path := "allOf"
			path = fmt.Sprintf("%s[%d]", path, 0)
			path = fmt.Sprintf("%s.%s", path, "properties")
			path = fmt.Sprintf("%s.%s", path, "name")
			_ = path
		}
	})
}
```

**Step 2: Run benchmarks**

Run: `go test ./internal/pathutil -bench=. -benchmem`
Expected: PathBuilder should show fewer allocations than FmtSprintf

**Step 3: Commit**

```bash
git add internal/pathutil/builder_bench_test.go
git commit -m "test(pathutil): add benchmarks for PathBuilder vs fmt.Sprintf"
```

---

## Task 6: Add Package Documentation

**Files:**
- Create: `internal/pathutil/doc.go`

**Step 1: Write package documentation**

```go
// internal/pathutil/doc.go

// Package pathutil provides efficient path building utilities for OpenAPI
// document traversal.
//
// The primary type is [PathBuilder], which uses push/pop semantics to build
// paths incrementally without allocating intermediate strings. This is
// particularly useful in recursive traversal where paths are built on each
// recursive call but only used when reporting errors or differences.
//
// # PathBuilder Usage
//
// Use [Get] to obtain a pooled PathBuilder, and [Put] to return it:
//
//	path := pathutil.Get()
//	defer pathutil.Put(path)
//
//	path.Push("properties")
//	path.Push(propName)
//	// ... recurse ...
//	path.Pop()
//	path.Pop()
//
//	// Only call String() when needed (e.g., reporting an error)
//	if hasError {
//	    return fmt.Errorf("error at %s", path.String())
//	}
//
// # Reference Builders
//
// The package also provides functions for building JSON Pointer references
// to OpenAPI components:
//
//	ref := pathutil.SchemaRef("Pet")      // "#/components/schemas/Pet"
//	ref := pathutil.DefinitionRef("Pet")  // "#/definitions/Pet"
//
// These use simple string concatenation which Go optimizes well for two
// operands, avoiding the overhead of fmt.Sprintf.
package pathutil
```

**Step 2: Verify documentation renders**

Run: `go doc ./internal/pathutil`
Expected: Package documentation displays correctly

**Step 3: Commit**

```bash
git add internal/pathutil/doc.go
git commit -m "docs(pathutil): add package documentation"
```

---

## Task 7: Run Full Test Suite and Verify

**Files:** None (verification only)

**Step 1: Run all pathutil tests**

Run: `go test ./internal/pathutil -v -race`
Expected: All tests PASS, no race conditions

**Step 2: Run benchmarks with memory stats**

Run: `go test ./internal/pathutil -bench=. -benchmem -count=3`
Expected: Consistent results showing allocation improvements

**Step 3: Verify build**

Run: `go build ./...`
Expected: Clean build, no errors

**Step 4: Run linter**

Run: `golangci-lint run ./internal/pathutil/...`
Expected: No lint errors

---

## Summary

After completing all tasks, the `internal/pathutil` package will contain:

| File | Purpose |
|------|---------|
| `builder.go` | PathBuilder type with Push, PushIndex, Pop, Reset, String |
| `pool.go` | sync.Pool management with Get/Put |
| `refs.go` | Reference string builders for all OAS component types |
| `doc.go` | Package documentation |
| `builder_test.go` | Unit tests |
| `builder_bench_test.go` | Benchmarks |

**Next phase:** Migration of existing packages (fixer, validator, walker, differ) to use pathutil. This should be done in separate PRs per package to keep changes reviewable.
