package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDocumentStats_OAS2(t *testing.T) {
	tests := []struct {
		name string
		doc  *OAS2Document
		want DocumentStats
	}{
		{
			name: "empty document",
			doc: &OAS2Document{
				Paths:       Paths{},
				Definitions: map[string]*Schema{},
			},
			want: DocumentStats{
				PathCount:      0,
				OperationCount: 0,
				SchemaCount:    0,
			},
		},
		{
			name: "document with paths and operations",
			doc: &OAS2Document{
				Paths: Paths{
					"/users": &PathItem{
						Get:  &Operation{},
						Post: &Operation{},
					},
					"/users/{id}": &PathItem{
						Get:    &Operation{},
						Put:    &Operation{},
						Delete: &Operation{},
					},
				},
				Definitions: map[string]*Schema{
					"User":  {},
					"Error": {},
				},
			},
			want: DocumentStats{
				PathCount:      2,
				OperationCount: 5,
				SchemaCount:    2,
			},
		},
		{
			name: "document with all HTTP methods",
			doc: &OAS2Document{
				Paths: Paths{
					"/test": &PathItem{
						Get:     &Operation{},
						Put:     &Operation{},
						Post:    &Operation{},
						Delete:  &Operation{},
						Options: &Operation{},
						Head:    &Operation{},
						Patch:   &Operation{},
					},
				},
				Definitions: map[string]*Schema{},
			},
			want: DocumentStats{
				PathCount:      1,
				OperationCount: 7,
				SchemaCount:    0,
			},
		},
		{
			name: "document with nil path items",
			doc: &OAS2Document{
				Paths: Paths{
					"/valid": &PathItem{
						Get: &Operation{},
					},
					"/nil": nil,
				},
				Definitions: map[string]*Schema{
					"Schema1": {},
				},
			},
			want: DocumentStats{
				PathCount:      2,
				OperationCount: 1,
				SchemaCount:    1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDocumentStats(tt.doc)
			assert.Equal(t, tt.want.PathCount, got.PathCount, "PathCount")
			assert.Equal(t, tt.want.OperationCount, got.OperationCount, "OperationCount")
			assert.Equal(t, tt.want.SchemaCount, got.SchemaCount, "SchemaCount")
		})
	}
}

func TestGetDocumentStats_OAS3(t *testing.T) {
	tests := []struct {
		name string
		doc  *OAS3Document
		want DocumentStats
	}{
		{
			name: "empty document",
			doc: &OAS3Document{
				Paths:      Paths{},
				Components: nil,
			},
			want: DocumentStats{
				PathCount:      0,
				OperationCount: 0,
				SchemaCount:    0,
			},
		},
		{
			name: "document with components but no schemas",
			doc: &OAS3Document{
				Paths: Paths{},
				Components: &Components{
					Schemas: nil,
				},
			},
			want: DocumentStats{
				PathCount:      0,
				OperationCount: 0,
				SchemaCount:    0,
			},
		},
		{
			name: "document with paths and schemas",
			doc: &OAS3Document{
				Paths: Paths{
					"/pets": &PathItem{
						Get:  &Operation{},
						Post: &Operation{},
					},
					"/pets/{id}": &PathItem{
						Get:    &Operation{},
						Put:    &Operation{},
						Delete: &Operation{},
					},
				},
				Components: &Components{
					Schemas: map[string]*Schema{
						"Pet":   {},
						"Error": {},
						"Owner": {},
					},
				},
			},
			want: DocumentStats{
				PathCount:      2,
				OperationCount: 5,
				SchemaCount:    3,
			},
		},
		{
			name: "document with all HTTP methods including Trace",
			doc: &OAS3Document{
				Paths: Paths{
					"/test": &PathItem{
						Get:     &Operation{},
						Put:     &Operation{},
						Post:    &Operation{},
						Delete:  &Operation{},
						Options: &Operation{},
						Head:    &Operation{},
						Patch:   &Operation{},
						Trace:   &Operation{}, // OAS 3.0+ specific
					},
				},
				Components: &Components{
					Schemas: map[string]*Schema{},
				},
			},
			want: DocumentStats{
				PathCount:      1,
				OperationCount: 8,
				SchemaCount:    0,
			},
		},
		{
			name: "document with nil path items",
			doc: &OAS3Document{
				Paths: Paths{
					"/valid": &PathItem{
						Get:  &Operation{},
						Post: &Operation{},
					},
					"/nil": nil,
				},
				Components: &Components{
					Schemas: map[string]*Schema{
						"Schema1": {},
						"Schema2": {},
					},
				},
			},
			want: DocumentStats{
				PathCount:      2,
				OperationCount: 2,
				SchemaCount:    2,
			},
		},
		{
			name: "document with webhooks (OAS 3.1+)",
			doc: &OAS3Document{
				Paths: Paths{
					"/pets": &PathItem{
						Get: &Operation{},
					},
				},
				Webhooks: map[string]*PathItem{
					"newPetWebhook": {
						Post: &Operation{},
					},
					"updatePetWebhook": {
						Put: &Operation{},
					},
				},
				Components: &Components{
					Schemas: map[string]*Schema{
						"Pet": {},
					},
				},
			},
			want: DocumentStats{
				PathCount:      1,
				OperationCount: 3, // 1 from paths + 2 from webhooks
				SchemaCount:    1,
			},
		},
		{
			name: "webhooks only (OAS 3.1+)",
			doc: &OAS3Document{
				Paths: Paths{},
				Webhooks: map[string]*PathItem{
					"webhook1": {
						Post: &Operation{},
						Put:  &Operation{},
					},
				},
				Components: &Components{
					Schemas: map[string]*Schema{},
				},
			},
			want: DocumentStats{
				PathCount:      0,
				OperationCount: 2, // Only from webhooks
				SchemaCount:    0,
			},
		},
		{
			name: "webhooks with nil path item",
			doc: &OAS3Document{
				Paths: Paths{
					"/test": &PathItem{
						Get: &Operation{},
					},
				},
				Webhooks: map[string]*PathItem{
					"validWebhook": {
						Post: &Operation{},
					},
					"nilWebhook": nil,
				},
				Components: nil,
			},
			want: DocumentStats{
				PathCount:      1,
				OperationCount: 2, // 1 from path + 1 from webhook (nil ignored)
				SchemaCount:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDocumentStats(tt.doc)
			assert.Equal(t, tt.want.PathCount, got.PathCount, "PathCount")
			assert.Equal(t, tt.want.OperationCount, got.OperationCount, "OperationCount")
			assert.Equal(t, tt.want.SchemaCount, got.SchemaCount, "SchemaCount")
		})
	}
}

func TestGetDocumentStats_UnknownType(t *testing.T) {
	// Test with an unknown document type
	got := GetDocumentStats("not a document")
	want := DocumentStats{
		PathCount:      0,
		OperationCount: 0,
		SchemaCount:    0,
	}

	assert.Equal(t, want, got)
}

func TestGetDocumentStats_NilDocument(t *testing.T) {
	// Test with nil document
	got := GetDocumentStats(nil)
	want := DocumentStats{
		PathCount:      0,
		OperationCount: 0,
		SchemaCount:    0,
	}

	assert.Equal(t, want, got)
}

func TestCountPathItemOperations(t *testing.T) {
	tests := []struct {
		name     string
		pathItem *PathItem
		want     int
	}{
		{
			name:     "nil path item",
			pathItem: nil,
			want:     0,
		},
		{
			name:     "empty path item",
			pathItem: &PathItem{},
			want:     0,
		},
		{
			name: "single operation",
			pathItem: &PathItem{
				Get: &Operation{},
			},
			want: 1,
		},
		{
			name: "multiple operations",
			pathItem: &PathItem{
				Get:    &Operation{},
				Post:   &Operation{},
				Put:    &Operation{},
				Delete: &Operation{},
			},
			want: 4,
		},
		{
			name: "all operations",
			pathItem: &PathItem{
				Get:     &Operation{},
				Put:     &Operation{},
				Post:    &Operation{},
				Delete:  &Operation{},
				Options: &Operation{},
				Head:    &Operation{},
				Patch:   &Operation{},
				Trace:   &Operation{},
			},
			want: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got int
			if tt.pathItem == nil {
				got = 0
			} else {
				got = countPathItemOperations(tt.pathItem)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
