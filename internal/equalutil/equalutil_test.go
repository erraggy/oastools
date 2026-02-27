package equalutil_test

import (
	"math"
	"testing"

	"github.com/erraggy/oastools/internal/equalutil"
	"github.com/erraggy/oastools/internal/testutil"
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
			b:    testutil.Ptr(3.14),
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    testutil.Ptr(3.14),
			b:    nil,
			want: false,
		},
		{
			name: "both same value",
			a:    testutil.Ptr(3.14),
			b:    testutil.Ptr(3.14),
			want: true,
		},
		{
			name: "both different values",
			a:    testutil.Ptr(3.14),
			b:    testutil.Ptr(2.71),
			want: false,
		},
		{
			name: "both zero",
			a:    testutil.Ptr(0.0),
			b:    testutil.Ptr(0.0),
			want: true,
		},
		{
			name: "negative values equal",
			a:    testutil.Ptr(-1.5),
			b:    testutil.Ptr(-1.5),
			want: true,
		},
		{
			name: "both NaN",
			a:    testutil.Ptr(math.NaN()),
			b:    testutil.Ptr(math.NaN()),
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
			b:    testutil.Ptr(42),
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    testutil.Ptr(42),
			b:    nil,
			want: false,
		},
		{
			name: "both same value",
			a:    testutil.Ptr(42),
			b:    testutil.Ptr(42),
			want: true,
		},
		{
			name: "both different values",
			a:    testutil.Ptr(42),
			b:    testutil.Ptr(100),
			want: false,
		},
		{
			name: "both zero",
			a:    testutil.Ptr(0),
			b:    testutil.Ptr(0),
			want: true,
		},
		{
			name: "negative values equal",
			a:    testutil.Ptr(-5),
			b:    testutil.Ptr(-5),
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
			b:    testutil.Ptr(true),
			want: false,
		},
		{
			name: "a nil, b non-nil false",
			a:    nil,
			b:    testutil.Ptr(false),
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    testutil.Ptr(true),
			b:    nil,
			want: false,
		},
		{
			name: "both true",
			a:    testutil.Ptr(true),
			b:    testutil.Ptr(true),
			want: true,
		},
		{
			name: "both false",
			a:    testutil.Ptr(false),
			b:    testutil.Ptr(false),
			want: true,
		},
		{
			name: "true vs false",
			a:    testutil.Ptr(true),
			b:    testutil.Ptr(false),
			want: false,
		},
		{
			name: "false vs true",
			a:    testutil.Ptr(false),
			b:    testutil.Ptr(true),
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
