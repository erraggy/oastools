package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: ptr, intPtr, and boolPtr helper functions are defined in schema_test_helpers.go

func TestParseResultEquals(t *testing.T) {
	// Create test documents
	oas3Doc1 := &OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: OASVersion303,
		Info:       &Info{Title: "Test API", Version: "1.0.0"},
	}
	oas3Doc2 := &OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: OASVersion303,
		Info:       &Info{Title: "Test API", Version: "1.0.0"},
	}
	oas3DocDifferent := &OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: OASVersion303,
		Info:       &Info{Title: "Different API", Version: "1.0.0"},
	}
	oas2Doc1 := &OAS2Document{
		Swagger:    "2.0",
		OASVersion: OASVersion20,
		Info:       &Info{Title: "Test API", Version: "1.0.0"},
	}
	oas2Doc2 := &OAS2Document{
		Swagger:    "2.0",
		OASVersion: OASVersion20,
		Info:       &Info{Title: "Test API", Version: "1.0.0"},
	}

	tests := []struct {
		name  string
		pr    *ParseResult
		other *ParseResult
		want  bool
	}{
		{
			name:  "both nil",
			pr:    nil,
			other: nil,
			want:  true,
		},
		{
			name:  "pr nil, other non-nil",
			pr:    nil,
			other: &ParseResult{Version: "3.0.3", OASVersion: OASVersion303},
			want:  false,
		},
		{
			name:  "pr non-nil, other nil",
			pr:    &ParseResult{Version: "3.0.3", OASVersion: OASVersion303},
			other: nil,
			want:  false,
		},
		{
			name: "same OAS3 documents",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
			},
			other: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc2,
			},
			want: true,
		},
		{
			name: "different OAS3 documents",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
			},
			other: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3DocDifferent,
			},
			want: false,
		},
		{
			name: "same OAS2 documents",
			pr: &ParseResult{
				Version:    "2.0",
				OASVersion: OASVersion20,
				Document:   oas2Doc1,
			},
			other: &ParseResult{
				Version:    "2.0",
				OASVersion: OASVersion20,
				Document:   oas2Doc2,
			},
			want: true,
		},
		{
			name: "different OASVersion",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
			},
			other: &ParseResult{
				Version:    "2.0",
				OASVersion: OASVersion20,
				Document:   oas2Doc1,
			},
			want: false,
		},
		{
			name: "different Version string",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
			},
			other: &ParseResult{
				Version:    "3.0.0",
				OASVersion: OASVersion303,
				Document:   oas3Doc2,
			},
			want: false,
		},
		{
			name: "ignores SourcePath",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
				SourcePath: "/path/to/spec1.yaml",
			},
			other: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc2,
				SourcePath: "/path/to/spec2.yaml",
			},
			want: true,
		},
		{
			name: "ignores SourceFormat",
			pr: &ParseResult{
				Version:      "3.0.3",
				OASVersion:   OASVersion303,
				Document:     oas3Doc1,
				SourceFormat: SourceFormatYAML,
			},
			other: &ParseResult{
				Version:      "3.0.3",
				OASVersion:   OASVersion303,
				Document:     oas3Doc2,
				SourceFormat: SourceFormatJSON,
			},
			want: true,
		},
		{
			name: "both nil documents",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   nil,
			},
			other: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   nil,
			},
			want: true,
		},
		{
			name: "one nil document",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
			},
			other: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   nil,
			},
			want: false,
		},
		{
			name: "OAS3 vs OAS2 document - type mismatch",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
			},
			other: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas2Doc1,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.Equals(tt.other)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseResultDocumentEquals(t *testing.T) {
	// Create test documents
	oas3Doc1 := &OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: OASVersion303,
		Info:       &Info{Title: "Test API", Version: "1.0.0"},
	}
	oas3Doc2 := &OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: OASVersion303,
		Info:       &Info{Title: "Test API", Version: "1.0.0"},
	}
	oas3DocDifferent := &OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: OASVersion303,
		Info:       &Info{Title: "Different API", Version: "1.0.0"},
	}

	tests := []struct {
		name  string
		pr    *ParseResult
		other *ParseResult
		want  bool
	}{
		{
			name:  "both nil",
			pr:    nil,
			other: nil,
			want:  true,
		},
		{
			name:  "pr nil, other non-nil",
			pr:    nil,
			other: &ParseResult{Document: oas3Doc1},
			want:  false,
		},
		{
			name:  "pr non-nil, other nil",
			pr:    &ParseResult{Document: oas3Doc1},
			other: nil,
			want:  false,
		},
		{
			name: "same documents",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
			},
			other: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc2,
			},
			want: true,
		},
		{
			name: "different documents",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
			},
			other: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3DocDifferent,
			},
			want: false,
		},
		{
			name: "ignores OASVersion difference",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
			},
			other: &ParseResult{
				Version:    "3.1.0",
				OASVersion: OASVersion310,
				Document:   oas3Doc2,
			},
			want: true,
		},
		{
			name: "ignores Version difference",
			pr: &ParseResult{
				Version:    "3.0.3",
				OASVersion: OASVersion303,
				Document:   oas3Doc1,
			},
			other: &ParseResult{
				Version:    "3.0.0",
				OASVersion: OASVersion303,
				Document:   oas3Doc2,
			},
			want: true,
		},
		{
			name: "both nil documents",
			pr: &ParseResult{
				Version:  "3.0.3",
				Document: nil,
			},
			other: &ParseResult{
				Version:  "3.0.3",
				Document: nil,
			},
			want: true,
		},
		{
			name: "one nil document",
			pr: &ParseResult{
				Document: oas3Doc1,
			},
			other: &ParseResult{
				Document: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.DocumentEquals(tt.other)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualDocument(t *testing.T) {
	oas3Doc1 := &OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: OASVersion303,
		Info:       &Info{Title: "Test API", Version: "1.0.0"},
	}
	oas3Doc2 := &OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: OASVersion303,
		Info:       &Info{Title: "Test API", Version: "1.0.0"},
	}
	oas3DocDifferent := &OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: OASVersion303,
		Info:       &Info{Title: "Different API", Version: "1.0.0"},
	}
	oas2Doc1 := &OAS2Document{
		Swagger:    "2.0",
		OASVersion: OASVersion20,
		Info:       &Info{Title: "Test API", Version: "1.0.0"},
	}
	oas2Doc2 := &OAS2Document{
		Swagger:    "2.0",
		OASVersion: OASVersion20,
		Info:       &Info{Title: "Test API", Version: "1.0.0"},
	}
	oas2DocDifferent := &OAS2Document{
		Swagger:    "2.0",
		OASVersion: OASVersion20,
		Info:       &Info{Title: "Different API", Version: "1.0.0"},
	}

	tests := []struct {
		name string
		a    any
		b    any
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b OAS3Document",
			a:    nil,
			b:    oas3Doc1,
			want: false,
		},
		{
			name: "a OAS3Document, b nil",
			a:    oas3Doc1,
			b:    nil,
			want: false,
		},
		{
			name: "same OAS3Documents",
			a:    oas3Doc1,
			b:    oas3Doc2,
			want: true,
		},
		{
			name: "different OAS3Documents",
			a:    oas3Doc1,
			b:    oas3DocDifferent,
			want: false,
		},
		{
			name: "same OAS2Documents",
			a:    oas2Doc1,
			b:    oas2Doc2,
			want: true,
		},
		{
			name: "different OAS2Documents",
			a:    oas2Doc1,
			b:    oas2DocDifferent,
			want: false,
		},
		{
			name: "OAS3 vs OAS2 - type mismatch",
			a:    oas3Doc1,
			b:    oas2Doc1,
			want: false,
		},
		{
			name: "OAS2 vs OAS3 - type mismatch",
			a:    oas2Doc1,
			b:    oas3Doc1,
			want: false,
		},
		{
			name: "unknown type - uses reflect.DeepEqual",
			a:    map[string]any{"key": "value"},
			b:    map[string]any{"key": "value"},
			want: true,
		},
		{
			name: "unknown type different - uses reflect.DeepEqual",
			a:    map[string]any{"key": "value1"},
			b:    map[string]any{"key": "value2"},
			want: false,
		},
		// Custom struct tests for reflect.DeepEqual fallback
		{
			name: "unknown type - same custom struct values via reflect.DeepEqual",
			a:    struct{ Name string }{Name: "test"},
			b:    struct{ Name string }{Name: "test"},
			want: true,
		},
		{
			name: "unknown type - different custom struct values via reflect.DeepEqual",
			a:    struct{ Name string }{Name: "test"},
			b:    struct{ Name string }{Name: "different"},
			want: false,
		},
		{
			name: "unknown type - different custom struct types via reflect.DeepEqual",
			a:    struct{ Name string }{Name: "test"},
			b:    struct{ Title string }{Title: "test"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalDocument(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
