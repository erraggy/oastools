package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExtensionFilter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ExtensionFilter
		wantErr bool
	}{
		{
			name:  "simple existence",
			input: "x-foo",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{{Key: "x-foo"}},
				},
			},
		},
		{
			name:  "key=value",
			input: "x-foo=bar",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{{Key: "x-foo", Value: strPtr("bar")}},
				},
			},
		},
		{
			name:  "negated existence",
			input: "!x-foo",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{{Key: "x-foo", Negated: true}},
				},
			},
		},
		{
			name:  "negated value",
			input: "x-foo!=bar",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{{Key: "x-foo", Value: strPtr("bar"), Negated: true}},
				},
			},
		},
		{
			name:  "AND operator",
			input: "x-foo+x-bar=1",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{
						{Key: "x-foo"},
						{Key: "x-bar", Value: strPtr("1")},
					},
				},
			},
		},
		{
			name:  "OR operator",
			input: "x-foo,x-bar=1",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{{Key: "x-foo"}},
					{{Key: "x-bar", Value: strPtr("1")}},
				},
			},
		},
		{
			name:  "mixed AND+OR",
			input: "x-a+x-b=1,!x-c",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{
						{Key: "x-a"},
						{Key: "x-b", Value: strPtr("1")},
					},
					{{Key: "x-c", Negated: true}},
				},
			},
		},
		{
			name:    "missing x- prefix",
			input:   "foo",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "bare equals",
			input:   "=value",
			wantErr: true,
		},
		{
			name:    "bare bang",
			input:   "!",
			wantErr: true,
		},
		{
			name:    "double negation",
			input:   "!x-foo!=bar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseExtensionFilter(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, len(tt.want.Groups), len(got.Groups))
			for i, group := range got.Groups {
				require.Equal(t, len(tt.want.Groups[i]), len(group))
				for j, expr := range group {
					wantExpr := tt.want.Groups[i][j]
					assert.Equal(t, wantExpr.Key, expr.Key)
					assert.Equal(t, wantExpr.Negated, expr.Negated)
					gotVal := "<nil>"
					if expr.Value != nil {
						gotVal = *expr.Value
					}
					wantVal := "<nil>"
					if wantExpr.Value != nil {
						wantVal = *wantExpr.Value
					}
					assert.Equal(t, wantVal, gotVal)
				}
			}
		})
	}
}

func TestExtensionFilter_Match(t *testing.T) {
	tests := []struct {
		name       string
		filter     string
		extensions map[string]any
		want       bool
	}{
		{
			name:       "existence match",
			filter:     "x-foo",
			extensions: map[string]any{"x-foo": "bar"},
			want:       true,
		},
		{
			name:       "existence no match",
			filter:     "x-foo",
			extensions: map[string]any{"x-bar": "baz"},
			want:       false,
		},
		{
			name:       "existence nil extensions",
			filter:     "x-foo",
			extensions: nil,
			want:       false,
		},
		{
			name:       "value match string",
			filter:     "x-foo=bar",
			extensions: map[string]any{"x-foo": "bar"},
			want:       true,
		},
		{
			name:       "value match bool as string",
			filter:     "x-internal=true",
			extensions: map[string]any{"x-internal": true},
			want:       true,
		},
		{
			name:       "value no match",
			filter:     "x-foo=bar",
			extensions: map[string]any{"x-foo": "baz"},
			want:       false,
		},
		{
			name:       "negated existence - key absent",
			filter:     "!x-foo",
			extensions: map[string]any{"x-bar": "baz"},
			want:       true,
		},
		{
			name:       "negated existence - key present",
			filter:     "!x-foo",
			extensions: map[string]any{"x-foo": "bar"},
			want:       false,
		},
		{
			name:       "negated value",
			filter:     "x-foo!=bar",
			extensions: map[string]any{"x-foo": "baz"},
			want:       true,
		},
		{
			name:       "AND - both match",
			filter:     "x-foo+x-bar",
			extensions: map[string]any{"x-foo": "1", "x-bar": "2"},
			want:       true,
		},
		{
			name:       "AND - one missing",
			filter:     "x-foo+x-bar",
			extensions: map[string]any{"x-foo": "1"},
			want:       false,
		},
		{
			name:       "OR - first matches",
			filter:     "x-foo,x-bar",
			extensions: map[string]any{"x-foo": "1"},
			want:       true,
		},
		{
			name:       "OR - second matches",
			filter:     "x-foo,x-bar",
			extensions: map[string]any{"x-bar": "1"},
			want:       true,
		},
		{
			name:       "OR - none match",
			filter:     "x-foo,x-bar",
			extensions: map[string]any{"x-baz": "1"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ParseExtensionFilter(tt.filter)
			require.NoError(t, err)
			got := f.Match(tt.extensions)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatExtensions(t *testing.T) {
	tests := []struct {
		name  string
		extra map[string]any
		want  string
	}{
		{name: "nil map", extra: nil, want: ""},
		{name: "empty map", extra: map[string]any{}, want: ""},
		{name: "no extensions", extra: map[string]any{"foo": "bar"}, want: ""},
		{name: "single extension", extra: map[string]any{"x-foo": "bar"}, want: "x-foo=bar"},
		{
			name:  "sorted output",
			extra: map[string]any{"x-beta": "2", "x-alpha": "1"},
			want:  "x-alpha=1, x-beta=2",
		},
		{
			name:  "ignores non-extension keys",
			extra: map[string]any{"x-real": true, "notAnExtension": "skip"},
			want:  "x-real=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatExtensions(tt.extra)
			assert.Equal(t, tt.want, got)
		})
	}
}

func strPtr(s string) *string { return &s }
