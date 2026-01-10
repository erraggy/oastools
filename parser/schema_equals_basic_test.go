package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSchemaEquals_Nil tests nil handling.
func TestSchemaEquals_Nil(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil",
			a:    nil,
			b:    &Schema{Type: "string"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &Schema{Type: "string"},
			b:    nil,
			want: false,
		},
		{
			name: "both empty schemas",
			a:    &Schema{},
			b:    &Schema{},
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

// TestSchemaEquals_StringFields tests all string field comparisons.
func TestSchemaEquals_StringFields(t *testing.T) {
	// Each test case checks that a different string field triggers inequality
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		// Ref
		{
			name: "same Ref",
			a:    &Schema{Ref: "#/components/schemas/Pet"},
			b:    &Schema{Ref: "#/components/schemas/Pet"},
			want: true,
		},
		{
			name: "different Ref",
			a:    &Schema{Ref: "#/components/schemas/Pet"},
			b:    &Schema{Ref: "#/components/schemas/User"},
			want: false,
		},
		// Schema
		{
			name: "same Schema",
			a:    &Schema{Schema: "https://json-schema.org/draft/2020-12/schema"},
			b:    &Schema{Schema: "https://json-schema.org/draft/2020-12/schema"},
			want: true,
		},
		{
			name: "different Schema",
			a:    &Schema{Schema: "https://json-schema.org/draft/2020-12/schema"},
			b:    &Schema{Schema: "https://json-schema.org/draft/07/schema"},
			want: false,
		},
		// Title
		{
			name: "same Title",
			a:    &Schema{Title: "Pet"},
			b:    &Schema{Title: "Pet"},
			want: true,
		},
		{
			name: "different Title",
			a:    &Schema{Title: "Pet"},
			b:    &Schema{Title: "User"},
			want: false,
		},
		// Description
		{
			name: "same Description",
			a:    &Schema{Description: "A pet in the store"},
			b:    &Schema{Description: "A pet in the store"},
			want: true,
		},
		{
			name: "different Description",
			a:    &Schema{Description: "A pet in the store"},
			b:    &Schema{Description: "A user account"},
			want: false,
		},
		// Pattern
		{
			name: "same Pattern",
			a:    &Schema{Pattern: "^[a-z]+$"},
			b:    &Schema{Pattern: "^[a-z]+$"},
			want: true,
		},
		{
			name: "different Pattern",
			a:    &Schema{Pattern: "^[a-z]+$"},
			b:    &Schema{Pattern: "^[A-Z]+$"},
			want: false,
		},
		// Format
		{
			name: "same Format",
			a:    &Schema{Format: "email"},
			b:    &Schema{Format: "email"},
			want: true,
		},
		{
			name: "different Format",
			a:    &Schema{Format: "email"},
			b:    &Schema{Format: "uri"},
			want: false,
		},
		// ContentEncoding
		{
			name: "same ContentEncoding",
			a:    &Schema{ContentEncoding: "base64"},
			b:    &Schema{ContentEncoding: "base64"},
			want: true,
		},
		{
			name: "different ContentEncoding",
			a:    &Schema{ContentEncoding: "base64"},
			b:    &Schema{ContentEncoding: "base32"},
			want: false,
		},
		// ContentMediaType
		{
			name: "same ContentMediaType",
			a:    &Schema{ContentMediaType: "application/json"},
			b:    &Schema{ContentMediaType: "application/json"},
			want: true,
		},
		{
			name: "different ContentMediaType",
			a:    &Schema{ContentMediaType: "application/json"},
			b:    &Schema{ContentMediaType: "text/plain"},
			want: false,
		},
		// CollectionFormat
		{
			name: "same CollectionFormat",
			a:    &Schema{CollectionFormat: "csv"},
			b:    &Schema{CollectionFormat: "csv"},
			want: true,
		},
		{
			name: "different CollectionFormat",
			a:    &Schema{CollectionFormat: "csv"},
			b:    &Schema{CollectionFormat: "pipes"},
			want: false,
		},
		// ID
		{
			name: "same ID",
			a:    &Schema{ID: "https://example.com/schema"},
			b:    &Schema{ID: "https://example.com/schema"},
			want: true,
		},
		{
			name: "different ID",
			a:    &Schema{ID: "https://example.com/schema"},
			b:    &Schema{ID: "https://other.com/schema"},
			want: false,
		},
		// Anchor
		{
			name: "same Anchor",
			a:    &Schema{Anchor: "myAnchor"},
			b:    &Schema{Anchor: "myAnchor"},
			want: true,
		},
		{
			name: "different Anchor",
			a:    &Schema{Anchor: "myAnchor"},
			b:    &Schema{Anchor: "otherAnchor"},
			want: false,
		},
		// DynamicRef
		{
			name: "same DynamicRef",
			a:    &Schema{DynamicRef: "#meta"},
			b:    &Schema{DynamicRef: "#meta"},
			want: true,
		},
		{
			name: "different DynamicRef",
			a:    &Schema{DynamicRef: "#meta"},
			b:    &Schema{DynamicRef: "#other"},
			want: false,
		},
		// DynamicAnchor
		{
			name: "same DynamicAnchor",
			a:    &Schema{DynamicAnchor: "meta"},
			b:    &Schema{DynamicAnchor: "meta"},
			want: true,
		},
		{
			name: "different DynamicAnchor",
			a:    &Schema{DynamicAnchor: "meta"},
			b:    &Schema{DynamicAnchor: "other"},
			want: false,
		},
		// Comment
		{
			name: "same Comment",
			a:    &Schema{Comment: "This is a comment"},
			b:    &Schema{Comment: "This is a comment"},
			want: true,
		},
		{
			name: "different Comment",
			a:    &Schema{Comment: "This is a comment"},
			b:    &Schema{Comment: "Different comment"},
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

// TestSchemaEquals_BooleanFields tests all boolean field comparisons.
func TestSchemaEquals_BooleanFields(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		// ReadOnly
		{
			name: "same ReadOnly true",
			a:    &Schema{ReadOnly: true},
			b:    &Schema{ReadOnly: true},
			want: true,
		},
		{
			name: "same ReadOnly false",
			a:    &Schema{ReadOnly: false},
			b:    &Schema{ReadOnly: false},
			want: true,
		},
		{
			name: "different ReadOnly",
			a:    &Schema{ReadOnly: true},
			b:    &Schema{ReadOnly: false},
			want: false,
		},
		// WriteOnly
		{
			name: "same WriteOnly true",
			a:    &Schema{WriteOnly: true},
			b:    &Schema{WriteOnly: true},
			want: true,
		},
		{
			name: "different WriteOnly",
			a:    &Schema{WriteOnly: true},
			b:    &Schema{WriteOnly: false},
			want: false,
		},
		// Deprecated
		{
			name: "same Deprecated true",
			a:    &Schema{Deprecated: true},
			b:    &Schema{Deprecated: true},
			want: true,
		},
		{
			name: "different Deprecated",
			a:    &Schema{Deprecated: true},
			b:    &Schema{Deprecated: false},
			want: false,
		},
		// Nullable
		{
			name: "same Nullable true",
			a:    &Schema{Nullable: true},
			b:    &Schema{Nullable: true},
			want: true,
		},
		{
			name: "different Nullable",
			a:    &Schema{Nullable: true},
			b:    &Schema{Nullable: false},
			want: false,
		},
		// UniqueItems
		{
			name: "same UniqueItems true",
			a:    &Schema{UniqueItems: true},
			b:    &Schema{UniqueItems: true},
			want: true,
		},
		{
			name: "different UniqueItems",
			a:    &Schema{UniqueItems: true},
			b:    &Schema{UniqueItems: false},
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

// TestSchemaEquals_PolymorphicType tests the Type field which can be string or []string.
func TestSchemaEquals_PolymorphicType(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		{
			name: "both nil Type",
			a:    &Schema{},
			b:    &Schema{},
			want: true,
		},
		{
			name: "same string type",
			a:    &Schema{Type: "string"},
			b:    &Schema{Type: "string"},
			want: true,
		},
		{
			name: "different string type",
			a:    &Schema{Type: "string"},
			b:    &Schema{Type: "integer"},
			want: false,
		},
		{
			name: "same []string type",
			a:    &Schema{Type: []string{"string", "null"}},
			b:    &Schema{Type: []string{"string", "null"}},
			want: true,
		},
		{
			name: "different []string type - different values",
			a:    &Schema{Type: []string{"string", "null"}},
			b:    &Schema{Type: []string{"integer", "null"}},
			want: false,
		},
		{
			name: "different []string type - different order",
			a:    &Schema{Type: []string{"string", "null"}},
			b:    &Schema{Type: []string{"null", "string"}},
			want: false,
		},
		{
			name: "string vs []string - type mismatch",
			a:    &Schema{Type: "string"},
			b:    &Schema{Type: []string{"string"}},
			want: false,
		},
		{
			name: "[]string vs string - type mismatch",
			a:    &Schema{Type: []string{"string"}},
			b:    &Schema{Type: "string"},
			want: false,
		},
		{
			name: "same empty []string",
			a:    &Schema{Type: []string{}},
			b:    &Schema{Type: []string{}},
			want: true,
		},
		{
			name: "nil Type vs string Type",
			a:    &Schema{Type: nil},
			b:    &Schema{Type: "string"},
			want: false,
		},
		{
			name: "string Type vs nil Type",
			a:    &Schema{Type: "string"},
			b:    &Schema{Type: nil},
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

// TestSchemaEquals_PolymorphicBounds tests ExclusiveMinimum and ExclusiveMaximum.
// These can be bool (OAS 3.0) or number (OAS 3.1+).
func TestSchemaEquals_PolymorphicBounds(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		// ExclusiveMinimum as bool
		{
			name: "ExclusiveMinimum both bool true",
			a:    &Schema{ExclusiveMinimum: true},
			b:    &Schema{ExclusiveMinimum: true},
			want: true,
		},
		{
			name: "ExclusiveMinimum both bool false",
			a:    &Schema{ExclusiveMinimum: false},
			b:    &Schema{ExclusiveMinimum: false},
			want: true,
		},
		{
			name: "ExclusiveMinimum bool true vs false",
			a:    &Schema{ExclusiveMinimum: true},
			b:    &Schema{ExclusiveMinimum: false},
			want: false,
		},
		// ExclusiveMinimum as float64
		{
			name: "ExclusiveMinimum same float64",
			a:    &Schema{ExclusiveMinimum: float64(5.0)},
			b:    &Schema{ExclusiveMinimum: float64(5.0)},
			want: true,
		},
		{
			name: "ExclusiveMinimum different float64",
			a:    &Schema{ExclusiveMinimum: float64(5.0)},
			b:    &Schema{ExclusiveMinimum: float64(10.0)},
			want: false,
		},
		// ExclusiveMinimum type mismatch
		{
			name: "ExclusiveMinimum bool vs float64",
			a:    &Schema{ExclusiveMinimum: true},
			b:    &Schema{ExclusiveMinimum: float64(1.0)},
			want: false,
		},
		{
			name: "ExclusiveMinimum float64 vs bool",
			a:    &Schema{ExclusiveMinimum: float64(1.0)},
			b:    &Schema{ExclusiveMinimum: true},
			want: false,
		},
		// ExclusiveMaximum as bool
		{
			name: "ExclusiveMaximum both bool true",
			a:    &Schema{ExclusiveMaximum: true},
			b:    &Schema{ExclusiveMaximum: true},
			want: true,
		},
		{
			name: "ExclusiveMaximum both bool false",
			a:    &Schema{ExclusiveMaximum: false},
			b:    &Schema{ExclusiveMaximum: false},
			want: true,
		},
		{
			name: "ExclusiveMaximum bool true vs false",
			a:    &Schema{ExclusiveMaximum: true},
			b:    &Schema{ExclusiveMaximum: false},
			want: false,
		},
		// ExclusiveMaximum as float64
		{
			name: "ExclusiveMaximum same float64",
			a:    &Schema{ExclusiveMaximum: float64(100.0)},
			b:    &Schema{ExclusiveMaximum: float64(100.0)},
			want: true,
		},
		{
			name: "ExclusiveMaximum different float64",
			a:    &Schema{ExclusiveMaximum: float64(100.0)},
			b:    &Schema{ExclusiveMaximum: float64(200.0)},
			want: false,
		},
		// ExclusiveMaximum type mismatch
		{
			name: "ExclusiveMaximum bool vs float64",
			a:    &Schema{ExclusiveMaximum: true},
			b:    &Schema{ExclusiveMaximum: float64(1.0)},
			want: false,
		},
		// nil handling
		{
			name: "ExclusiveMinimum both nil",
			a:    &Schema{ExclusiveMinimum: nil},
			b:    &Schema{ExclusiveMinimum: nil},
			want: true,
		},
		{
			name: "ExclusiveMinimum nil vs value",
			a:    &Schema{ExclusiveMinimum: nil},
			b:    &Schema{ExclusiveMinimum: true},
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

// TestSchemaEquals_PointerFields tests pointer fields like Maximum, Minimum, etc.
func TestSchemaEquals_PointerFields(t *testing.T) {
	tests := []struct {
		name string
		a    *Schema
		b    *Schema
		want bool
	}{
		// Maximum
		{
			name: "Maximum both nil",
			a:    &Schema{Maximum: nil},
			b:    &Schema{Maximum: nil},
			want: true,
		},
		{
			name: "Maximum one nil one set",
			a:    &Schema{Maximum: nil},
			b:    &Schema{Maximum: ptr(100.0)},
			want: false,
		},
		{
			name: "Maximum same values",
			a:    &Schema{Maximum: ptr(100.0)},
			b:    &Schema{Maximum: ptr(100.0)},
			want: true,
		},
		{
			name: "Maximum different values",
			a:    &Schema{Maximum: ptr(100.0)},
			b:    &Schema{Maximum: ptr(200.0)},
			want: false,
		},
		// Minimum
		{
			name: "Minimum both nil",
			a:    &Schema{Minimum: nil},
			b:    &Schema{Minimum: nil},
			want: true,
		},
		{
			name: "Minimum one nil one set",
			a:    &Schema{Minimum: ptr(0.0)},
			b:    &Schema{Minimum: nil},
			want: false,
		},
		{
			name: "Minimum same values",
			a:    &Schema{Minimum: ptr(0.0)},
			b:    &Schema{Minimum: ptr(0.0)},
			want: true,
		},
		{
			name: "Minimum different values",
			a:    &Schema{Minimum: ptr(0.0)},
			b:    &Schema{Minimum: ptr(1.0)},
			want: false,
		},
		// MultipleOf
		{
			name: "MultipleOf both nil",
			a:    &Schema{MultipleOf: nil},
			b:    &Schema{MultipleOf: nil},
			want: true,
		},
		{
			name: "MultipleOf same values",
			a:    &Schema{MultipleOf: ptr(2.0)},
			b:    &Schema{MultipleOf: ptr(2.0)},
			want: true,
		},
		{
			name: "MultipleOf different values",
			a:    &Schema{MultipleOf: ptr(2.0)},
			b:    &Schema{MultipleOf: ptr(5.0)},
			want: false,
		},
		// MaxLength
		{
			name: "MaxLength both nil",
			a:    &Schema{MaxLength: nil},
			b:    &Schema{MaxLength: nil},
			want: true,
		},
		{
			name: "MaxLength one nil one set",
			a:    &Schema{MaxLength: intPtr(100)},
			b:    &Schema{MaxLength: nil},
			want: false,
		},
		{
			name: "MaxLength same values",
			a:    &Schema{MaxLength: intPtr(100)},
			b:    &Schema{MaxLength: intPtr(100)},
			want: true,
		},
		{
			name: "MaxLength different values",
			a:    &Schema{MaxLength: intPtr(100)},
			b:    &Schema{MaxLength: intPtr(200)},
			want: false,
		},
		// MinLength
		{
			name: "MinLength same values",
			a:    &Schema{MinLength: intPtr(1)},
			b:    &Schema{MinLength: intPtr(1)},
			want: true,
		},
		{
			name: "MinLength different values",
			a:    &Schema{MinLength: intPtr(1)},
			b:    &Schema{MinLength: intPtr(5)},
			want: false,
		},
		// MaxItems
		{
			name: "MaxItems same values",
			a:    &Schema{MaxItems: intPtr(10)},
			b:    &Schema{MaxItems: intPtr(10)},
			want: true,
		},
		{
			name: "MaxItems different values",
			a:    &Schema{MaxItems: intPtr(10)},
			b:    &Schema{MaxItems: intPtr(20)},
			want: false,
		},
		// MinItems
		{
			name: "MinItems same values",
			a:    &Schema{MinItems: intPtr(0)},
			b:    &Schema{MinItems: intPtr(0)},
			want: true,
		},
		{
			name: "MinItems different values",
			a:    &Schema{MinItems: intPtr(0)},
			b:    &Schema{MinItems: intPtr(1)},
			want: false,
		},
		// MaxProperties
		{
			name: "MaxProperties same values",
			a:    &Schema{MaxProperties: intPtr(50)},
			b:    &Schema{MaxProperties: intPtr(50)},
			want: true,
		},
		{
			name: "MaxProperties different values",
			a:    &Schema{MaxProperties: intPtr(50)},
			b:    &Schema{MaxProperties: intPtr(100)},
			want: false,
		},
		// MinProperties
		{
			name: "MinProperties same values",
			a:    &Schema{MinProperties: intPtr(1)},
			b:    &Schema{MinProperties: intPtr(1)},
			want: true,
		},
		{
			name: "MinProperties different values",
			a:    &Schema{MinProperties: intPtr(1)},
			b:    &Schema{MinProperties: intPtr(2)},
			want: false,
		},
		// MaxContains
		{
			name: "MaxContains same values",
			a:    &Schema{MaxContains: intPtr(5)},
			b:    &Schema{MaxContains: intPtr(5)},
			want: true,
		},
		{
			name: "MaxContains different values",
			a:    &Schema{MaxContains: intPtr(5)},
			b:    &Schema{MaxContains: intPtr(10)},
			want: false,
		},
		// MinContains
		{
			name: "MinContains same values",
			a:    &Schema{MinContains: intPtr(1)},
			b:    &Schema{MinContains: intPtr(1)},
			want: true,
		},
		{
			name: "MinContains different values",
			a:    &Schema{MinContains: intPtr(1)},
			b:    &Schema{MinContains: intPtr(2)},
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
