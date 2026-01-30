---
name: gen-test
description: Generate test scaffolding for a Go file following oastools conventions. Usage: /gen-test <file.go> [function names...]
---

# gen-test

Generate test scaffolding for Go files following oastools testing conventions.

**Usage:**
- `/gen-test parser/resolver.go` - Generate tests for all exported functions
- `/gen-test validator/validate.go ValidateSpec` - Generate test for specific function

## Step 1: Analyze Target File

Read the specified file and identify:
1. Package name
2. Exported functions/methods to test
3. Existing test file (if any)
4. Dependencies and types used

> **Note:** In the commands below, `$FILE` refers to the path provided by the user when invoking the skill.
> For example, if the user runs `/gen-test parser/resolver.go`, then `$FILE` = `parser/resolver.go`.

```bash
# Check if test file already exists
# Example: parser/resolver.go -> parser/resolver_test.go
ls -la "${FILE%%.go}_test.go" 2>/dev/null
```

## Step 2: Find Testing Patterns

Look for existing test patterns in the same package:

```bash
# Find existing tests in the package
# Example: parser/resolver.go -> parser/*_test.go
ls -la $(dirname "$FILE")/*_test.go 2>/dev/null | head -5
```

Read 1-2 existing test files to understand:
- Table-driven test patterns used
- Test helper usage (`testutil`, `require`, `assert`)
- Fixture/testdata patterns
- Subtests structure

## Step 3: Generate Test Scaffolding

Create or append to the `*_test.go` file following these conventions:

### oastools Testing Conventions

1. **Use testify**: `github.com/stretchr/testify/require` and `assert`
2. **Table-driven tests**: Use `tests := []struct{...}` pattern
3. **Subtests**: Use `t.Run(tt.name, func(t *testing.T) {...})`
4. **Descriptive names**: Test case names should describe the scenario
5. **testutil helpers**: Use `testutil.LoadSpec()` for loading OAS fixtures

### Template Structure

```go
func TestFunctionName(t *testing.T) {
	tests := []struct {
		name    string
		input   InputType
		want    OutputType
		wantErr bool
	}{
		{
			name:  "valid input returns expected output",
			input: validInput,
			want:  expectedOutput,
		},
		{
			name:    "invalid input returns error",
			input:   invalidInput,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FunctionName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

### For Methods on Types

```go
func TestTypeName_MethodName(t *testing.T) {
	// Setup
	obj := NewTypeName(...)

	// Test cases
	tests := []struct {
		name string
		// ...
	}{
		// ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ...
		})
	}
}
```

## Step 4: Add Edge Cases

Always include test cases for:
- ✅ Happy path (valid input)
- ❌ Error cases (invalid input)
- 🔲 Nil/empty inputs
- 🔄 Boundary conditions

## Step 5: Verify Tests Compile

After generating, verify the tests compile:

```bash
go test -c ./path/to/package
```

## Step 6: Summary

Present the generated test scaffolding and ask if the user wants to:
1. Run the tests (`go test -v ./path/to/package -run TestName`)
2. Add more test cases
3. Generate tests for additional functions
