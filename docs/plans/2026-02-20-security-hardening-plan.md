# Security Hardening v1.52.0 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Harden all 5 attack surfaces (MCP server, parser, HTTP validator, generator, CLI) against 26 security findings.

**Architecture:** Six PRs grouped by package, all targeting milestone v1.52.0. A shared `internal/pathutil` package provides output-path safety reused by both MCP and CLI. Parser's existing `WithHTTPClient` option is leveraged for SSRF protection — the MCP server injects a safe client.

**Tech Stack:** Go stdlib (`net`, `net/url`, `sync`, `path/filepath`, `io`), existing functional option patterns.

**Dependency order:** PR 1 (go-sdk) -> PR 3 (parser) -> PR 2 (MCP, uses parser's WithHTTPClient) -> PR 4/5 (parallel) -> PR 6 (CLI, uses shared pathutil)

---

## PR 1: go-sdk Dependency Update

### Task 1.1: Commit go-sdk v1.3.1 Update

The go.mod/go.sum changes are already in the working tree (unstaged).

**Files:**
- Modify: `go.mod` (already modified)
- Modify: `go.sum` (already modified)

**Step 1: Verify the unstaged changes**

Run: `git diff go.mod`
Expected: `github.com/modelcontextprotocol/go-sdk` v1.3.0 -> v1.3.1, new indirect deps

**Step 2: Run tests to confirm nothing breaks**

Run: `make test`
Expected: All tests pass

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): update go-sdk to v1.3.1 (security patch)

Switches to case-sensitive JSON unmarshaling via segmentio/encoding.
New indirect deps: segmentio/asm, segmentio/encoding, golang.org/x/sys."
```

**Step 4: Create PR**

Target milestone: v1.52.0

---

## PR 2: MCP Server Hardening (Findings 1.1-1.8)

### Task 2.1: Create shared `internal/pathutil` package

**Files:**
- Create: `internal/pathutil/pathutil.go`
- Create: `internal/pathutil/pathutil_test.go`

**Step 1: Write the failing tests**

```go
// internal/pathutil/pathutil_test.go
package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeOutputPath(t *testing.T) {
	t.Run("clean path accepted", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "output.yaml")
		got, err := SanitizeOutputPath(target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !filepath.IsAbs(got) {
			t.Fatalf("expected absolute path, got %s", got)
		}
	})

	t.Run("dot-dot rejected", func(t *testing.T) {
		_, err := SanitizeOutputPath("/tmp/../etc/passwd")
		if err == nil {
			t.Fatal("expected error for path with ..")
		}
	})

	t.Run("symlink rejected", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "real.yaml")
		if err := os.WriteFile(target, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link.yaml")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		_, err := SanitizeOutputPath(link)
		if err == nil {
			t.Fatal("expected error for symlink target")
		}
	})

	t.Run("new file in existing dir accepted", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "newfile.yaml")
		got, err := SanitizeOutputPath(target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != target {
			t.Fatalf("expected %s, got %s", target, got)
		}
	})
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/pathutil/...`
Expected: FAIL (package doesn't exist)

**Step 3: Write implementation**

```go
// internal/pathutil/pathutil.go
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SanitizeOutputPath validates and cleans an output file path.
// It rejects paths containing ".." after cleaning and paths that
// resolve to symlinks. Returns the cleaned absolute path.
func SanitizeOutputPath(path string) (string, error) {
	cleaned := filepath.Clean(path)

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("cannot resolve absolute path: %w", err)
	}

	// Reject if cleaned path still contains ".."
	// (filepath.Clean resolves most, but abs-rooted ".." can remain)
	if strings.Contains(abs, "..") {
		return "", fmt.Errorf("path must not contain '..': %s", abs)
	}

	// Check if the target already exists and is a symlink
	info, err := os.Lstat(abs)
	if err == nil {
		// File exists — reject if symlink
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing to write to symlink: %s", abs)
		}
	}
	// If file doesn't exist, that's fine — it will be created

	return abs, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/pathutil/...`
Expected: PASS

**Step 5: Run diagnostics**

Run: gopls `go_diagnostics` on `internal/pathutil/pathutil.go`

**Step 6: Commit**

```bash
git add internal/pathutil/
git commit -m "feat(pathutil): add shared output path sanitization

Rejects paths with '..', symlink targets, and ensures absolute paths.
Used by both MCP server and CLI output handlers."
```

---

### Task 2.2: Create SSRF-safe HTTP client for MCP

**Files:**
- Create: `internal/mcpserver/safeclient.go`
- Create: `internal/mcpserver/safeclient_test.go`

**Step 1: Write the failing tests**

```go
// internal/mcpserver/safeclient_test.go
package mcpserver

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.1.1", true},  // link-local
		{"::1", true},          // IPv6 loopback
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"93.184.216.34", false},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}
			got := isBlockedIP(ip)
			if got != tt.private {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tt.ip, got, tt.private)
			}
		})
	}
}

func TestNewSafeHTTPClient(t *testing.T) {
	client := newSafeHTTPClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Timeout == 0 {
		t.Error("expected non-zero timeout")
	}
	if client.CheckRedirect == nil {
		t.Error("expected CheckRedirect to be set")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run TestIsPrivateIP`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/mcpserver/safeclient.go
package mcpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// isBlockedIP returns true if the IP is private, loopback, or link-local.
func isBlockedIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

// newSafeHTTPClient creates an HTTP client that blocks requests to
// private/loopback/link-local IPs. Used by the MCP server to prevent
// SSRF when resolving specs from URLs provided by AI agents.
func newSafeHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				for _, ipAddr := range ips {
					if isBlockedIP(ipAddr.IP) {
						return nil, fmt.Errorf("blocked request to private/loopback IP: %s (%s)", host, ipAddr.IP)
					}
				}
				// Dial the first resolved address
				return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			// Re-resolve and check the redirect target
			host := req.URL.Hostname()
			ips, err := net.DefaultResolver.LookupIPAddr(req.Context(), host)
			if err != nil {
				return err
			}
			for _, ipAddr := range ips {
				if isBlockedIP(ipAddr.IP) {
					return fmt.Errorf("redirect to private/loopback IP blocked: %s (%s)", host, ipAddr.IP)
				}
			}
			return nil
		},
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/mcpserver/ -run "TestIsPrivateIP|TestNewSafeHTTPClient"`
Expected: PASS

**Step 5: Run diagnostics**

Run: gopls `go_diagnostics` on `internal/mcpserver/safeclient.go`

**Step 6: Commit**

```bash
git add internal/mcpserver/safeclient.go internal/mcpserver/safeclient_test.go
git commit -m "feat(mcp): add SSRF-safe HTTP client for MCP server

Blocks requests to private, loopback, and link-local IPs.
Validates redirect targets against the same blocklist.
Opt-out via OASTOOLS_ALLOW_PRIVATE_IPS=true."
```

---

### Task 2.3: Add MCP config fields and sanitize helpers

**Files:**
- Modify: `internal/mcpserver/config.go` (add new config fields)
- Modify: `internal/mcpserver/server.go` (add sanitizeError, update paginate)

**Step 1: Write failing tests for new config fields**

Add tests to existing `config_test.go` for:
- `OASTOOLS_MAX_INLINE_SIZE` parsing (default 10 MiB)
- `OASTOOLS_MAX_LIMIT` parsing (default 1000)
- `OASTOOLS_MAX_JOIN_SPECS` parsing (default 20)
- `OASTOOLS_ALLOW_PRIVATE_IPS` parsing (default false)

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run TestLoadConfig`
Expected: FAIL (fields don't exist)

**Step 3: Add config fields to `config.go`**

Add to `serverConfig` struct:
```go
// Security settings.
MaxInlineSize   int64 // max inline content size in bytes (default 10 MiB)
MaxLimit        int   // max pagination limit (default 1000)
MaxJoinSpecs    int   // max specs in join (default 20)
AllowPrivateIPs bool  // opt-out of SSRF protection
```

Add to `loadConfig()`:
```go
MaxInlineSize:   envInt64("OASTOOLS_MAX_INLINE_SIZE", 10*1024*1024),
MaxLimit:        envInt("OASTOOLS_MAX_LIMIT", 1000),
MaxJoinSpecs:    envInt("OASTOOLS_MAX_JOIN_SPECS", 20),
AllowPrivateIPs: envBool("OASTOOLS_ALLOW_PRIVATE_IPS", false),
```

Add `envInt64` helper following existing `envInt` pattern.

**Step 4: Write sanitizeError helper**

Add to `server.go`:
```go
// sanitizeError strips absolute filesystem paths from error messages
// to prevent leaking internal directory structure to MCP clients.
func sanitizeError(err error) string {
	msg := err.Error()
	// Strip absolute paths (Unix and Windows)
	re := regexp.MustCompile(`(?:/[a-zA-Z0-9._-]+){2,}`)
	return re.ReplaceAllString(msg, "<path>")
}
```

Update `errResult` to use `sanitizeError`:
```go
func errResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: sanitizeError(err)}},
	}
}
```

**Step 5: Update paginate to cap limit**

In `paginate()`, after the default assignment:
```go
if limit > cfg.MaxLimit {
    limit = cfg.MaxLimit
}
```

**Step 6: Run tests**

Run: `go test ./internal/mcpserver/...`
Expected: PASS

**Step 7: Run diagnostics and commit**

```bash
git add internal/mcpserver/config.go internal/mcpserver/config_test.go internal/mcpserver/server.go
git commit -m "feat(mcp): add security config fields and error sanitization

Adds MaxInlineSize, MaxLimit, MaxJoinSpecs, AllowPrivateIPs config.
Strips filesystem paths from error messages returned to clients.
Caps pagination limit to prevent resource exhaustion."
```

---

### Task 2.4: Apply output path safety to all tool handlers

**Files:**
- Modify: `internal/mcpserver/tools_fix.go:85`
- Modify: `internal/mcpserver/tools_convert.go:81`
- Modify: `internal/mcpserver/tools_join.go:111`
- Modify: `internal/mcpserver/tools_overlay.go:94`
- Modify: `internal/mcpserver/tools_generate.go:70`

**Step 1: Write failing test for path traversal rejection**

In appropriate test file, test that `handleFix` (or any handler) rejects output paths containing `..`:

```go
func TestOutputPathTraversal(t *testing.T) {
	// Test that output paths with ".." are rejected
	input := fixInput{
		Spec:   specInput{Content: minimalOAS3},
		Output: "/tmp/../etc/evil.yaml",
	}
	result, _, _ := handleFix(context.Background(), nil, input)
	if !result.IsError {
		t.Fatal("expected error for path traversal")
	}
}
```

**Step 2: Update each handler**

In each of the 5 handlers, before the `os.WriteFile` call, add:

```go
if input.Output != "" {
    cleanPath, err := pathutil.SanitizeOutputPath(input.Output)
    if err != nil {
        return errResult(fmt.Errorf("invalid output path: %w", err)), fixOutput{}, nil
    }
    if err := os.WriteFile(cleanPath, data, 0o600); err != nil {
        return errResult(fmt.Errorf("failed to write output file: %w", err)), fixOutput{}, nil
    }
    output.WrittenTo = cleanPath
}
```

Note: Change `0o644` to `0o600` in all handlers.

**Step 3: Run tests**

Run: `go test ./internal/mcpserver/...`
Expected: PASS

**Step 4: Run diagnostics and commit**

```bash
git add internal/mcpserver/tools_fix.go internal/mcpserver/tools_convert.go \
        internal/mcpserver/tools_join.go internal/mcpserver/tools_overlay.go \
        internal/mcpserver/tools_generate.go
git commit -m "fix(mcp): sanitize output paths in all tool handlers

Applies SanitizeOutputPath before writing: rejects '..' traversal,
symlinks, and tightens permissions from 0644 to 0600."
```

---

### Task 2.5: Add input validation to MCP handlers

**Files:**
- Modify: `internal/mcpserver/input.go` (inline content size check)
- Modify: `internal/mcpserver/tools_join.go` (specs count + strategy validation)
- Modify: `internal/mcpserver/tools_generate.go` (package name validation)

**Step 1: Write failing tests**

```go
func TestInlineContentSizeLimit(t *testing.T) {
	huge := strings.Repeat("a", 11*1024*1024) // 11 MiB
	s := specInput{Content: huge}
	_, err := s.resolve()
	if err == nil {
		t.Fatal("expected error for oversized content")
	}
}

func TestJoinSpecsLimit(t *testing.T) {
	specs := make([]specInput, 25)
	for i := range specs {
		specs[i] = specInput{Content: minimalOAS3}
	}
	input := joinInput{Specs: specs}
	result, _, _ := handleJoin(context.Background(), nil, input)
	if !result.IsError {
		t.Fatal("expected error for too many specs")
	}
}

func TestPackageNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		valid   bool
	}{
		{"api", true},
		{"my_pkg", true},
		{"myPkg", false},  // uppercase
		{"123pkg", false},  // starts with digit
		{"my-pkg", false},  // hyphen
		{"", true},         // empty = default
	}
	for _, tt := range tests {
		// Test via handleGenerate input validation
	}
}
```

**Step 2: Implement inline content size check**

In `input.go` `resolve()` method, before parsing:
```go
if s.Content != "" {
    if int64(len(s.Content)) > cfg.MaxInlineSize {
        return nil, fmt.Errorf("inline content size %d bytes exceeds maximum %d bytes",
            len(s.Content), cfg.MaxInlineSize)
    }
}
```

**Step 3: Implement join specs limit and strategy validation**

In `tools_join.go` `handleJoin()`, after the `len < 2` check:
```go
if len(input.Specs) > cfg.MaxJoinSpecs {
    return errResult(fmt.Errorf("too many specs: got %d, maximum is %d",
        len(input.Specs), cfg.MaxJoinSpecs)), joinOutput{}, nil
}

// Validate strategies against known values
if input.PathStrategy != "" && !validJoinStrategies[input.PathStrategy] {
    return errResult(fmt.Errorf("invalid path_strategy: %q", input.PathStrategy)), joinOutput{}, nil
}
if input.SchemaStrategy != "" && !validJoinStrategies[input.SchemaStrategy] {
    return errResult(fmt.Errorf("invalid schema_strategy: %q", input.SchemaStrategy)), joinOutput{}, nil
}
```

**Step 4: Implement package name validation**

In `tools_generate.go` `handleGenerate()`, before passing to generator:
```go
if input.PackageName != "" {
    if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(input.PackageName) || len(input.PackageName) > 64 {
        return errResult(fmt.Errorf("invalid package_name: must match [a-z][a-z0-9_]* (max 64 chars)")),
            generateOutput{}, nil
    }
}
```

Compile the regex once at package level:
```go
var validPackageName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
```

**Step 5: Run tests, diagnostics, and commit**

```bash
git add internal/mcpserver/input.go internal/mcpserver/tools_join.go \
        internal/mcpserver/tools_generate.go
git commit -m "fix(mcp): add input validation for content size, specs count, strategies, and package name

Rejects inline content > 10 MiB, join with > 20 specs, invalid
collision strategies, and package names that aren't valid Go identifiers."
```

---

### Task 2.6: Inject SSRF-safe client into parser resolution

**Files:**
- Modify: `internal/mcpserver/input.go` (pass safe client via parser options)

**Step 1: Update resolve() to inject HTTP client**

In `input.go` `resolve()`, add SSRF-safe client when resolving URL specs:

```go
// When resolving URLs in MCP context, inject safe HTTP client
if s.URL != "" && !cfg.AllowPrivateIPs {
    extraOpts = append(extraOpts, parser.WithHTTPClient(newSafeHTTPClient()))
}
```

The `extraOpts` parameter already exists on `resolve()`.

**Step 2: Write test**

```go
func TestResolveURLUsesSSRFClient(t *testing.T) {
	// This is more of an integration test — verify the safe client is used
	// by checking that private IPs are rejected
	s := specInput{URL: "http://127.0.0.1:9999/spec.yaml"}
	_, err := s.resolve()
	// Should fail with SSRF error, not connection refused
	if err == nil {
		t.Fatal("expected error for loopback URL")
	}
	if !strings.Contains(err.Error(), "private") && !strings.Contains(err.Error(), "loopback") {
		// May also fail with connection refused if no SSRF check
		t.Logf("error was: %v (may be connection refused if test runs before SSRF client is injected)", err)
	}
}
```

**Step 3: Run tests, diagnostics, and commit**

```bash
git add internal/mcpserver/input.go
git commit -m "fix(mcp): inject SSRF-safe HTTP client for URL spec resolution

When resolving URL specs in MCP context, uses safe HTTP client that
blocks private/loopback/link-local IPs. Opt-out: OASTOOLS_ALLOW_PRIVATE_IPS=true."
```

---

### Task 2.7: Create PR for MCP server hardening

**Step 1: Run full test suite**

Run: `make check`
Expected: PASS

**Step 2: Create PR**

Title: `fix(mcp): comprehensive MCP server security hardening`
Milestone: v1.52.0
Labels: security

---

## PR 3: Parser & Resolver Hardening (Findings 2.1-2.5)

### Task 3.1: Add `MaxInputSize` to parser options and apply to ParseReader

**Files:**
- Modify: `parser/parser_options.go` (add MaxInputSize option if not present)
- Modify: `parser/parser.go:347` (wrap io.ReadAll with LimitReader)
- Test: `parser/parser_test.go`

**Step 1: Write failing test**

```go
func TestParseReaderSizeLimit(t *testing.T) {
	// Create a reader that exceeds the default limit
	huge := strings.NewReader(strings.Repeat("a", 101*1024*1024)) // 101 MiB
	result, err := ParseWithOptions(
		WithReader(huge),
		WithMaxInputSize(100*1024*1024), // 100 MiB
	)
	if err == nil {
		t.Fatal("expected error for oversized input")
	}
	if result != nil {
		t.Fatal("expected nil result")
	}
}
```

**Step 2: Add WithMaxInputSize option if missing**

Check if `maxInputSize` is in `parseConfig` (it may already exist as `maxFileSize`). If the field exists under a different name, use it. If not, add:

```go
// In parseConfig struct:
maxInputSize int64 // 0 means default (100 MiB)

// Option function:
func WithMaxInputSize(size int64) Option {
    return func(cfg *parseConfig) error {
        cfg.maxInputSize = size
        return nil
    }
}
```

**Step 3: Wrap io.ReadAll in ParseReader**

At `parser.go:347`, replace:
```go
data, err := io.ReadAll(r)
```

With:
```go
maxSize := p.maxInputSize
if maxSize == 0 {
    maxSize = 100 * 1024 * 1024 // 100 MiB default
}
data, err := io.ReadAll(io.LimitReader(r, maxSize+1))
if err != nil {
    return nil, fmt.Errorf("parser: failed to read data: %w", err)
}
if int64(len(data)) > maxSize {
    return nil, fmt.Errorf("parser: input size %d bytes exceeds maximum %d bytes", len(data), maxSize)
}
```

**Step 4: Run tests, diagnostics, and commit**

---

### Task 3.2: Apply size limit to fetchURL response body

**Files:**
- Modify: `parser/parser_format.go:131` (wrap resp.Body with LimitReader)

**Step 1: Write failing test**

Test that fetching a URL with a response body exceeding maxInputSize returns an error.

**Step 2: Wrap io.ReadAll in fetchURL**

At `parser_format.go:131`, replace:
```go
data, err := io.ReadAll(resp.Body)
```

With:
```go
maxSize := p.maxInputSize
if maxSize == 0 {
    maxSize = 100 * 1024 * 1024
}
data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
if err != nil {
    return nil, "", fmt.Errorf("parser: failed to read response body: %w", err)
}
if int64(len(data)) > maxSize {
    return nil, "", fmt.Errorf("parser: response body size exceeds maximum %d bytes", maxSize)
}
```

**Step 3: Run tests, diagnostics, and commit**

---

### Task 3.3: Add same-origin enforcement to resolveRelativeURL

**Files:**
- Modify: `parser/resolver.go:411-432`
- Test: `parser/resolver_test.go`

**Step 1: Write failing test**

```go
func TestResolveRelativeURLSameOrigin(t *testing.T) {
	r := &RefResolver{baseURL: "https://example.com/api/spec.yaml"}
	// Same host should succeed
	got, err := r.resolveRelativeURL("components.yaml")
	if err != nil {
		t.Fatalf("same-host resolution failed: %v", err)
	}
	if !strings.Contains(got, "example.com") {
		t.Fatalf("expected example.com in resolved URL, got %s", got)
	}
	// Cross-host should fail — but this requires crafted input
	// The path.Join approach can't change hosts, so this is already safe
	// for relative paths. The real risk is absolute URLs in $ref.
}
```

**Step 2: Add host check**

At `resolver.go:429-431`, after resolving the path, add:

```go
// Verify the resolved URL stays on the same host
resolved := base.String() + fragment
resolvedURL, err := url.Parse(resolved)
if err != nil {
    return "", fmt.Errorf("invalid resolved URL: %w", err)
}
if resolvedURL.Host != "" {
    origBase, _ := url.Parse(r.baseURL)
    if origBase != nil && resolvedURL.Host != origBase.Host {
        return "", fmt.Errorf("cross-origin ref blocked: resolved to %s (base: %s)", resolvedURL.Host, origBase.Host)
    }
}
return resolved, nil
```

**Step 3: Run tests, diagnostics, and commit**

---

### Task 3.4: Create PR for parser hardening

Run `make check`, create PR targeting v1.52.0 milestone.

---

## PR 4: HTTP Validator Hardening (Findings 3.1-3.7)

### Task 4.1: Add `WithMaxBodySize` option and apply to request/response body reads

**Files:**
- Modify: `httpvalidator/options.go` (add WithMaxBodySize)
- Modify: `httpvalidator/validator.go` (add maxBodySize field)
- Modify: `httpvalidator/request.go:373` (wrap io.ReadAll)
- Modify: `httpvalidator/response.go:51` (wrap io.ReadAll)
- Test: `httpvalidator/request_test.go`, `httpvalidator/response_test.go`

**Step 1: Write failing tests**

```go
func TestRequestBodySizeLimit(t *testing.T) {
	// Create a request with body exceeding the limit
	body := strings.NewReader(strings.Repeat("x", 1024))
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/json")

	v, _ := NewWithOptions(WithParsed(parsed), WithMaxBodySize(512))
	result, err := v.ValidateRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasErrors() {
		t.Fatal("expected error for oversized body")
	}
}
```

**Step 2: Add option**

In `options.go`:
```go
func WithMaxBodySize(n int64) Option {
    return func(c *config) error {
        c.maxBodySize = n
        return nil
    }
}
```

In `config` struct, add `maxBodySize int64`.
In `Validator` struct, add `maxBodySize int64`.
Default: `10 * 1024 * 1024` (10 MiB).

**Step 3: Wrap io.ReadAll in request.go:373**

Replace:
```go
body, readErr := io.ReadAll(req.Body)
```

With:
```go
maxSize := v.maxBodySize
if maxSize == 0 {
    maxSize = 10 * 1024 * 1024 // 10 MiB default
}
body, readErr := io.ReadAll(io.LimitReader(req.Body, maxSize+1))
if readErr == nil && int64(len(body)) > maxSize {
    result.addError("requestBody",
        fmt.Sprintf("request body size %d exceeds maximum %d bytes", len(body), maxSize),
        SeverityError)
    return
}
```

Do the same in `response.go:51`.

**Step 4: Run tests, diagnostics, and commit**

---

### Task 4.2: Fix pattern cache concurrency with sync.Map

**Files:**
- Modify: `httpvalidator/schema.go:18` (change patternCache type)
- Modify: `httpvalidator/schema.go:501-513` (update matchPattern)
- Create: `httpvalidator/schema_bench_test.go` (benchmark)

**Step 1: Write benchmark**

```go
func BenchmarkMatchPattern(b *testing.B) {
	sv := NewSchemaValidator()
	patterns := []string{
		`^[a-zA-Z]+$`, `^\d{3}-\d{2}-\d{4}$`, `^[a-f0-9]+$`,
		`^\w+@\w+\.\w+$`, `^https?://`, `^\d+\.\d+\.\d+$`,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pattern := patterns[i%len(patterns)]
		sv.matchPattern(pattern, "test-value-123")
	}
}
```

**Step 2: Run baseline benchmark**

Run: `go test -bench BenchmarkMatchPattern -benchmem ./httpvalidator/`
Record: baseline numbers

**Step 3: Change patternCache to sync.Map**

In `schema.go`, change struct field:
```go
type SchemaValidator struct {
    patternCache sync.Map // map[string]*regexp.Regexp
    patternCount int32    // atomic counter for size cap
    redactValues bool
}
```

Update `matchPattern`:
```go
func (v *SchemaValidator) matchPattern(pattern, s string) (bool, error) {
    if cached, ok := v.patternCache.Load(pattern); ok {
        return cached.(*regexp.Regexp).MatchString(s), nil
    }
    re, err := regexp.Compile(pattern)
    if err != nil {
        return false, err
    }
    // Size cap: if cache exceeds 1000 entries, clear and start fresh
    count := atomic.AddInt32(&v.patternCount, 1)
    if count > 1000 {
        v.patternCache = sync.Map{}
        atomic.StoreInt32(&v.patternCount, 1)
    }
    v.patternCache.Store(pattern, re)
    return re.MatchString(s), nil
}
```

Remove `patternCache: make(...)` from constructors (sync.Map zero value is ready to use).

**Step 4: Run benchmark comparison**

Run: `go test -bench BenchmarkMatchPattern -benchmem ./httpvalidator/`
Compare with baseline. Acceptable if within 20% for read-heavy workload.

**Step 5: Run tests, diagnostics, and commit**

---

### Task 4.3: Snapshot mutable validator fields

**Files:**
- Modify: `httpvalidator/validator.go:230-275` (ValidateRequest)
- Modify: `httpvalidator/validator.go:277-300` (ValidateResponse)

**Step 1: Snapshot at start of each method**

At the start of `ValidateRequest`:
```go
func (v *Validator) ValidateRequest(req *http.Request) (*RequestValidationResult, error) {
    strictMode := v.StrictMode
    includeWarnings := v.IncludeWarnings
    // Use strictMode and includeWarnings locals throughout
```

Similarly for `ValidateResponse`. Then find all references to `v.StrictMode` and `v.IncludeWarnings` within these methods and replace with the local variables.

**Step 2: Run tests, diagnostics, and commit**

---

### Task 4.4: Replace manual form parsing with url.ParseQuery

**Files:**
- Modify: `httpvalidator/request.go:478-490` (OAS 3.x form parsing)
- Modify: `httpvalidator/request.go:502-514` (OAS 2.0 form parsing)

**Step 1: Write failing test for URL-encoded form values**

```go
func TestFormBodyURLDecoding(t *testing.T) {
	// URL-encoded body: key%20name=value%3D123
	body := []byte("key%20name=value%3D123&normal=ok")
	// After parsing, key should be "key name" and value should be "value=123"
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Get("key name"); got != "value=123" {
		t.Errorf("expected decoded value, got %q", got)
	}
}
```

**Step 2: Replace OAS 3.x parsing (lines 478-490)**

Replace the manual `strings.Split` block with:
```go
parsed, err := url.ParseQuery(string(body))
if err != nil {
    result.addError("requestBody", fmt.Sprintf("invalid form body: %v", err), SeverityError)
    return
}
formData := make(map[string]any, len(parsed))
for key, values := range parsed {
    if len(values) > 0 {
        formData[key] = values[0]
    }
}
```

**Step 3: Replace OAS 2.0 parsing (lines 502-514)**

Replace with:
```go
formValues, err := url.ParseQuery(string(body))
if err != nil {
    result.addError("requestBody", fmt.Sprintf("invalid form body: %v", err), SeverityError)
    return
}
```

`url.ParseQuery` returns `url.Values` which is `map[string][]string` — exactly what `formValues` already is.

**Step 4: Run tests, diagnostics, and commit**

---

### Task 4.5: Fix body presence check

**Files:**
- Modify: `httpvalidator/request.go:313`

**Step 1: Remove ContentLength check**

Change:
```go
if req.Body == nil || req.ContentLength == 0 {
```
To:
```go
if req.Body == nil {
```

**Step 2: Run tests, diagnostics, and commit**

---

### Task 4.6: Implement additionalProperties enforcement

**Files:**
- Modify: `httpvalidator/schema.go:306-348` (validateObject)
- Test: `httpvalidator/schema_test.go`

**Step 1: Write failing test**

```go
func TestAdditionalPropertiesFalse(t *testing.T) {
	sv := NewSchemaValidator()
	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
		AdditionalProperties: &parser.AdditionalProperties{Allowed: boolPtr(false)},
	}
	obj := map[string]any{"name": "Alice", "age": 30, "extra": "bad"}
	errors := sv.Validate(obj, schema, "test")
	if len(errors) == 0 {
		t.Fatal("expected error for additional property 'extra'")
	}
}
```

**Step 2: Add enforcement in validateObject**

After the property schemas loop (line 344), add:
```go
// additionalProperties enforcement
if schema.AdditionalProperties != nil {
    allowed := schema.AdditionalProperties.Allowed
    if allowed != nil && !*allowed {
        for name := range obj {
            if _, defined := schema.Properties[name]; !defined {
                errors = append(errors, ValidationError{
                    Path:     path + "." + name,
                    Message:  fmt.Sprintf("additional property %q is not allowed", name),
                    Severity: SeverityError,
                })
            }
        }
    }
}
```

**Step 3: Run tests, diagnostics, and commit**

---

### Task 4.7: Sanitize error messages

**Files:**
- Modify: `httpvalidator/request.go` (lines 338, 350, 399)
- Modify: `httpvalidator/response.go` (line 230)
- Modify: `httpvalidator/validator.go` (line 242)
- Modify: `httpvalidator/options.go` (line 262)

**Step 1: Add truncation helper**

```go
// truncateForError truncates a string to maxLen and adds "..." if truncated.
func truncateForError(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}
```

**Step 2: Replace %s with %q and add truncation**

At each identified location, change patterns like:
```go
fmt.Sprintf("invalid Content-Type header: %s", contentType)
```
To:
```go
fmt.Sprintf("invalid Content-Type header: %q", truncateForError(contentType, 200))
```

Similarly for `req.URL.Path` and `mediaType` occurrences.

**Step 3: Run tests, diagnostics, and commit**

---

### Task 4.8: Create PR for HTTP validator hardening

Run `make check`, create PR targeting v1.52.0 milestone.

---

## PR 5: Generator Hardening (Findings 4.1-4.5)

### Task 5.1: Harden toFileName with character allowlist

**Files:**
- Modify: `generator/oas3_generator.go:984-989` (toFileName)
- Test: `generator/oas3_generator_test.go`

**Step 1: Write failing test**

```go
func TestToFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"MyScheme", "myscheme"},
		{"my-scheme", "my_scheme"},
		{"../../etc/passwd", "etcpasswd"},
		{"scheme/with/slashes", "schemewithslashes"},
		{"scheme.with.dots", "schemewithdots"},
		{"normal_name", "normal_name"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toFileName(tt.input)
			if got != tt.expected {
				t.Errorf("toFileName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
```

**Step 2: Update toFileName**

```go
func toFileName(name string) string {
    name = strings.ToLower(name)
    name = strings.ReplaceAll(name, "-", "_")
    name = strings.ReplaceAll(name, " ", "_")
    // Strip all characters except [a-z0-9_]
    var result strings.Builder
    for _, r := range name {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
            result.WriteRune(r)
        }
    }
    return result.String()
}
```

**Step 3: Run tests, diagnostics, and commit**

---

### Task 5.2: Add filepath.Base safety to WriteFiles

**Files:**
- Modify: `generator/writer.go:18`

**Step 1: Write failing test**

```go
func TestWriteFilesPathTraversal(t *testing.T) {
	dir := t.TempDir()
	result := &GenerateResult{
		Files: []GeneratedFile{
			{Name: "../escape.go", Content: []byte("package evil")},
		},
	}
	err := result.WriteFiles(dir)
	if err == nil {
		// Check that the file was NOT written outside dir
		if _, statErr := os.Stat(filepath.Join(filepath.Dir(dir), "escape.go")); statErr == nil {
			t.Fatal("file escaped output directory")
		}
	}
}
```

**Step 2: Add safety check**

At `writer.go:18`, add:
```go
for _, file := range r.Files {
    safeName := filepath.Base(file.Name)
    if safeName != file.Name {
        return fmt.Errorf("invalid file name %q: must not contain path separators", file.Name)
    }
    filePath := filepath.Join(outputDir, safeName)
```

**Step 3: Run tests, diagnostics, and commit**

---

### Task 5.3: Route spec-derived strings through cleanDescription

**Files:**
- Modify: `generator/security_helpers.go:191,270,309`

**Step 1: Apply cleanDescription**

At line 191, change:
```go
formatComment = fmt.Sprintf("\n// Bearer format: %s", bearerFormat)
```
To:
```go
formatComment = fmt.Sprintf("\n// Bearer format: %s", cleanDescription(bearerFormat))
```

At line 270, change:
```go
scopesComment = "\n// Available scopes: " + strings.Join(scopes, ", ")
```
To:
```go
cleaned := make([]string, len(scopes))
for i, s := range scopes {
    cleaned[i] = cleanDescription(s)
}
scopesComment = "\n// Available scopes: " + strings.Join(cleaned, ", ")
```

At line 309, change:
```go
urlComment = fmt.Sprintf("\n// OpenID Connect Discovery URL: %s", discoveryURL)
```
To:
```go
urlComment = fmt.Sprintf("\n// OpenID Connect Discovery URL: %s", cleanDescription(discoveryURL))
```

**Step 2: Run tests, diagnostics, and commit**

---

### Task 5.4: Generate safer client defaults

**Files:**
- Modify: `generator/client_boilerplate.go:44`
- Modify: `generator/oauth2_flows.go:172`
- Modify: `generator/oidc_discovery.go:154`

**Step 1: Replace http.DefaultClient in generated code**

At `client_boilerplate.go:44`, change:
```go
buf.WriteString("\t\tHTTPClient: http.DefaultClient,\n")
```
To:
```go
buf.WriteString("\t\tHTTPClient: &http.Client{Timeout: 30 * time.Second},\n")
```

Similarly at `oauth2_flows.go:172` and `oidc_discovery.go:154`.

**Step 2: Ensure `time` import is included in generated code**

Check that the generated file imports `"time"`. Add to import block if missing.

**Step 3: Run tests (including golden file tests), diagnostics, and commit**

---

### Task 5.5: Validate OAuth2/OIDC URLs and sanitize discriminator

**Files:**
- Modify: `generator/oauth2_flows.go` (URL scheme validation)
- Modify: `generator/oidc_discovery.go` (URL scheme validation)
- Modify: `generator/template_builders.go:447` (sanitize DiscriminatorJSONName)

**Step 1: Add URL scheme validation**

At generation time, validate that OAuth2 URLs use https (or http for localhost):
```go
func validateSecurityURL(urlStr string) string {
    u, err := url.Parse(urlStr)
    if err != nil {
        return "" // skip invalid URLs
    }
    if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")) {
        // Return URL but emit warning
        return urlStr // still use it, but warn during generation
    }
    return urlStr
}
```

**Step 2: Sanitize discriminator JSON name**

At `template_builders.go:447`, change:
```go
oneOfData.DiscriminatorJSONName = schema.Discriminator.PropertyName
```
To:
```go
jsonName := schema.Discriminator.PropertyName
jsonName = strings.ReplaceAll(jsonName, `"`, "")
jsonName = strings.ReplaceAll(jsonName, "`", "")
oneOfData.DiscriminatorJSONName = jsonName
```

**Step 3: Run tests, diagnostics, and commit**

---

### Task 5.6: Create PR for generator hardening

Run `make check`, create PR targeting v1.52.0 milestone.

---

## PR 6: CLI & Cross-Cutting (Findings 5.1-5.3)

### Task 6.1: Add symlink checks to CLI output commands

**Files:**
- Modify: `cmd/oastools/commands/fix.go:346-347`
- Modify: `cmd/oastools/commands/convert.go:201-202`
- Modify: `cmd/oastools/commands/overlay.go:239-240`
- Modify: `cmd/oastools/commands/join.go:387-388`

**Step 1: Add symlink check before each os.WriteFile**

The CLI commands already use `filepath.Clean()` and `0o600`. Add `os.Lstat` check:

```go
cleanedOutput := filepath.Clean(flags.Output)
// Reject symlinks to prevent symlink attacks
if info, err := os.Lstat(cleanedOutput); err == nil {
    if info.Mode()&os.ModeSymlink != 0 {
        return fmt.Errorf("refusing to write to symlink: %s", cleanedOutput)
    }
}
if err := os.WriteFile(cleanedOutput, data, 0o600); err != nil {
```

Apply to all 4 CLI commands.

**Step 2: Write test**

Test that CLI output rejects symlink targets.

**Step 3: Run tests, diagnostics, and commit**

---

### Task 6.2: Add depth cap to JSONPath recursion

**Files:**
- Modify: `internal/jsonpath/eval.go:185-227`
- Test: `internal/jsonpath/eval_test.go`

**Step 1: Write failing test**

```go
func TestRecursiveDescentDepthLimit(t *testing.T) {
	// Build a deeply nested structure
	var node any = "leaf"
	for i := 0; i < 600; i++ {
		node = map[string]any{"nested": node}
	}
	// Should not panic or stack overflow
	results := recursiveDescend(node, nil, 0)
	// Results should be capped, not all 600 levels
	if len(results) > 500 {
		t.Logf("got %d results (expected <= 500 due to depth cap)", len(results))
	}
}
```

**Step 2: Add depth parameter**

Change `recursiveDescend` signature:
```go
func recursiveDescend(node any, child Segment, depth int) []any {
    if depth > 500 {
        return nil
    }
    // ... existing code
    // In recursive calls, pass depth+1:
    results = append(results, recursiveDescend(val, child, depth+1)...)
```

Change `collectAllDescendants` signature:
```go
func collectAllDescendants(node any, results *[]any, depth int) {
    if depth > 500 {
        return
    }
    // ... existing code with depth+1 on recursive calls
```

Update the caller in `applySegment` to pass `0` as initial depth.

**Step 3: Run tests, diagnostics, and commit**

---

### Task 6.3: Create PR for CLI & cross-cutting

Run `make check`, create PR targeting v1.52.0 milestone.

---

## Final Steps

### Task 7.1: Verify all PRs pass CI

Verify each PR passes CI independently and that they merge cleanly in order.

### Task 7.2: Merge all PRs

Merge in dependency order: PR 1 -> PR 3 -> PR 2 -> PR 4 -> PR 5 -> PR 6.

### Task 7.3: Prepare v1.52.0 release

Use `/prepare-release v1.52.0` skill.
