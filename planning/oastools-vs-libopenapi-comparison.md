# oastools Feature Gap Analysis

**Date:** December 2025
**oastools Version:** v1.30.1
**Reference Library:** libopenapi v0.30.4

---

## Purpose

This document identifies features present in libopenapi (github.com/pb33f/libopenapi) that are missing from oastools, to inform the oastools development roadmap. It also notes architectural differences that explain design trade-offs.

## Architectural Context

**oastools** uses a 10-package modular architecture where each package handles a distinct concern (parsing, validation, fixing, conversion, joining, overlay, diffing, generation, building, errors).

**libopenapi** uses a two-tier "porcelain/plumbing" architecture that preserves complete source document metadata (line numbers, column positions, YAML node structure).

---

## Feature Comparison Matrix

| Capability | oastools | libopenapi | Notes |
|------------|----------|------------|-------|
| **OAS 2.0 (Swagger)** | ✅ Full | ✅ Full | Both parse and validate |
| **OAS 3.0.x** | ✅ Full | ✅ Full | Primary target for both |
| **OAS 3.1.x** | ✅ Full | ✅ Full | JSON Schema 2020-12 alignment |
| **OAS 3.2.0** | ✅ Full | ✅ Full | Both recently added support |
| **JSON/YAML parsing** | ✅ | ✅ | Both auto-detect format |
| **Format preservation** | ✅ | ✅ | Both preserve input format |
| **Lossless parsing** | ⚠️ Partial | ✅ Full | libopenapi preserves line/column numbers |
| **Source maps** | ✅ Optional | ✅ Built-in | oastools opt-in via `WithSourceMap(true)` |
| **Reference resolution** | ✅ Full | ✅ Full | Both handle circular refs |
| **External $ref (file)** | ✅ | ✅ | Both support local file refs |
| **External $ref (HTTP)** | ✅ | ✅ | Both support remote refs |
| **Spec validation** | ✅ Severity levels | ✅ Via libopenapi-validator | oastools has Error/Warning/Info |
| **HTTP request/response validation** | ✅ Via httpvalidator | ✅ Via libopenapi-validator | Both provide runtime HTTP validation |
| **Auto-fixing** | ✅ Dedicated fixer package | ⚠️ Via vacuum --fix | oastools integrated, libopenapi external |
| **Version conversion (2.0 ↔ 3.x)** | ✅ Bidirectional | ❌ Not available | oastools feature |
| **Multi-spec merging** | ✅ With collision strategies | ⚠️ Basic bundler | oastools has Accept/Rename/Error strategies |
| **OpenAPI Overlay v1.0.0** | ✅ Native support | ❌ Not available | oastools feature |
| **Breaking change detection** | ✅ Integrated differ | ✅ what-changed engine | Both provide detection |
| **Code generation** | ✅ Client/Server/Types | ❌ Not included | oastools feature |
| **Programmatic building** | ✅ Fluent builder API | ⚠️ Manual struct creation | oastools more ergonomic |
| **Semantic deduplication** | ✅ Schema consolidation | ❌ Not available | oastools reduces document size |
| **CLI tool** | ✅ Full-featured | ❌ Library only | libopenapi powers vacuum CLI |
| **Structured errors** | ✅ oaserrors package | ✅ Error types | Both provide rich error context |
| **DeepCopy methods** | ✅ Code-generated | ❌ Manual | oastools 37x faster cloning |

---

## Architectural Deep Dive

### oastools: 10-Package Modular Architecture

oastools organizes functionality into discrete packages, each with a single responsibility:

```
github.com/erraggy/oastools
├── parser      → Parse OAS files (JSON/YAML, file/URL/reader)
├── validator   → Validate with severity levels (Error/Warning/Info)
├── fixer       → Auto-fix common validation errors
├── converter   → Convert between OAS 2.0 and 3.x (bidirectional)
├── joiner      → Merge multiple specs with collision strategies
├── overlay     → Apply OpenAPI Overlay v1.0.0 transformations
├── differ      → Detect breaking changes between versions
├── generator   → Generate Go client/server code with security support
├── builder     → Programmatically construct OAS documents
└── oaserrors   → Structured error types for programmatic handling
```

This design enables the **parse-once pattern**, where a document is parsed once and the `ParseResult` is passed to multiple operations:

```go
// Parse once, reuse for multiple operations
result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))

// Each *Parsed method skips re-parsing (massive performance gain)
validator.ValidateParsed(result, true, false)   // 31x faster
converter.ConvertParsed(result, "3.0.3")        // 9x faster  
differ.DiffParsed(result1, result2)             // 81x faster
joiner.JoinParsed([]ParseResult{result, other}) // 150x faster
```

### libopenapi: Porcelain/Plumbing Two-Tier Architecture

libopenapi provides two API layers for every type:

```go
// High-level (Porcelain) - easy to use
schema := v3Model.Model.Components.Schemas.GetOrZero("Pet").Schema()
fmt.Println(schema.Type)  // Direct property access

// Low-level (Plumbing) - full AST access via GoLow()
lowSchema := schema.GoLow()
fmt.Printf("Line %d, Column %d\n", 
    lowSchema.Type.KeyNode.Line, 
    lowSchema.Type.KeyNode.Column)
```

This lossless parsing preserves:
- Line and column numbers for every key and value
- Original YAML node structure
- Comments (where supported)
- Key ordering via `orderedmap.Map`

The architecture excels for tooling that needs precise source locations (linters, IDEs, diff engines) but adds overhead for simple parsing scenarios.

---

## Dependency Comparison

### oastools Dependencies

```
github.com/erraggy/oastools
├── go.yaml.in/yaml/v4      (YAML parsing)
└── golang.org/x/tools      (Code generation - imports analysis)

Test-only: github.com/stretchr/testify
```

oastools explicitly minimizes dependencies, with only two runtime dependencies required.

### libopenapi Dependencies

```
github.com/pb33f/libopenapi
├── github.com/pb33f/ordered-map/v2  (Ordered map implementation)
├── gopkg.in/yaml.v3                  (YAML parsing)
├── github.com/santhosh-tekuri/jsonschema (JSON Schema validation)
├── github.com/vmware-labs/yaml-jsonpath  (JSONPath queries)
└── (additional transitive dependencies)
```

For validation, libopenapi-validator adds:
```
github.com/pb33f/libopenapi-validator
├── github.com/santhosh-tekuri/jsonschema/v6
└── (shares libopenapi dependencies)
```

---

## Performance Analysis

### oastools Published Benchmarks

oastools provides comprehensive benchmark data (from benchmarks.md, Apple M4, Go 1.24):

**Parser Performance:**

| Document Size | Parse Time | Memory | Allocations |
|---------------|------------|--------|-------------|
| Small OAS3 | 142 μs | 203 KB | 2,128 |
| Medium OAS3 | 1.1 ms | 1.4 MB | 17,000 |
| Large OAS3 | 11.6 ms | 18.5 MB | 170,000 |

**Parse-Once Performance Gains:**

| Operation | Full (parse+op) | Pre-parsed | Speedup |
|-----------|-----------------|------------|---------|
| Validate | 240 μs | 7.5 μs | **31x** |
| Fix | 303 μs | 86 μs | **3.5x** |
| Convert | 152 μs | 3.2 μs | **47x** |
| Join (2 docs) | 110 μs | 732 ns | **150x** |
| Diff | 463 μs | 5.7 μs | **81x** |

**Code Generation Performance:**

| Mode | Time | Memory | Allocations |
|------|------|--------|-------------|
| Types only | 39 μs | 28 KB | 724 |
| Client | 272 μs | 187 KB | 4,088 |
| Server | 57 μs | 48 KB | 1,040 |
| All modes | 249 μs | 182 KB | 3,882 |

### libopenapi Performance Characteristics

libopenapi markets itself as "high performance" but does not publish comparative benchmarks. Known optimizations include:

- `sync.Pool` for `strings.Builder` in hash generation
- Parallel translation functions (`TranslateMapParallel`, `TranslateSliceParallel`)
- Caching layer with hit/miss tracking (`GetHighCache()`, `GetHighCacheHits()`)
- Internal benchmark tests exist but aren't documented

The vacuum linter claims "world's fastest OpenAPI linter" but without comparative data against tools like Spectral.

---

## Unique Capabilities

### oastools Exclusive Features

**1. Version Conversion (converter package)**

Bidirectional conversion between OAS 2.0 and 3.x with issue tracking:

```go
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)

// Issues categorized by severity
for _, issue := range result.Issues {
    fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Path, issue.Message)
}
```

No equivalent exists in libopenapi. The closest alternative is kin-openapi's unidirectional 2→3 converter.

**2. OpenAPI Overlay v1.0.0 Support (overlay package)**

Native implementation of the OAI Overlay Specification:

```go
result, _ := overlay.ApplyWithOptions(
    overlay.WithSpecFilePath("api.yaml"),
    overlay.WithOverlayFilePath("production.yaml"),
)

// Or dry-run to preview changes
preview, _ := overlay.DryRunWithOptions(
    overlay.WithSpecFilePath("api.yaml"),
    overlay.WithOverlayFilePath("changes.yaml"),
)
```

libopenapi has no Overlay support. The separate speakeasy-api/openapi-overlay library exists but isn't integrated.

**3. Code Generation (generator package)**

Generate Go client/server code with OAuth2/OIDC support:

```go
result, _ := generator.GenerateWithOptions(
    generator.WithFilePath("api.yaml"),
    generator.WithPackageName("api"),
    generator.WithClient(true),
    generator.WithServer(true),
    generator.WithOAuth2Flows(true),
)
```

libopenapi explicitly excludes code generation. External tools like oapi-codegen fill this gap.

**4. Multi-Spec Merging with Collision Strategies (joiner package)**

```go
result, _ := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
    joiner.WithPathStrategy(joiner.StrategyAcceptLeft),    // or AcceptRight, Rename, Error
    joiner.WithSchemaStrategy(joiner.StrategyRename),
)
```

libopenapi's bundler provides basic multi-file support but without explicit collision handling.

**5. Semantic Schema Deduplication (builder package)**

```go
spec := builder.New(parser.OASVersion320,
    builder.WithSemanticDeduplication(true),
)
// Structurally identical schemas are automatically consolidated
```

### libopenapi Exclusive Features

**1. Lossless Parsing with Full AST Access**

Every high-level type provides `GoLow()` for source location access:

```go
schema := v3Model.Model.Components.Schemas.GetOrZero("Pet").Schema()
lowSchema := schema.GoLow()

// Access exact source positions
line := lowSchema.Type.KeyNode.Line
column := lowSchema.Type.KeyNode.Column
```

oastools offers optional source maps via `WithSourceMap(true)` but with less granular access.

**2. HTTP Request/Response Validation**

**oastools**: Via httpvalidator package (as of this PR):

```go
v, _ := httpvalidator.New(parsed)

// Validate incoming requests
result, _ := v.ValidateRequest(request)

// Validate responses (middleware-friendly)
result, _ := v.ValidateResponseData(request, statusCode, headers, body)
```

**libopenapi**: Via libopenapi-validator:

```go
validator, _ := validator.NewValidator(document)

// Validate incoming requests against the spec
valid, errs := validator.ValidateHttpRequest(request)

// Validate responses
valid, errs := validator.ValidateHttpResponse(request, response)
```

**3. RenderAndReload Pattern**

Synchronize high-level changes back to low-level model:

```go
// Make changes to high-level model
v3Model.Model.Info.Title = "New Title"

// Re-render and reload to sync line numbers
rawBytes, newDoc, newModel, errs := document.RenderAndReload()
```

**4. Configurable Breaking Change Rules**

Customize what constitutes a breaking change:

```go
customRules := &model.BreakingRulesConfig{
    Operation: &model.OperationRules{
        OperationID: &model.BreakingChangeRule{
            Modified: boolPtr(false), // Make operationId changes non-breaking
        },
    },
}
model.SetActiveBreakingRulesConfig(customRules)
```

---

## Testing and Quality

### oastools

- **3,000+ tests** across all packages
- **90+ benchmarks** with I/O-isolated variants for accurate regression detection
- Real-world API corpus: Discord, Stripe, GitHub, Microsoft Graph (34MB), DigitalOcean, Google Maps, Asana, GitLab, Slack
- Comprehensive coverage with positive, negative, and edge cases
- CI/CD via GitHub Actions with `make check` (fmt, lint, test, tidy)

### libopenapi

- Extensive test suite (specific count not published)
- Test specs include Petstore, Stripe, and complex circular reference cases
- 47 contributors, 765 GitHub stars
- Powers production tools: vacuum, openapi-changes, wiretap
- Used by Speakeasy, Mattermost, Scalar, APIdeck

---

## API Design Patterns

### oastools: Dual API Style

**Functional Options (convenience):**
```go
result, _ := parser.ParseWithOptions(
    parser.WithFilePath("api.yaml"),
    parser.WithResolveRefs(true),
    parser.WithValidateStructure(true),
)
```

**Struct-Based (reusable instances):**
```go
p := parser.New(
    parser.WithResolveRefs(true),
)
result1, _ := p.Parse("api1.yaml", false, true)
result2, _ := p.Parse("api2.yaml", false, true)
```

### libopenapi: Document-Centric

```go
// Parse document
document, err := libopenapi.NewDocument(specBytes)

// Build version-specific model
v3Model, errs := document.BuildV3Model()

// Access model
for name, schema := range v3Model.Model.Components.Schemas.FromOldest() {
    fmt.Printf("Schema: %s\n", name)
}
```

---

## Benchmark Comparison Guide

The following benchmark suite enables direct performance comparison. This is designed for execution in a Claude Code session.

### Setup Requirements

```bash
# Create benchmark directory
mkdir -p /tmp/openapi-benchmark/testdata
cd /tmp/openapi-benchmark

# Download test specifications
curl -o testdata/petstore.yaml https://petstore3.swagger.io/api/v3/openapi.yaml
curl -o testdata/stripe.yaml https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.yaml

# Initialize module
go mod init benchmark
go get github.com/erraggy/oastools@latest
go get github.com/pb33f/libopenapi@latest
```

### Benchmark Code

Create `benchmark_test.go`:

```go
package benchmark

import (
	"os"
	"testing"

	// oastools imports
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
	"github.com/erraggy/oastools/differ"

	// libopenapi imports
	"github.com/pb33f/libopenapi"
	libvalidator "github.com/pb33f/libopenapi-validator"
)

// Test data loaded once at package init
var (
	petstoreSpec []byte
	stripeSpec   []byte
)

func init() {
	var err error
	petstoreSpec, err = os.ReadFile("testdata/petstore.yaml")
	if err != nil {
		panic("Missing testdata/petstore.yaml")
	}
	stripeSpec, err = os.ReadFile("testdata/stripe.yaml")
	if err != nil {
		panic("Missing testdata/stripe.yaml - download from github.com/stripe/openapi")
	}
}

// ==================== PARSING BENCHMARKS ====================

// oastools parsing - uses Go 1.24+ b.Loop()
func BenchmarkParse_oastools_Petstore(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(petstoreSpec)))
	for b.Loop() {
		result, err := parser.ParseWithOptions(parser.WithBytes(petstoreSpec))
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

func BenchmarkParse_oastools_Stripe(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(stripeSpec)))
	for b.Loop() {
		result, err := parser.ParseWithOptions(parser.WithBytes(stripeSpec))
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

// libopenapi parsing
func BenchmarkParse_libopenapi_Petstore(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(petstoreSpec)))
	for b.Loop() {
		doc, err := libopenapi.NewDocument(petstoreSpec)
		if err != nil {
			b.Fatal(err)
		}
		_, errs := doc.BuildV3Model()
		if len(errs) > 0 {
			b.Fatal(errs)
		}
	}
}

func BenchmarkParse_libopenapi_Stripe(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(stripeSpec)))
	for b.Loop() {
		doc, err := libopenapi.NewDocument(stripeSpec)
		if err != nil {
			b.Fatal(err)
		}
		_, errs := doc.BuildV3Model()
		if len(errs) > 0 {
			b.Fatal(errs)
		}
	}
}

// ==================== VALIDATION BENCHMARKS ====================

// Pre-parsed validation (oastools parse-once pattern)
func BenchmarkValidateParsed_oastools(b *testing.B) {
	// Parse once outside measurement loop
	result, err := parser.ParseWithOptions(parser.WithBytes(petstoreSpec))
	if err != nil {
		b.Fatal(err)
	}
	
	v := validator.New(validator.WithIncludeWarnings(true))
	
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		vResult, err := v.ValidateParsed(result)
		if err != nil {
			b.Fatal(err)
		}
		_ = vResult
	}
}

// Pre-parsed validation (libopenapi)
func BenchmarkValidateParsed_libopenapi(b *testing.B) {
	// Parse once outside measurement loop
	doc, err := libopenapi.NewDocument(petstoreSpec)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		v, errs := libvalidator.NewValidator(doc)
		if len(errs) > 0 {
			b.Fatal(errs)
		}
		valid, validationErrs := v.ValidateDocument()
		_ = valid
		_ = validationErrs
	}
}

// ==================== DIFF BENCHMARKS ====================

// Diff with pre-parsed documents (oastools)
func BenchmarkDiffParsed_oastools(b *testing.B) {
	source, _ := parser.ParseWithOptions(parser.WithBytes(petstoreSpec))
	target, _ := parser.ParseWithOptions(parser.WithBytes(petstoreSpec))
	
	d := differ.New()
	
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		result, err := d.DiffParsed(source, target)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

// Diff with pre-parsed documents (libopenapi)
func BenchmarkDiffParsed_libopenapi(b *testing.B) {
	original, _ := libopenapi.NewDocument(petstoreSpec)
	modified, _ := libopenapi.NewDocument(petstoreSpec)
	
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		changes, err := libopenapi.CompareDocuments(original, modified)
		if err != nil {
			b.Fatal(err)
		}
		_ = changes
	}
}

// ==================== MEMORY PROFILE ====================

func BenchmarkMemoryProfile_oastools_Stripe(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(stripeSpec)))
	for b.Loop() {
		result, _ := parser.ParseWithOptions(parser.WithBytes(stripeSpec))
		_ = result
	}
}

func BenchmarkMemoryProfile_libopenapi_Stripe(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(stripeSpec)))
	for b.Loop() {
		doc, _ := libopenapi.NewDocument(stripeSpec)
		model, _ := doc.BuildV3Model()
		_ = model
	}
}
```

### Execution Commands

```bash
# Run all benchmarks with memory stats
go test -bench=. -benchmem -count=5 -timeout=30m | tee results.txt

# Run parsing benchmarks only
go test -bench=BenchmarkParse -benchmem -count=5

# Run with CPU profiling
go test -bench=BenchmarkParse_oastools_Stripe -cpuprofile=cpu.prof -benchmem
go tool pprof -http=:8080 cpu.prof

# Run with memory profiling
go test -bench=BenchmarkMemoryProfile -memprofile=mem.prof -benchmem
go tool pprof -http=:8080 mem.prof

# Compare results with benchstat
go install golang.org/x/perf/cmd/benchstat@latest
benchstat results.txt
```

### Expected Output Format

```
BenchmarkParse_oastools_Petstore-10      8234   142156 ns/op   125.4 MB/s   202678 B/op   2128 allocs/op
BenchmarkParse_libopenapi_Petstore-10    5421   223456 ns/op    79.8 MB/s   312456 B/op   3456 allocs/op
```

---

## Feature Gap Summary: What oastools Could Add

Based on this analysis, the following libopenapi features are candidates for oastools implementation:

### High Priority (Gap in oastools) - ADDRESSED IN THIS PR

1. ~~**HTTP request/response validation**~~ - ✅ Implemented via `httpvalidator` package in this PR
2. ~~**Configurable breaking change rules**~~ - ✅ Implemented via `differ` rules configuration in this PR

### Lower Priority (Different design trade-offs)

1. **Lossless parsing with GoLow()** - libopenapi preserves exact line/column for every key/value; oastools has opt-in source maps
2. **RenderAndReload pattern** - for syncing high-level model changes back to low-level AST

---

## Ecosystem Comparison

### oastools Ecosystem

- **CLI Tool:** `oastools` binary with all functionality
- **Documentation:** GitHub Pages at erraggy.github.io/oastools
- **Installation:** `go install`, Homebrew tap, pre-built binaries

### libopenapi Ecosystem

- **vacuum:** OpenAPI linter (Spectral-compatible)
- **openapi-changes:** Breaking change detection CLI
- **wiretap:** Live API compliance validation
- **libopenapi-validator:** HTTP request/response validation

The pb33f ecosystem is more fragmented (multiple repositories) but offers specialized tools. oastools is more integrated (single repository) but less mature in the linting space.

---

## Summary

This analysis originally identified two features present in libopenapi that oastools lacked:

1. ~~**HTTP request/response validation**~~ - ✅ **NOW IMPLEMENTED** via `httpvalidator` package (this PR)
2. ~~**Configurable breaking change rules**~~ - ✅ **NOW IMPLEMENTED** via `differ` rules configuration (this PR)

**Update**: Both identified gaps have been addressed in this PR. The remaining architectural differences (lossless parsing, RenderAndReload) reflect different design priorities rather than missing features.
