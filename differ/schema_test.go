package differ

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/testutil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	d.diffSchemaRecursiveUnified(schema1, schema2, "test.schema", visited, result)

	assert.Empty(t, result.Changes, "Expected no changes for identical circular structures")
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
	d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

	// We expect changes (property added/removed), but no infinite loop
	assert.NotEmpty(t, result.Changes, "Expected changes for different structures")
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
			d.Mode = ModeBreaking // Need breaking mode for severity checks
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

			require.NotEmpty(t, result.Changes, "Expected changes but got none")

			// Find the property addition/removal change
			found := false
			for _, change := range result.Changes {
				if tt.removedProp != "" && change.Type == ChangeTypeRemoved && change.Path == "test.schema.properties."+tt.removedProp {
					assert.Equal(t, tt.expectedRemoved, change.Severity, "Expected removed property severity %v, got %v", tt.expectedRemoved, change.Severity)
					found = true
					break
				}
				if tt.addedProp != "" && change.Type == ChangeTypeAdded && change.Path == "test.schema.properties."+tt.addedProp {
					assert.Equal(t, tt.expectedAdded, change.Severity, "Expected added property severity %v, got %v", tt.expectedAdded, change.Severity)
					found = true
					break
				}
			}
			assert.True(t, found, "Did not find expected property change")
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

			d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

			hasChanges := len(result.Changes) > 0
			assert.Equal(t, tt.expectChanges, hasChanges, "Expected changes=%v, got %d changes", tt.expectChanges, len(result.Changes))
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
			d.Mode = ModeBreaking // Need breaking mode for severity checks
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

			hasChanges := len(result.Changes) > 0
			assert.Equal(t, tt.expectChanges, hasChanges, "Expected changes=%v, got %d changes", tt.expectChanges, len(result.Changes))

			if tt.expectChanges && tt.expectedSeverity != Severity(0) {
				require.NotEmpty(t, result.Changes, "Expected changes but got none")
				assert.Equal(t, tt.expectedSeverity, result.Changes[0].Severity, "Expected severity %v, got %v", tt.expectedSeverity, result.Changes[0].Severity)
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

	d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

	require.NotEmpty(t, result.Changes, "Expected changes for nested property addition")

	// Verify the path includes the nested structure
	foundNestedChange := false
	for _, change := range result.Changes {
		if change.Path == "test.schema.properties.user.properties.address.properties.zip" {
			foundNestedChange = true
			assert.Equal(t, ChangeTypeAdded, change.Type, "Expected added change, got %v", change.Type)
		}
	}

	assert.True(t, foundNestedChange, "Did not find expected nested property change")
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
	d.diffSchemaRecursiveUnified(schemaA, targetA, "test.schema", visited, result)

	// Identical circular structures should have no changes
	assert.Empty(t, result.Changes, "Expected no changes for identical circular structures")
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

	d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

	require.NotEmpty(t, result.Changes, "Expected changes for items property addition")

	// Verify the path includes items
	foundItemsChange := false
	for _, change := range result.Changes {
		if change.Path == "test.schema.items.properties.email" {
			foundItemsChange = true
		}
	}

	assert.True(t, foundItemsChange, "Did not find expected items property change")
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

	d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

	require.NotEmpty(t, result.Changes, "Expected changes for additionalProperties schema modification")

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

	assert.True(t, foundChange, "Did not find expected additionalProperties change. Found %d changes", len(result.Changes))
}

// TestDiffSchemaUnknownTypesIdentical tests that identical unknown types are skipped
func TestDiffSchemaUnknownTypesIdentical(t *testing.T) {
	// Create schemas with unknown types (e.g., unresolved $ref maps)
	unknownType := map[string]any{"$ref": "#/components/schemas/Pet"}

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

	d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

	// Identical unknown types should be skipped, no changes reported
	assert.Empty(t, result.Changes, "Expected no changes for identical unknown types")
}

// TestDiffSchemaUnknownTypesDifferent tests that different unknown types are reported
func TestDiffSchemaUnknownTypesDifferent(t *testing.T) {
	source := &parser.Schema{
		Type:  "array",
		Items: map[string]any{"$ref": "#/components/schemas/Pet"},
	}

	target := &parser.Schema{
		Type:  "array",
		Items: &parser.Schema{Type: "string"}, // Changed to actual schema
	}

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

	// Different types should be reported
	require.NotEmpty(t, result.Changes, "Expected changes for different types")
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

	d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

	require.NotEmpty(t, result.Changes, "Expected changes in simple mode")

	// Simple mode should not have severity
	for _, change := range result.Changes {
		assert.Equal(t, Severity(0), change.Severity, "Simple mode should not have severity, got %v", change.Severity)
	}
}

// TestDiffSchemaItemsSimpleMode tests Items diffing in simple mode
func TestDiffSchemaItemsSimpleMode(t *testing.T) {
	tests := []struct {
		name          string
		sourceItems   any
		targetItems   any
		expectChanges bool
	}{
		{
			name:          "Items nil to nil",
			sourceItems:   nil,
			targetItems:   nil,
			expectChanges: false,
		},
		{
			name:          "Items nil to Schema",
			sourceItems:   nil,
			targetItems:   &parser.Schema{Type: "string"},
			expectChanges: true,
		},
		{
			name:          "Items Schema to nil",
			sourceItems:   &parser.Schema{Type: "string"},
			targetItems:   nil,
			expectChanges: true,
		},
		{
			name:          "Items type change",
			sourceItems:   &parser.Schema{Type: "string"},
			targetItems:   &parser.Schema{Type: "number"},
			expectChanges: true,
		},
		{
			name:          "Items bool change",
			sourceItems:   true,
			targetItems:   false,
			expectChanges: true,
		},
		{
			name:          "Items Schema to bool",
			sourceItems:   &parser.Schema{Type: "string"},
			targetItems:   true,
			expectChanges: true,
		},
		{
			name:          "Items bool to Schema",
			sourceItems:   false,
			targetItems:   &parser.Schema{Type: "string"},
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

			d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

			hasChanges := len(result.Changes) > 0
			assert.Equal(t, tt.expectChanges, hasChanges, "Expected changes=%v, got %d changes", tt.expectChanges, len(result.Changes))

			// Verify no severity in simple mode
			for _, change := range result.Changes {
				assert.Equal(t, Severity(0), change.Severity, "Simple mode should not have severity, got %v", change.Severity)
			}
		})
	}
}

// TestDiffSchemaAdditionalPropertiesSimpleMode tests AdditionalProperties in simple mode
func TestDiffSchemaAdditionalPropertiesSimpleMode(t *testing.T) {
	tests := []struct {
		name             string
		sourceAdditional any
		targetAdditional any
		expectChanges    bool
	}{
		{
			name:             "AdditionalProperties nil to nil",
			sourceAdditional: nil,
			targetAdditional: nil,
			expectChanges:    false,
		},
		{
			name:             "AdditionalProperties nil to false",
			sourceAdditional: nil,
			targetAdditional: false,
			expectChanges:    true,
		},
		{
			name:             "AdditionalProperties false to nil",
			sourceAdditional: false,
			targetAdditional: nil,
			expectChanges:    true,
		},
		{
			name:             "AdditionalProperties true to false",
			sourceAdditional: true,
			targetAdditional: false,
			expectChanges:    true,
		},
		{
			name:             "AdditionalProperties Schema to Schema",
			sourceAdditional: &parser.Schema{Type: "string"},
			targetAdditional: &parser.Schema{Type: "number"},
			expectChanges:    true,
		},
		{
			name:             "AdditionalProperties bool to Schema",
			sourceAdditional: false,
			targetAdditional: &parser.Schema{Type: "string"},
			expectChanges:    true,
		},
		{
			name:             "AdditionalProperties Schema to bool",
			sourceAdditional: &parser.Schema{Type: "string"},
			targetAdditional: true,
			expectChanges:    true,
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

			d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

			hasChanges := len(result.Changes) > 0
			assert.Equal(t, tt.expectChanges, hasChanges, "Expected changes=%v, got %d changes", tt.expectChanges, len(result.Changes))

			// Verify no severity in simple mode
			for _, change := range result.Changes {
				assert.Equal(t, Severity(0), change.Severity, "Simple mode should not have severity, got %v", change.Severity)
			}
		})
	}
}

// TestDiffSchemaPropertiesSimpleMode tests property diffing in simple mode
func TestDiffSchemaPropertiesSimpleMode(t *testing.T) {
	source := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name":    {Type: "string"},
			"age":     {Type: "integer"},
			"removed": {Type: "string"},
		},
	}

	target := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name":  {Type: "string"},
			"age":   {Type: "number"}, // Changed type
			"email": {Type: "string"}, // Added
		},
	}

	d := New()
	result := &DiffResult{}
	visited := newSchemaVisited()

	d.diffSchemaRecursiveUnified(source, target, "test.schema", visited, result)

	require.NotEmpty(t, result.Changes, "Expected changes for property modifications")

	// Verify we have changes for removed, added, and modified properties
	foundRemoved := false
	foundAdded := false
	foundModified := false

	for _, change := range result.Changes {
		// Verify no severity in simple mode
		assert.Equal(t, Severity(0), change.Severity, "Simple mode should not have severity, got %v for change at %s", change.Severity, change.Path)

		if change.Path == "test.schema.properties.removed" && change.Type == ChangeTypeRemoved {
			foundRemoved = true
		}
		if change.Path == "test.schema.properties.email" && change.Type == ChangeTypeAdded {
			foundAdded = true
		}
		if change.Path == "test.schema.properties.age.type" && change.Type == ChangeTypeModified {
			foundModified = true
		}
	}

	assert.True(t, foundRemoved, "Expected to find removed property change")
	assert.True(t, foundAdded, "Expected to find added property change")
	assert.True(t, foundModified, "Expected to find modified property type change")
}

// TestDiffSchemaAllOf tests allOf composition diffing
func TestDiffSchemaAllOf(t *testing.T) {
	tests := []struct {
		name           string
		source         []*parser.Schema
		target         []*parser.Schema
		expectedCount  int
		mode           string
		checkSeverity  bool
		expectedSevere int // Count of Error or Critical severity changes
	}{
		{
			name: "AllOf schemas identical",
			source: []*parser.Schema{
				{Type: "string"},
				{Type: "object"},
			},
			target: []*parser.Schema{
				{Type: "string"},
				{Type: "object"},
			},
			expectedCount:  0,
			mode:           "breaking",
			checkSeverity:  false,
			expectedSevere: 0,
		},
		{
			name: "AllOf schema added (breaking mode - Error severity)",
			source: []*parser.Schema{
				{Type: "string"},
			},
			target: []*parser.Schema{
				{Type: "string"},
				{Type: "object"},
			},
			expectedCount:  1,
			mode:           "breaking",
			checkSeverity:  true,
			expectedSevere: 1, // Adding allOf is Error in breaking mode
		},
		{
			name: "AllOf schema removed (breaking mode - Info severity)",
			source: []*parser.Schema{
				{Type: "string"},
				{Type: "object"},
			},
			target: []*parser.Schema{
				{Type: "string"},
			},
			expectedCount:  1,
			mode:           "breaking",
			checkSeverity:  true,
			expectedSevere: 0, // Removing allOf is Info in breaking mode
		},
		{
			name: "AllOf schema modified",
			source: []*parser.Schema{
				{Type: "string", MinLength: testutil.Ptr(5)},
			},
			target: []*parser.Schema{
				{Type: "string", MinLength: testutil.Ptr(10)},
			},
			expectedCount:  1,
			mode:           "breaking",
			checkSeverity:  false,
			expectedSevere: 0,
		},
		{
			name: "AllOf schema added (simple mode - no severity)",
			source: []*parser.Schema{
				{Type: "string"},
			},
			target: []*parser.Schema{
				{Type: "string"},
				{Type: "object"},
			},
			expectedCount:  1,
			mode:           "simple",
			checkSeverity:  false,
			expectedSevere: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &parser.Schema{AllOf: tt.source}
			target := &parser.Schema{AllOf: tt.target}

			sourceDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": source,
					},
				},
			}
			targetDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": target,
					},
				},
			}

			differ := New()
			if tt.mode == "breaking" {
				differ.Mode = ModeBreaking
			}

			result, err := differ.DiffParsed(
				parser.ParseResult{Document: sourceDoc, OASVersion: parser.OASVersion310},
				parser.ParseResult{Document: targetDoc, OASVersion: parser.OASVersion310},
			)
			require.NoError(t, err)

			assert.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))

			if tt.checkSeverity {
				severeCount := 0
				for _, c := range result.Changes {
					if c.Severity == SeverityError || c.Severity == SeverityCritical {
						severeCount++
					}
				}
				assert.Equal(t, tt.expectedSevere, severeCount, "Expected %d severe changes, got %d", tt.expectedSevere, severeCount)
			}
		})
	}
}

// TestDiffSchemaAnyOf tests anyOf composition diffing
func TestDiffSchemaAnyOf(t *testing.T) {
	tests := []struct {
		name          string
		source        []*parser.Schema
		target        []*parser.Schema
		expectedCount int
		mode          string
	}{
		{
			name: "AnyOf schemas identical",
			source: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			target: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			expectedCount: 0,
			mode:          "breaking",
		},
		{
			name: "AnyOf schema added",
			source: []*parser.Schema{
				{Type: "string"},
			},
			target: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			expectedCount: 1,
			mode:          "breaking",
		},
		{
			name: "AnyOf schema removed",
			source: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			target: []*parser.Schema{
				{Type: "string"},
			},
			expectedCount: 1,
			mode:          "breaking",
		},
		{
			name: "AnyOf schema modified",
			source: []*parser.Schema{
				{Type: "string", MinLength: testutil.Ptr(5)},
			},
			target: []*parser.Schema{
				{Type: "string", MinLength: testutil.Ptr(10)},
			},
			expectedCount: 1,
			mode:          "breaking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &parser.Schema{AnyOf: tt.source}
			target := &parser.Schema{AnyOf: tt.target}

			sourceDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": source,
					},
				},
			}
			targetDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": target,
					},
				},
			}

			differ := New()
			if tt.mode == "breaking" {
				differ.Mode = ModeBreaking
			}

			result, err := differ.DiffParsed(
				parser.ParseResult{Document: sourceDoc, OASVersion: parser.OASVersion310},
				parser.ParseResult{Document: targetDoc, OASVersion: parser.OASVersion310},
			)
			require.NoError(t, err)

			assert.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
		})
	}
}

// TestDiffSchemaOneOf tests oneOf composition diffing
func TestDiffSchemaOneOf(t *testing.T) {
	tests := []struct {
		name          string
		source        []*parser.Schema
		target        []*parser.Schema
		expectedCount int
		mode          string
	}{
		{
			name: "OneOf schemas identical",
			source: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			target: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			expectedCount: 0,
			mode:          "breaking",
		},
		{
			name: "OneOf schema added",
			source: []*parser.Schema{
				{Type: "string"},
			},
			target: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			expectedCount: 1,
			mode:          "breaking",
		},
		{
			name: "OneOf schema removed",
			source: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			target: []*parser.Schema{
				{Type: "string"},
			},
			expectedCount: 1,
			mode:          "breaking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &parser.Schema{OneOf: tt.source}
			target := &parser.Schema{OneOf: tt.target}

			sourceDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": source,
					},
				},
			}
			targetDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": target,
					},
				},
			}

			differ := New()
			if tt.mode == "breaking" {
				differ.Mode = ModeBreaking
			}

			result, err := differ.DiffParsed(
				parser.ParseResult{Document: sourceDoc, OASVersion: parser.OASVersion310},
				parser.ParseResult{Document: targetDoc, OASVersion: parser.OASVersion310},
			)
			require.NoError(t, err)

			assert.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
		})
	}
}

// TestDiffSchemaNot tests not schema diffing
func TestDiffSchemaNot(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Schema
		target        *parser.Schema
		expectedCount int
		mode          string
	}{
		{
			name:          "Not schemas identical",
			source:        &parser.Schema{Type: "string"},
			target:        &parser.Schema{Type: "string"},
			expectedCount: 0,
			mode:          "breaking",
		},
		{
			name:          "Not schema added",
			source:        nil,
			target:        &parser.Schema{Type: "string"},
			expectedCount: 1,
			mode:          "breaking",
		},
		{
			name:          "Not schema removed",
			source:        &parser.Schema{Type: "string"},
			target:        nil,
			expectedCount: 1,
			mode:          "breaking",
		},
		{
			name:          "Not schema modified",
			source:        &parser.Schema{Type: "string", MinLength: testutil.Ptr(5)},
			target:        &parser.Schema{Type: "string", MinLength: testutil.Ptr(10)},
			expectedCount: 1,
			mode:          "breaking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &parser.Schema{Not: tt.source}
			target := &parser.Schema{Not: tt.target}

			sourceDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": source,
					},
				},
			}
			targetDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": target,
					},
				},
			}

			differ := New()
			if tt.mode == "breaking" {
				differ.Mode = ModeBreaking
			}

			result, err := differ.DiffParsed(
				parser.ParseResult{Document: sourceDoc, OASVersion: parser.OASVersion310},
				parser.ParseResult{Document: targetDoc, OASVersion: parser.OASVersion310},
			)
			require.NoError(t, err)

			assert.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
		})
	}
}

// TestDiffSchemaConditional tests conditional schema diffing (if/then/else)
func TestDiffSchemaConditional(t *testing.T) {
	tests := []struct {
		name          string
		sourceIf      *parser.Schema
		sourceThen    *parser.Schema
		sourceElse    *parser.Schema
		targetIf      *parser.Schema
		targetThen    *parser.Schema
		targetElse    *parser.Schema
		expectedCount int
		mode          string
	}{
		{
			name:          "Conditional schemas identical",
			sourceIf:      &parser.Schema{Type: "string"},
			sourceThen:    &parser.Schema{MinLength: testutil.Ptr(5)},
			sourceElse:    &parser.Schema{MaxLength: testutil.Ptr(10)},
			targetIf:      &parser.Schema{Type: "string"},
			targetThen:    &parser.Schema{MinLength: testutil.Ptr(5)},
			targetElse:    &parser.Schema{MaxLength: testutil.Ptr(10)},
			expectedCount: 0,
			mode:          "breaking",
		},
		{
			name:          "If schema added",
			sourceIf:      nil,
			sourceThen:    nil,
			sourceElse:    nil,
			targetIf:      &parser.Schema{Type: "string"},
			targetThen:    nil,
			targetElse:    nil,
			expectedCount: 1,
			mode:          "breaking",
		},
		{
			name:          "Then schema added",
			sourceIf:      &parser.Schema{Type: "string"},
			sourceThen:    nil,
			sourceElse:    nil,
			targetIf:      &parser.Schema{Type: "string"},
			targetThen:    &parser.Schema{MinLength: testutil.Ptr(5)},
			targetElse:    nil,
			expectedCount: 1,
			mode:          "breaking",
		},
		{
			name:          "Else schema added",
			sourceIf:      &parser.Schema{Type: "string"},
			sourceThen:    nil,
			sourceElse:    nil,
			targetIf:      &parser.Schema{Type: "string"},
			targetThen:    nil,
			targetElse:    &parser.Schema{MaxLength: testutil.Ptr(10)},
			expectedCount: 1,
			mode:          "breaking",
		},
		{
			name:          "If schema removed",
			sourceIf:      &parser.Schema{Type: "string"},
			sourceThen:    nil,
			sourceElse:    nil,
			targetIf:      nil,
			targetThen:    nil,
			targetElse:    nil,
			expectedCount: 1,
			mode:          "breaking",
		},
		{
			name:          "If schema modified",
			sourceIf:      &parser.Schema{Type: "string", MinLength: testutil.Ptr(5)},
			sourceThen:    nil,
			sourceElse:    nil,
			targetIf:      &parser.Schema{Type: "string", MinLength: testutil.Ptr(10)},
			targetThen:    nil,
			targetElse:    nil,
			expectedCount: 1,
			mode:          "breaking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &parser.Schema{
				If:   tt.sourceIf,
				Then: tt.sourceThen,
				Else: tt.sourceElse,
			}
			target := &parser.Schema{
				If:   tt.targetIf,
				Then: tt.targetThen,
				Else: tt.targetElse,
			}

			sourceDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": source,
					},
				},
			}
			targetDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": target,
					},
				},
			}

			differ := New()
			if tt.mode == "breaking" {
				differ.Mode = ModeBreaking
			}

			result, err := differ.DiffParsed(
				parser.ParseResult{Document: sourceDoc, OASVersion: parser.OASVersion310},
				parser.ParseResult{Document: targetDoc, OASVersion: parser.OASVersion310},
			)
			require.NoError(t, err)

			assert.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
		})
	}
}

// TestDiffSchemaCompositionCycles tests composition with circular references
func TestDiffSchemaCompositionCycles(t *testing.T) {
	// Create a schema with circular allOf reference
	sourceSchema := &parser.Schema{
		Type: "object",
	}
	sourceSchema.AllOf = []*parser.Schema{sourceSchema} // Self-reference

	targetSchema := &parser.Schema{
		Type: "object",
	}
	targetSchema.AllOf = []*parser.Schema{targetSchema} // Self-reference

	sourceDoc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Node": sourceSchema,
			},
		},
	}
	targetDoc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Node": targetSchema,
			},
		},
	}

	differ := New()
	differ.Mode = ModeBreaking

	result, err := differ.DiffParsed(
		parser.ParseResult{Document: sourceDoc, OASVersion: parser.OASVersion310},
		parser.ParseResult{Document: targetDoc, OASVersion: parser.OASVersion310},
	)
	require.NoError(t, err)

	// Should handle cycles without infinite loop
	// No changes expected since both have same circular structure
	assert.Empty(t, result.Changes, "Expected 0 changes for identical circular schemas")
}

// TestDiffSchemaCompositionSimpleMode tests composition fields in simple mode (no severity)
func TestDiffSchemaCompositionSimpleMode(t *testing.T) {
	tests := []struct {
		name          string
		sourceAllOf   []*parser.Schema
		targetAllOf   []*parser.Schema
		sourceAnyOf   []*parser.Schema
		targetAnyOf   []*parser.Schema
		sourceOneOf   []*parser.Schema
		targetOneOf   []*parser.Schema
		sourceNot     *parser.Schema
		targetNot     *parser.Schema
		expectedCount int
	}{
		{
			name: "AllOf added in simple mode",
			sourceAllOf: []*parser.Schema{
				{Type: "string"},
			},
			targetAllOf: []*parser.Schema{
				{Type: "string"},
				{Type: "object"},
			},
			expectedCount: 1,
		},
		{
			name: "AllOf removed in simple mode",
			sourceAllOf: []*parser.Schema{
				{Type: "string"},
				{Type: "object"},
			},
			targetAllOf: []*parser.Schema{
				{Type: "string"},
			},
			expectedCount: 1,
		},
		{
			name: "AnyOf added in simple mode",
			sourceAnyOf: []*parser.Schema{
				{Type: "string"},
			},
			targetAnyOf: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			expectedCount: 1,
		},
		{
			name: "AnyOf removed in simple mode",
			sourceAnyOf: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			targetAnyOf: []*parser.Schema{
				{Type: "string"},
			},
			expectedCount: 1,
		},
		{
			name: "OneOf added in simple mode",
			sourceOneOf: []*parser.Schema{
				{Type: "string"},
			},
			targetOneOf: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			expectedCount: 1,
		},
		{
			name: "OneOf removed in simple mode",
			sourceOneOf: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			targetOneOf: []*parser.Schema{
				{Type: "string"},
			},
			expectedCount: 1,
		},
		{
			name:          "Not added in simple mode",
			sourceNot:     nil,
			targetNot:     &parser.Schema{Type: "string"},
			expectedCount: 1,
		},
		{
			name:          "Not removed in simple mode",
			sourceNot:     &parser.Schema{Type: "string"},
			targetNot:     nil,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &parser.Schema{
				AllOf: tt.sourceAllOf,
				AnyOf: tt.sourceAnyOf,
				OneOf: tt.sourceOneOf,
				Not:   tt.sourceNot,
			}
			target := &parser.Schema{
				AllOf: tt.targetAllOf,
				AnyOf: tt.targetAnyOf,
				OneOf: tt.targetOneOf,
				Not:   tt.targetNot,
			}

			sourceDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": source,
					},
				},
			}
			targetDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": target,
					},
				},
			}

			differ := New()
			// Simple mode is the default, but be explicit
			differ.Mode = ModeSimple

			result, err := differ.DiffParsed(
				parser.ParseResult{Document: sourceDoc, OASVersion: parser.OASVersion310},
				parser.ParseResult{Document: targetDoc, OASVersion: parser.OASVersion310},
			)
			require.NoError(t, err)

			assert.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
		})
	}
}

// TestDiffSchemaConditionalSimpleMode tests conditional schemas in simple mode
func TestDiffSchemaConditionalSimpleMode(t *testing.T) {
	tests := []struct {
		name          string
		sourceIf      *parser.Schema
		sourceThen    *parser.Schema
		sourceElse    *parser.Schema
		targetIf      *parser.Schema
		targetThen    *parser.Schema
		targetElse    *parser.Schema
		expectedCount int
	}{
		{
			name:          "If added in simple mode",
			sourceIf:      nil,
			targetIf:      &parser.Schema{Type: "string"},
			expectedCount: 1,
		},
		{
			name:          "Then added in simple mode",
			sourceIf:      &parser.Schema{Type: "string"},
			targetIf:      &parser.Schema{Type: "string"},
			sourceThen:    nil,
			targetThen:    &parser.Schema{MinLength: testutil.Ptr(5)},
			expectedCount: 1,
		},
		{
			name:          "Else added in simple mode",
			sourceIf:      &parser.Schema{Type: "string"},
			targetIf:      &parser.Schema{Type: "string"},
			sourceElse:    nil,
			targetElse:    &parser.Schema{MaxLength: testutil.Ptr(10)},
			expectedCount: 1,
		},
		{
			name:          "If removed in simple mode",
			sourceIf:      &parser.Schema{Type: "string"},
			targetIf:      nil,
			expectedCount: 1,
		},
		{
			name:          "Then removed in simple mode",
			sourceIf:      &parser.Schema{Type: "string"},
			targetIf:      &parser.Schema{Type: "string"},
			sourceThen:    &parser.Schema{MinLength: testutil.Ptr(5)},
			targetThen:    nil,
			expectedCount: 1,
		},
		{
			name:          "Else removed in simple mode",
			sourceIf:      &parser.Schema{Type: "string"},
			targetIf:      &parser.Schema{Type: "string"},
			sourceElse:    &parser.Schema{MaxLength: testutil.Ptr(10)},
			targetElse:    nil,
			expectedCount: 1,
		},
		{
			name:          "Multiple conditional changes",
			sourceIf:      &parser.Schema{Type: "string"},
			sourceThen:    &parser.Schema{MinLength: testutil.Ptr(5)},
			targetIf:      &parser.Schema{Type: "integer"},
			targetElse:    &parser.Schema{MaxLength: testutil.Ptr(10)},
			expectedCount: 3, // if modified, then removed, else added
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &parser.Schema{
				If:   tt.sourceIf,
				Then: tt.sourceThen,
				Else: tt.sourceElse,
			}
			target := &parser.Schema{
				If:   tt.targetIf,
				Then: tt.targetThen,
				Else: tt.targetElse,
			}

			sourceDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": source,
					},
				},
			}
			targetDoc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": target,
					},
				},
			}

			differ := New()
			differ.Mode = ModeSimple

			result, err := differ.DiffParsed(
				parser.ParseResult{Document: sourceDoc, OASVersion: parser.OASVersion310},
				parser.ParseResult{Document: targetDoc, OASVersion: parser.OASVersion310},
			)
			require.NoError(t, err)

			assert.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
		})
	}
}

// TestDiffSchemaAllOfModified tests recursive comparison within allOf schemas
func TestDiffSchemaAllOfModified(t *testing.T) {
	source := &parser.Schema{
		AllOf: []*parser.Schema{
			{Type: "string", MinLength: testutil.Ptr(5)},
			{Type: "object", Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
			}},
		},
	}
	target := &parser.Schema{
		AllOf: []*parser.Schema{
			{Type: "string", MinLength: testutil.Ptr(10)}, // Changed constraint
			{Type: "object", Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
				"age":  {Type: "integer"}, // Added property
			}},
		},
	}

	sourceDoc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"TestSchema": source,
			},
		},
	}
	targetDoc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"TestSchema": target,
			},
		},
	}

	differ := New()
	differ.Mode = ModeSimple

	result, err := differ.DiffParsed(
		parser.ParseResult{Document: sourceDoc, OASVersion: parser.OASVersion310},
		parser.ParseResult{Document: targetDoc, OASVersion: parser.OASVersion310},
	)
	require.NoError(t, err)

	// Should detect minLength change and property addition
	assert.GreaterOrEqual(t, len(result.Changes), 2, "Expected at least 2 changes (minLength and property), got %d", len(result.Changes))

	// Check that changes are in allOf context
	foundAllOfChange := false
	for _, c := range result.Changes {
		if strings.Contains(c.Path, "allOf[") {
			foundAllOfChange = true
			break
		}
	}
	assert.True(t, foundAllOfChange, "Expected to find changes within allOf schemas")
}
