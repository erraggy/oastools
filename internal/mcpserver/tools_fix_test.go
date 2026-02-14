package mcpserver

import (
	"context"
	"testing"

	"github.com/erraggy/oastools/fixer"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// specWithDuplicateOperationIds has two operations sharing the same operationId.
const specWithDuplicateOperationIds = `openapi: "3.0.0"
info:
  title: Dup Test
  version: "1.0.0"
paths:
  /cats:
    get:
      operationId: listAnimals
      responses:
        "200":
          description: OK
  /dogs:
    get:
      operationId: listAnimals
      responses:
        "200":
          description: OK
`

// specWithMissingRef references a schema that does not exist.
const specWithMissingRef = `openapi: "3.0.0"
info:
  title: Stub Test
  version: "1.0.0"
paths:
  /items:
    get:
      operationId: listItems
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/MissingSchema"
`

// specWithMissingPathParam has a path template variable without a declared parameter.
const specWithMissingPathParam = `openapi: "3.0.0"
info:
  title: Path Param Test
  version: "1.0.0"
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        "200":
          description: OK
`

func TestFixTool_DuplicateOperationIds(t *testing.T) {
	input := fixInput{
		Spec:                     specInput{Content: specWithDuplicateOperationIds},
		FixDuplicateOperationIds: true,
	}
	_, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", output.Version)
	assert.GreaterOrEqual(t, output.FixCount, 1)
	assert.NotEmpty(t, output.Fixes)

	// At least one fix should be a duplicate-operation-id fix.
	found := false
	for _, f := range output.Fixes {
		if f.Type == string(fixer.FixTypeDuplicateOperationId) {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a duplicate-operation-id fix")
}

func TestFixTool_DryRun(t *testing.T) {
	input := fixInput{
		Spec:                     specInput{Content: specWithDuplicateOperationIds},
		FixDuplicateOperationIds: true,
		DryRun:                   true,
	}
	_, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	// Dry run should still report fixes that would be applied.
	assert.GreaterOrEqual(t, output.FixCount, 1)
	assert.NotEmpty(t, output.Fixes)
	// Document should be empty in dry-run mode.
	assert.Empty(t, output.Document)
}

func TestFixTool_StubMissingRefs(t *testing.T) {
	input := fixInput{
		Spec:            specInput{Content: specWithMissingRef},
		StubMissingRefs: true,
		IncludeDocument: true,
	}
	_, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", output.Version)
	assert.GreaterOrEqual(t, output.FixCount, 1)
	assert.NotEmpty(t, output.Document)
	// The fixed document should contain the stubbed schema.
	assert.Contains(t, output.Document, "MissingSchema")
}

func TestFixTool_MissingPathParameter(t *testing.T) {
	input := fixInput{
		Spec: specInput{Content: specWithMissingPathParam},
	}
	_, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", output.Version)
	assert.GreaterOrEqual(t, output.FixCount, 1)

	found := false
	for _, f := range output.Fixes {
		if f.Type == string(fixer.FixTypeMissingPathParameter) {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a missing-path-parameter fix")
}

func TestFixTool_IncludeDocument(t *testing.T) {
	input := fixInput{
		Spec:            specInput{Content: specWithMissingPathParam},
		IncludeDocument: true,
	}
	_, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.NotEmpty(t, output.Document)
	assert.Contains(t, output.Document, "userId")
}

func TestFixTool_IncludeDocument_DryRun_NoDocument(t *testing.T) {
	// When both IncludeDocument and DryRun are set, the document should
	// not be included because dry run does not actually apply fixes.
	input := fixInput{
		Spec:            specInput{Content: specWithMissingPathParam},
		IncludeDocument: true,
		DryRun:          true,
	}
	_, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Empty(t, output.Document)
	assert.GreaterOrEqual(t, output.FixCount, 1)
}

func TestFixTool_InvalidSpec(t *testing.T) {
	input := fixInput{
		Spec: specInput{Content: "not valid yaml: ["},
	}
	result, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Version)
}

func TestFixTool_NoInputProvided(t *testing.T) {
	input := fixInput{
		Spec: specInput{},
	}
	result, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Version)
}

func TestFixTool_NoFixesNeeded(t *testing.T) {
	// A clean spec that has no fixable issues.
	clean := `openapi: "3.0.0"
info:
  title: Clean API
  version: "1.0.0"
paths:
  /health:
    get:
      operationId: getHealth
      responses:
        "200":
          description: OK
`
	input := fixInput{
		Spec: specInput{Content: clean},
	}
	_, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", output.Version)
	assert.Equal(t, 0, output.FixCount)
	assert.Empty(t, output.Fixes)
}
