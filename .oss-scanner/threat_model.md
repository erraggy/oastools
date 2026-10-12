# oastools threat model

oastools is a Go library and CLI that parses, validates, fixes, converts, joins, diffs, walks and generates code from OpenAPI Specification documents (OAS 2.0 to 3.2), and includes an MCP server (internal/mcpserver).

## Untrusted input

- OpenAPI documents (JSON/YAML) from files, stdin, or URLs, including `$ref` resolution (local files, remote HTTP).
- Overlay documents and JSONPath expressions (internal/jsonpath).
- HTTP requests and responses checked by httpvalidator.
- MCP tool arguments received by the MCP server.

## In scope

- Crashes, hangs, or unbounded memory/CPU on crafted documents (deeply nested or cyclic schemas, YAML anchors/aliases, huge inputs).
- `$ref` handling that reads files outside the intended base path or makes unintended network requests (path traversal, SSRF).
- Code generator output that allows injection from spec strings (names, descriptions, enums, paths) into generated Go code.
- Incorrect results that make validation or breaking-change detection pass for invalid or incompatible documents.

## Out of scope

- Findings that require a malicious local user running the CLI with their own privileges.
- Vulnerabilities in third-party dependencies with no reachable path from oastools.
- docs/ site tooling and benchmark scripts.

## Severity guidance

Highest: memory-safety-like failures, arbitrary file read or network access via `$ref`, code injection in generated output. Medium: denial of service from crafted input. Low: incorrect validation diagnostics.

## Reports

Email the primary contact; a proposed patch with a regression test is preferred.
