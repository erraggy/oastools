package differ

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// TestDiffSchemaCircularReferences tests that circular references don't cause infinite loops
func TestDiffSchemaCircularReferences(t *testing.T) {
	// Create schemas with circular references
	schema1 := &parser.Schema{
		Type:       "object",
		Properties: make(map[string]*parser.Schema),
	}
	// Self-reference
	schema1.Properties["self"] = schema1

	schema2 := &parser.Schema{
		Type:       "object",
		Properties: make(map[string]*parser.Schema),
	}
	// Self-reference
	schema2.Properties["self"] = schema2

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	// Should not panic or infinite loop
	d.diffSchemaRecursiveBreaking(schema1, schema2, "test.schema", visited, result)

	if len(result.Changes) > 0 {
		t.Errorf("Expected no changes for identical circular structures, got %d", len(result.Changes))
	}
}

// TestDiffSchemaCircularReferencesTargetOnly tests cycle detection when only target has a cycle
func TestDiffSchemaCircularReferencesTargetOnly(t *testing.T) {
	// Source: simple non-circular schema
	source := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	}

	// Target: schema with circular reference
	target := &parser.Schema{
		Type:       "object",
		Properties: make(map[string]*parser.Schema),
	}
	target.Properties["self"] = target

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	// Should not panic or infinite loop
	d.diffSchemaRecursiveBreaking(source, target, "test.schema", visited, result)

	// We expect changes (property added/removed), but no infinite loop
	if len(result.Changes) == 0 {
		t.Error("Expected changes for different structures")
	}
}

// TestDiffSchemaPropertiesRequired tests severity for required vs optional properties
func TestDiffSchemaPropertiesRequired(t *testing.T) {
	tests := []struct {
		name            string
		sourceRequired  []string
		targetRequired  []string
		removedProp     string
		addedProp       string
		expectedRemoved Severity
		expectedAdded   Severity
	}{
		{
			name:            "removed required property",
			sourceRequired:  []string{"name"},
			targetRequired:  []string{},
			removedProp:     "name",
			expectedRemoved: SeverityError,
		},
		{
			name:            "removed optional property",
			sourceRequired:  []string{},
			targetRequired:  []string{},
			removedProp:     "name",
			expectedRemoved: SeverityWarning,
		},
		{
			name:           "added required property",
			sourceRequired: []string{},
			targetRequired: []string{"email"},
			addedProp:      "email",
			expectedAdded:  SeverityWarning,
		},
		{
			name:           "added optional property",
			sourceRequired: []string{},
			targetRequired: []string{},
			addedProp:      "email",
			expectedAdded:  SeverityInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &parser.Schema{
				Type:       "object",
				Required:   tt.sourceRequired,
				Properties: make(map[string]*parser.Schema),
			}
			target := &parser.Schema{
				Type:       "object",
				Required:   tt.targetRequired,
				Properties: make(map[string]*parser.Schema),
			}

			if tt.removedProp != "" {
				source.Properties[tt.removedProp] = &parser.Schema{Type: "string"}
			}
			if tt.addedProp != "" {
				target.Properties[tt.addedProp] = &parser.Schema{Type: "string"}
			}

			d := New()
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaRecursiveBreaking(source, target, "test.schema", visited, result)

			if len(result.Changes) == 0 {
				t.Fatal("Expected changes but got none")
			}

			// Find the property addition/removal change
			found := false
			for _, change := range result.Changes {
				if tt.removedProp != "" && change.Type == ChangeTypeRemoved && change.Path == "test.schema.properties."+tt.removedProp {
					if change.Severity != tt.expectedRemoved {
						t.Errorf("Expected removed property severity %v, got %v", tt.expectedRemoved, change.Severity)
					}
					found = true
					break
				}
				if tt.addedProp != "" && change.Type == ChangeTypeAdded && change.Path == "test.schema.properties."+tt.addedProp {
					if change.Severity != tt.expectedAdded {
						t.Errorf("Expected added property severity %v, got %v", tt.expectedAdded, change.Severity)
					}
					found = true
					break
				}
			}
			if !found {
				t.Error("Did not find expected property change")
			}
		})
	}
}

// TestDiffSchemaItemsTypeChange tests Items field type changes
func TestDiffSchemaItemsTypeChange(t *testing.T) {
	tests := []struct {
		name          string
		sourceItems   any
		targetItems   any
		expectChanges bool
	}{
		{
			name:          "Items: Schema to Schema (different)",
			sourceItems:   &parser.Schema{Type: "string"},
			targetItems:   &parser.Schema{Type: "number"},
			expectChanges: true,
		},
		{
			name:          "Items: bool to bool (same)",
			sourceItems:   true,
			targetItems:   true,
			expectChanges: false,
		},
		{
			name:          "Items: bool true to false",
			sourceItems:   true,
			targetItems:   false,
			expectChanges: true,
		},
		{
			name:          "Items: Schema to bool",
			sourceItems:   &parser.Schema{Type: "string"},
			targetItems:   false,
			expectChanges: true,
		},
		{
			name:          "Items: bool to Schema",
			sourceItems:   true,
			targetItems:   &parser.Schema{Type: "string"},
			expectChanges: true,
		},
		{
			name:          "Items: nil to Schema",
			sourceItems:   nil,
			targetItems:   &parser.Schema{Type: "string"},
			expectChanges: true,
		},
		{
			name:          "Items: Schema to nil",
			sourceItems:   &parser.Schema{Type: "string"},
			targetItems:   nil,
			expectChanges: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &parser.Schema{
				Type:  "array",
				Items: tt.sourceItems,
			}
			target := &parser.Schema{
				Type:  "array",
				Items: tt.targetItems,
			}

			d := New()
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaRecursiveBreaking(source, target, "test.schema", visited, result)

			hasChanges := len(result.Changes) > 0
			if hasChanges != tt.expectChanges {
				t.Errorf("Expected changes=%v, got %d changes", tt.expectChanges, len(result.Changes))
			}
		})
	}
}

// TestDiffSchemaAdditionalPropertiesBreaking tests additionalProperties field with severity
func TestDiffSchemaAdditionalPropertiesBreaking(t *testing.T) {
	tests := []struct {
		name             string
		sourceAdditional any
		targetAdditional any
		expectChanges    bool
		expectedSeverity Severity
	}{
		{
			name:             "additionalProperties: true to false (restricting)",
			sourceAdditional: true,
			targetAdditional: false,
			expectChanges:    true,
			expectedSeverity: SeverityError,
		},
		{
			name:             "additionalProperties: false to true (relaxing)",
			sourceAdditional: false,
			targetAdditional: true,
			expectChanges:    true,
			expectedSeverity: SeverityInfo,
		},
		{
			name:             "additionalProperties: true to true (same)",
			sourceAdditional: true,
			targetAdditional: true,
			expectChanges:    false,
		},
		{
			name:             "additionalProperties: Schema to Schema (different)",
			sourceAdditional: &parser.Schema{Type: "string"},
			targetAdditional: &parser.Schema{Type: "number"},
			expectChanges:    true,
		},
		{
			name:             "additionalProperties: nil to false",
			sourceAdditional: nil,
			targetAdditional: false,
			expectChanges:    true,
			expectedSeverity: SeverityError,
		},
		{
			name:             "additionalProperties: false to nil",
			sourceAdditional: false,
			targetAdditional: nil,
			expectChanges:    true,
			expectedSeverity: SeverityInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &parser.Schema{
				Type:                 "object",
				AdditionalProperties: tt.sourceAdditional,
			}
			target := &parser.Schema{
				Type:                 "object",
				AdditionalProperties: tt.targetAdditional,
			}

			d := New()
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaRecursiveBreaking(source, target, "test.schema", visited, result)

			hasChanges := len(result.Changes) > 0
			if hasChanges != tt.expectChanges {
				t.Errorf("Expected changes=%v, got %d changes", tt.expectChanges, len(result.Changes))
			}

			if tt.expectChanges && tt.expectedSeverity != Severity(0) {
				if len(result.Changes) == 0 {
					t.Fatal("Expected changes but got none")
				}
				if result.Changes[0].Severity != tt.expectedSeverity {
					t.Errorf("Expected severity %v, got %v", tt.expectedSeverity, result.Changes[0].Severity)
				}
			}
		})
	}
}

// TestDiffSchemaNestedProperties tests deeply nested property changes
func TestDiffSchemaNestedProperties(t *testing.T) {
	source := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"user": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"address": {
						Type: "object",
						Properties: map[string]*parser.Schema{
							"street": {Type: "string"},
							"city":   {Type: "string"},
						},
					},
				},
			},
		},
	}

	target := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"user": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"address": {
						Type: "object",
						Properties: map[string]*parser.Schema{
							"street": {Type: "string"},
							"city":   {Type: "string"},
							"zip":    {Type: "string"}, // Added nested property
						},
					},
				},
			},
		},
	}

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	d.diffSchemaRecursiveBreaking(source, target, "test.schema", visited, result)

	if len(result.Changes) == 0 {
		t.Fatal("Expected changes for nested property addition")
	}

	// Verify the path includes the nested structure
	foundNestedChange := false
	for _, change := range result.Changes {
		if change.Path == "test.schema.properties.user.properties.address.properties.zip" {
			foundNestedChange = true
			if change.Type != ChangeTypeAdded {
				t.Errorf("Expected added change, got %v", change.Type)
			}
		}
	}

	if !foundNestedChange {
		t.Error("Did not find expected nested property change")
	}
}

// TestDiffSchemaComplexCircular tests complex circular reference scenarios
func TestDiffSchemaComplexCircular(t *testing.T) {
	// Create mutually referencing schemas
	schemaA := &parser.Schema{
		Type:       "object",
		Properties: make(map[string]*parser.Schema),
	}
	schemaB := &parser.Schema{
		Type:       "object",
		Properties: make(map[string]*parser.Schema),
	}

	// A references B, B references A
	schemaA.Properties["b"] = schemaB
	schemaB.Properties["a"] = schemaA

	// Create identical target structure
	targetA := &parser.Schema{
		Type:       "object",
		Properties: make(map[string]*parser.Schema),
	}
	targetB := &parser.Schema{
		Type:       "object",
		Properties: make(map[string]*parser.Schema),
	}
	targetA.Properties["b"] = targetB
	targetB.Properties["a"] = targetA

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	// Should not panic or infinite loop
	d.diffSchemaRecursiveBreaking(schemaA, targetA, "test.schema", visited, result)

	// Identical circular structures should have no changes
	if len(result.Changes) > 0 {
		t.Errorf("Expected no changes for identical circular structures, got %d", len(result.Changes))
	}
}

// TestDiffSchemaItemsRecursive tests recursive diffing of Items schemas
func TestDiffSchemaItemsRecursive(t *testing.T) {
	source := &parser.Schema{
		Type: "array",
		Items: &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id":   {Type: "string"},
				"name": {Type: "string"},
			},
		},
	}

	target := &parser.Schema{
		Type: "array",
		Items: &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id":    {Type: "string"},
				"name":  {Type: "string"},
				"email": {Type: "string"}, // Added property in items
			},
		},
	}

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	d.diffSchemaRecursiveBreaking(source, target, "test.schema", visited, result)

	if len(result.Changes) == 0 {
		t.Fatal("Expected changes for items property addition")
	}

	// Verify the path includes items
	foundItemsChange := false
	for _, change := range result.Changes {
		if change.Path == "test.schema.items.properties.email" {
			foundItemsChange = true
		}
	}

	if !foundItemsChange {
		t.Error("Did not find expected items property change")
	}
}

// TestDiffSchemaAdditionalPropertiesRecursive tests recursive diffing of AdditionalProperties schemas
func TestDiffSchemaAdditionalPropertiesRecursive(t *testing.T) {
	source := &parser.Schema{
		Type: "object",
		AdditionalProperties: &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"value": {Type: "string"},
			},
		},
	}

	target := &parser.Schema{
		Type: "object",
		AdditionalProperties: &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"value": {Type: "number"}, // Changed type
			},
		},
	}

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	d.diffSchemaRecursiveBreaking(source, target, "test.schema", visited, result)

	if len(result.Changes) == 0 {
		t.Fatal("Expected changes for additionalProperties schema modification")
	}

	// Verify the path includes additionalProperties
	foundChange := false
	for _, change := range result.Changes {
		// Look for any change within additionalProperties
		if len(change.Path) > len("test.schema.additionalProperties") &&
			change.Path[:len("test.schema.additionalProperties")] == "test.schema.additionalProperties" {
			foundChange = true
			break
		}
	}

	if !foundChange {
		t.Errorf("Did not find expected additionalProperties change. Found %d changes:", len(result.Changes))
		for _, ch := range result.Changes {
			t.Logf("  - %s: %s", ch.Path, ch.Message)
		}
	}
}

// TestDiffSchemaUnknownTypesIdentical tests that identical unknown types are skipped
func TestDiffSchemaUnknownTypesIdentical(t *testing.T) {
	// Create schemas with unknown types (e.g., unresolved $ref maps)
	unknownType := map[string]interface{}{"$ref": "#/components/schemas/Pet"}

	source := &parser.Schema{
		Type:  "array",
		Items: unknownType,
	}

	target := &parser.Schema{
		Type:  "array",
		Items: unknownType,
	}

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	d.diffSchemaRecursiveBreaking(source, target, "test.schema", visited, result)

	// Identical unknown types should be skipped, no changes reported
	if len(result.Changes) > 0 {
		t.Errorf("Expected no changes for identical unknown types, got %d", len(result.Changes))
	}
}

// TestDiffSchemaUnknownTypesDifferent tests that different unknown types are reported
func TestDiffSchemaUnknownTypesDifferent(t *testing.T) {
	source := &parser.Schema{
		Type:  "array",
		Items: map[string]interface{}{"$ref": "#/components/schemas/Pet"},
	}

	target := &parser.Schema{
		Type:  "array",
		Items: &parser.Schema{Type: "string"}, // Changed to actual schema
	}

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	d.diffSchemaRecursiveBreaking(source, target, "test.schema", visited, result)

	// Different types should be reported
	if len(result.Changes) == 0 {
		t.Fatal("Expected changes for different types")
	}
}

// TestDiffSchemaSimpleMode tests that simple mode works without severity
func TestDiffSchemaSimpleMode(t *testing.T) {
	source := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	}

	target := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name":  {Type: "string"},
			"email": {Type: "string"},
		},
	}

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	d.diffSchemaRecursive(source, target, "test.schema", visited, result)

	if len(result.Changes) == 0 {
		t.Fatal("Expected changes in simple mode")
	}

	// Simple mode should not have severity
	for _, change := range result.Changes {
		if change.Severity != Severity(0) {
			t.Errorf("Simple mode should not have severity, got %v", change.Severity)
		}
	}
}
