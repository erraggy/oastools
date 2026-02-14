package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const walkResponsesTestSpec = `openapi: "3.0.0"
info:
  title: Walk Responses Test
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List pets
      responses:
        "200":
          description: A list of pets
        "400":
          description: Bad request
        "500":
          description: Internal server error
    post:
      summary: Create a pet
      responses:
        "201":
          description: Pet created
        "400":
          description: Invalid input
  /pets/{petId}:
    get:
      summary: Get a pet
      responses:
        "200":
          description: A single pet
        "404":
          description: Pet not found
        default:
          description: Unexpected error
`

func callWalkResponses(t *testing.T, input walkResponsesInput) (*mcp.CallToolResult, walkResponsesOutput) {
	t.Helper()
	result, out, err := handleWalkResponses(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	if out == nil {
		return result, walkResponsesOutput{}
	}
	wo, ok := out.(walkResponsesOutput)
	require.True(t, ok, "expected walkResponsesOutput, got %T", out)
	return result, wo
}

func TestWalkResponses_AllResponses(t *testing.T) {
	input := walkResponsesInput{
		Spec: specInput{Content: walkResponsesTestSpec},
	}
	_, output := callWalkResponses(t, input)

	assert.Equal(t, 8, output.Total)
	assert.Equal(t, 8, output.Matched)
	assert.Equal(t, 8, output.Returned)
	require.Len(t, output.Summaries, 8)
}

func TestWalkResponses_FilterByStatus(t *testing.T) {
	input := walkResponsesInput{
		Spec:   specInput{Content: walkResponsesTestSpec},
		Status: "200",
	}
	_, output := callWalkResponses(t, input)

	assert.Equal(t, 2, output.Matched)
	require.Len(t, output.Summaries, 2)
	for _, s := range output.Summaries {
		assert.Equal(t, "200", s.Status)
	}
}

func TestWalkResponses_FilterByStatusWildcard(t *testing.T) {
	input := walkResponsesInput{
		Spec:   specInput{Content: walkResponsesTestSpec},
		Status: "4xx",
	}
	_, output := callWalkResponses(t, input)

	assert.Equal(t, 3, output.Matched)
	for _, s := range output.Summaries {
		assert.True(t, s.Status[0] == '4', "expected 4xx status, got %s", s.Status)
	}
}

func TestWalkResponses_FilterByDefault(t *testing.T) {
	input := walkResponsesInput{
		Spec:   specInput{Content: walkResponsesTestSpec},
		Status: "default",
	}
	_, output := callWalkResponses(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "default", output.Summaries[0].Status)
	assert.Equal(t, "Unexpected error", output.Summaries[0].Description)
}

func TestWalkResponses_FilterByPath(t *testing.T) {
	input := walkResponsesInput{
		Spec: specInput{Content: walkResponsesTestSpec},
		Path: "/pets",
	}
	_, output := callWalkResponses(t, input)

	assert.Equal(t, 5, output.Matched)
	for _, s := range output.Summaries {
		assert.Equal(t, "/pets", s.Path)
	}
}

func TestWalkResponses_FilterByMethod(t *testing.T) {
	input := walkResponsesInput{
		Spec:   specInput{Content: walkResponsesTestSpec},
		Method: "post",
	}
	_, output := callWalkResponses(t, input)

	assert.Equal(t, 2, output.Matched)
	for _, s := range output.Summaries {
		assert.Equal(t, "POST", s.Method)
	}
}

func TestWalkResponses_DetailMode(t *testing.T) {
	input := walkResponsesInput{
		Spec:   specInput{Content: walkResponsesTestSpec},
		Status: "default",
		Detail: true,
	}
	_, output := callWalkResponses(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Nil(t, output.Summaries)
	require.Len(t, output.Responses, 1)
	assert.Equal(t, "default", output.Responses[0].Status)
	assert.NotNil(t, output.Responses[0].Response)
	assert.Equal(t, "Unexpected error", output.Responses[0].Response.Description)
}

func TestWalkResponses_Limit(t *testing.T) {
	input := walkResponsesInput{
		Spec:  specInput{Content: walkResponsesTestSpec},
		Limit: 3,
	}
	_, output := callWalkResponses(t, input)

	assert.Equal(t, 8, output.Total)
	assert.Equal(t, 8, output.Matched)
	assert.Equal(t, 3, output.Returned)
	assert.Len(t, output.Summaries, 3)
}

func TestWalkResponses_InvalidSpec(t *testing.T) {
	input := walkResponsesInput{
		Spec: specInput{Content: "not valid yaml: ["},
	}
	result, _ := callWalkResponses(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkResponses_NoMatches(t *testing.T) {
	input := walkResponsesInput{
		Spec:   specInput{Content: walkResponsesTestSpec},
		Status: "302",
	}
	_, output := callWalkResponses(t, input)

	assert.Equal(t, 0, output.Matched)
	assert.Nil(t, output.Summaries)
}

func TestStatusCodeMatches(t *testing.T) {
	tests := []struct {
		statusCode string
		filter     string
		want       bool
	}{
		{"200", "200", true},
		{"200", "201", false},
		{"200", "2xx", true},
		{"404", "4xx", true},
		{"500", "5xx", true},
		{"200", "4xx", false},
		{"default", "default", true},
		{"default", "200", false},
		{"200", "default", false},
	}
	for _, tt := range tests {
		t.Run(tt.statusCode+"_"+tt.filter, func(t *testing.T) {
			assert.Equal(t, tt.want, statusCodeMatches(tt.statusCode, tt.filter))
		})
	}
}
