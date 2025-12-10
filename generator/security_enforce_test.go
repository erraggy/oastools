package generator

import (
	"strings"
	"testing"

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
	if !strings.Contains(result, "package api") {
		t.Error("expected package declaration")
	}

	// Check imports
	if !strings.Contains(result, `"fmt"`) {
		t.Error("expected fmt import")
	}

	// Check SecurityRequirement type
	if !strings.Contains(result, "type SecurityRequirement struct") {
		t.Error("expected SecurityRequirement struct")
	}
	if !strings.Contains(result, "Scheme string") {
		t.Error("expected Scheme field")
	}
	if !strings.Contains(result, "Scopes []string") {
		t.Error("expected Scopes field")
	}

	// Check operation security map
	if !strings.Contains(result, "var OperationSecurity = map[string][]SecurityRequirement") {
		t.Error("expected OperationSecurity map")
	}
	if !strings.Contains(result, `"listUsers"`) {
		t.Error("expected listUsers operation")
	}
	if !strings.Contains(result, `"user.read"`) {
		t.Error("expected user.read scope")
	}

	// Check global security
	if !strings.Contains(result, "var GlobalSecurity = []SecurityRequirement") {
		t.Error("expected GlobalSecurity")
	}
	if !strings.Contains(result, `Scheme: "api_key"`) {
		t.Error("expected api_key scheme in global security")
	}

	// Check SecurityValidator
	if !strings.Contains(result, "type SecurityValidator struct") {
		t.Error("expected SecurityValidator struct")
	}
	if !strings.Contains(result, "func NewSecurityValidator()") {
		t.Error("expected NewSecurityValidator function")
	}
	if !strings.Contains(result, "func (v *SecurityValidator) ConfigureScheme") {
		t.Error("expected ConfigureScheme method")
	}
	if !strings.Contains(result, "func (v *SecurityValidator) ValidateOperation") {
		t.Error("expected ValidateOperation method")
	}
}

func TestSecurityEnforceGenerator_EmptySecurity(t *testing.T) {
	g := NewSecurityEnforceGenerator("api")

	result := g.GenerateSecurityEnforceFile(nil, nil)

	// Should still have the basic structure
	if !strings.Contains(result, "type SecurityRequirement struct") {
		t.Error("expected SecurityRequirement struct")
	}
	if !strings.Contains(result, "var OperationSecurity = map[string][]SecurityRequirement") {
		t.Error("expected OperationSecurity map")
	}

	// Should NOT have global security when empty
	if strings.Contains(result, "var GlobalSecurity") {
		t.Error("did not expect GlobalSecurity for empty security")
	}
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
		if got != tt.want {
			t.Errorf("quotedStrings(%v) = %q, want %q", tt.input, got, tt.want)
		}
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

	// listUsers should have its own security
	listUsersSec, ok := result["listUsers"]
	if !ok {
		t.Error("expected listUsers in result")
	}
	if len(listUsersSec) != 1 {
		t.Errorf("expected 1 security requirement for listUsers, got %d", len(listUsersSec))
	}
	if _, hasOAuth2 := listUsersSec[0]["oauth2"]; !hasOAuth2 {
		t.Error("expected oauth2 scheme for listUsers")
	}

	// createUser should have global security
	createUserSec, ok := result["createUser"]
	if !ok {
		t.Error("expected createUser in result")
	}
	if len(createUserSec) != 1 {
		t.Errorf("expected 1 security requirement for createUser, got %d", len(createUserSec))
	}
	if _, hasGlobal := createUserSec[0]["global_auth"]; !hasGlobal {
		t.Error("expected global_auth scheme for createUser")
	}
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

	listPetsSec, ok := result["listPets"]
	if !ok {
		t.Error("expected listPets in result")
	}
	if len(listPetsSec) != 1 {
		t.Errorf("expected 1 security requirement for listPets, got %d", len(listPetsSec))
	}
}

func TestExtractOperationSecurityOAS3_Empty(t *testing.T) {
	doc := &parser.OAS3Document{}
	result := ExtractOperationSecurityOAS3(doc)

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}
