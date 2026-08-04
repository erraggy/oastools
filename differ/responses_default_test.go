package differ

import (
	"strings"
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
		name         string
		source       *parser.Responses
		target       *parser.Responses
		mode         DiffMode
		wantPath     string
		wantType     ChangeType
		wantSeverity *Severity
		wantNoDiff   bool
	}{
		{
			// Graded like a removed success code: the default covers every
			// code not listed individually, so losing it can leave the
			// operation with no documented success response at all.
			name:         "default removed is breaking",
			source:       withDefault("fallback"),
			target:       withoutDefault(),
			mode:         ModeBreaking,
			wantPath:     "test[default]",
			wantType:     ChangeTypeRemoved,
			wantSeverity: SeverityPtr(SeverityError),
		},
		{
			// Still reported outside breaking mode, where severity is not
			// graded at all: addChange zeroes it for every change.
			name:     "default removed outside breaking mode",
			source:   withDefault("fallback"),
			target:   withoutDefault(),
			mode:     ModeSimple,
			wantPath: "test[default]",
			wantType: ChangeTypeRemoved,
		},
		{
			name:         "default added",
			source:       withoutDefault(),
			target:       withDefault("fallback"),
			mode:         ModeBreaking,
			wantPath:     "test[default]",
			wantType:     ChangeTypeAdded,
			wantSeverity: SeverityPtr(SeverityInfo),
		},
		{
			name:     "default description changed",
			source:   withDefault("fallback"),
			target:   withDefault("something else"),
			mode:     ModeBreaking,
			wantPath: "test[default].description",
			wantType: ChangeTypeModified,
		},
		{
			name:       "default unchanged",
			source:     withDefault("fallback"),
			target:     withDefault("fallback"),
			mode:       ModeBreaking,
			wantNoDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = tt.mode
			result := &DiffResult{}

			d.diffResponsesUnified(tt.source, tt.target, "test", result)

			if tt.wantNoDiff {
				assert.Empty(t, result.Changes)
				return
			}

			seen := make([]string, 0, len(result.Changes))
			var match *Change
			for i, c := range result.Changes {
				seen = append(seen, string(c.Type)+" "+c.Path)
				if c.Path == tt.wantPath && c.Type == tt.wantType {
					match = &result.Changes[i]
				}
			}
			require.NotNil(t, match, "want a %s change at %q; got %v", tt.wantType, tt.wantPath, seen)

			if tt.wantSeverity != nil {
				assert.Equal(t, *tt.wantSeverity, match.Severity,
					"severity of %s at %q", tt.wantType, tt.wantPath)
			}
		})
	}
}

// TestDiffResponsesUnifiedObservesExtensions covers extensions on the Responses
// Object itself. They are held apart from the status codes, so the Codes loops
// do not see them either.
func TestDiffResponsesUnifiedObservesExtensions(t *testing.T) {
	withExtra := func(extra map[string]any) *parser.Responses {
		return &parser.Responses{
			Codes: map[string]*parser.Response{"200": {Description: "OK"}},
			Extra: extra,
		}
	}

	tests := []struct {
		name       string
		source     *parser.Responses
		target     *parser.Responses
		wantType   ChangeType
		wantNoDiff bool
	}{
		{
			name:     "extension value modified",
			source:   withExtra(map[string]any{"x-note": "one"}),
			target:   withExtra(map[string]any{"x-note": "two"}),
			wantType: ChangeTypeModified,
		},
		{
			name:     "extension added",
			source:   withExtra(nil),
			target:   withExtra(map[string]any{"x-note": "one"}),
			wantType: ChangeTypeAdded,
		},
		{
			name:     "extension removed",
			source:   withExtra(map[string]any{"x-note": "one"}),
			target:   withExtra(nil),
			wantType: ChangeTypeRemoved,
		},
		{
			name:       "extension unchanged",
			source:     withExtra(map[string]any{"x-note": "one"}),
			target:     withExtra(map[string]any{"x-note": "one"}),
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

			seen := make([]string, 0, len(result.Changes))
			found := false
			for _, c := range result.Changes {
				seen = append(seen, string(c.Type)+" "+c.Path)
				if c.Type == tt.wantType && strings.Contains(c.Path, "x-note") {
					found = true
				}
			}
			assert.True(t, found, "want a %s change naming x-note; got %v", tt.wantType, seen)
		})
	}
}
