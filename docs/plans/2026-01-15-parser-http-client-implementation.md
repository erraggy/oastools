# Parser WithHTTPClient Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `WithHTTPClient(*http.Client)` option to the parser package, enabling custom HTTP client configuration for URL fetching.

**Architecture:** Extend the existing functional options pattern by adding an `HTTPClient` field to both `Parser` and `parseConfig` structs, creating a `WithHTTPClient` option function, and modifying `fetchURL` to use the custom client when provided. Warn (not error) when both `HTTPClient` and `InsecureSkipVerify` are set.

**Tech Stack:** Go stdlib `net/http`, `net/http/httptest` for testing

---

## Task 1: Add HTTPClient Field to parseConfig

**Files:**
- Modify: `parser/parser.go:556-557`

**Step 1: Add httpClient field after userAgent**

In `parser/parser.go`, find line 556-557:
```go
	userAgent          string
	logger             Logger
```

Change to:
```go
	userAgent  string
	httpClient *http.Client
	logger     Logger
```

**Step 2: Verify build succeeds**

Run: `go build ./parser`
Expected: Success (no errors)

**Step 3: Commit**

```bash
git add parser/parser.go
git commit -m "feat(parser): add httpClient field to parseConfig"
```

---

## Task 2: Add HTTPClient Field to Parser Struct

**Files:**
- Modify: `parser/parser.go:39-42`

**Step 1: Add HTTPClient field after UserAgent**

In `parser/parser.go`, find lines 39-42:
```go
	UserAgent string
	// Logger is the structured logger for debug output
	// If nil, logging is disabled (default)
	Logger Logger
```

Change to:
```go
	UserAgent string
	// HTTPClient is the HTTP client used for fetching URLs.
	// If nil, a default client with 30-second timeout is created.
	// When set, InsecureSkipVerify is ignored (configure TLS on your client's transport).
	HTTPClient *http.Client
	// Logger is the structured logger for debug output
	// If nil, logging is disabled (default)
	Logger Logger
```

**Step 2: Verify build succeeds**

Run: `go build ./parser`
Expected: Success (no errors)

**Step 3: Commit**

```bash
git add parser/parser.go
git commit -m "feat(parser): add HTTPClient field to Parser struct"
```

---

## Task 3: Write Failing Test for WithHTTPClient Option

**Files:**
- Modify: `parser/parser_options_test.go`

**Step 1: Write the failing test**

Add at end of file:
```go
// TestWithHTTPClient tests the WithHTTPClient option
func TestWithHTTPClient(t *testing.T) {
	t.Run("sets client in config", func(t *testing.T) {
		customClient := &http.Client{Timeout: 60 * time.Second}
		cfg := &parseConfig{}
		opt := WithHTTPClient(customClient)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Same(t, customClient, cfg.httpClient)
	})

	t.Run("accepts nil client", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithHTTPClient(nil)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Nil(t, cfg.httpClient)
	})
}
```

Add to imports at top of file:
```go
	"time"
```

**Step 2: Run test to verify it fails**

Run: `go test ./parser -run TestWithHTTPClient -v`
Expected: FAIL with "undefined: WithHTTPClient"

---

## Task 4: Implement WithHTTPClient Option

**Files:**
- Modify: `parser/parser.go:713` (after WithUserAgent)

**Step 1: Add WithHTTPClient function**

After line 713 (end of WithUserAgent), add:
```go

// WithHTTPClient sets a custom HTTP client for fetching URLs.
// When set, the client is used as-is for all HTTP requests.
// The InsecureSkipVerify option is ignored when a custom client is provided
// (configure TLS settings on your client's transport instead).
//
// If the client is nil, this option has no effect (default client is used).
//
// Example with custom timeout:
//
//	client := &http.Client{Timeout: 60 * time.Second}
//	result, err := parser.ParseWithOptions(
//	    parser.WithFilePath("https://example.com/api.yaml"),
//	    parser.WithHTTPClient(client),
//	)
//
// Example with proxy:
//
//	proxyURL, _ := url.Parse("http://proxy.example.com:8080")
//	client := &http.Client{
//	    Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
//	}
//	result, err := parser.ParseWithOptions(
//	    parser.WithFilePath("https://internal.corp/api.yaml"),
//	    parser.WithHTTPClient(client),
//	)
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *parseConfig) error {
		cfg.httpClient = client
		return nil
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./parser -run TestWithHTTPClient -v`
Expected: PASS

**Step 3: Commit**

```bash
git add parser/parser.go parser/parser_options_test.go
git commit -m "feat(parser): add WithHTTPClient option function"
```

---

## Task 5: Transfer HTTPClient in ParseWithOptions

**Files:**
- Modify: `parser/parser.go:595-596`

**Step 1: Add HTTPClient transfer**

In `parser/parser.go`, find lines 595-596:
```go
		UserAgent:          cfg.userAgent,
		Logger:             cfg.logger,
```

Change to:
```go
		UserAgent:          cfg.userAgent,
		HTTPClient:         cfg.httpClient,
		Logger:             cfg.logger,
```

**Step 2: Verify build succeeds**

Run: `go build ./parser`
Expected: Success

**Step 3: Commit**

```bash
git add parser/parser.go
git commit -m "feat(parser): transfer HTTPClient from config to Parser"
```

---

## Task 6: Write Failing Integration Test

**Files:**
- Modify: `parser/parser_options_test.go`

**Step 1: Write the failing integration test**

Add at end of file:
```go
// TestParseWithOptions_HTTPClient tests custom HTTP client integration
func TestParseWithOptions_HTTPClient(t *testing.T) {
	t.Run("uses custom client for URL parsing", func(t *testing.T) {
		requestReceived := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestReceived = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`))
		}))
		defer server.Close()

		customClient := &http.Client{Timeout: 5 * time.Second}
		result, err := ParseWithOptions(
			WithFilePath(server.URL),
			WithHTTPClient(customClient),
		)

		require.NoError(t, err)
		assert.True(t, requestReceived)
		assert.Equal(t, "3.0.0", result.Version)
	})

	t.Run("custom client timeout is respected", func(t *testing.T) {
		// Server that delays response longer than client timeout
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`))
		}))
		defer server.Close()

		// Client with very short timeout
		shortTimeoutClient := &http.Client{Timeout: 50 * time.Millisecond}
		_, err := ParseWithOptions(
			WithFilePath(server.URL),
			WithHTTPClient(shortTimeoutClient),
		)

		require.Error(t, err)
		// Error message varies by Go version, just check it's a timeout-related error
		assert.True(t, strings.Contains(err.Error(), "deadline") || strings.Contains(err.Error(), "timeout"))
	})
}
```

**Step 2: Run test to verify timeout test fails (client not used yet)**

Run: `go test ./parser -run TestParseWithOptions_HTTPClient -v`
Expected: First test PASS (default client works), second test FAIL (timeout not respected because custom client not used in fetchURL)

---

## Task 7: Modify fetchURL to Use Custom Client

**Files:**
- Modify: `parser/parser.go:329-347`

**Step 1: Modify fetchURL to check HTTPClient**

In `parser/parser.go`, find lines 329-347:
```go
func (p *Parser) fetchURL(urlStr string) ([]byte, string, error) {
	// Create HTTP client with timeout
	// Configure TLS if InsecureSkipVerify is enabled
	var client *http.Client
	if p.InsecureSkipVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // User explicitly requested insecure mode
			},
		}
		client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	} else {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
```

Change to:
```go
func (p *Parser) fetchURL(urlStr string) ([]byte, string, error) {
	// Create HTTP client with timeout
	// Use custom client if provided, otherwise create default
	var client *http.Client
	if p.HTTPClient != nil {
		client = p.HTTPClient
		if p.InsecureSkipVerify {
			p.log().Warn("InsecureSkipVerify ignored when HTTPClient provided; configure TLS on your client's transport")
		}
	} else if p.InsecureSkipVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // User explicitly requested insecure mode
			},
		}
		client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	} else {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./parser -run TestParseWithOptions_HTTPClient -v`
Expected: PASS (both tests)

**Step 3: Run all parser tests to ensure no regressions**

Run: `go test ./parser -v -count=1`
Expected: PASS

**Step 4: Commit**

```bash
git add parser/parser.go
git commit -m "feat(parser): use custom HTTPClient in fetchURL when provided"
```

---

## Task 8: Test Custom Transport

**Files:**
- Modify: `parser/parser_options_test.go`

**Step 1: Add roundTripperFunc helper and test**

Add at end of file:
```go
// roundTripperFunc is a helper for testing custom transports
type roundTripperFunc struct {
	fn func(*http.Request) (*http.Response, error)
}

func (r *roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.fn(req)
}

func TestParseWithOptions_HTTPClient_CustomTransport(t *testing.T) {
	t.Run("custom transport is used", func(t *testing.T) {
		transportUsed := false
		customTransport := &roundTripperFunc{
			fn: func(req *http.Request) (*http.Response, error) {
				transportUsed = true
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`)),
					Header: make(http.Header),
				}, nil
			},
		}
		customClient := &http.Client{Transport: customTransport}

		result, err := ParseWithOptions(
			WithFilePath("https://example.com/api.yaml"),
			WithHTTPClient(customClient),
		)

		require.NoError(t, err)
		assert.True(t, transportUsed, "Custom transport should have been used")
		assert.Equal(t, "3.0.0", result.Version)
	})

	t.Run("user agent still applied with custom client", func(t *testing.T) {
		var receivedUA string
		customTransport := &roundTripperFunc{
			fn: func(req *http.Request) (*http.Response, error) {
				receivedUA = req.Header.Get("User-Agent")
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`)),
					Header: make(http.Header),
				}, nil
			},
		}
		customClient := &http.Client{Transport: customTransport}

		_, err := ParseWithOptions(
			WithFilePath("https://example.com/api.yaml"),
			WithHTTPClient(customClient),
			WithUserAgent("custom-agent/1.0"),
		)

		require.NoError(t, err)
		assert.Equal(t, "custom-agent/1.0", receivedUA)
	})
}
```

Add `"io"` to imports at top of file.

**Step 2: Run tests to verify they pass**

Run: `go test ./parser -run TestParseWithOptions_HTTPClient_CustomTransport -v`
Expected: PASS

**Step 3: Commit**

```bash
git add parser/parser_options_test.go
git commit -m "test(parser): add custom transport tests for HTTPClient"
```

---

## Task 9: Test InsecureSkipVerify Warning

**Files:**
- Modify: `parser/parser_options_test.go`

**Step 1: Add test for warning interaction**

Add at end of file:
```go
func TestParseWithOptions_HTTPClient_InsecureInteraction(t *testing.T) {
	t.Run("warns when both HTTPClient and InsecureSkipVerify set", func(t *testing.T) {
		var logMessages []string
		mockLogger := &mockTestLogger{
			warnFunc: func(msg string, args ...any) {
				logMessages = append(logMessages, msg)
			},
		}

		customTransport := &roundTripperFunc{
			fn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`)),
					Header: make(http.Header),
				}, nil
			},
		}
		customClient := &http.Client{Transport: customTransport}

		_, err := ParseWithOptions(
			WithFilePath("https://example.com/api.yaml"),
			WithHTTPClient(customClient),
			WithInsecureSkipVerify(true),
			WithLogger(mockLogger),
		)

		require.NoError(t, err)
		require.Len(t, logMessages, 1)
		assert.Contains(t, logMessages[0], "InsecureSkipVerify ignored")
	})
}

// mockTestLogger implements Logger for testing
type mockTestLogger struct {
	warnFunc func(msg string, args ...any)
}

func (m *mockTestLogger) Debug(msg string, args ...any) {}
func (m *mockTestLogger) Info(msg string, args ...any)  {}
func (m *mockTestLogger) Warn(msg string, args ...any) {
	if m.warnFunc != nil {
		m.warnFunc(msg, args...)
	}
}
func (m *mockTestLogger) Error(msg string, args ...any) {}
```

**Step 2: Run test to verify it passes**

Run: `go test ./parser -run TestParseWithOptions_HTTPClient_InsecureInteraction -v`
Expected: PASS

**Step 3: Commit**

```bash
git add parser/parser_options_test.go
git commit -m "test(parser): verify InsecureSkipVerify warning with HTTPClient"
```

---

## Task 10: Test Struct-Based API

**Files:**
- Modify: `parser/parser_url_test.go`

**Step 1: Read current file to find insertion point**

Find the end of the file to add the test.

**Step 2: Add struct API test**

Add at end of `parser/parser_url_test.go`:
```go
func TestParser_Parse_WithHTTPClient(t *testing.T) {
	t.Run("struct API uses HTTPClient field", func(t *testing.T) {
		requestReceived := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestReceived = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`))
		}))
		defer server.Close()

		p := New()
		p.HTTPClient = &http.Client{Timeout: 5 * time.Second}

		result, err := p.Parse(server.URL)

		require.NoError(t, err)
		assert.True(t, requestReceived)
		assert.Equal(t, "3.0.0", result.Version)
	})
}
```

Ensure imports include `"time"`.

**Step 3: Run test to verify it passes**

Run: `go test ./parser -run TestParser_Parse_WithHTTPClient -v`
Expected: PASS

**Step 4: Commit**

```bash
git add parser/parser_url_test.go
git commit -m "test(parser): verify struct API respects HTTPClient field"
```

---

## Task 11: Add Benchmarks

**Files:**
- Modify: `parser/parser_bench_test.go`

**Step 1: Add benchmarks at end of file**

```go
func BenchmarkParseURL_CustomClient(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(petstoreYAMLData)
	}))
	defer server.Close()

	// Reusable client (simulates real-world usage with connection pooling)
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := ParseWithOptions(
			WithFilePath(server.URL),
			WithHTTPClient(client),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseURL_DefaultClient(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(petstoreYAMLData)
	}))
	defer server.Close()

	b.ResetTimer()
	for b.Loop() {
		_, err := ParseWithOptions(
			WithFilePath(server.URL),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}
```

Ensure imports include `"net/http"`, `"net/http/httptest"`, and `"time"`.

**Step 2: Run benchmarks to verify they work**

Run: `go test ./parser -bench=BenchmarkParseURL -benchmem -count=1`
Expected: Benchmarks run successfully

**Step 3: Commit**

```bash
git add parser/parser_bench_test.go
git commit -m "bench(parser): add HTTPClient vs default client benchmarks"
```

---

## Task 12: Update doc.go

**Files:**
- Modify: `parser/doc.go:49-50`

**Step 1: Add HTTP Client Configuration section**

In `parser/doc.go`, after line 50 (after the HTTP refs paragraph), add:

```go
//
// # HTTP Client Configuration
//
// By default, the parser creates an HTTP client with a 30-second timeout for
// fetching remote specifications. For advanced use cases, provide a custom
// HTTP client:
//
//	// Custom timeout for slow networks
//	client := &http.Client{Timeout: 60 * time.Second}
//	result, err := parser.ParseWithOptions(
//	    parser.WithFilePath("https://api.example.com/openapi.yaml"),
//	    parser.WithHTTPClient(client),
//	)
//
//	// Corporate proxy
//	proxyURL, _ := url.Parse("http://proxy.corp.internal:8080")
//	client := &http.Client{
//	    Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
//	}
//	result, err := parser.ParseWithOptions(
//	    parser.WithFilePath("https://internal-api.corp/spec.yaml"),
//	    parser.WithHTTPClient(client),
//	)
//
// When a custom client is provided, the InsecureSkipVerify option is ignored.
// Configure TLS settings directly on your client's transport instead.
```

**Step 2: Verify godoc renders correctly**

Run: `go doc -all github.com/erraggy/oastools/parser | grep -A5 "HTTP Client"`
Expected: Shows the new section

**Step 3: Commit**

```bash
git add parser/doc.go
git commit -m "docs(parser): add HTTP Client Configuration section to doc.go"
```

---

## Task 13: Add Example Tests

**Files:**
- Modify: `parser/example_test.go`

**Step 1: Add example functions**

Add after the existing examples (around line 80):
```go

// ExampleWithHTTPClient demonstrates using a custom HTTP client with a longer timeout.
func ExampleWithHTTPClient() {
	// Create a custom HTTP client with longer timeout for slow networks
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	result, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
		parser.WithHTTPClient(client),
	)
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Version: %s\n", result.Version)
	// Output:
	// Version: 3.0.3
}

// ExampleWithHTTPClient_proxy demonstrates configuring a proxy for corporate environments.
func ExampleWithHTTPClient_proxy() {
	// This example shows the configuration pattern for corporate proxies.
	// In a real scenario, you would use an actual proxy URL.

	// Configure client to use corporate proxy (example configuration)
	// proxyURL, _ := url.Parse("http://proxy.example.com:8080")
	// client := &http.Client{
	//     Transport: &http.Transport{
	//         Proxy: http.ProxyURL(proxyURL),
	//     },
	//     Timeout: 30 * time.Second,
	// }

	// For this example, we use a direct client
	client := &http.Client{Timeout: 30 * time.Second}

	result, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/petstore-3.0.yaml"),
		parser.WithHTTPClient(client),
	)
	if err != nil {
		log.Fatalf("failed to parse: %v", err)
	}

	fmt.Printf("Parsed: %s\n", result.Version)
	// Output:
	// Parsed: 3.0.3
}
```

Add `"time"` to imports.

**Step 2: Run examples to verify they pass**

Run: `go test ./parser -run Example -v`
Expected: PASS

**Step 3: Commit**

```bash
git add parser/example_test.go
git commit -m "docs(parser): add WithHTTPClient examples"
```

---

## Task 14: Update README.md

**Files:**
- Modify: `README.md`

**Step 1: Find Parser section and add HTTP client docs**

Search for the Parser section in README.md and add after the existing parser examples:

```markdown
### HTTP Client Configuration

For advanced scenarios like custom timeouts, proxies, or authentication:

```go
// Custom timeout for slow networks
client := &http.Client{Timeout: 120 * time.Second}
result, _ := parser.ParseWithOptions(
    parser.WithFilePath("https://api.example.com/openapi.yaml"),
    parser.WithHTTPClient(client),
)

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

When a custom client is provided, `InsecureSkipVerify` is ignored—configure TLS on your client's transport instead.
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add HTTP client configuration to README"
```

---

## Task 15: Update Developer Guide

**Files:**
- Modify: `docs/developer-guide.md`

**Step 1: Add detailed HTTP client section**

Add a new section with detailed examples for:
- Custom timeout configuration
- Proxy setup
- Authentication via custom transport (e.g., adding Authorization header)
- Connection pooling benefits
- Testing with mock transports

**Step 2: Commit**

```bash
git add docs/developer-guide.md
git commit -m "docs: add detailed HTTP client docs to developer guide"
```

---

## Task 16: Final Validation

**Step 1: Run full test suite**

Run: `go test ./parser -v -race -count=1`
Expected: All tests PASS, no race conditions

**Step 2: Run linter**

Run: `make lint`
Expected: No lint errors

**Step 3: Run full check suite**

Run: `make check`
Expected: All checks PASS

**Step 4: Verify test coverage**

Run: `go test ./parser -coverprofile=coverage.out && go tool cover -func=coverage.out | grep -E "(WithHTTPClient|fetchURL)"`
Expected: Good coverage on new code

---

## Files Modified Summary

| File | Changes |
|------|---------|
| `parser/parser.go` | Add `HTTPClient` field (Parser + parseConfig), `WithHTTPClient` option, update `fetchURL` |
| `parser/parser_options_test.go` | Unit + integration tests, custom transport tests, warning test |
| `parser/parser_url_test.go` | Struct API test |
| `parser/parser_bench_test.go` | Benchmarks |
| `parser/doc.go` | HTTP Client Configuration section |
| `parser/example_test.go` | Runnable examples |
| `README.md` | HTTP client configuration examples |
| `docs/developer-guide.md` | Detailed HTTP client documentation |
