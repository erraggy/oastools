package differ

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// TestDiffResponseLinksUnified tests response links comparison
func TestDiffResponseLinksUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        map[string]*parser.Link
		target        map[string]*parser.Link
		mode          DiffMode
		expectedCount int
		checkAdded    string
		checkRemoved  string
	}{
		{
			name:          "both empty - no changes",
			source:        map[string]*parser.Link{},
			target:        map[string]*parser.Link{},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
		{
			name:   "link added",
			source: map[string]*parser.Link{},
			target: map[string]*parser.Link{
				"GetUser": {
					OperationID: "getUser",
					Description: "Get user by ID",
				},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkAdded:    "test.links.GetUser",
		},
		{
			name: "link removed",
			source: map[string]*parser.Link{
				"GetUser": {
					OperationID: "getUser",
					Description: "Get user by ID",
				},
			},
			target:        map[string]*parser.Link{},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkRemoved:  "test.links.GetUser",
		},
		{
			name: "link modified - operationId changed",
			source: map[string]*parser.Link{
				"GetUser": {
					OperationID: "getUser",
				},
			},
			target: map[string]*parser.Link{
				"GetUser": {
					OperationID: "getUserById",
				},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "link modified - operationRef changed",
			source: map[string]*parser.Link{
				"GetUser": {
					OperationRef: "#/paths/~1users~1{id}/get",
				},
			},
			target: map[string]*parser.Link{
				"GetUser": {
					OperationRef: "#/paths/~1users~1{userId}/get",
				},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "link modified - description changed",
			source: map[string]*parser.Link{
				"GetUser": {
					OperationID: "getUser",
					Description: "Original description",
				},
			},
			target: map[string]*parser.Link{
				"GetUser": {
					OperationID: "getUser",
					Description: "Updated description",
				},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "multiple links - added and removed",
			source: map[string]*parser.Link{
				"GetUser": {OperationID: "getUser"},
			},
			target: map[string]*parser.Link{
				"GetOrder": {OperationID: "getOrder"},
			},
			mode:          ModeBreaking,
			expectedCount: 2, // 1 removed, 1 added
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffResponseLinksUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s (type: %v)", c.Path, c.Message, c.Type)
				}
				return
			}

			// Check specific added change
			if tt.checkAdded != "" {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkAdded && c.Type == ChangeTypeAdded {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected added change at path %s not found", tt.checkAdded)
				}
			}

			// Check specific removed change
			if tt.checkRemoved != "" {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkRemoved && c.Type == ChangeTypeRemoved {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected removed change at path %s not found", tt.checkRemoved)
				}
			}
		})
	}
}

// TestDiffResponseExamplesUnified tests response examples comparison
func TestDiffResponseExamplesUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        map[string]any
		target        map[string]any
		mode          DiffMode
		expectedCount int
		checkAdded    string
		checkRemoved  string
	}{
		{
			name:          "both empty - no changes",
			source:        map[string]any{},
			target:        map[string]any{},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
		{
			name:   "example added",
			source: map[string]any{},
			target: map[string]any{
				"success": map[string]any{"id": 1, "name": "test"},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkAdded:    "test.examples.success",
		},
		{
			name: "example removed",
			source: map[string]any{
				"success": map[string]any{"id": 1, "name": "test"},
			},
			target:        map[string]any{},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkRemoved:  "test.examples.success",
		},
		{
			name: "multiple examples - added and removed",
			source: map[string]any{
				"example1": "value1",
			},
			target: map[string]any{
				"example2": "value2",
			},
			mode:          ModeBreaking,
			expectedCount: 2, // 1 removed, 1 added
		},
		{
			name: "example unchanged - no diff",
			source: map[string]any{
				"example": "value",
			},
			target: map[string]any{
				"example": "value",
			},
			mode:          ModeBreaking,
			expectedCount: 0, // examples are only added/removed, not deep-compared
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffResponseExamplesUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s (type: %v)", c.Path, c.Message, c.Type)
				}
				return
			}

			// Check specific added change
			if tt.checkAdded != "" {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkAdded && c.Type == ChangeTypeAdded {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected added change at path %s not found", tt.checkAdded)
				}
			}

			// Check specific removed change
			if tt.checkRemoved != "" {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkRemoved && c.Type == ChangeTypeRemoved {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected removed change at path %s not found", tt.checkRemoved)
				}
			}
		})
	}
}

// TestDiffMediaTypeUnified tests media type comparison
func TestDiffMediaTypeUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.MediaType
		target        *parser.MediaType
		mode          DiffMode
		expectedCount int
		checkPath     string
	}{
		{
			name: "schema added",
			source: &parser.MediaType{
				Schema: nil,
			},
			target: &parser.MediaType{
				Schema: &parser.Schema{Type: "object"},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.schema",
		},
		{
			name: "schema removed",
			source: &parser.MediaType{
				Schema: &parser.Schema{Type: "object"},
			},
			target: &parser.MediaType{
				Schema: nil,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.schema",
		},
		{
			name: "schema changed",
			source: &parser.MediaType{
				Schema: &parser.Schema{Type: "string"},
			},
			target: &parser.MediaType{
				Schema: &parser.Schema{Type: "object"},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.schema.type",
		},
		{
			name: "no changes",
			source: &parser.MediaType{
				Schema: &parser.Schema{Type: "object"},
			},
			target: &parser.MediaType{
				Schema: &parser.Schema{Type: "object"},
			},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
		{
			name: "both schemas nil - no changes",
			source: &parser.MediaType{
				Schema: nil,
			},
			target: &parser.MediaType{
				Schema: nil,
			},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffMediaTypeUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s", c.Path, c.Message)
				}
			}

			if tt.checkPath != "" && tt.expectedCount > 0 {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected change at path %s not found", tt.checkPath)
				}
			}
		})
	}
}

// TestDiffResponseContentUnified tests response content map comparison
func TestDiffResponseContentUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        map[string]*parser.MediaType
		target        map[string]*parser.MediaType
		mode          DiffMode
		expectedCount int
	}{
		{
			name:          "both empty - no changes",
			source:        map[string]*parser.MediaType{},
			target:        map[string]*parser.MediaType{},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
		{
			name:   "media type added",
			source: map[string]*parser.MediaType{},
			target: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "object"}},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "media type removed",
			source: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "object"}},
			},
			target:        map[string]*parser.MediaType{},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "media type modified",
			source: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "string"}},
			},
			target: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "object"}},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffResponseContentUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s", c.Path, c.Message)
				}
			}
		})
	}
}

// TestDiffResponseHeadersUnified tests response headers comparison
func TestDiffResponseHeadersUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        map[string]*parser.Header
		target        map[string]*parser.Header
		mode          DiffMode
		expectedCount int
	}{
		{
			name:          "both empty - no changes",
			source:        map[string]*parser.Header{},
			target:        map[string]*parser.Header{},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
		{
			name:   "header added",
			source: map[string]*parser.Header{},
			target: map[string]*parser.Header{
				"X-Request-Id": {Description: "Request ID", Required: false},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "header removed",
			source: map[string]*parser.Header{
				"X-Request-Id": {Description: "Request ID", Required: false},
			},
			target:        map[string]*parser.Header{},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "header modified - required changed",
			source: map[string]*parser.Header{
				"X-Request-Id": {Description: "Request ID", Required: false},
			},
			target: map[string]*parser.Header{
				"X-Request-Id": {Description: "Request ID", Required: true},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffResponseHeadersUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s", c.Path, c.Message)
				}
			}
		})
	}
}

// TestDiffHeaderUnified tests individual header comparison
func TestDiffHeaderUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Header
		target        *parser.Header
		mode          DiffMode
		expectedCount int
		checkPath     string
	}{
		{
			name: "description changed",
			source: &parser.Header{
				Description: "Original",
			},
			target: &parser.Header{
				Description: "Updated",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.description",
		},
		{
			name: "required changed optional to required - error",
			source: &parser.Header{
				Required: false,
			},
			target: &parser.Header{
				Required: true,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.required",
		},
		{
			name: "required changed required to optional - warning",
			source: &parser.Header{
				Required: true,
			},
			target: &parser.Header{
				Required: false,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.required",
		},
		{
			name: "type changed",
			source: &parser.Header{
				Type: "string",
			},
			target: &parser.Header{
				Type: "integer",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.type",
		},
		{
			name: "deprecated changed - simple mode",
			source: &parser.Header{
				Deprecated: false,
			},
			target: &parser.Header{
				Deprecated: true,
			},
			mode:          ModeSimple,
			expectedCount: 1,
			checkPath:     "test.deprecated",
		},
		{
			name: "style changed - simple mode",
			source: &parser.Header{
				Style: "simple",
			},
			target: &parser.Header{
				Style: "form",
			},
			mode:          ModeSimple,
			expectedCount: 1,
			checkPath:     "test.style",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffHeaderUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s", c.Path, c.Message)
				}
				return
			}

			if tt.checkPath != "" && tt.expectedCount > 0 {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected change at path %s not found", tt.checkPath)
				}
			}
		})
	}
}

// TestDiffResponsesUnified tests responses object comparison
func TestDiffResponsesUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Responses
		target        *parser.Responses
		mode          DiffMode
		expectedCount int
	}{
		{
			name:          "both nil - no changes",
			source:        nil,
			target:        nil,
			mode:          ModeBreaking,
			expectedCount: 0,
		},
		{
			name:   "responses added",
			source: nil,
			target: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {Description: "OK"},
				},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "responses removed",
			source: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {Description: "OK"},
				},
			},
			target:        nil,
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "success code removed - error",
			source: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {Description: "OK"},
				},
			},
			target: &parser.Responses{
				Codes: map[string]*parser.Response{},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "error code added - warning",
			source: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {Description: "OK"},
				},
			},
			target: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {Description: "OK"},
					"400": {Description: "Bad Request"},
				},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "success code added - info",
			source: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {Description: "OK"},
				},
			},
			target: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {Description: "OK"},
					"201": {Description: "Created"},
				},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffResponsesUnified(tt.source, tt.target, "test.responses", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s (severity: %v)", c.Path, c.Message, c.Severity)
				}
			}
		})
	}
}

// TestDiffLinkUnified tests individual link comparison
func TestDiffLinkUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Link
		target        *parser.Link
		mode          DiffMode
		expectedCount int
		checkPath     string
	}{
		{
			name: "operationRef changed",
			source: &parser.Link{
				OperationRef: "#/paths/~1users/get",
			},
			target: &parser.Link{
				OperationRef: "#/paths/~1users~1{id}/get",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.operationRef",
		},
		{
			name: "operationId changed",
			source: &parser.Link{
				OperationID: "getUsers",
			},
			target: &parser.Link{
				OperationID: "listUsers",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.operationId",
		},
		{
			name: "description changed",
			source: &parser.Link{
				Description: "Original",
			},
			target: &parser.Link{
				Description: "Updated",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.description",
		},
		{
			name: "no changes",
			source: &parser.Link{
				OperationID: "getUser",
				Description: "Get user",
			},
			target: &parser.Link{
				OperationID: "getUser",
				Description: "Get user",
			},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffLinkUnified(tt.source, tt.target, "test", result)

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for _, c := range result.Changes {
					t.Logf("Change: %s - %s", c.Path, c.Message)
				}
				return
			}

			if tt.checkPath != "" && tt.expectedCount > 0 {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected change at path %s not found", tt.checkPath)
				}
			}
		})
	}
}
