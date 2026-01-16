# Design: Parser HTTP Client Configuration

> Validated: 2026-01-15

## Overview

Extend the `parser` package to accept a custom `*http.Client` for fetching URLs, enabling users to configure timeouts, proxies, transport settings, and authentication.

## Motivation

Currently, the parser creates a new `http.Client` internally in `fetchURL()` with hardcoded settings (30-second timeout, optional TLS skip). Users cannot:

- Configure custom timeouts for slow networks
- Use corporate proxies or custom transports
- Add authentication headers via custom transports
- Inject mock clients for testing
- Reuse connection pools across parse operations
- Apply rate limiting or circuit breakers

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Type | `*http.Client` (concrete) | Familiar to Go devs, stdlib convention, users can still mock via custom `Transport` |
| Conflict handling | Warn log | When both `HTTPClient` and `InsecureSkipVerify` are set, warn and ignore `InsecureSkipVerify` |
| Documentation | Full | All 4 docs: doc.go, example_test.go, README.md, developer-guide.md |

## API Surface

### New Option Function

```go
// WithHTTPClient sets a custom HTTP client for fetching URLs.
// When set, the client is used as-is for all HTTP requests.
// The InsecureSkipVerify option is ignored when a custom client is provided
// (configure TLS settings on your client's transport instead).
//
// If the client is nil, this option has no effect (default client is used).
func WithHTTPClient(client *http.Client) Option
```

### Updated Parser Struct

```go
type Parser struct {
    // ... existing fields ...
    UserAgent  string
    HTTPClient *http.Client  // NEW: Custom HTTP client for URL fetching
    Logger     Logger
    // ... rest of fields ...
}
```

### Usage Examples

```go
// Functional options API
result, err := parser.ParseWithOptions(
    parser.WithFilePath("https://api.example.com/openapi.yaml"),
    parser.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
)

// Struct-based API
p := parser.New()
p.HTTPClient = &http.Client{Timeout: 60 * time.Second}
result, err := p.Parse("https://api.example.com/openapi.yaml")
```

## Behavior

- When `HTTPClient` is `nil` (default): Creates internal client with 30s timeout (current behavior)
- When `HTTPClient` is set: Uses provided client as-is
- When both `HTTPClient` and `InsecureSkipVerify` are set: **Warns** and ignores `InsecureSkipVerify`
- User-Agent header is still applied regardless of client source

## Implementation

### Phase 1: Core Changes

**File: `parser/parser.go`**

1. Add `httpClient *http.Client` field to `parseConfig` struct
2. Add `HTTPClient *http.Client` field to `Parser` struct (after `UserAgent`)
3. Create `WithHTTPClient(client *http.Client) Option` function
4. Update `ParseWithOptions` to transfer `cfg.httpClient` to `Parser.HTTPClient`
5. Modify `fetchURL` to use custom client when provided:

```go
func (p *Parser) fetchURL(urlStr string) ([]byte, string, error) {
    var client *http.Client

    if p.HTTPClient != nil {
        client = p.HTTPClient
        if p.InsecureSkipVerify {
            p.log().Warn("InsecureSkipVerify ignored when HTTPClient provided; configure TLS on your client's transport")
        }
    } else {
        // existing client creation logic (unchanged)
    }
    // ... rest unchanged ...
}
```

### Phase 2: Testing

**File: `parser/parser_options_test.go`**

| Test | Purpose |
|------|---------|
| `TestWithHTTPClient/sets_client_in_config` | Verify non-nil client is stored |
| `TestWithHTTPClient/accepts_nil_client` | Verify nil is accepted without error |
| `TestParseWithOptions_HTTPClient/uses_custom_client` | httptest.Server verifies request received |
| `TestParseWithOptions_HTTPClient/custom_timeout_respected` | Short timeout + slow server = error |
| `TestParseWithOptions_HTTPClient/custom_transport_used` | roundTripperFunc mock |
| `TestParseWithOptions_HTTPClient/user_agent_applied` | Verify User-Agent header |
| `TestParseWithOptions_HTTPClient_InsecureInteraction` | Verify warning logged |

**File: `parser/parser_url_test.go`**

| Test | Purpose |
|------|---------|
| `TestParser_Parse_WithHTTPClient` | Verify struct API respects HTTPClient field |

**File: `parser/parser_bench_test.go`**

| Benchmark | Purpose |
|-----------|---------|
| `BenchmarkParseURL_CustomClient` | Reusable client with connection pooling |
| `BenchmarkParseURL_DefaultClient` | Baseline (new client each call) |

### Phase 3: Documentation

1. **`parser/doc.go`** — Add "HTTP Client Configuration" section with examples
2. **`parser/example_test.go`** — Add `ExampleWithHTTPClient()` and `ExampleWithHTTPClient_proxy()`
3. **`README.md`** — Add HTTP client configuration section
4. **`docs/developer-guide.md`** — Add detailed HTTP client documentation

### Phase 4: Validation

```bash
make check  # Full test suite, lint, coverage
```

## Files Modified

| File | Changes |
|------|---------|
| `parser/parser.go` | Add `HTTPClient` field, `WithHTTPClient` option, update `fetchURL` |
| `parser/parser_options_test.go` | Unit + integration tests |
| `parser/parser_url_test.go` | Struct API test |
| `parser/parser_bench_test.go` | Benchmarks |
| `parser/doc.go` | HTTP Client Configuration section |
| `parser/example_test.go` | Runnable examples |
| `README.md` | HTTP client examples |
| `docs/developer-guide.md` | Detailed HTTP client documentation |

## Risk Assessment

| Risk | Level | Mitigation |
|------|-------|------------|
| Backward compatibility | Low | Nil client preserves existing behavior |
| API surface | Low | Single new option function |
| TLS configuration confusion | Medium | Warn log when both options set, clear documentation |
