---
name: corpus-test
description: "Run corpus regression tests against real-world OpenAPI specs. Usage: /corpus-test [short]"
---

# corpus-test

Run integration tests against the corpus of real-world OpenAPI specifications.

**Usage:**
- `/corpus-test` — run all corpus tests
- `/corpus-test short` — exclude large specs (faster)

## Step 1: Check Corpus Availability

Use the Glob tool to check for corpus files:

```
pattern: testdata/corpus/*.{json,yaml,yml}
```

If no corpus files found, download them:

```bash
make corpus-download
```

## Step 2: Run Corpus Tests

Based on the mode argument:

**Full mode** (default):
```bash
make test-corpus
```

**Short mode** (argument = "short"):
```bash
make test-corpus-short
```

Capture and analyze the output.

## Step 3: Analyze Results

> **Note:** Corpus specs are real-world documents (Plaid, DigitalOcean, GitHub, etc.) that often contain validation errors intentionally. Validation failures are expected — the key metric is whether the fixer resolves them.

For each corpus spec, report:
- ✅ Parse/validate/fix succeeded
- ❌ Failures with error details (fixer failed to resolve validation errors)
- ⚠️ New warnings compared to expected behavior

## Step 4: Report

```
## Corpus Test Results

| Spec | Parse | Validate | Fix | Notes |
|------|-------|----------|-----|-------|
| petstore.yaml | ✅ | ✅ | ✅ | |
| github-api.json | ✅ | ❌ | ⚠️ | [details] |
| ... | ... | ... | ... | ... |

### Summary
- **Passed**: N/M specs
- **Failed**: [list]
- **New issues**: [list]
```

## Step 5: Offer Actions

If failures are found:
1. **Investigate** — deep dive into specific failures
2. **Compare with main** — check if failures are regressions
3. **File issues** — create GitHub issues for new failures