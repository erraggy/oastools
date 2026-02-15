package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const diffBaseSpec = `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
`

const diffRevisedSpec = `openapi: "3.0.0"
info:
  title: Test API
  version: "2.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
    post:
      operationId: createPet
      responses:
        "201":
          description: Created
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
`

const diffBreakingSpec = `openapi: "3.0.0"
info:
  title: Test API
  version: "2.0.0"
paths: {}
`

func TestDiffTool_DetectsChanges(t *testing.T) {
	input := diffInput{
		Base:     specInput{Content: diffBaseSpec},
		Revision: specInput{Content: diffRevisedSpec},
	}
	_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Greater(t, output.TotalChanges, 0, "should detect changes between base and revised specs")
	assert.NotEmpty(t, output.Changes)
	assert.NotEmpty(t, output.Summary)

	// Verify change structure has all expected fields populated.
	for _, c := range output.Changes {
		assert.NotEmpty(t, c.Severity, "change should have a severity")
		assert.NotEmpty(t, c.Type, "change should have a type")
		assert.NotEmpty(t, c.Message, "change should have a message")
	}
}

func TestDiffTool_BreakingOnly(t *testing.T) {
	// The breaking spec removes the /pets endpoint entirely, which is a breaking change.
	input := diffInput{
		Base:         specInput{Content: diffBaseSpec},
		Revision:     specInput{Content: diffBreakingSpec},
		BreakingOnly: true,
	}
	_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Greater(t, output.BreakingCount, 0, "should have breaking changes")

	// All displayed changes should be breaking (critical or error severity).
	for _, c := range output.Changes {
		assert.Contains(t, []string{"critical", "error"}, c.Severity,
			"breaking_only should only include critical/error changes, got: %s", c.Severity)
	}
}

func TestDiffTool_NoChanges(t *testing.T) {
	input := diffInput{
		Base:     specInput{Content: diffBaseSpec},
		Revision: specInput{Content: diffBaseSpec},
	}
	_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, 0, output.TotalChanges)
	assert.Equal(t, 0, output.BreakingCount)
	assert.Empty(t, output.Changes)
	assert.Equal(t, "No changes detected.", output.Summary)
}

func TestDiffTool_InvalidBase(t *testing.T) {
	input := diffInput{
		Base:     specInput{Content: "not valid yaml: ["},
		Revision: specInput{Content: diffBaseSpec},
	}
	result, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Changes)
}

func TestDiffTool_InvalidRevision(t *testing.T) {
	input := diffInput{
		Base:     specInput{Content: diffBaseSpec},
		Revision: specInput{Content: "not valid yaml: ["},
	}
	result, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Changes)
}

func TestDiffTool_NoInfo(t *testing.T) {
	input := diffInput{
		Base:     specInput{Content: diffBaseSpec},
		Revision: specInput{Content: diffRevisedSpec},
		NoInfo:   true,
	}
	_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	// With NoInfo, info-level changes should be suppressed.
	for _, c := range output.Changes {
		assert.NotEqual(t, "info", c.Severity,
			"no_info should suppress info-level changes")
	}
}

func TestDiffTool_MissingInput(t *testing.T) {
	input := diffInput{
		Base:     specInput{},
		Revision: specInput{Content: diffBaseSpec},
	}
	result, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Changes)
}

func TestDiffTool_Pagination(t *testing.T) {
	// Baseline: get total change count.
	_, baseline, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, diffInput{
		Base:     specInput{Content: diffBaseSpec},
		Revision: specInput{Content: diffRevisedSpec},
	})
	require.NoError(t, err)
	require.Greater(t, baseline.TotalChanges, 2, "need at least 3 changes for pagination test")

	t.Run("limit", func(t *testing.T) {
		_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, diffInput{
			Base:     specInput{Content: diffBaseSpec},
			Revision: specInput{Content: diffRevisedSpec},
			Limit:    1,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.TotalChanges, output.TotalChanges)
		assert.Equal(t, 1, output.Returned)
		assert.Len(t, output.Changes, 1)
		assert.NotEmpty(t, output.Summary)
	})

	t.Run("offset", func(t *testing.T) {
		_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, diffInput{
			Base:     specInput{Content: diffBaseSpec},
			Revision: specInput{Content: diffRevisedSpec},
			Offset:   1,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.TotalChanges, output.TotalChanges)
		assert.Equal(t, baseline.TotalChanges-1, output.Returned)
	})

	t.Run("offset and limit", func(t *testing.T) {
		_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, diffInput{
			Base:     specInput{Content: diffBaseSpec},
			Revision: specInput{Content: diffRevisedSpec},
			Offset:   1,
			Limit:    2,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.TotalChanges, output.TotalChanges)
		assert.Equal(t, 2, output.Returned)
		assert.Len(t, output.Changes, 2)
	})

	t.Run("offset beyond total", func(t *testing.T) {
		_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, diffInput{
			Base:     specInput{Content: diffBaseSpec},
			Revision: specInput{Content: diffRevisedSpec},
			Offset:   baseline.TotalChanges,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.TotalChanges, output.TotalChanges)
		assert.Equal(t, 0, output.Returned)
		assert.Nil(t, output.Changes)
		// Summary should still reflect the full result.
		assert.NotEmpty(t, output.Summary)
	})

	t.Run("counts unchanged by pagination", func(t *testing.T) {
		_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, diffInput{
			Base:     specInput{Content: diffBaseSpec},
			Revision: specInput{Content: diffRevisedSpec},
			Limit:    1,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.BreakingCount, output.BreakingCount)
		assert.Equal(t, baseline.WarningCount, output.WarningCount)
		assert.Equal(t, baseline.InfoCount, output.InfoCount)
	})

	t.Run("negative offset returns no changes", func(t *testing.T) {
		_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, diffInput{
			Base:     specInput{Content: diffBaseSpec},
			Revision: specInput{Content: diffRevisedSpec},
			Offset:   -1,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.TotalChanges, output.TotalChanges)
		assert.Equal(t, 0, output.Returned)
		assert.Nil(t, output.Changes)
	})

	t.Run("negative limit uses default", func(t *testing.T) {
		_, output, err := handleDiff(context.Background(), &mcp.CallToolRequest{}, diffInput{
			Base:     specInput{Content: diffBaseSpec},
			Revision: specInput{Content: diffRevisedSpec},
			Limit:    -5,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.TotalChanges, output.TotalChanges)
		assert.Equal(t, baseline.TotalChanges, output.Returned)
	})
}
