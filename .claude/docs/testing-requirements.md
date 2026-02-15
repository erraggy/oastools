# Testing Requirements

**CRITICAL: All exported functionality MUST have comprehensive test coverage.**

## Coverage Requirements

Test coverage must include:

1. **Exported Functions** - Package-level convenience functions and struct methods
2. **Exported Types** - Struct initialization, fields, type conversions
3. **Exported Constants** - Verify expected values

## Coverage Types

- **Positive Cases**: Valid inputs work correctly
- **Negative Cases**: Error handling with invalid inputs, missing files, malformed data
- **Edge Cases**: Boundary conditions, empty inputs, nil values
- **Integration**: Components working together (parse then validate, parse then join)

## Codecov Patch Coverage

**70% patch coverage required** on all PRs (configured in `.codecov.yml`).

```bash
# Verify coverage locally
go test -coverprofile=cover.out ./package/
go tool cover -func=cover.out | tail -1
```

Test all branches including nil checks and error paths—they count against patch coverage.

## Known Test Stability Issues

**TestCircularReferenceDetection** (`parser/resolver_test.go`): If this test hangs, check `parser/resolver.go` for:

1. Deep copying in `resolveRefsRecursive` (not shallow copy)
2. Parameterized defer in `ResolveLocal` (captures ref by value)
