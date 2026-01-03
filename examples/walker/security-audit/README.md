# Security Audit

Demonstrates auditing OpenAPI specifications for security issues using custom validation rules with the walker package.

## What You'll Learn

- Implementing custom validation rules beyond schema validation
- Pattern matching on field names for sensitive data detection
- Categorizing issues by severity (ERROR, WARNING, INFO)
- Building security-focused linting tools for API specifications

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/walker/security-audit
go run main.go
```

## Expected Output

```
Security Audit Report
=====================

Security Schemes Available:
  - apiKey (apiKey)
  - bearerAuth (http)

Findings by Severity:

[ERROR] (5 findings)
  $.components.schemas['Credentials']
    Sensitive field 'password' found - ensure proper handling
  $.components.schemas['Credentials']
    Sensitive field 'secret' found - ensure proper handling
  $.components.schemas['Credentials']
    Sensitive field 'token' found - ensure proper handling
  $.components.schemas['User']
    Sensitive field 'apiKey' found - ensure proper handling
  $.components.schemas['User']
    Sensitive field 'password' found - ensure proper handling

[WARNING] (3 findings)
  $.paths['/users'].post
    Operation has no security requirements
  $.paths['/users/{userId}'].delete
    Operation has no security requirements
  $.paths['/users/{userId}'].put
    Operation has no security requirements

[INFO] (2 findings)
  $.paths['/_system/health']
    Internal endpoint detected - verify access controls
  $.paths['/internal/debug']
    Internal endpoint detected - verify access controls

Summary: 5 errors, 3 warnings, 2 info
```

## Files

| File | Purpose |
|------|---------|
| main.go | Security audit logic using multiple walker handlers |
| go.mod | Module definition with local replace directive |
| specs/api-to-audit.yaml | Sample API with intentional security issues for demonstration |

## Key Concepts

### Custom Validation Beyond Schema Validation

The walker enables validation rules that go beyond structural schema validation:

```go
// Check for operations missing security requirements
walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
    if len(op.Security) == 0 && !isInternalPath(currentPath) {
        findings = append(findings, Finding{
            Severity: "WARNING",
            Path:     path,
            Message:  "Operation has no security requirements",
        })
    }
    return walker.Continue
}),
```

This catches security gaps that a schema validator would miss.

### Severity-Based Issue Categorization

Organizing findings by severity helps prioritize remediation:

- **ERROR**: Critical issues requiring immediate attention (sensitive data exposure)
- **WARNING**: Security gaps that should be addressed (missing authentication)
- **INFO**: Items for review that may be intentional (internal endpoints)

### Sensitive Data Detection Patterns

Pattern matching identifies potentially sensitive fields:

```go
sensitivePatterns := []string{"password", "secret", "token", "apikey", "credential", "key"}

walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
    for propName := range schema.Properties {
        for _, pattern := range sensitivePatterns {
            if strings.Contains(strings.ToLower(propName), pattern) {
                // Found sensitive field
            }
        }
    }
    return walker.Continue
}),
```

## Use Cases

- **CI/CD Security Gates**: Fail builds when security issues exceed thresholds
- **Compliance Audits**: Verify APIs meet security requirements
- **API Review Automation**: Pre-review checks before manual security review
- **Security Policy Enforcement**: Ensure all public endpoints require authentication
- **Sensitive Data Inventory**: Track which schemas contain sensitive fields

## Next Steps

- [Walker Deep Dive](../../../walker/deep_dive.md) - Complete walker documentation
- [Public API Filter](../public-api-filter/) - Filter endpoints by visibility
- [API Statistics](../api-statistics/) - Collect API metrics in a single pass

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
