# Detecting API Breaking Changes

## Overview

The `differ` package classifies changes between OpenAPI specifications by severity based on their impact on API consumers. This document provides a comprehensive reference for understanding breaking changes and how to handle them.

## Severity Levels

### Critical (API consumers WILL break)

Changes classified as **Critical** will definitively break existing API consumers. These require immediate attention and typically demand a major version bump.

**Examples:**

- Removed endpoints (`DELETE /paths/users/{id}`)
- Removed operations (removing `GET`, `POST`, `DELETE`, etc. from an existing path)
- Removed required request parameters
- Changed parameter location (`query` → `header`, `path` → `query`)
- Removed required request body fields
- Changed response status codes (removing `200`, `201`, or other success codes)
- Removed required response fields
- Removed security schemes that operations depend on

**Impact:** Consumers making requests will receive errors or unexpected responses.

**Action Required:** Major version bump (e.g., `v1` → `v2`)

---

### Error (API consumers MAY break)

Changes classified as **Error** severity have a high likelihood of breaking consumers, especially those not written defensively.

**Examples:**

- Changed parameter or property types (`string` → `integer`, `number` → `string`)
- Added new required parameters or fields (existing clients won't send them)
- Removed optional fields that were commonly used
- Changed enum values to a more restricted set (`["A", "B", "C"]` → `["A", "B"]`)
- Removed response content types (`application/json` → only `application/xml`)
- Changed from single value to array or vice versa
- Made existing optional parameter required

**Impact:** Consumers may receive validation errors, type errors, or unexpected behavior.

**Action Required:** Major version bump for public APIs, thorough testing for internal APIs

---

### Warning (API consumers SHOULD be aware)

Changes classified as **Warning** typically won't break consumers but may affect behavior or require updates for best practices.

**Examples:**

- Added new optional parameters
- Deprecated operations, parameters, or fields
- Changed descriptions that affect semantics (e.g., documenting new behavior)
- Added new enum values to an expandable set (`["A", "B"]` → `["A", "B", "C"]`)
- Changed examples or default values
- Modified error response structures
- Changed authentication requirements (added optional security)

**Impact:** Consumers may miss new functionality or receive deprecation warnings.

**Action Required:** Minor version bump, document in release notes

---

### Info (Non-breaking changes)

Changes classified as **Info** are backward compatible and generally improve the API without affecting existing consumers.

**Examples:**

- Added new optional fields
- Relaxed constraints (`maxLength` increased, `minimum` decreased)
- Added new endpoints
- Added new operations to existing paths
- Improved documentation, descriptions, or examples
- Added new response status codes (e.g., adding `201` when `200` already exists)
- Added new content types (while keeping existing ones)
- Added nullable to optional fields

**Impact:** No negative impact on consumers; may enable new use cases.

**Action Required:** Patch version bump, document as enhancement

---

## Version Compatibility

### Semantic Versioning Guidance

Based on the severity of changes detected by the differ:

| Severity Level | Version Change | Example | Reason |
|---------------|----------------|---------|--------|
| **Critical** or **Error** | MAJOR | `1.0.0 → 2.0.0` | Breaking change |
| **Warning** | MINOR | `1.0.0 → 1.1.0` | New features, deprecations |
| **Info** | PATCH | `1.0.0 → 1.0.1` | Backward-compatible enhancements |

**Mixed Changes:**
If a release contains changes of multiple severities, use the highest severity level to determine the version bump.

For example, if you have:

- 3 Critical changes
- 5 Warning changes
- 10 Info changes

Use a MAJOR version bump (`1.0.0 → 2.0.0`).

---

## Detection and Usage

### Using the differ Package

The `differ` package provides two modes:

1. **ModeSimple:** Detects all changes without severity classification
2. **ModeBreaking:** Classifies changes by severity (Critical, Error, Warning, Info)

**Example:**

```go
import "github.com/erraggy/oastools/differ"

d := differ.New()
d.Mode = differ.ModeBreaking
d.IncludeInfo = true

result, err := d.Diff("api-v1.yaml", "api-v2.yaml")
if err != nil {
    log.Fatal(err)
}

// Check for breaking changes
if result.HasBreakingChanges {
    fmt.Printf("⚠️  Found %d breaking changes\n", result.BreakingCount)

    // Review critical changes
    for _, change := range result.Changes {
        if change.Severity == differ.SeverityCritical {
            fmt.Printf("CRITICAL: %s\n", change.Message)
        }
    }
}
```

### Command Line

```bash
# Detect breaking changes
oastools diff --breaking api-v1.yaml api-v2.yaml

# Focus on breaking and error-level changes only
oastools diff --breaking --no-info api-v1.yaml api-v2.yaml
```

---

## Common Scenarios

### Adding a New Endpoint

```yaml
# v1.yaml
paths:
  /users:
    get: ...

# v2.yaml (added /posts)
paths:
  /users:
    get: ...
  /posts:
    get: ...
```

**Severity:** Info (non-breaking)
**Version Change:** Patch (`1.0.0 → 1.0.1`)

---

### Removing a Required Parameter

```yaml
# v1.yaml
paths:
  /users:
    get:
      parameters:
        - name: id
          in: query
          required: true

# v2.yaml (removed required parameter)
paths:
  /users:
    get:
      parameters: []
```

**Severity:** Critical (breaking)
**Version Change:** Major (`1.0.0 → 2.0.0`)
**Impact:** Existing clients sending `id` parameter may not receive expected data

---

### Making an Optional Parameter Required

```yaml
# v1.yaml
parameters:
  - name: filter
    in: query
    required: false

# v2.yaml
parameters:
  - name: filter
    in: query
    required: true
```

**Severity:** Error (likely breaking)
**Version Change:** Major (`1.0.0 → 2.0.0`)
**Impact:** Existing clients not sending `filter` will receive validation errors

---

### Adding a New Optional Parameter

```yaml
# v1.yaml
parameters:
  - name: id
    required: true

# v2.yaml (added optional parameter)
parameters:
  - name: id
    required: true
  - name: filter
    required: false
```

**Severity:** Warning (non-breaking, but noteworthy)
**Version Change:** Minor (`1.0.0 → 1.1.0`)
**Impact:** Existing clients continue to work; new functionality available

---

### Changing a Response Type

```yaml
# v1.yaml
responses:
  200:
    schema:
      type: string

# v2.yaml
responses:
  200:
    schema:
      type: object
```

**Severity:** Critical (breaking)
**Version Change:** Major (`1.0.0 → 2.0.0`)
**Impact:** Clients expecting a string will fail to parse the response

---

## Best Practices

### For API Developers

1. **Always run differ before releasing** to understand the impact of your changes
2. **Use ModeBreaking** to get severity classifications
3. **Document all breaking changes** in release notes with migration guides
4. **Consider deprecation first** before removing features (use Warning-level changes)
5. **Version your API properly** based on severity levels

### For API Consumers

1. **Monitor deprecation warnings** to prepare for future breaking changes
2. **Write defensive code** that handles new optional fields gracefully
3. **Pin to specific major versions** of APIs you depend on
4. **Test against new API versions** before upgrading
5. **Subscribe to API change notifications** if available

### For CI/CD Integration

```bash
#!/bin/bash
# Example: Fail CI if breaking changes are detected

oastools diff \
  --breaking \
  current-api.yaml \
  proposed-api.yaml \
  > diff-result.txt

# Check for breaking changes
if grep -q "CRITICAL\|ERROR" diff-result.txt; then
  echo "❌ Breaking changes detected! Review required."
  exit 1
fi

echo "✓ No breaking changes detected"
exit 0
```

---

## Additional Resources

- **OpenAPI Specification:** [spec.openapis.org/oas](https://spec.openapis.org/oas/)
- **Semantic Versioning:** [semver.org](https://semver.org/)
- **oastools differ package:** [pkg.go.dev/github.com/erraggy/oastools/differ](https://pkg.go.dev/github.com/erraggy/oastools/differ)

For questions or issues, please [open an issue](https://github.com/erraggy/oastools/issues).
