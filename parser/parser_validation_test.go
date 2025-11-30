package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseValidationErrors(t *testing.T) {
	parser := New()
	data := []byte(`
swagger: "2.0"
paths: {}
`)
	result, err := parser.ParseBytes(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for missing required fields")
	}

	// Should have errors for missing info
	hasInfoError := false
	for _, err := range result.Errors {
		// Check if error message mentions missing info field
		errMsg := err.Error()
		if strings.Contains(errMsg, "info") && strings.Contains(errMsg, "missing") {
			hasInfoError = true
			break
		}
	}
	if !hasInfoError {
		t.Errorf("Expected error for missing info field, got: %v", result.Errors)
	}
}

func TestParseWithValidationDisabled(t *testing.T) {
	parser := New()
	parser.ValidateStructure = false

	data := []byte(`
swagger: "2.0"
paths: {}
`)
	result, err := parser.ParseBytes(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Errors) > 0 {
		t.Error("Should not have validation errors when validation is disabled")
	}
}

// TestWebhooksVersionValidation tests that webhooks are properly validated based on version
func TestWebhooksVersionValidation(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		includeWebhooks bool
		expectError     bool
		errorContains   string
	}{
		{
			name:            "Webhooks in OAS 3.0.0 should error",
			version:         "3.0.0",
			includeWebhooks: true,
			expectError:     true,
			errorContains:   "webhooks",
		},
		{
			name:            "Webhooks in OAS 3.0.1 should error",
			version:         "3.0.1",
			includeWebhooks: true,
			expectError:     true,
			errorContains:   "webhooks",
		},
		{
			name:            "Webhooks in OAS 3.1.0 should be valid",
			version:         "3.1.0",
			includeWebhooks: true,
			expectError:     false,
		},
		{
			name:            "Webhooks in OAS 3.2.0 should be valid",
			version:         "3.2.0",
			includeWebhooks: true,
			expectError:     false,
		},
		{
			name:            "No webhooks in OAS 3.0.0 should be valid",
			version:         "3.0.0",
			includeWebhooks: false,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()

			webhooksSection := ""
			if tt.includeWebhooks {
				webhooksSection = `
webhooks:
  newPet:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        '200':
          description: Success
`
			}

			data := []byte(`openapi: "` + tt.version + `"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
` + webhooksSection)

			result, err := parser.ParseBytes(data)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			hasWebhookError := false
			for _, e := range result.Errors {
				if strings.Contains(e.Error(), tt.errorContains) {
					hasWebhookError = true
					break
				}
			}

			if tt.expectError && !hasWebhookError {
				t.Errorf("Expected error containing '%s' for version %s with webhooks, but got errors: %v",
					tt.errorContains, tt.version, result.Errors)
			}

			if !tt.expectError && hasWebhookError {
				t.Errorf("Did not expect webhook error for version %s, but got: %v",
					tt.version, result.Errors)
			}
		})
	}
}

// TestPathsRequirementVersionValidation tests that paths requirement is properly validated based on version
func TestPathsRequirementVersionValidation(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		includePaths    bool
		includeWebhooks bool
		expectError     bool
		errorContains   string
	}{
		{
			name:            "OAS 3.0.0 requires paths",
			version:         "3.0.0",
			includePaths:    false,
			includeWebhooks: false,
			expectError:     true,
			errorContains:   "paths",
		},
		{
			name:            "OAS 3.0.2 requires paths",
			version:         "3.0.2",
			includePaths:    false,
			includeWebhooks: false,
			expectError:     true,
			errorContains:   "paths",
		},
		{
			name:            "OAS 3.1.0 requires paths or webhooks",
			version:         "3.1.0",
			includePaths:    false,
			includeWebhooks: false,
			expectError:     true,
			errorContains:   "paths",
		},
		{
			name:            "OAS 3.1.0 with webhooks is valid",
			version:         "3.1.0",
			includePaths:    false,
			includeWebhooks: true,
			expectError:     false,
		},
		{
			name:            "OAS 3.2.0 with webhooks is valid",
			version:         "3.2.0",
			includePaths:    false,
			includeWebhooks: true,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()

			pathsSection := ""
			if tt.includePaths {
				pathsSection = `paths:
  /test:
    get:
      responses:
        '200':
          description: Success
`
			}

			webhooksSection := ""
			if tt.includeWebhooks {
				webhooksSection = `webhooks:
  newPet:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        '200':
          description: Success
`
			}

			data := []byte(`openapi: "` + tt.version + `"
info:
  title: Test API
  version: 1.0.0
` + pathsSection + webhooksSection)

			result, err := parser.ParseBytes(data)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			hasExpectedError := false
			for _, e := range result.Errors {
				if strings.Contains(e.Error(), tt.errorContains) {
					hasExpectedError = true
					break
				}
			}

			if tt.expectError && !hasExpectedError {
				t.Errorf("Expected error containing '%s' for version %s, but got errors: %v",
					tt.errorContains, tt.version, result.Errors)
			}

			if !tt.expectError && len(result.Errors) > 0 {
				t.Errorf("Did not expect errors for version %s, but got: %v",
					tt.version, result.Errors)
			}
		})
	}
}

// TestInvalidVersionValidation tests that invalid version strings are properly rejected
func TestInvalidVersionValidation(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Version 4.0.0 should be rejected",
			version:       "4.0.0",
			expectError:   true,
			errorContains: "invalid OAS version",
		},
		{
			name:          "Version 2.5.0 should be rejected",
			version:       "2.5.0",
			expectError:   true,
			errorContains: "invalid OAS version",
		},
		{
			name:          "Version 5.0.0 should be rejected",
			version:       "5.0.0",
			expectError:   true,
			errorContains: "invalid OAS version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()

			data := []byte(`openapi: "` + tt.version + `"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
`)

			result, err := parser.ParseBytes(data)
			if tt.expectError {
				assert.Nil(t, result)
				assert.ErrorContains(t, err, tt.errorContains)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestInvalidStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode string
		oasVersion string
		expectErr  bool
	}{
		{"Valid 200", "200", "2.0", false},
		{"Valid 404", "404", "3.0.0", false},
		{"Valid 2XX wildcard", "2XX", "3.0.0", false},
		{"Valid 5XX wildcard", "5XX", "2.0", false},
		{"Valid default", "default", "3.0.0", false},
		{"Valid extension field x-custom", "x-custom", "3.0.0", false},
		{"Valid extension field x-rate-limit", "x-rate-limit", "2.0", false},
		{"Valid extension field x-", "x-", "3.0.0", false},
		{"Invalid 99 - too low", "99", "3.0.0", true},
		{"Invalid 600 - too high", "600", "2.0", true},
		{"Invalid 6XX - out of range wildcard", "6XX", "3.0.0", true},
		{"Invalid XXX - all wildcards", "XXX", "3.0.0", true},
		{"Invalid 2X3 - mixed wildcard", "2X3", "2.0", true},
		{"Invalid empty string", "", "3.0.0", true},
		{"Invalid two chars", "20", "3.0.0", true},
		{"Invalid four chars", "2000", "2.0", true},
		{"Invalid non-numeric", "abc", "3.0.0", true},
		{"Invalid x without dash", "x", "3.0.0", true},
		{"Invalid xCustom without dash", "xCustom", "2.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var spec string
			if tt.oasVersion == "2.0" {
				spec = `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '` + tt.statusCode + `':
          description: Test response
`
			} else {
				spec = `openapi: "` + tt.oasVersion + `"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '` + tt.statusCode + `':
          description: Test response
`
			}

			parser := New()
			result, err := parser.ParseBytes([]byte(spec))

			// Check for invalid status code error in either parse error or validation errors
			// Parse error check (fail-fast during unmarshaling)
			hasStatusCodeError := err != nil && strings.Contains(err.Error(), "invalid status code")

			// Check validation errors (caught during validation phase)
			if !hasStatusCodeError && result != nil {
				for _, e := range result.Errors {
					if strings.Contains(e.Error(), "invalid status code") {
						hasStatusCodeError = true
						break
					}
				}
			}

			if tt.expectErr && !hasStatusCodeError {
				t.Errorf("Expected invalid status code error for '%s', but got no such error. Parse error: %v, Validation errors: %v",
					tt.statusCode, err, result.Errors)
			}

			if !tt.expectErr && hasStatusCodeError {
				t.Errorf("Did not expect invalid status code error for '%s', but got one. Parse error: %v, Validation errors: %v",
					tt.statusCode, err, result.Errors)
			}

			// For valid status codes, ensure parsing succeeded
			if !tt.expectErr && err != nil {
				t.Errorf("Expected successful parse for valid status code '%s', but got parse error: %v",
					tt.statusCode, err)
			}
		})
	}
}

func TestDuplicateOperationIds(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		expectErr bool
		errorMsg  string
	}{
		{
			name: "OAS 2.0 - Duplicate operationId",
			spec: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
  /accounts:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`,
			expectErr: true,
			errorMsg:  "duplicate operationId",
		},
		{
			name: "OAS 3.0 - Duplicate operationId",
			spec: `openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
  /accounts:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`,
			expectErr: true,
			errorMsg:  "duplicate operationId",
		},
		{
			name: "OAS 3.1 - Unique operationIds",
			spec: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
  /accounts:
    get:
      operationId: getAccount
      responses:
        '200':
          description: Success
`,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			result, err := parser.ParseBytes([]byte(tt.spec))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			hasDuplicateError := false
			for _, e := range result.Errors {
				if strings.Contains(e.Error(), tt.errorMsg) {
					hasDuplicateError = true
					break
				}
			}

			if tt.expectErr && !hasDuplicateError {
				t.Errorf("Expected duplicate operationId error, but got none. Errors: %v", result.Errors)
			}

			if !tt.expectErr && hasDuplicateError {
				t.Errorf("Did not expect duplicate operationId error, but got one. Errors: %v", result.Errors)
			}
		})
	}
}
