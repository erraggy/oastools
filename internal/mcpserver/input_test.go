package mcpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecInput_ResolveFile(t *testing.T) {
	// Use an existing testdata file from the repo
	input := specInput{File: "../../testdata/petstore-3.0.yaml"}
	result, err := input.resolve()
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Version)
}

func TestSpecInput_ResolveContent(t *testing.T) {
	content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`
	input := specInput{Content: content}
	result, err := input.resolve()
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "3.0.0", result.Version)
}

func TestSpecInput_ResolveNoneProvided(t *testing.T) {
	input := specInput{}
	_, err := input.resolve()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of file, url, or content must be provided")
}

func TestSpecInput_ResolveMultipleProvided(t *testing.T) {
	input := specInput{File: "foo.yaml", Content: "bar"}
	_, err := input.resolve()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of file, url, or content must be provided")
}

func TestSpecInput_ResolveFileNotFound(t *testing.T) {
	input := specInput{File: "/nonexistent/path.yaml"}
	_, err := input.resolve()
	assert.Error(t, err)
}
