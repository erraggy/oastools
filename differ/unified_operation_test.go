package differ

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiffParameterUnified tests parameter comparison with all covered paths
func TestDiffParameterUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Parameter
		target        *parser.Parameter
		mode          DiffMode
		expectedCount int
		checkSeverity Severity
		checkPath     string
		checkMessage  string
	}{
		{
			name: "required changed optional to required - error",
			source: &parser.Parameter{
				Name:     "id",
				In:       "query",
				Required: false,
			},
			target: &parser.Parameter{
				Name:     "id",
				In:       "query",
				Required: true,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityError,
			checkPath:     "test.param.required",
		},
		{
			name: "required changed required to optional - warning",
			source: &parser.Parameter{
				Name:     "id",
				In:       "query",
				Required: true,
			},
			target: &parser.Parameter{
				Name:     "id",
				In:       "query",
				Required: false,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityWarning,
			checkPath:     "test.param.required",
		},
		{
			name: "type changed - incompatible",
			source: &parser.Parameter{
				Name: "id",
				In:   "query",
				Type: "string",
			},
			target: &parser.Parameter{
				Name: "id",
				In:   "query",
				Type: "integer",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityError,
			checkPath:     "test.param.type",
		},
		{
			name: "type changed - compatible (integer to number)",
			source: &parser.Parameter{
				Name: "id",
				In:   "query",
				Type: "integer",
			},
			target: &parser.Parameter{
				Name: "id",
				In:   "query",
				Type: "number",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityWarning,
			checkPath:     "test.param.type",
		},
		{
			name: "format changed",
			source: &parser.Parameter{
				Name:   "id",
				In:     "query",
				Format: "int32",
			},
			target: &parser.Parameter{
				Name:   "id",
				In:     "query",
				Format: "int64",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityWarning,
			checkPath:     "test.param.format",
		},
		{
			name: "schema added",
			source: &parser.Parameter{
				Name:   "id",
				In:     "query",
				Schema: nil,
			},
			target: &parser.Parameter{
				Name:   "id",
				In:     "query",
				Schema: &parser.Schema{Type: "string"},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityInfo,
			checkPath:     "test.param.schema",
		},
		{
			name: "schema removed",
			source: &parser.Parameter{
				Name:   "id",
				In:     "query",
				Schema: &parser.Schema{Type: "string"},
			},
			target: &parser.Parameter{
				Name:   "id",
				In:     "query",
				Schema: nil,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityWarning,
			checkPath:     "test.param.schema",
		},
		{
			name: "schema modified - type change",
			source: &parser.Parameter{
				Name:   "id",
				In:     "query",
				Schema: &parser.Schema{Type: "string"},
			},
			target: &parser.Parameter{
				Name:   "id",
				In:     "query",
				Schema: &parser.Schema{Type: "integer"},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityError,
			checkPath:     "test.param.schema.type",
		},
		{
			name: "no changes",
			source: &parser.Parameter{
				Name:     "id",
				In:       "query",
				Required: true,
				Type:     "string",
			},
			target: &parser.Parameter{
				Name:     "id",
				In:       "query",
				Required: true,
				Type:     "string",
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

			d.diffParameterUnified(tt.source, tt.target, "test.param", result)

			require.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))

			if tt.expectedCount > 0 && tt.checkPath != "" {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						if tt.checkSeverity != 0 {
							assert.Equal(t, tt.checkSeverity, c.Severity, "Expected severity %v, got %v for path %s", tt.checkSeverity, c.Severity, c.Path)
						}
						break
					}
				}
				assert.True(t, found, "Expected change at path %s not found", tt.checkPath)
			}
		})
	}
}

// TestDiffRequestBodyUnified tests request body comparison
func TestDiffRequestBodyUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.RequestBody
		target        *parser.RequestBody
		mode          DiffMode
		expectedCount int
		checkSeverity Severity
		checkPath     string
	}{
		{
			name:          "both nil - no changes",
			source:        nil,
			target:        nil,
			mode:          ModeBreaking,
			expectedCount: 0,
		},
		{
			name:   "request body added - optional",
			source: nil,
			target: &parser.RequestBody{
				Required: false,
				Content: map[string]*parser.MediaType{
					"application/json": {Schema: &parser.Schema{Type: "object"}},
				},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityInfo,
			checkPath:     "test.requestBody",
		},
		{
			name:   "request body added - required",
			source: nil,
			target: &parser.RequestBody{
				Required: true,
				Content: map[string]*parser.MediaType{
					"application/json": {Schema: &parser.Schema{Type: "object"}},
				},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityWarning,
			checkPath:     "test.requestBody",
		},
		{
			name: "request body removed - optional",
			source: &parser.RequestBody{
				Required: false,
				Content: map[string]*parser.MediaType{
					"application/json": {Schema: &parser.Schema{Type: "object"}},
				},
			},
			target:        nil,
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityWarning,
			checkPath:     "test.requestBody",
		},
		{
			name: "request body removed - required",
			source: &parser.RequestBody{
				Required: true,
				Content: map[string]*parser.MediaType{
					"application/json": {Schema: &parser.Schema{Type: "object"}},
				},
			},
			target:        nil,
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityError,
			checkPath:     "test.requestBody",
		},
		{
			name: "required changed optional to required - error",
			source: &parser.RequestBody{
				Required: false,
				Content:  map[string]*parser.MediaType{},
			},
			target: &parser.RequestBody{
				Required: true,
				Content:  map[string]*parser.MediaType{},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityError,
			checkPath:     "test.requestBody.required",
		},
		{
			name: "required changed required to optional - info",
			source: &parser.RequestBody{
				Required: true,
				Content:  map[string]*parser.MediaType{},
			},
			target: &parser.RequestBody{
				Required: false,
				Content:  map[string]*parser.MediaType{},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkSeverity: SeverityInfo,
			checkPath:     "test.requestBody.required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffRequestBodyUnified(tt.source, tt.target, "test.requestBody", result)

			require.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))

			if tt.expectedCount > 0 && tt.checkPath != "" {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						if tt.checkSeverity != 0 {
							assert.Equal(t, tt.checkSeverity, c.Severity, "Expected severity %v, got %v for path %s", tt.checkSeverity, c.Severity, c.Path)
						}
						break
					}
				}
				assert.True(t, found, "Expected change at path %s not found", tt.checkPath)
			}
		})
	}
}

// TestDiffRequestBodyMediaTypeUnified tests request body media type comparison
func TestDiffRequestBodyMediaTypeUnified(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffRequestBodyMediaTypeUnified(tt.source, tt.target, "test", result)

			assert.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
		})
	}
}

// TestDiffRequestBodyContentUnified tests request body content map comparison
func TestDiffRequestBodyContentUnified(t *testing.T) {
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

			d.diffRequestBodyContentUnified(tt.source, tt.target, "test", result)

			assert.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
		})
	}
}

// TestDiffOperationUnified_OperationId tests operation comparison focusing on operationId
func TestDiffOperationUnified_OperationId(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.Operation
		target        *parser.Operation
		mode          DiffMode
		expectedCount int
		checkPath     string
	}{
		{
			name: "operationId changed",
			source: &parser.Operation{
				OperationID: "getUsers",
			},
			target: &parser.Operation{
				OperationID: "listUsers",
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.op.operationId",
		},
		{
			name: "deprecated changed false to true",
			source: &parser.Operation{
				Deprecated: false,
			},
			target: &parser.Operation{
				Deprecated: true,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.op.deprecated",
		},
		{
			name: "deprecated changed true to false",
			source: &parser.Operation{
				Deprecated: true,
			},
			target: &parser.Operation{
				Deprecated: false,
			},
			mode:          ModeBreaking,
			expectedCount: 1,
			checkPath:     "test.op.deprecated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffOperationUnified(tt.source, tt.target, "test.op", result)

			require.GreaterOrEqual(t, len(result.Changes), tt.expectedCount, "Expected at least %d changes, got %d", tt.expectedCount, len(result.Changes))

			if tt.checkPath != "" {
				found := false
				for _, c := range result.Changes {
					if c.Path == tt.checkPath {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected change at path %s not found", tt.checkPath)
			}
		})
	}
}

// TestDiffParametersUnified tests parameter slice comparison
func TestDiffParametersUnified(t *testing.T) {
	tests := []struct {
		name          string
		source        []*parser.Parameter
		target        []*parser.Parameter
		mode          DiffMode
		expectedCount int
	}{
		{
			name:          "both empty - no changes",
			source:        []*parser.Parameter{},
			target:        []*parser.Parameter{},
			mode:          ModeBreaking,
			expectedCount: 0,
		},
		{
			name:   "required parameter added",
			source: []*parser.Parameter{},
			target: []*parser.Parameter{
				{Name: "id", In: "query", Required: true},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name:   "optional parameter added",
			source: []*parser.Parameter{},
			target: []*parser.Parameter{
				{Name: "id", In: "query", Required: false},
			},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "required parameter removed",
			source: []*parser.Parameter{
				{Name: "id", In: "query", Required: true},
			},
			target:        []*parser.Parameter{},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "optional parameter removed",
			source: []*parser.Parameter{
				{Name: "id", In: "query", Required: false},
			},
			target:        []*parser.Parameter{},
			mode:          ModeBreaking,
			expectedCount: 1,
		},
		{
			name: "parameter modified",
			source: []*parser.Parameter{
				{Name: "id", In: "query", Type: "string"},
			},
			target: []*parser.Parameter{
				{Name: "id", In: "query", Type: "integer"},
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

			d.diffParametersUnified(tt.source, tt.target, "test.params", result)

			assert.Len(t, result.Changes, tt.expectedCount, "Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
		})
	}
}
