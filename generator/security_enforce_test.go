package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

func TestSecurityEnforceGenerator_GenerateSecurityEnforceFile(t *testing.T) {
	g := NewSecurityEnforceGenerator("api")

	opSecurity := OperationSecurityRequirements{
		"listUsers": {
			{"oauth2": []string{"user.read"}},
		},
		"createUser": {
			{"oauth2": []string{"user.write"}},
		},
	}

	globalSecurity := []parser.SecurityRequirement{
		{"api_key": {}},
	}

	result := g.GenerateSecurityEnforceFile(opSecurity, globalSecurity)

	// Check package declaration
	assert.Contains(t, result, "package api", "expected package declaration")

	// Check imports
	assert.Contains(t, result, `"fmt"`, "expected fmt import")

	// Check SecurityRequirement type
	assert.Contains(t, result, "type SecurityRequirement struct", "expected SecurityRequirement struct")
	assert.Contains(t, result, "Scheme string", "expected Scheme field")
	assert.Contains(t, result, "Scopes []string", "expected Scopes field")

	// Check operation security map
	assert.Contains(t, result, "var OperationSecurity = map[string][]SecurityRequirement", "expected OperationSecurity map")
	assert.Contains(t, result, `"listUsers"`, "expected listUsers operation")
	assert.Contains(t, result, `"user.read"`, "expected user.read scope")

	// Check global security
	assert.Contains(t, result, "var GlobalSecurity = []SecurityRequirement", "expected GlobalSecurity")
	assert.Contains(t, result, `Scheme: "api_key"`, "expected api_key scheme in global security")

	// Check SecurityValidator
	assert.Contains(t, result, "type SecurityValidator struct", "expected SecurityValidator struct")
	assert.Contains(t, result, "func NewSecurityValidator()", "expected NewSecurityValidator function")
	assert.Contains(t, result, "func (v *SecurityValidator) ConfigureScheme", "expected ConfigureScheme method")
	assert.Contains(t, result, "func (v *SecurityValidator) ValidateOperation", "expected ValidateOperation method")
}

func TestSecurityEnforceGenerator_EmptySecurity(t *testing.T) {
	g := NewSecurityEnforceGenerator("api")

	result := g.GenerateSecurityEnforceFile(nil, nil)

	// Should still have the basic structure
	assert.Contains(t, result, "type SecurityRequirement struct", "expected SecurityRequirement struct")
	assert.Contains(t, result, "var OperationSecurity = map[string][]SecurityRequirement", "expected OperationSecurity map")

	// Should NOT have global security when empty
	assert.NotContains(t, result, "var GlobalSecurity", "did not expect GlobalSecurity for empty security")
}

func TestQuotedStrings(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{[]string{"a", "b", "c"}, `"a", "b", "c"`},
		{[]string{"single"}, `"single"`},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		got := quotedStrings(tt.input)
		assert.Equal(t, tt.want, got, "quotedStrings(%v)", tt.input)
	}
}

func TestExtractOperationSecurityOAS3(t *testing.T) {
	doc := &parser.OAS3Document{
		Security: []parser.SecurityRequirement{
			{"global_auth": {}},
		},
		Paths: map[string]*parser.PathItem{
			"/users": {
				Get: &parser.Operation{
					OperationID: "listUsers",
					Security: []parser.SecurityRequirement{
						{"oauth2": []string{"user.read"}},
					},
				},
				Post: &parser.Operation{
					OperationID: "createUser",
					// No operation-level security, should use global
				},
			},
		},
	}

	result := ExtractOperationSecurityOAS3(doc)

	// ListUsers (transformed from listUsers) should have its own security
	listUsersSec, ok := result["ListUsers"]
	require.True(t, ok, "expected ListUsers in result")
	assert.Len(t, listUsersSec, 1, "expected 1 security requirement for ListUsers")
	_, hasOAuth2 := listUsersSec[0]["oauth2"]
	assert.True(t, hasOAuth2, "expected oauth2 scheme for ListUsers")

	// CreateUser (transformed from createUser) should have global security
	createUserSec, ok := result["CreateUser"]
	require.True(t, ok, "expected CreateUser in result")
	assert.Len(t, createUserSec, 1, "expected 1 security requirement for CreateUser")
	_, hasGlobal := createUserSec[0]["global_auth"]
	assert.True(t, hasGlobal, "expected global_auth scheme for CreateUser")
}

func TestExtractOperationSecurityOAS2(t *testing.T) {
	doc := &parser.OAS2Document{
		Security: []parser.SecurityRequirement{
			{"api_key": {}},
		},
		Paths: map[string]*parser.PathItem{
			"/pets": {
				Get: &parser.Operation{
					OperationID: "listPets",
				},
			},
		},
	}

	result := ExtractOperationSecurityOAS2(doc)

	// ListPets (transformed from listPets) should have global security
	listPetsSec, ok := result["ListPets"]
	require.True(t, ok, "expected ListPets in result")
	assert.Len(t, listPetsSec, 1, "expected 1 security requirement for ListPets")
}

func TestExtractOperationSecurityOAS3_Empty(t *testing.T) {
	doc := &parser.OAS3Document{}
	result := ExtractOperationSecurityOAS3(doc)

	assert.Empty(t, result, "expected empty result")
}
