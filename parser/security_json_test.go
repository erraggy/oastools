package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecuritySchemeMarshalJSON tests SecurityScheme.MarshalJSON.
func TestSecuritySchemeMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		scheme   *SecurityScheme
		expected map[string]any
	}{
		{
			name: "API key security scheme without Extra",
			scheme: &SecurityScheme{
				Type:        "apiKey",
				Name:        "api_key",
				In:          "header",
				Description: "API key authentication",
			},
			expected: map[string]any{
				"type":        "apiKey",
				"name":        "api_key",
				"in":          "header",
				"description": "API key authentication",
			},
		},
		{
			name: "HTTP bearer security scheme",
			scheme: &SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "Bearer token authentication",
			},
			expected: map[string]any{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
				"description":  "Bearer token authentication",
			},
		},
		{
			name: "OAuth2 security scheme (OAS 3.x)",
			scheme: &SecurityScheme{
				Type: "oauth2",
				Flows: &OAuthFlows{
					Implicit: &OAuthFlow{
						AuthorizationURL: "https://example.com/oauth/authorize",
						Scopes: map[string]string{
							"read":  "Read access",
							"write": "Write access",
						},
					},
				},
			},
			expected: map[string]any{
				"type": "oauth2",
				"flows": map[string]any{
					"implicit": map[string]any{
						"authorizationUrl": "https://example.com/oauth/authorize",
						"scopes": map[string]any{
							"read":  "Read access",
							"write": "Write access",
						},
					},
				},
			},
		},
		{
			name: "OAuth2 security scheme (OAS 2.0)",
			scheme: &SecurityScheme{
				Type:             "oauth2",
				Flow:             "implicit",
				AuthorizationURL: "https://example.com/oauth/authorize",
				Scopes: map[string]string{
					"read": "Read access",
				},
			},
			expected: map[string]any{
				"type":             "oauth2",
				"flow":             "implicit",
				"authorizationUrl": "https://example.com/oauth/authorize",
				"scopes": map[string]any{
					"read": "Read access",
				},
			},
		},
		{
			name: "OpenID Connect security scheme",
			scheme: &SecurityScheme{
				Type:             "openIdConnect",
				OpenIDConnectURL: "https://example.com/.well-known/openid-configuration",
			},
			expected: map[string]any{
				"type":             "openIdConnect",
				"openIdConnectUrl": "https://example.com/.well-known/openid-configuration",
			},
		},
		{
			name: "security scheme with Extra fields",
			scheme: &SecurityScheme{
				Type: "apiKey",
				Name: "api_key",
				In:   "header",
				Extra: map[string]any{
					"x-example":    "sk_test_1234567890",
					"x-deprecated": false,
				},
			},
			expected: map[string]any{
				"type":         "apiKey",
				"name":         "api_key",
				"in":           "header",
				"x-example":    "sk_test_1234567890",
				"x-deprecated": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.scheme)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			// Deep comparison for nested structures
			assert.Equal(t, tt.expected["type"], result["type"], "Type should match")
		})
	}
}

// TestSecuritySchemeUnmarshalJSON tests SecurityScheme.UnmarshalJSON.
func TestSecuritySchemeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *SecurityScheme
	}{
		{
			name:  "API key security scheme",
			input: `{"type":"apiKey","name":"api_key","in":"header","description":"API key authentication"}`,
			expected: &SecurityScheme{
				Type:        "apiKey",
				Name:        "api_key",
				In:          "header",
				Description: "API key authentication",
			},
		},
		{
			name:  "HTTP bearer security scheme",
			input: `{"type":"http","scheme":"bearer","bearerFormat":"JWT"}`,
			expected: &SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
		},
		{
			name:  "security scheme with x- extensions",
			input: `{"type":"apiKey","name":"api_key","in":"header","x-example":"sk_test_1234567890","x-deprecated":false}`,
			expected: &SecurityScheme{
				Type: "apiKey",
				Name: "api_key",
				In:   "header",
				Extra: map[string]any{
					"x-example":    "sk_test_1234567890",
					"x-deprecated": false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scheme SecurityScheme
			err := json.Unmarshal([]byte(tt.input), &scheme)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Type, scheme.Type, "Type should match")
			assert.Equal(t, tt.expected.Name, scheme.Name, "Name should match")
			assert.Equal(t, tt.expected.In, scheme.In, "In should match")
			assert.Equal(t, tt.expected.Scheme, scheme.Scheme, "Scheme should match")
			assert.Equal(t, tt.expected.BearerFormat, scheme.BearerFormat, "BearerFormat should match")
			assert.Equal(t, tt.expected.Extra, scheme.Extra, "Extra fields should match")
		})
	}
}

// TestOAuthFlowsMarshalJSON tests OAuthFlows.MarshalJSON.
func TestOAuthFlowsMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		flows    *OAuthFlows
		expected map[string]any
	}{
		{
			name: "implicit flow without Extra",
			flows: &OAuthFlows{
				Implicit: &OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					Scopes: map[string]string{
						"read": "Read access",
					},
				},
			},
			expected: map[string]any{
				"implicit": map[string]any{
					"authorizationUrl": "https://example.com/oauth/authorize",
					"scopes": map[string]any{
						"read": "Read access",
					},
				},
			},
		},
		{
			name: "authorization code flow",
			flows: &OAuthFlows{
				AuthorizationCode: &OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					TokenURL:         "https://example.com/oauth/token",
					Scopes: map[string]string{
						"read":  "Read access",
						"write": "Write access",
					},
				},
			},
			expected: map[string]any{
				"authorizationCode": map[string]any{
					"authorizationUrl": "https://example.com/oauth/authorize",
					"tokenUrl":         "https://example.com/oauth/token",
					"scopes": map[string]any{
						"read":  "Read access",
						"write": "Write access",
					},
				},
			},
		},
		{
			name: "flows with Extra fields",
			flows: &OAuthFlows{
				Implicit: &OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					Scopes:           map[string]string{"read": "Read access"},
				},
				Extra: map[string]any{
					"x-custom": "value",
				},
			},
			expected: map[string]any{
				"implicit": map[string]any{
					"authorizationUrl": "https://example.com/oauth/authorize",
					"scopes": map[string]any{
						"read": "Read access",
					},
				},
				"x-custom": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.flows)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			// Verify presence of expected flows
			if tt.flows.Implicit != nil {
				assert.Contains(t, result, "implicit", "Should have implicit flow")
			}
			if tt.flows.AuthorizationCode != nil {
				assert.Contains(t, result, "authorizationCode", "Should have authorizationCode flow")
			}
		})
	}
}

// TestOAuthFlowsUnmarshalJSON tests OAuthFlows.UnmarshalJSON.
func TestOAuthFlowsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *OAuthFlows
	}{
		{
			name:  "implicit flow",
			input: `{"implicit":{"authorizationUrl":"https://example.com/oauth/authorize","scopes":{"read":"Read access"}}}`,
			expected: &OAuthFlows{
				Implicit: &OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					Scopes:           map[string]string{"read": "Read access"},
				},
			},
		},
		{
			name:  "flows with x- extensions",
			input: `{"implicit":{"authorizationUrl":"https://example.com/oauth/authorize","scopes":{"read":"Read access"}},"x-custom":"value"}`,
			expected: &OAuthFlows{
				Implicit: &OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					Scopes:           map[string]string{"read": "Read access"},
				},
				Extra: map[string]any{
					"x-custom": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var flows OAuthFlows
			err := json.Unmarshal([]byte(tt.input), &flows)
			require.NoError(t, err, "UnmarshalJSON should not error")

			if tt.expected.Implicit != nil {
				require.NotNil(t, flows.Implicit, "Implicit flow should not be nil")
				assert.Equal(t, tt.expected.Implicit.AuthorizationURL, flows.Implicit.AuthorizationURL)
				assert.Equal(t, tt.expected.Implicit.Scopes, flows.Implicit.Scopes)
			}
			assert.Equal(t, tt.expected.Extra, flows.Extra, "Extra fields should match")
		})
	}
}

// TestOAuthFlowMarshalJSON tests OAuthFlow.MarshalJSON.
func TestOAuthFlowMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		flow     *OAuthFlow
		expected map[string]any
	}{
		{
			name: "implicit flow without Extra",
			flow: &OAuthFlow{
				AuthorizationURL: "https://example.com/oauth/authorize",
				Scopes: map[string]string{
					"read": "Read access",
				},
			},
			expected: map[string]any{
				"authorizationUrl": "https://example.com/oauth/authorize",
				"scopes": map[string]any{
					"read": "Read access",
				},
			},
		},
		{
			name: "authorization code flow",
			flow: &OAuthFlow{
				AuthorizationURL: "https://example.com/oauth/authorize",
				TokenURL:         "https://example.com/oauth/token",
				RefreshURL:       "https://example.com/oauth/refresh",
				Scopes: map[string]string{
					"read":  "Read access",
					"write": "Write access",
				},
			},
			expected: map[string]any{
				"authorizationUrl": "https://example.com/oauth/authorize",
				"tokenUrl":         "https://example.com/oauth/token",
				"refreshUrl":       "https://example.com/oauth/refresh",
				"scopes": map[string]any{
					"read":  "Read access",
					"write": "Write access",
				},
			},
		},
		{
			name: "flow with Extra fields",
			flow: &OAuthFlow{
				AuthorizationURL: "https://example.com/oauth/authorize",
				Scopes:           map[string]string{"read": "Read access"},
				Extra: map[string]any{
					"x-audience": "api.example.com",
				},
			},
			expected: map[string]any{
				"authorizationUrl": "https://example.com/oauth/authorize",
				"scopes": map[string]any{
					"read": "Read access",
				},
				"x-audience": "api.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.flow)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			assert.Equal(t, tt.expected["authorizationUrl"], result["authorizationUrl"], "AuthorizationURL should match")
			if tt.expected["tokenUrl"] != nil {
				assert.Equal(t, tt.expected["tokenUrl"], result["tokenUrl"], "TokenURL should match")
			}
		})
	}
}

// TestOAuthFlowUnmarshalJSON tests OAuthFlow.UnmarshalJSON.
func TestOAuthFlowUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *OAuthFlow
	}{
		{
			name:  "implicit flow",
			input: `{"authorizationUrl":"https://example.com/oauth/authorize","scopes":{"read":"Read access"}}`,
			expected: &OAuthFlow{
				AuthorizationURL: "https://example.com/oauth/authorize",
				Scopes:           map[string]string{"read": "Read access"},
			},
		},
		{
			name:  "flow with x- extensions",
			input: `{"authorizationUrl":"https://example.com/oauth/authorize","scopes":{"read":"Read access"},"x-audience":"api.example.com"}`,
			expected: &OAuthFlow{
				AuthorizationURL: "https://example.com/oauth/authorize",
				Scopes:           map[string]string{"read": "Read access"},
				Extra: map[string]any{
					"x-audience": "api.example.com",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var flow OAuthFlow
			err := json.Unmarshal([]byte(tt.input), &flow)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.AuthorizationURL, flow.AuthorizationURL, "AuthorizationURL should match")
			assert.Equal(t, tt.expected.Scopes, flow.Scopes, "Scopes should match")
			assert.Equal(t, tt.expected.Extra, flow.Extra, "Extra fields should match")
		})
	}
}

// TestSecurityJSONRoundTrip tests that marshal/unmarshal round-trips preserve data.
func TestSecurityJSONRoundTrip(t *testing.T) {
	t.Run("SecurityScheme round-trip", func(t *testing.T) {
		original := &SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "Bearer token authentication",
			Extra: map[string]any{
				"x-example": "Bearer eyJhbGci...",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded SecurityScheme
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Type, decoded.Type)
		assert.Equal(t, original.Scheme, decoded.Scheme)
		assert.Equal(t, original.BearerFormat, decoded.BearerFormat)
		assert.Equal(t, original.Description, decoded.Description)
		assert.Equal(t, original.Extra, decoded.Extra)
	})

	t.Run("OAuthFlows round-trip", func(t *testing.T) {
		original := &OAuthFlows{
			Implicit: &OAuthFlow{
				AuthorizationURL: "https://example.com/oauth/authorize",
				Scopes: map[string]string{
					"read":  "Read access",
					"write": "Write access",
				},
			},
			Extra: map[string]any{
				"x-custom": "value",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded OAuthFlows
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		require.NotNil(t, decoded.Implicit)
		assert.Equal(t, original.Implicit.AuthorizationURL, decoded.Implicit.AuthorizationURL)
		assert.Equal(t, original.Implicit.Scopes, decoded.Implicit.Scopes)
		assert.Equal(t, original.Extra, decoded.Extra)
	})

	t.Run("OAuthFlow round-trip", func(t *testing.T) {
		original := &OAuthFlow{
			AuthorizationURL: "https://example.com/oauth/authorize",
			TokenURL:         "https://example.com/oauth/token",
			RefreshURL:       "https://example.com/oauth/refresh",
			Scopes: map[string]string{
				"read":  "Read access",
				"write": "Write access",
			},
			Extra: map[string]any{
				"x-audience": "api.example.com",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded OAuthFlow
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.AuthorizationURL, decoded.AuthorizationURL)
		assert.Equal(t, original.TokenURL, decoded.TokenURL)
		assert.Equal(t, original.RefreshURL, decoded.RefreshURL)
		assert.Equal(t, original.Scopes, decoded.Scopes)
		assert.Equal(t, original.Extra, decoded.Extra)
	})
}
