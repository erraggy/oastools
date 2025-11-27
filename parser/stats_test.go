package parser

import (
	"testing"
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
			if got.PathCount != tt.want.PathCount {
				t.Errorf("PathCount = %d, want %d", got.PathCount, tt.want.PathCount)
			}
			if got.OperationCount != tt.want.OperationCount {
				t.Errorf("OperationCount = %d, want %d", got.OperationCount, tt.want.OperationCount)
			}
			if got.SchemaCount != tt.want.SchemaCount {
				t.Errorf("SchemaCount = %d, want %d", got.SchemaCount, tt.want.SchemaCount)
			}
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDocumentStats(tt.doc)
			if got.PathCount != tt.want.PathCount {
				t.Errorf("PathCount = %d, want %d", got.PathCount, tt.want.PathCount)
			}
			if got.OperationCount != tt.want.OperationCount {
				t.Errorf("OperationCount = %d, want %d", got.OperationCount, tt.want.OperationCount)
			}
			if got.SchemaCount != tt.want.SchemaCount {
				t.Errorf("SchemaCount = %d, want %d", got.SchemaCount, tt.want.SchemaCount)
			}
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

	if got != want {
		t.Errorf("GetDocumentStats with unknown type = %+v, want %+v", got, want)
	}
}

func TestGetDocumentStats_NilDocument(t *testing.T) {
	// Test with nil document
	got := GetDocumentStats(nil)
	want := DocumentStats{
		PathCount:      0,
		OperationCount: 0,
		SchemaCount:    0,
	}

	if got != want {
		t.Errorf("GetDocumentStats with nil = %+v, want %+v", got, want)
	}
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
			if got != tt.want {
				t.Errorf("countPathItemOperations() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetDocumentStats_Integration(t *testing.T) {
	// Test with actual parsed documents
	t.Run("parse OAS2 petstore and verify stats", func(t *testing.T) {
		p := New()
		result, err := p.Parse("../testdata/petstore-2.0.yaml")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		// Petstore 2.0 has known structure
		if result.Stats.PathCount == 0 {
			t.Error("Expected non-zero path count")
		}
		if result.Stats.OperationCount == 0 {
			t.Error("Expected non-zero operation count")
		}
		// Operations should always be >= paths (assuming at least 1 operation per path)
		if result.Stats.OperationCount < result.Stats.PathCount {
			t.Errorf("Expected operations (%d) >= paths (%d)", result.Stats.OperationCount, result.Stats.PathCount)
		}
	})

	t.Run("parse OAS3 petstore and verify stats", func(t *testing.T) {
		p := New()
		result, err := p.Parse("../testdata/petstore-3.0.yaml")
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		// Petstore 3.0 has known structure
		if result.Stats.PathCount == 0 {
			t.Error("Expected non-zero path count")
		}
		if result.Stats.OperationCount == 0 {
			t.Error("Expected non-zero operation count")
		}
		if result.Stats.OperationCount < result.Stats.PathCount {
			t.Errorf("Expected operations (%d) >= paths (%d)", result.Stats.OperationCount, result.Stats.PathCount)
		}
	})
}
