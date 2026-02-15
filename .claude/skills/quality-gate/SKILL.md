---
name: quality-gate
description: "Run full validation suite (make check + gopls diagnostics) and report structured pass/fail. Usage: /quality-gate [package...]"
---

# quality-gate

Run the complete quality pipeline and report structured results.

**Usage:**
- `/quality-gate` — validate all changed packages
- `/quality-gate parser validator` — validate specific packages

## Step 1: Identify Scope

Determine which packages to check.

If packages were specified as arguments, use those.

Otherwise, find changed `.go` files:

```bash
git diff --name-only HEAD
```

Then extract unique package directories from the output using string processing — do NOT use piped bash commands (they trigger permission prompts). Convert relative paths to absolute paths (prepend the repo root) for Step 3.

## Step 2: Run `make check`

This runs tidy, fmt, lint, and tests in one command:

```bash
make check
```

Record pass/fail and capture any output.

## Step 3: Run gopls Diagnostics

Use the `go_diagnostics` MCP tool on changed `.go` files (NOT bash). Pass absolute paths.

Categorize results:
- 🔴 **Errors** — must fix
- 🟡 **Warnings** — should fix
- 💡 **Hints** — fix for performance (5-15% impact documented)

## Step 4: Report

Present results in this format:

```
## Quality Gate Results

### `make check`: ✅ PASS / ❌ FAIL
[details if failed]

### gopls Diagnostics
| File | Level | Message | Suggested Fix (if any) |
|------|-------|---------|------------------------|
| ... | ... | ... | ... |

### Verdict: ✅ READY / ❌ NOT READY

Issues to address:
1. [issue]
2. [issue]
```

## Step 5: Offer Actions

If there are fixable issues, offer:
1. **Fix all** — address all findings automatically
2. **Fix errors only** — address blocking issues
3. **Skip** — acknowledge and continue