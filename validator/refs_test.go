package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRefValidation tests that $ref values are properly validated
func TestRefValidation(t *testing.T) {
	testCases := []struct {
		name          string
		oasVersion    string
		content       string
		expectError   bool
		errorContains string
	}{
		{
			name:       "OAS 2.0 - Valid ref to definitions",
			oasVersion: "2.0",
			content: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          schema:
            $ref: "#/definitions/Pet"
definitions:
  Pet:
    type: object
    properties:
      name:
        type: string`,
			expectError: false,
		},
		{
			name:       "OAS 2.0 - Invalid ref using OAS 3.x format",
			oasVersion: "2.0",
			content: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          schema:
            $ref: "#/components/schemas/Pet"
definitions:
  Pet:
    type: object`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
		{
			name:       "OAS 2.0 - Ref to non-existent definition",
			oasVersion: "2.0",
			content: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          schema:
            $ref: "#/definitions/NonExistent"
definitions:
  Pet:
    type: object`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
		{
			name:       "OAS 3.0 - Valid ref to components/schemas",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string`,
			expectError: false,
		},
		{
			name:       "OAS 3.0 - Invalid ref using OAS 2.0 format",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/definitions/Pet"
components:
  schemas:
    Pet:
      type: object`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
		{
			name:       "OAS 3.0 - Ref to non-existent schema",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/NonExistent"
components:
  schemas:
    Pet:
      type: object`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
		{
			name:       "OAS 3.0 - Valid ref in nested schema",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: object
                properties:
                  pet:
                    $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object`,
			expectError: false,
		},
		{
			name:       "OAS 3.0 - Invalid ref in nested schema",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: object
                properties:
                  pet:
                    $ref: "#/components/schemas/NonExistent"
components:
  schemas:
    Pet:
      type: object`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
		{
			name:       "OAS 3.0 - Valid ref to parameter",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: "#/components/parameters/PetId"
      responses:
        '200':
          description: Success
components:
  parameters:
    PetId:
      name: petId
      in: path
      required: true
      schema:
        type: string`,
			expectError: false,
		},
		{
			name:       "OAS 3.0 - Invalid ref to parameter",
			oasVersion: "3.0.3",
			content: `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: "#/components/parameters/NonExistent"
      responses:
        '200':
          description: Success
components:
  parameters:
    PetId:
      name: petId
      in: path
      required: true
      schema:
        type: string`,
			expectError:   true,
			errorContains: "does not resolve to a valid component",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write test file
			tmpFile := filepath.Join(t.TempDir(), "test.yaml")
			err := os.WriteFile(tmpFile, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Validate
			v := New()
			result, err := v.Validate(tmpFile)
			if err != nil {
				t.Fatalf("Validate failed: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			hasRefError := false
			for _, validationErr := range result.Errors {
				if tc.errorContains != "" && validationErr.Field == "$ref" {
					if !strings.Contains(validationErr.Message, tc.errorContains) {
						t.Errorf("Expected error containing '%s', got: %s", tc.errorContains, validationErr.Message)
					}
					hasRefError = true
				}
			}

			if tc.expectError && !hasRefError {
				t.Errorf("Expected $ref validation error, but got none. Errors: %v", result.Errors)
			}

			if !tc.expectError && hasRefError {
				t.Errorf("Did not expect $ref validation error, but got one")
			}
		})
	}
}

// TestValidate_RefEndingWithSlash_OAS2 tests that $ref ending with / reports an error
func TestValidate_RefEndingWithSlash_OAS2(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]*parser.PathItem{
			"/test": {
				Get: &parser.Operation{
					OperationID: "getTest",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Schema: &parser.Schema{
									Ref: "#/definitions/", // Ref ending with /
								},
							},
						},
					},
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"ValidSchema": {
				Type: "object",
			},
		},
	}

	parseResult := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	assert.False(t, result.Valid, "Document should be invalid with $ref ending in /")
	assert.NotEmpty(t, result.Errors, "Should have validation errors")

	// Check for ref ending with / error
	foundError := false
	for _, e := range result.Errors {
		if e.Field == "$ref" && strings.Contains(e.Message, "references an empty schema name") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about $ref referencing empty schema name")
}

// TestValidate_RefEndingWithSlash_OAS3 tests that $ref ending with / reports an error in OAS 3.x
func TestValidate_RefEndingWithSlash_OAS3(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]*parser.PathItem{
			"/test": {
				Get: &parser.Operation{
					OperationID: "getTest",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Ref: "#/components/schemas/", // Ref ending with /
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"ValidSchema": {
					Type: "object",
				},
			},
		},
	}

	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	assert.False(t, result.Valid, "Document should be invalid with $ref ending in /")
	assert.NotEmpty(t, result.Errors, "Should have validation errors")

	// Check for ref ending with / error
	foundError := false
	for _, e := range result.Errors {
		if e.Field == "$ref" && strings.Contains(e.Message, "references an empty schema name") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about $ref referencing empty schema name")
}
