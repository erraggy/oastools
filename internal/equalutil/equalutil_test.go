package equalutil_test

import (
	"math"
	"testing"

	"github.com/erraggy/oastools/internal/equalutil"
	"github.com/stretchr/testify/assert"
)

func TestEqualPtr_float64(t *testing.T) {
	tests := []struct {
		name string
		a    *float64
		b    *float64
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
			b:    new(3.14),
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    new(3.14),
			b:    nil,
			want: false,
		},
		{
			name: "both same value",
			a:    new(3.14),
			b:    new(3.14),
			want: true,
		},
		{
			name: "both different values",
			a:    new(3.14),
			b:    new(2.71),
			want: false,
		},
		{
			name: "both zero",
			a:    new(0.0),
			b:    new(0.0),
			want: true,
		},
		{
			name: "negative values equal",
			a:    new(-1.5),
			b:    new(-1.5),
			want: true,
		},
		{
			name: "both NaN",
			a:    new(math.NaN()),
			b:    new(math.NaN()),
			want: false, // NaN != NaN per IEEE 754
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalutil.EqualPtr(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualPtr_int(t *testing.T) {
	tests := []struct {
		name string
		a    *int
		b    *int
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
			b:    new(42),
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    new(42),
			b:    nil,
			want: false,
		},
		{
			name: "both same value",
			a:    new(42),
			b:    new(42),
			want: true,
		},
		{
			name: "both different values",
			a:    new(42),
			b:    new(100),
			want: false,
		},
		{
			name: "both zero",
			a:    new(0),
			b:    new(0),
			want: true,
		},
		{
			name: "negative values equal",
			a:    new(-5),
			b:    new(-5),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalutil.EqualPtr(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualPtr_bool(t *testing.T) {
	tests := []struct {
		name string
		a    *bool
		b    *bool
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil true",
			a:    nil,
			b:    new(true),
			want: false,
		},
		{
			name: "a nil, b non-nil false",
			a:    nil,
			b:    new(false),
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    new(true),
			b:    nil,
			want: false,
		},
		{
			name: "both true",
			a:    new(true),
			b:    new(true),
			want: true,
		},
		{
			name: "both false",
			a:    new(false),
			b:    new(false),
			want: true,
		},
		{
			name: "true vs false",
			a:    new(true),
			b:    new(false),
			want: false,
		},
		{
			name: "false vs true",
			a:    new(false),
			b:    new(true),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalutil.EqualPtr(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
