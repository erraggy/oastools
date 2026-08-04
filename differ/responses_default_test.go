package differ

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// TestDiffResponsesUnifiedObservesDefault covers the default response, which
// has its own field on parser.Responses rather than a Codes entry. A scan of
// Codes alone cannot see it, so removing one would be reported as no change at
// all: the two documents would be called identical.
func TestDiffResponsesUnifiedObservesDefault(t *testing.T) {
	withDefault := func(description string) *parser.Responses {
		return &parser.Responses{
			Default: &parser.Response{Description: description},
			Codes:   map[string]*parser.Response{"200": {Description: "OK"}},
		}
	}
	withoutDefault := func() *parser.Responses {
		return &parser.Responses{
			Codes: map[string]*parser.Response{"200": {Description: "OK"}},
		}
	}

	tests := []struct {
		name       string
		source     *parser.Responses
		target     *parser.Responses
		wantPath   string
		wantType   ChangeType
		wantNoDiff bool
	}{
		{
			name:     "default removed",
			source:   withDefault("fallback"),
			target:   withoutDefault(),
			wantPath: "test[default]",
			wantType: ChangeTypeRemoved,
		},
		{
			name:     "default added",
			source:   withoutDefault(),
			target:   withDefault("fallback"),
			wantPath: "test[default]",
			wantType: ChangeTypeAdded,
		},
		{
			name:     "default description changed",
			source:   withDefault("fallback"),
			target:   withDefault("something else"),
			wantPath: "test[default].description",
			wantType: ChangeTypeModified,
		},
		{
			name:       "default unchanged",
			source:     withDefault("fallback"),
			target:     withDefault("fallback"),
			wantNoDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = ModeBreaking
			result := &DiffResult{}

			d.diffResponsesUnified(tt.source, tt.target, "test", result)

			if tt.wantNoDiff {
				assert.Empty(t, result.Changes)
				return
			}

			paths := make([]string, 0, len(result.Changes))
			found := false
			for _, c := range result.Changes {
				paths = append(paths, string(c.Type)+" "+c.Path)
				if c.Path == tt.wantPath && c.Type == tt.wantType {
					found = true
				}
			}
			assert.True(t, found, "want a %s change at %q; got %v", tt.wantType, tt.wantPath, paths)
		})
	}
}

// TestDiffResponsesUnifiedObservesExtensions covers extensions on the Responses
// Object itself. They are held apart from the status codes, so the Codes loops
// do not see them either.
func TestDiffResponsesUnifiedObservesExtensions(t *testing.T) {
	source := &parser.Responses{
		Codes: map[string]*parser.Response{"200": {Description: "OK"}},
		Extra: map[string]any{"x-note": "one"},
	}
	target := &parser.Responses{
		Codes: map[string]*parser.Response{"200": {Description: "OK"}},
		Extra: map[string]any{"x-note": "two"},
	}

	d := New()
	d.Mode = ModeBreaking
	result := &DiffResult{}

	d.diffResponsesUnified(source, target, "test", result)

	require.NotEmpty(t, result.Changes, "a changed extension value is a change")
	found := false
	for _, c := range result.Changes {
		if c.Type == ChangeTypeModified {
			found = true
		}
	}
	assert.True(t, found, "want the extension reported as modified; got %+v", result.Changes)
}
