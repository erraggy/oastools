package validator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

// TestValidateOAuth2Flows tests the validateOAuth2Flows method.
func TestValidateOAuth2Flows(t *testing.T) {
	tests := []struct {
		name          string
		flows         *parser.OAuthFlows
		expectErrors  int
		errorMessages []string
	}{
		{
			name: "valid implicit flow",
			flows: &parser.OAuthFlows{
				Implicit: &parser.OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					Scopes:           map[string]string{"read": "Read access"},
				},
			},
			expectErrors: 0,
		},
		{
			name: "implicit flow missing authorizationUrl",
			flows: &parser.OAuthFlows{
				Implicit: &parser.OAuthFlow{
					Scopes: map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  1,
			errorMessages: []string{"Implicit flow must have authorizationUrl"},
		},
		{
			name: "implicit flow invalid authorizationUrl",
			flows: &parser.OAuthFlows{
				Implicit: &parser.OAuthFlow{
					AuthorizationURL: "not-a-valid-url",
					Scopes:           map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  1,
			errorMessages: []string{"Invalid URL format for authorizationUrl"},
		},
		{
			name: "valid password flow",
			flows: &parser.OAuthFlows{
				Password: &parser.OAuthFlow{
					TokenURL: "https://example.com/oauth/token",
					Scopes:   map[string]string{"read": "Read access"},
				},
			},
			expectErrors: 0,
		},
		{
			name: "password flow missing tokenUrl",
			flows: &parser.OAuthFlows{
				Password: &parser.OAuthFlow{
					Scopes: map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  1,
			errorMessages: []string{"Password flow must have tokenUrl"},
		},
		{
			name: "password flow invalid tokenUrl",
			flows: &parser.OAuthFlows{
				Password: &parser.OAuthFlow{
					TokenURL: "invalid-url",
					Scopes:   map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  1,
			errorMessages: []string{"Invalid URL format for tokenUrl"},
		},
		{
			name: "valid clientCredentials flow",
			flows: &parser.OAuthFlows{
				ClientCredentials: &parser.OAuthFlow{
					TokenURL: "https://example.com/oauth/token",
					Scopes:   map[string]string{"read": "Read access"},
				},
			},
			expectErrors: 0,
		},
		{
			name: "clientCredentials flow missing tokenUrl",
			flows: &parser.OAuthFlows{
				ClientCredentials: &parser.OAuthFlow{
					Scopes: map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  1,
			errorMessages: []string{"Client credentials flow must have tokenUrl"},
		},
		{
			name: "clientCredentials flow invalid tokenUrl",
			flows: &parser.OAuthFlows{
				ClientCredentials: &parser.OAuthFlow{
					TokenURL: "bad-url",
					Scopes:   map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  1,
			errorMessages: []string{"Invalid URL format for tokenUrl"},
		},
		{
			name: "valid authorizationCode flow",
			flows: &parser.OAuthFlows{
				AuthorizationCode: &parser.OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					TokenURL:         "https://example.com/oauth/token",
					Scopes:           map[string]string{"read": "Read access"},
				},
			},
			expectErrors: 0,
		},
		{
			name: "authorizationCode flow missing authorizationUrl",
			flows: &parser.OAuthFlows{
				AuthorizationCode: &parser.OAuthFlow{
					TokenURL: "https://example.com/oauth/token",
					Scopes:   map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  1,
			errorMessages: []string{"Authorization code flow must have authorizationUrl"},
		},
		{
			name: "authorizationCode flow missing tokenUrl",
			flows: &parser.OAuthFlows{
				AuthorizationCode: &parser.OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					Scopes:           map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  1,
			errorMessages: []string{"Authorization code flow must have tokenUrl"},
		},
		{
			name: "authorizationCode flow invalid authorizationUrl",
			flows: &parser.OAuthFlows{
				AuthorizationCode: &parser.OAuthFlow{
					AuthorizationURL: "invalid",
					TokenURL:         "https://example.com/oauth/token",
					Scopes:           map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  1,
			errorMessages: []string{"Invalid URL format for authorizationUrl"},
		},
		{
			name: "authorizationCode flow invalid tokenUrl",
			flows: &parser.OAuthFlows{
				AuthorizationCode: &parser.OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					TokenURL:         "invalid",
					Scopes:           map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  1,
			errorMessages: []string{"Invalid URL format for tokenUrl"},
		},
		{
			name: "authorizationCode flow both URLs invalid",
			flows: &parser.OAuthFlows{
				AuthorizationCode: &parser.OAuthFlow{
					AuthorizationURL: "bad-auth-url",
					TokenURL:         "bad-token-url",
					Scopes:           map[string]string{"read": "Read access"},
				},
			},
			expectErrors:  2,
			errorMessages: []string{"Invalid URL format for authorizationUrl", "Invalid URL format for tokenUrl"},
		},
		{
			name: "multiple flows all valid",
			flows: &parser.OAuthFlows{
				Implicit: &parser.OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					Scopes:           map[string]string{"read": "Read access"},
				},
				Password: &parser.OAuthFlow{
					TokenURL: "https://example.com/oauth/token",
					Scopes:   map[string]string{"write": "Write access"},
				},
				ClientCredentials: &parser.OAuthFlow{
					TokenURL: "https://example.com/oauth/token",
					Scopes:   map[string]string{"admin": "Admin access"},
				},
				AuthorizationCode: &parser.OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					TokenURL:         "https://example.com/oauth/token",
					Scopes:           map[string]string{"full": "Full access"},
				},
			},
			expectErrors: 0,
		},
		{
			name: "multiple flows with errors",
			flows: &parser.OAuthFlows{
				Implicit: &parser.OAuthFlow{
					// Missing authorizationUrl
					Scopes: map[string]string{"read": "Read access"},
				},
				Password: &parser.OAuthFlow{
					// Missing tokenUrl
					Scopes: map[string]string{"write": "Write access"},
				},
			},
			expectErrors:  2,
			errorMessages: []string{"Implicit flow must have authorizationUrl", "Password flow must have tokenUrl"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := &ValidationResult{
				Valid:  true,
				Errors: []ValidationError{},
			}

			v.validateOAuth2Flows(tt.flows, "components.securitySchemes.oauth2", result, "https://spec.openapis.org/oas/v3.1.0")

			assert.Equal(t, tt.expectErrors, len(result.Errors), "Expected %d errors, got %d", tt.expectErrors, len(result.Errors))

			// Check that expected error messages are present
			for _, expectedMsg := range tt.errorMessages {
				found := false
				for _, err := range result.Errors {
					if strings.Contains(err.Message, expectedMsg) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected error message containing %q not found", expectedMsg)
			}

			// Verify error structure for first error if any
			if len(result.Errors) > 0 {
				err := result.Errors[0]
				assert.NotEmpty(t, err.Path, "Error path should not be empty")
				assert.NotEmpty(t, err.Message, "Error message should not be empty")
				assert.NotEmpty(t, err.SpecRef, "Error SpecRef should not be empty")
				assert.Equal(t, SeverityError, err.Severity, "Error severity should be SeverityError")
			}
		})
	}
}

// TestValidateOAuth2FlowsWithRefreshURL tests OAuth2 flows with refreshUrl.
func TestValidateOAuth2FlowsWithRefreshURL(t *testing.T) {
	tests := []struct {
		name         string
		flows        *parser.OAuthFlows
		expectErrors int
	}{
		{
			name: "password flow with valid refreshUrl",
			flows: &parser.OAuthFlows{
				Password: &parser.OAuthFlow{
					TokenURL:   "https://example.com/oauth/token",
					RefreshURL: "https://example.com/oauth/refresh",
					Scopes:     map[string]string{"read": "Read access"},
				},
			},
			expectErrors: 0,
		},
		{
			name: "authorizationCode flow with valid refreshUrl",
			flows: &parser.OAuthFlows{
				AuthorizationCode: &parser.OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					TokenURL:         "https://example.com/oauth/token",
					RefreshURL:       "https://example.com/oauth/refresh",
					Scopes:           map[string]string{"read": "Read access"},
				},
			},
			expectErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			result := &ValidationResult{
				Valid:  true,
				Errors: []ValidationError{},
			}

			v.validateOAuth2Flows(tt.flows, "components.securitySchemes.oauth2", result, "https://spec.openapis.org/oas/v3.1.0")

			assert.Equal(t, tt.expectErrors, len(result.Errors), "Expected %d errors, got %d", tt.expectErrors, len(result.Errors))
		})
	}
}

// TestGetJSONSchemaRef tests the getJSONSchemaRef function.
func TestGetJSONSchemaRef(t *testing.T) {
	result := getJSONSchemaRef()
	expected := "https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html"
	assert.Equal(t, expected, result, "getJSONSchemaRef() should return JSON Schema Draft 2020-12 URL")
	assert.NotEmpty(t, result, "getJSONSchemaRef() should return a non-empty string")
}
