# gopls Workflow

## Import Management

**Use Go tooling for imports and formatting instead of manual management.**

After editing Go source files:
```bash
goimports -w <file>   # Auto-organizes imports and formats
gofmt -w <file>       # Formats code (goimports includes this)
```

`goimports` automatically:
- Adds missing imports
- Removes unused imports
- Groups imports (stdlib, external, internal)
- Applies `gofmt` formatting

When refactoring, don't manually adjust import blocks - run `goimports` and let Go's tooling handle it.

## gopls Diagnostics

**CRITICAL: Always run `go_diagnostics` on modified files and address ALL findings, including hints.**

The gopls MCP server (`go_diagnostics` tool) provides invaluable code quality feedback. **Even "hint" level suggestions can have significant performance impact.**

### Proven Impact
In v1.22.2, addressing gopls hints (unnecessary type conversions, redundant nil checks, modern Go idioms) resulted in **5-15% performance improvements** across most packages. These weren't errors or warnings—they were hints that had been ignored for some time.

### Workflow
1. After modifying Go files, run `go_diagnostics` with the file paths
2. Address **all** severity levels: errors, warnings, **and hints**
3. Hints suggest modern Go idioms and stdlib usage that improve both readability and performance

## Common Hints and Fixes

| Hint | Fix |
|------|-----|
| "Loop can be simplified using slices.Contains" | Replace manual contains loops with `slices.Contains(slice, item)` |
| "Replace m[k]=v loop with maps.Copy" | Use `maps.Copy(dst, src)` for map copying |
| "Constant reflect.Ptr should be inlined" | Use `reflect.Pointer` (reflect.Ptr is deprecated) |
| "Ranging over SplitSeq is more efficient" | Use `for part := range strings.SplitSeq(s, sep)` instead of `for _, part := range strings.Split(s, sep)` |
| "for loop can be modernized using range over int" | Use `for i := range n` instead of `for i := 0; i < n; i++` |

## Running Diagnostics

Via gopls MCP:
```
go_diagnostics with file paths
```
