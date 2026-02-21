# Security Hardening Design

**Date**: 2026-02-20
**Status**: Approved
**Milestone**: [v1.52.0](https://github.com/erraggy/oastools/milestone/7) (target: Feb 22, 2026)
**Catalyst**: `github.com/modelcontextprotocol/go-sdk` v1.3.0 -> v1.3.1 (case-sensitive JSON unmarshaling security patch)

## Summary

Comprehensive security hardening across 5 attack surfaces: MCP server, parser/resolver, HTTP validator, code generator, and CLI/file I/O. Addresses 7 High-severity and 19 Medium-severity findings from a full codebase audit. All changes ship in a single release (v1.52.0) across multiple PRs grouped by package.

## Guiding Principles

### User Direction vs Malicious Actor

The codebase has two trust models:

| Context | Trust Level | Rationale |
|---------|------------|-----------|
| **CLI** | Trust the user | User invokes commands directly; restrictions (SSRF blocking, private IP warnings) would be unwelcome — `curl` doesn't do this either |
| **MCP Server** | Protect against untrusted input | AI agents may be manipulated; input is not directly user-controlled |

This distinction drives different defaults: CLI has zero restrictions on target IPs/URLs, while MCP blocks private IPs with opt-out via `OASTOOLS_ALLOW_PRIVATE_IPS=true`.

### Stdlib Over Hand-Rolled

Anywhere a hand-rolled implementation duplicates stdlib-provided functionality, replace it. This reduces attack surface and ensures security patches flow through automatically.

### No Breaking API Changes

All hardening adds safety without changing public API signatures or behavior for valid inputs.

## Section 1: MCP Server Hardening

### 1.1 Output Path Safety (High)

**Finding**: `os.WriteFile(input.Output, data, 0o644)` in `tools_fix.go:85`, `tools_convert.go:81`, `tools_join.go:111`, `tools_overlay.go:94`, `tools_generate.go:70` — no path sanitization.

**Fix**: New `sanitizeOutputPath(path string) (string, error)` helper in `internal/mcpserver/pathutil.go`:
- `filepath.Clean` and `filepath.Abs` the path
- Reject paths containing `..` after cleaning
- `os.Lstat` check — reject if target is a symlink
- Change file permissions from `0o644` to `0o600`
- All 5 tool handlers call this before writing

### 1.2 SSRF Protection — MCP Only (High)

**Finding**: `parser.fetchURL` makes HTTP requests with no restrictions on target IP.

**Fix**: MCP-only SSRF protection via `parser.WithHTTPClient`:
- New `internal/mcpserver/safeclient.go` creates an `http.Client` with custom `DialContext` that blocks private/loopback/link-local IPs using `net.IP.IsPrivate()`, `IsLoopback()`, `IsLinkLocalUnicast()`
- Custom `CheckRedirect` that re-validates each redirect target
- The safe client is injected into parser via `parser.WithHTTPClient(safeClient)` when resolving specs in MCP context
- CLI path unchanged — no SSRF restrictions
- Opt-out: `OASTOOLS_ALLOW_PRIVATE_IPS=true` env var disables the IP check

### 1.3 Input Size Bounds (Medium)

**Finding**: `specInput.Content` field at `input.go:228` has no size limit.

**Fix**: Reject inline content exceeding 10 MiB before parsing. Configurable via `OASTOOLS_MAX_INLINE_SIZE` env var.

### 1.4 Pagination Safety (Medium)

**Finding**: `paginate` function at `server.go:139-151` accepts unbounded `limit`.

**Fix**: Cap `limit` at 1000 (configurable via `OASTOOLS_MAX_LIMIT`). Already existing `OASTOOLS_WALK_LIMIT` provides the default, this adds an upper bound.

### 1.5 Join Specs Bound (Medium)

**Finding**: `tools_join.go:48-59` accepts unbounded `specs` array.

**Fix**: Cap at 20 specs (configurable via `OASTOOLS_MAX_JOIN_SPECS`).

### 1.6 Package Name Validation (Medium)

**Finding**: `tools_generate.go:52-53` doesn't validate `package_name` as a Go identifier.

**Fix**: Validate against `^[a-z][a-z0-9_]*$` (max 64 chars) before passing to generator.

### 1.7 Join Strategy Validation (Medium)

**Finding**: `tools_join.go:66-71` doesn't validate `path_strategy`/`schema_strategy` against known values.

**Fix**: Validate against `validJoinStrategies` set before passing to joiner.

### 1.8 Error Sanitization (Medium)

**Finding**: `errResult` at `server.go:173-178` exposes raw OS error paths.

**Fix**: Strip absolute filesystem paths from error messages before returning to client. New `sanitizeError(err error) string` helper.

## Section 2: Parser & Resolver Hardening

### 2.1 Input Size Limit (High)

**Finding**: `io.ReadAll(r)` at `parser.go:347` (`ParseReader`) is unbounded for stdin input.

**Fix**: New `MaxInputSize` field on parser options (default 100 MiB). Wrap reader with `io.LimitReader`. Applies to both `ParseReader` and `fetchURL`.

### 2.2 HTTP Client Injection (High)

**Finding**: `fetchURL` at `parser_format.go:77-139` uses a bare `http.Client` with no way to customize.

**Fix**: New `WithHTTPClient(client *http.Client)` parser option. MCP server injects its SSRF-safe client. CLI uses default (no restrictions). If no custom client is provided, parser uses `http.DefaultClient`.

### 2.3 Fetch Response Size (Medium)

**Finding**: `io.ReadAll(resp.Body)` at `parser_format.go:131` is unbounded.

**Fix**: Wrap `resp.Body` with `io.LimitReader(resp.Body, maxInputSize+1)` — the +1 pattern detects "exceeded" vs "exactly at limit".

### 2.4 Redirect Safety (Medium)

**Finding**: `http.Client` at `parser_format.go:93-101` follows redirects with no validation.

**Fix**: When the injected client has a `CheckRedirect`, it's respected. The MCP safe client validates redirect targets against the private IP blocklist. Default client follows Go's standard 10-redirect limit.

### 2.5 Same-Origin Enforcement (Medium)

**Finding**: `resolveRelativeURL` at `resolver.go:412-432` resolves relative URLs without verifying the result stays on the same host.

**Fix**: After resolving, verify `resolved.Host == base.Host`. If not, return an error. This prevents open-redirect-style attacks where a relative ref resolves to a different host.

## Section 3: HTTP Validator Hardening

### 3.1 Body Size Limits (High)

**Finding**: `io.ReadAll` at `request.go:373` and `response.go:51` — unbounded body reads.

**Fix**: New `WithMaxBodySize(n int64)` option (default 10 MiB, aligned with AWS API Gateway and Spring Boot defaults). Bodies exceeding the limit return a validation error rather than OOM.

### 3.2 Pattern Cache Concurrency (High)

**Finding**: `patternCache` map at `schema.go:503-510` has no synchronization — data race under concurrent validation.

**Fix**: Replace `map[string]*regexp.Regexp` with `sync.Map`. Add size cap of 1000 entries (evict-all when exceeded). **Benchmark before merging** to verify no performance regression — `sync.Map` has different performance characteristics than plain maps for write-heavy workloads.

### 3.3 Validator Field Race (Medium)

**Finding**: Mutable public fields `StrictMode` and `IncludeWarnings` at `validator.go:43-48` — data race if modified during concurrent validation.

**Fix**: Snapshot `StrictMode` and `IncludeWarnings` into local variables at the start of each `Validate*` method. Public fields remain for API compatibility but are read once.

### 3.4 Form Body Parsing — Stdlib Replacement (Medium)

**Finding**: Manual `strings.Split(string(body), "&")` parsing at `request.go:478-495` doesn't URL-decode values.

**Fix**: Replace with `url.ParseQuery(string(body))`. This is a stdlib replacement that handles URL-decoding, empty values, and edge cases correctly.

### 3.5 Body Presence Check (Medium)

**Finding**: `req.ContentLength == 0` at `request.go:313` used to detect missing body — incorrect for chunked encoding.

**Fix**: Remove `ContentLength == 0` check; rely solely on `req.Body == nil` for body presence detection.

### 3.6 Additional Properties Enforcement (Medium)

**Finding**: `additionalProperties: false` at `schema.go:306-348` is parsed but not enforced.

**Fix**: When `additionalProperties` is `false`, check for extra properties in the input object and emit a validation error for each unexpected key.

### 3.7 Error Sanitization (Medium)

**Finding**: Raw URL paths and Content-Type values echoed in errors at `validator.go:242` and `request.go:337-342` — log injection risk.

**Fix**: Use `%q` (quoted) instead of `%s` for user-supplied values in error messages. Truncate to 200 characters.

## Section 4: Generator Hardening

### 4.1 Path Traversal in Filenames (High)

**Finding**: `toFileName` at `oas3_generator.go:984` doesn't strip `/` or `.` — scheme names like `../../etc/passwd` can escape `outputDir`.

**Fix**: Apply `[a-z0-9_]` character allowlist (matching the existing `sanitizeGroupName` at `file_splitter.go:779`). Also add `filepath.Base` safety check in `WriteFiles` at `writer.go:18`.

### 4.2 Comment Injection (Medium)

**Finding**: `BearerFormat`, `OpenIDConnectURL`, and scope names at `security_helpers.go:190,270,309` bypass `cleanDescription` — can inject `*/` to break out of comments.

**Fix**: Route all spec-derived strings through `cleanDescription` before embedding in generated code.

### 4.3 Generated Client Defaults (Medium)

**Finding**: `http.DefaultClient` in generated boilerplate at `client_boilerplate.go:44` — no timeout.

**Fix**: Generate `&http.Client{Timeout: 30 * time.Second}` as the default client in generated code. Users can still override.

### 4.4 OAuth2/OIDC URL Validation (Medium)

**Finding**: Generated OAuth2 flow code at `oauth2_flows.go:172` and OIDC discovery at `oidc_discovery.go:154` use URLs from spec without validation.

**Fix**: Validate URL scheme is `https` (or `http` for localhost) at code generation time. Emit a warning comment in generated code for non-HTTPS URLs.

### 4.5 Discriminator JSON Name (Medium)

**Finding**: `.DiscriminatorJSONName` at `template_builders.go:447` is set from raw spec value, used unquoted in struct tags.

**Fix**: Strip `"` and backtick characters from discriminator JSON names before embedding in struct tags.

## Section 5: CLI & Cross-Cutting

### 5.1 Symlink Safety on Output (Medium)

**Finding**: `os.WriteFile` at `cmd/oastools/commands/fix.go:346`, `convert.go:201`, `overlay.go:239`, `join.go:387` follows symlinks.

**Fix**: `os.Lstat` check before `os.WriteFile` — reject if target is a symlink. Reuse the MCP `sanitizeOutputPath` logic via a shared `internal/pathutil` package.

### 5.2 JSONPath Recursion Depth (Medium)

**Finding**: `recursiveDescend`/`collectAllDescendants` at `internal/jsonpath/eval.go:185-227` — unbounded recursion on deeply nested documents.

**Fix**: Depth cap of 500. Return error when exceeded.

### 5.3 Stdlib Replacement Sweep (Medium)

**Finding**: Hand-rolled implementations that should use stdlib equivalents.

**Fix**: Sweep the entire codebase for non-standard implementations. Known instance: `request.go:478-495` form parsing. Identify and replace any others found during implementation.

### 5.4 go-sdk Dependency Update

**Change**: `github.com/modelcontextprotocol/go-sdk` v1.3.0 -> v1.3.1

This is the catalyst for the audit. The security patch switches to case-sensitive JSON unmarshaling via `segmentio/encoding`. New indirect dependencies: `segmentio/asm v1.2.1`, `segmentio/encoding v0.5.3`, `golang.org/x/sys v0.41.0`.

## Well-Mitigated Areas (No Changes Needed)

| Area | Why It's Fine |
|------|--------------|
| `$ref` file path traversal (`resolver.go:224-248`) | Blocked via `filepath.Rel` |
| Circular `$ref` resolution (`resolver.go:450-546`) | Handled via resolving map + `maxRefDepth` |
| YAML bombs (`go.yaml.in/yaml/v4`) | Alias-ratio throttle built into library |
| JSON case folding (`parser/common_json.go`) | Not security-relevant in this context |
| MCP env var parsing (`internal/mcpserver/config.go`) | Well-validated via typed parsers |

## PR Strategy

All PRs target the `v1.52.0` milestone. Grouped by package to minimize merge conflicts and enable parallel review:

| PR | Scope | Findings |
|----|-------|----------|
| 1 | go-sdk update + go.sum | 5.4 |
| 2 | MCP server hardening | 1.1-1.8 |
| 3 | Parser & resolver hardening | 2.1-2.5 |
| 4 | HTTP validator hardening | 3.1-3.7 |
| 5 | Generator hardening | 4.1-4.5 |
| 6 | CLI & cross-cutting | 5.1-5.3 |

## Testing Strategy

| Area | Approach |
|------|----------|
| MCP output path safety | Unit tests: `..` traversal, symlink, absolute paths outside allowed dirs |
| SSRF protection | Unit tests: private IPs blocked, loopback blocked, link-local blocked, opt-out works |
| Input bounds | Unit tests: content exceeding limit rejected, at-limit accepted |
| Pattern cache | Benchmark: `sync.Map` vs plain map, verify no regression |
| Body size limits | Unit tests: oversized body returns error, at-limit body succeeds |
| Form parsing | Unit tests: URL-encoded values decoded correctly |
| Path traversal (generator) | Unit tests: `../` stripped, only `[a-z0-9_]` survive |
| Symlink safety | Unit tests: symlink targets rejected, regular files accepted |
| JSONPath depth | Unit tests: depth 500 succeeds, depth 501 errors |
| Stdlib sweep | Review-time: identify and test each replacement |

## Backward Compatibility

- All new options have defaults matching current behavior for valid inputs
- No public API signature changes
- MCP tools gain validation but reject only invalid/malicious inputs
- Generated code gains safer defaults (timeout, scheme validation) but users can still override
