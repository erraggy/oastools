package differ

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

// TestDiffSchemaUnevaluatedProperties tests the unevaluatedProperties comparison
func TestDiffSchemaUnevaluatedProperties(t *testing.T) {
	tests := []struct {
		name           string
		source         any
		target         any
		expectChanges  bool
		expectedChange ChangeType
	}{
		{
			name:          "both nil",
			source:        nil,
			target:        nil,
			expectChanges: false,
		},
		{
			name:           "added bool false",
			source:         nil,
			target:         false,
			expectChanges:  true,
			expectedChange: ChangeTypeAdded,
		},
		{
			name:           "added bool true",
			source:         nil,
			target:         true,
			expectChanges:  true,
			expectedChange: ChangeTypeAdded,
		},
		{
			name:           "added schema",
			source:         nil,
			target:         &parser.Schema{Type: "object"},
			expectChanges:  true,
			expectedChange: ChangeTypeAdded,
		},
		{
			name:           "removed",
			source:         false,
			target:         nil,
			expectChanges:  true,
			expectedChange: ChangeTypeRemoved,
		},
		{
			name:           "bool changed",
			source:         true,
			target:         false,
			expectChanges:  true,
			expectedChange: ChangeTypeModified,
		},
		{
			name:          "bool same",
			source:        true,
			target:        true,
			expectChanges: false,
		},
		{
			name:           "type changed bool to schema",
			source:         true,
			target:         &parser.Schema{Type: "string"},
			expectChanges:  true,
			expectedChange: ChangeTypeModified,
		},
		{
			name:           "unknown source type",
			source:         "invalid",
			target:         true,
			expectChanges:  true,
			expectedChange: ChangeTypeModified,
		},
		{
			name:           "unknown target type",
			source:         true,
			target:         42,
			expectChanges:  true,
			expectedChange: ChangeTypeModified,
		},
		{
			name:          "both unknown",
			source:        "invalid",
			target:        42,
			expectChanges: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaUnevaluatedPropertiesUnified(tt.source, tt.target, "schema", visited, result)

			if tt.expectChanges {
				assert.NotEmpty(t, result.Changes, "Expected changes but got none")
			} else {
				assert.Empty(t, result.Changes, "Expected no changes but got %d", len(result.Changes))
			}
			if tt.expectChanges && len(result.Changes) > 0 {
				assert.Equal(t, tt.expectedChange, result.Changes[0].Type)
			}
		})
	}
}

// TestDiffSchemaUnevaluatedItems tests the unevaluatedItems comparison
func TestDiffSchemaUnevaluatedItems(t *testing.T) {
	tests := []struct {
		name           string
		source         any
		target         any
		expectChanges  bool
		expectedChange ChangeType
	}{
		{
			name:          "both nil",
			source:        nil,
			target:        nil,
			expectChanges: false,
		},
		{
			name:           "added",
			source:         nil,
			target:         false,
			expectChanges:  true,
			expectedChange: ChangeTypeAdded,
		},
		{
			name:           "removed",
			source:         true,
			target:         nil,
			expectChanges:  true,
			expectedChange: ChangeTypeRemoved,
		},
		{
			name:           "bool changed",
			source:         true,
			target:         false,
			expectChanges:  true,
			expectedChange: ChangeTypeModified,
		},
		{
			name:          "schema same",
			source:        &parser.Schema{Type: "string"},
			target:        &parser.Schema{Type: "string"},
			expectChanges: false,
		},
		{
			name:           "unknown source",
			source:         []string{"invalid"},
			target:         true,
			expectChanges:  true,
			expectedChange: ChangeTypeModified,
		},
		{
			name:           "unknown target",
			source:         false,
			target:         map[string]any{"foo": "bar"},
			expectChanges:  true,
			expectedChange: ChangeTypeModified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaUnevaluatedItemsUnified(tt.source, tt.target, "schema", visited, result)

			if tt.expectChanges {
				assert.NotEmpty(t, result.Changes, "Expected changes but got none")
			} else {
				assert.Empty(t, result.Changes, "Expected no changes but got %d", len(result.Changes))
			}
			if tt.expectChanges && len(result.Changes) > 0 {
				assert.Equal(t, tt.expectedChange, result.Changes[0].Type)
			}
		})
	}
}

// TestDiffSchemaContentFields tests contentEncoding, contentMediaType, contentSchema
func TestDiffSchemaContentFields(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Schema
		target        *parser.Schema
		expectChanges int
	}{
		{
			name:          "no content fields",
			source:        &parser.Schema{Type: "string"},
			target:        &parser.Schema{Type: "string"},
			expectChanges: 0,
		},
		{
			name:          "contentEncoding added",
			source:        &parser.Schema{Type: "string"},
			target:        &parser.Schema{Type: "string", ContentEncoding: "base64"},
			expectChanges: 1,
		},
		{
			name:          "contentEncoding removed",
			source:        &parser.Schema{Type: "string", ContentEncoding: "base64"},
			target:        &parser.Schema{Type: "string"},
			expectChanges: 1,
		},
		{
			name:          "contentEncoding changed",
			source:        &parser.Schema{Type: "string", ContentEncoding: "base64"},
			target:        &parser.Schema{Type: "string", ContentEncoding: "quoted-printable"},
			expectChanges: 1,
		},
		{
			name:          "contentMediaType added",
			source:        &parser.Schema{Type: "string"},
			target:        &parser.Schema{Type: "string", ContentMediaType: "application/json"},
			expectChanges: 1,
		},
		{
			name:          "contentMediaType removed",
			source:        &parser.Schema{Type: "string", ContentMediaType: "application/json"},
			target:        &parser.Schema{Type: "string"},
			expectChanges: 1,
		},
		{
			name:          "contentSchema added",
			source:        &parser.Schema{Type: "string"},
			target:        &parser.Schema{Type: "string", ContentSchema: &parser.Schema{Type: "object"}},
			expectChanges: 1,
		},
		{
			name:          "contentSchema removed",
			source:        &parser.Schema{Type: "string", ContentSchema: &parser.Schema{Type: "object"}},
			target:        &parser.Schema{Type: "string"},
			expectChanges: 1,
		},
		{
			name: "all content fields changed",
			source: &parser.Schema{
				Type:             "string",
				ContentEncoding:  "base64",
				ContentMediaType: "application/json",
				ContentSchema:    &parser.Schema{Type: "object"},
			},
			target: &parser.Schema{
				Type:             "string",
				ContentEncoding:  "quoted-printable",
				ContentMediaType: "text/plain",
				ContentSchema:    &parser.Schema{Type: "array"},
			},
			expectChanges: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaContentFieldsUnified(tt.source, tt.target, "schema", visited, result)

			assert.Len(t, result.Changes, tt.expectChanges)
		})
	}
}

// TestDiffSchemaPrefixItems tests prefixItems comparison
func TestDiffSchemaPrefixItems(t *testing.T) {
	tests := []struct {
		name          string
		source        []*parser.Schema
		target        []*parser.Schema
		expectChanges int
	}{
		{
			name:          "both nil",
			source:        nil,
			target:        nil,
			expectChanges: 0,
		},
		{
			name:   "added",
			source: nil,
			target: []*parser.Schema{
				{Type: "string"},
			},
			expectChanges: 1,
		},
		{
			name: "removed",
			source: []*parser.Schema{
				{Type: "string"},
			},
			target:        nil,
			expectChanges: 1,
		},
		{
			name: "same length different content",
			source: []*parser.Schema{
				{Type: "string"},
			},
			target: []*parser.Schema{
				{Type: "integer"},
			},
			expectChanges: 1,
		},
		{
			name: "item added",
			source: []*parser.Schema{
				{Type: "string"},
			},
			target: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			expectChanges: 1,
		},
		{
			name: "item removed",
			source: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			target: []*parser.Schema{
				{Type: "string"},
			},
			expectChanges: 1,
		},
		{
			name: "same content",
			source: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			target: []*parser.Schema{
				{Type: "string"},
				{Type: "integer"},
			},
			expectChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaPrefixItemsUnified(tt.source, tt.target, "schema", visited, result)

			assert.Len(t, result.Changes, tt.expectChanges)
		})
	}
}

// TestDiffSchemaContains tests contains comparison
func TestDiffSchemaContains(t *testing.T) {
	tests := []struct {
		name           string
		source         *parser.Schema
		target         *parser.Schema
		expectChanges  bool
		expectedChange ChangeType
	}{
		{
			name:          "both nil",
			source:        nil,
			target:        nil,
			expectChanges: false,
		},
		{
			name:           "added",
			source:         nil,
			target:         &parser.Schema{Type: "string"},
			expectChanges:  true,
			expectedChange: ChangeTypeAdded,
		},
		{
			name:           "removed",
			source:         &parser.Schema{Type: "string"},
			target:         nil,
			expectChanges:  true,
			expectedChange: ChangeTypeRemoved,
		},
		{
			name:          "same",
			source:        &parser.Schema{Type: "string"},
			target:        &parser.Schema{Type: "string"},
			expectChanges: false,
		},
		{
			name:           "different",
			source:         &parser.Schema{Type: "string"},
			target:         &parser.Schema{Type: "integer"},
			expectChanges:  true,
			expectedChange: ChangeTypeModified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaContainsUnified(tt.source, tt.target, "schema", visited, result)

			if tt.expectChanges {
				assert.NotEmpty(t, result.Changes, "Expected changes but got none")
			} else {
				assert.Empty(t, result.Changes, "Expected no changes but got %d", len(result.Changes))
			}
		})
	}
}

// TestDiffSchemaPropertyNames tests propertyNames comparison
func TestDiffSchemaPropertyNames(t *testing.T) {
	tests := []struct {
		name           string
		source         *parser.Schema
		target         *parser.Schema
		expectChanges  bool
		expectedChange ChangeType
	}{
		{
			name:          "both nil",
			source:        nil,
			target:        nil,
			expectChanges: false,
		},
		{
			name:           "added",
			source:         nil,
			target:         &parser.Schema{Pattern: "^[a-z]+$"},
			expectChanges:  true,
			expectedChange: ChangeTypeAdded,
		},
		{
			name:           "removed",
			source:         &parser.Schema{Pattern: "^[a-z]+$"},
			target:         nil,
			expectChanges:  true,
			expectedChange: ChangeTypeRemoved,
		},
		{
			name:          "same",
			source:        &parser.Schema{Pattern: "^[a-z]+$"},
			target:        &parser.Schema{Pattern: "^[a-z]+$"},
			expectChanges: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaPropertyNamesUnified(tt.source, tt.target, "schema", visited, result)

			if tt.expectChanges {
				assert.NotEmpty(t, result.Changes, "Expected changes but got none")
			} else {
				assert.Empty(t, result.Changes, "Expected no changes but got %d", len(result.Changes))
			}
		})
	}
}

// TestDiffSchemaDependentSchemas tests dependentSchemas comparison
func TestDiffSchemaDependentSchemas(t *testing.T) {
	tests := []struct {
		name          string
		source        map[string]*parser.Schema
		target        map[string]*parser.Schema
		expectChanges int
	}{
		{
			name:          "both nil",
			source:        nil,
			target:        nil,
			expectChanges: 0,
		},
		{
			name:   "added",
			source: nil,
			target: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
			expectChanges: 1,
		},
		{
			name: "removed",
			source: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
			target:        nil,
			expectChanges: 1,
		},
		{
			name: "key added",
			source: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
			target: map[string]*parser.Schema{
				"name":  {Type: "object"},
				"email": {Type: "string"},
			},
			expectChanges: 1,
		},
		{
			name: "key removed",
			source: map[string]*parser.Schema{
				"name":  {Type: "object"},
				"email": {Type: "string"},
			},
			target: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
			expectChanges: 1,
		},
		{
			name: "schema changed",
			source: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
			target: map[string]*parser.Schema{
				"name": {Type: "array"},
			},
			expectChanges: 1,
		},
		{
			name: "same",
			source: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
			target: map[string]*parser.Schema{
				"name": {Type: "object"},
			},
			expectChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{}
			visited := newSchemaVisited()

			d.diffSchemaDependentSchemasUnified(tt.source, tt.target, "schema", visited, result)

			assert.Len(t, result.Changes, tt.expectChanges)
		})
	}
}

// TestDiffMediaTypesUnified tests mediaTypes comparison (OAS 3.2+)
func TestDiffMediaTypesUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        map[string]*parser.MediaType
		target        map[string]*parser.MediaType
		expectChanges int
	}{
		{
			name:          "both nil",
			source:        nil,
			target:        nil,
			expectChanges: 0,
		},
		{
			name:   "added",
			source: nil,
			target: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "object"}},
			},
			expectChanges: 1,
		},
		{
			name: "removed",
			source: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "object"}},
			},
			target:        nil,
			expectChanges: 1,
		},
		{
			name: "key added",
			source: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "object"}},
			},
			target: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "object"}},
				"application/xml":  {Schema: &parser.Schema{Type: "object"}},
			},
			expectChanges: 1,
		},
		{
			name: "same",
			source: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "object"}},
			},
			target: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "object"}},
			},
			expectChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{}

			d.diffMediaTypesUnified(tt.source, tt.target, "components.mediaTypes", result)

			assert.Len(t, result.Changes, tt.expectChanges)
		})
	}
}
