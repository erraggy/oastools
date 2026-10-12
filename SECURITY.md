# Security Policy

## Supported Versions

Security fixes are made on the latest release. Please upgrade to the most recent version before reporting.

## Reporting a Vulnerability

Please report suspected vulnerabilities privately by email to <oastools@robnrob.com>. Do not open a public issue for a security problem.

Include the affected version, a description of the issue, and a minimal OpenAPI document or command that reproduces it where possible.

You can expect an acknowledgement of your report, and a fix or an explanation once it has been assessed. Reporters are credited in the release notes only if they ask for it, using the name or alias they approve.

## Scope

oastools processes untrusted OpenAPI documents, so the areas of most interest are:

- Crashes, hangs or unbounded memory use on crafted documents.
- `$ref` resolution that reads unintended files or makes unintended network requests.
- Code generator output that allows injection from specification strings.
- The MCP server (`internal/mcpserver`) and its tool arguments.

See [`.oss-scanner/threat_model.md`](.oss-scanner/threat_model.md) for more detail.
