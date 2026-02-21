# Why oastools?

oastools is designed around three principles: **minimal dependencies**, **production-grade quality**, and **performance**. This page explains what that means in practice.

## 📦 Minimal Dependencies

```text
github.com/erraggy/oastools
├── go.yaml.in/yaml/v4                    (YAML parsing)
├── golang.org/x/text                     (Title casing)
├── golang.org/x/tools                    (Code generation — imports analysis)
└── github.com/modelcontextprotocol/go-sdk (MCP server)
```

Unlike many OpenAPI tools that pull in dozens of transitive dependencies, oastools is designed to be self-contained. The `stretchr/testify` dependency is test-only and not included in your production builds.

## ✅ Battle-Tested Quality

The entire toolchain is validated against a corpus of 10 real-world production APIs:

| Domain          | APIs                                    |
|-----------------|-----------------------------------------|
| FinTech         | Stripe, Plaid                           |
| Developer Tools | GitHub, DigitalOcean                    |
| Communications  | Discord (OAS 3.1)                       |
| Enterprise      | Microsoft Graph (34MB, 18k+ operations) |
| Location        | Google Maps                             |
| Public          | US National Weather Service             |
| Reference       | Petstore (OAS 2.0)                      |
| Productivity    | Asana                                   |

This corpus spans OAS 2.0 through 3.1, JSON and YAML formats, and document sizes from 20KB to 34MB.

## ⚡ Performance

Pre-parsed workflows eliminate redundant parsing when processing multiple operations:

| Method             | Speedup      |
|--------------------|--------------|
| `ValidateParsed()` | 31x faster   |
| `ConvertParsed()`  | 47x faster   |
| `JoinParsed()`     | 150x faster  |
| `DiffParsed()`     | 81x faster   |
| `FixParsed()`      | ~60x faster  |
| `ApplyParsed()`    | ~11x faster  |

JSON marshaling is optimized for 25-32% better performance with 29-37% fewer allocations. See the [whitepaper performance section](whitepaper.md#17-performance-analysis) for detailed analysis.

## 🔒 Type-Safe Document Cloning

All parser types include generated `DeepCopy()` methods for safe document mutation. Unlike JSON marshal/unmarshal approaches used by other tools, oastools provides:

- **Type preservation** — Polymorphic fields maintain their actual types (e.g., `Schema.Type` as `string` vs `[]string` for OAS 3.1)
- **Version-aware copying** — Handles OAS version differences correctly (`ExclusiveMinimum` as bool in 3.0 vs number in 3.1)
- **Extension preservation** — All `x-*` extension fields are deep copied
- **Performance** — Direct struct copying without serialization overhead

```go
// Safe mutation without affecting the original
copy := result.OAS3Document.DeepCopy()
copy.Info.Title = "Modified API"
```

All OAS types also provide `Equals()` methods for structural comparison.

## 🛡️ Enterprise-Grade Error Handling

The `oaserrors` package provides structured error types that work with Go's standard `errors.Is()` and `errors.As()`:

```go
import (
    "errors"
    "github.com/erraggy/oastools/oaserrors"
    "github.com/erraggy/oastools/parser"
)

result, err := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
if err != nil {
    // Check error category with errors.Is()
    if errors.Is(err, oaserrors.ErrPathTraversal) {
        log.Fatal("Security: path traversal attempt blocked")
    }

    // Extract details with errors.As()
    var refErr *oaserrors.ReferenceError
    if errors.As(err, &refErr) {
        log.Printf("Failed to resolve: %s (type: %s)", refErr.Ref, refErr.RefType)
    }
}
```

Error types include `ParseError`, `ReferenceError`, `ValidationError`, `ResourceLimitError`, `ConversionError`, and `ConfigError`.

## ⚙️ Configurable Resource Limits

Protect against resource exhaustion with configurable limits:

```go
result, err := parser.ParseWithOptions(
    parser.WithFilePath("api.yaml"),
    parser.WithMaxRefDepth(50),           // Max $ref nesting (default: 100)
    parser.WithMaxCachedDocuments(200),   // Max cached external docs (default: 100)
    parser.WithMaxFileSize(20*1024*1024), // Max file size in bytes (default: 10MB)
)
```

## 🌐 HTTP Client Configuration

For advanced scenarios like custom timeouts, proxies, or authentication:

```go
// Custom timeout for slow networks
client := &http.Client{Timeout: 120 * time.Second}
result, _ := parser.ParseWithOptions(
    parser.WithFilePath("https://api.example.com/openapi.yaml"),
    parser.WithHTTPClient(client),
)
```

```go
// Corporate proxy
proxyURL, _ := url.Parse("http://proxy.corp:8080")
client := &http.Client{
    Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
}
result, _ := parser.ParseWithOptions(
    parser.WithFilePath("https://internal-api.corp/spec.yaml"),
    parser.WithHTTPClient(client),
)
```

When a custom client is provided, `InsecureSkipVerify` is ignored — configure TLS on your client's transport instead.
