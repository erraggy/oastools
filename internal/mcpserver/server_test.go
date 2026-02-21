package mcpserver

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaginate(t *testing.T) {
	items := []int{0, 1, 2, 3, 4}

	tests := []struct {
		name   string
		items  []int
		offset int
		limit  int
		want   []int
	}{
		{
			name:   "default limit returns all when under 100",
			items:  items,
			offset: 0,
			limit:  0,
			want:   []int{0, 1, 2, 3, 4},
		},
		{
			name:   "explicit limit",
			items:  items,
			offset: 0,
			limit:  2,
			want:   []int{0, 1},
		},
		{
			name:   "offset only",
			items:  items,
			offset: 2,
			limit:  0,
			want:   []int{2, 3, 4},
		},
		{
			name:   "offset and limit",
			items:  items,
			offset: 1,
			limit:  2,
			want:   []int{1, 2},
		},
		{
			name:   "offset at end",
			items:  items,
			offset: 4,
			limit:  2,
			want:   []int{4},
		},
		{
			name:   "offset beyond end",
			items:  items,
			offset: 5,
			limit:  2,
			want:   nil,
		},
		{
			name:   "negative offset",
			items:  items,
			offset: -1,
			limit:  2,
			want:   nil,
		},
		{
			name:   "limit exceeds remaining",
			items:  items,
			offset: 3,
			limit:  10,
			want:   []int{3, 4},
		},
		{
			name:   "nil slice",
			items:  nil,
			offset: 0,
			limit:  2,
			want:   nil,
		},
		{
			name:   "empty slice",
			items:  []int{},
			offset: 0,
			limit:  2,
			want:   nil,
		},
		{
			name:   "negative limit treated as default",
			items:  items,
			offset: 0,
			limit:  -1,
			want:   []int{0, 1, 2, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := paginate(tt.items, tt.offset, tt.limit)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetailLimit(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"zero returns default", 0, 25},
		{"negative returns default", -1, 25},
		{"explicit 50", 50, 50},
		{"explicit 10", 10, 10},
		{"explicit 200", 200, 200},
		{"boundary 1", 1, 1},
		{"max int returns itself", math.MaxInt, math.MaxInt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, detailLimit(tt.input))
		})
	}
}

func TestPaginate_OverflowLimit(t *testing.T) {
	items := []int{0, 1, 2}
	got := paginate(items, 1, math.MaxInt)
	assert.Equal(t, []int{1, 2}, got)
}

func TestPaginate_DefaultLimit(t *testing.T) {
	items := make([]int, 150)
	for i := range items {
		items[i] = i
	}
	got := paginate(items, 0, 0)
	assert.Len(t, got, 100, "default limit should cap at 100 items")
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error returns empty string",
			err:  nil,
			want: "",
		},
		{
			name: "strips absolute path",
			err:  fmt.Errorf("failed to open /home/user/secret/api.yaml: no such file"),
			want: "failed to open <path>: no such file",
		},
		{
			name: "preserves non-path content",
			err:  fmt.Errorf("invalid JSON at line 5"),
			want: "invalid JSON at line 5",
		},
		{
			name: "strips multiple paths",
			err:  fmt.Errorf("diff /tmp/a.yaml vs /tmp/b.yaml failed"),
			want: "diff <path> vs <path> failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPaginate_MaxLimitCap(t *testing.T) {
	// Generate items exceeding MaxLimit.
	items := make([]int, 1500)
	for i := range items {
		items[i] = i
	}
	// Request a limit higher than MaxLimit (default 1000).
	got := paginate(items, 0, 1500)
	assert.Len(t, got, cfg.MaxLimit, "limit should be capped at MaxLimit")
}
