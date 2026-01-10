package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSchemaEquals_Extensions tests the Extra map comparison.
func TestSchemaEquals_Extensions(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "both nil Extra",
			a:    &Schema{Extra: nil},
			b:    &Schema{Extra: nil},
			want: true,
		},
		{
			name: "both empty Extra",
			a:    &Schema{Extra: map[string]any{}},
			b:    &Schema{Extra: map[string]any{}},
			want: true,
		},
		{
			name: "nil vs empty Extra",
			a:    &Schema{Extra: nil},
			b:    &Schema{Extra: map[string]any{}},
			want: true,
		},
		{
			name: "same Extra",
			a:    &Schema{Extra: map[string]any{"x-custom": "value"}},
			b:    &Schema{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra values",
			a:    &Schema{Extra: map[string]any{"x-custom": "value1"}},
			b:    &Schema{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		{
			name: "different Extra keys",
			a:    &Schema{Extra: map[string]any{"x-custom1": "value"}},
			b:    &Schema{Extra: map[string]any{"x-custom2": "value"}},
			want: false,
		},
		{
			name: "nested extension values same",
			a: &Schema{Extra: map[string]any{
				"x-nested": map[string]any{
					"level1": map[string]any{
						"level2": "deep value",
					},
				},
			}},
			b: &Schema{Extra: map[string]any{
				"x-nested": map[string]any{
					"level1": map[string]any{
						"level2": "deep value",
					},
				},
			}},
			want: true,
		},
		{
			name: "nested extension values different",
			a: &Schema{Extra: map[string]any{
				"x-nested": map[string]any{
					"level1": "value1",
				},
			}},
			b: &Schema{Extra: map[string]any{
				"x-nested": map[string]any{
					"level1": "value2",
				},
			}},
			want: false,
		},
		{
			name: "extra with mixed types",
			a: &Schema{Extra: map[string]any{
				"x-string": "value",
				"x-number": float64(42),
				"x-bool":   true,
				"x-array":  []any{1, 2, 3},
			}},
			b: &Schema{Extra: map[string]any{
				"x-string": "value",
				"x-number": float64(42),
				"x-bool":   true,
				"x-array":  []any{1, 2, 3},
			}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equals(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaEquals_Discriminator tests Discriminator comparison.
func TestSchemaEquals_Discriminator(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "both nil Discriminator",
			a:    &Schema{Discriminator: nil},
			b:    &Schema{Discriminator: nil},
			want: true,
		},
		{
			name: "nil vs non-nil Discriminator",
			a:    &Schema{Discriminator: nil},
			b:    &Schema{Discriminator: &Discriminator{PropertyName: "type"}},
			want: false,
		},
		{
			name: "same Discriminator",
			a:    &Schema{Discriminator: &Discriminator{PropertyName: "type"}},
			b:    &Schema{Discriminator: &Discriminator{PropertyName: "type"}},
			want: true,
		},
		{
			name: "different PropertyName",
			a:    &Schema{Discriminator: &Discriminator{PropertyName: "type"}},
			b:    &Schema{Discriminator: &Discriminator{PropertyName: "kind"}},
			want: false,
		},
		{
			name: "same Mapping",
			a: &Schema{Discriminator: &Discriminator{
				PropertyName: "type",
				Mapping: map[string]string{
					"dog": "#/components/schemas/Dog",
					"cat": "#/components/schemas/Cat",
				},
			}},
			b: &Schema{Discriminator: &Discriminator{
				PropertyName: "type",
				Mapping: map[string]string{
					"dog": "#/components/schemas/Dog",
					"cat": "#/components/schemas/Cat",
				},
			}},
			want: true,
		},
		{
			name: "different Mapping",
			a: &Schema{Discriminator: &Discriminator{
				PropertyName: "type",
				Mapping: map[string]string{
					"dog": "#/components/schemas/Dog",
				},
			}},
			b: &Schema{Discriminator: &Discriminator{
				PropertyName: "type",
				Mapping: map[string]string{
					"cat": "#/components/schemas/Cat",
				},
			}},
			want: false,
		},
		{
			name: "discriminator with extra",
			a: &Schema{Discriminator: &Discriminator{
				PropertyName: "type",
				Extra:        map[string]any{"x-custom": "value"},
			}},
			b: &Schema{Discriminator: &Discriminator{
				PropertyName: "type",
				Extra:        map[string]any{"x-custom": "value"},
			}},
			want: true,
		},
		{
			name: "discriminator with different extra",
			a: &Schema{Discriminator: &Discriminator{
				PropertyName: "type",
				Extra:        map[string]any{"x-custom": "value1"},
			}},
			b: &Schema{Discriminator: &Discriminator{
				PropertyName: "type",
				Extra:        map[string]any{"x-custom": "value2"},
			}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equals(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaEquals_XML tests XML comparison.
func TestSchemaEquals_XML(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "both nil XML",
			a:    &Schema{XML: nil},
			b:    &Schema{XML: nil},
			want: true,
		},
		{
			name: "nil vs non-nil XML",
			a:    &Schema{XML: nil},
			b:    &Schema{XML: &XML{Name: "item"}},
			want: false,
		},
		{
			name: "same XML",
			a:    &Schema{XML: &XML{Name: "item", Namespace: "http://example.com"}},
			b:    &Schema{XML: &XML{Name: "item", Namespace: "http://example.com"}},
			want: true,
		},
		{
			name: "different Name",
			a:    &Schema{XML: &XML{Name: "item"}},
			b:    &Schema{XML: &XML{Name: "element"}},
			want: false,
		},
		{
			name: "different Namespace",
			a:    &Schema{XML: &XML{Namespace: "http://example.com"}},
			b:    &Schema{XML: &XML{Namespace: "http://other.com"}},
			want: false,
		},
		{
			name: "different Prefix",
			a:    &Schema{XML: &XML{Prefix: "ns1"}},
			b:    &Schema{XML: &XML{Prefix: "ns2"}},
			want: false,
		},
		{
			name: "different Attribute",
			a:    &Schema{XML: &XML{Attribute: true}},
			b:    &Schema{XML: &XML{Attribute: false}},
			want: false,
		},
		{
			name: "different Wrapped",
			a:    &Schema{XML: &XML{Wrapped: true}},
			b:    &Schema{XML: &XML{Wrapped: false}},
			want: false,
		},
		{
			name: "full XML same",
			a: &Schema{XML: &XML{
				Name:      "items",
				Namespace: "http://example.com",
				Prefix:    "ex",
				Attribute: false,
				Wrapped:   true,
			}},
			b: &Schema{XML: &XML{
				Name:      "items",
				Namespace: "http://example.com",
				Prefix:    "ex",
				Attribute: false,
				Wrapped:   true,
			}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equals(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSchemaEquals_ExternalDocs tests ExternalDocs comparison.
func TestSchemaEquals_ExternalDocs(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "ExternalDocs both nil",
			a:    &Schema{ExternalDocs: nil},
			b:    &Schema{ExternalDocs: nil},
			want: true,
		},
		{
			name: "ExternalDocs nil vs non-nil",
			a:    &Schema{ExternalDocs: nil},
			b:    &Schema{ExternalDocs: &ExternalDocs{URL: "https://example.com"}},
			want: false,
		},
		{
			name: "ExternalDocs same",
			a:    &Schema{ExternalDocs: &ExternalDocs{URL: "https://example.com", Description: "API docs"}},
			b:    &Schema{ExternalDocs: &ExternalDocs{URL: "https://example.com", Description: "API docs"}},
			want: true,
		},
		{
			name: "ExternalDocs different URL",
			a:    &Schema{ExternalDocs: &ExternalDocs{URL: "https://example.com"}},
			b:    &Schema{ExternalDocs: &ExternalDocs{URL: "https://other.com"}},
			want: false,
		},
		{
			name: "ExternalDocs different Description",
			a:    &Schema{ExternalDocs: &ExternalDocs{Description: "Description 1"}},
			b:    &Schema{ExternalDocs: &ExternalDocs{Description: "Description 2"}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equals(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
