package pathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathParamRegex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		want    []string // expected captured group values
	}{
		{
			name:    "single parameter",
			input:   "/pets/{petId}",
			wantLen: 1,
			want:    []string{"petId"},
		},
		{
			name:    "multiple parameters",
			input:   "/pets/{petId}/owners/{ownerId}",
			wantLen: 2,
			want:    []string{"petId", "ownerId"},
		},
		{
			name:    "no parameters",
			input:   "/pets/all",
			wantLen: 0,
		},
		{
			name:    "parameter at start",
			input:   "{version}/pets",
			wantLen: 1,
			want:    []string{"version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := PathParamRegex.FindAllStringSubmatch(tt.input, -1)
			require.Len(t, matches, tt.wantLen)
			for i, match := range matches {
				require.True(t, len(match) >= 2, "match %d has no capture group", i)
				assert.Equal(t, tt.want[i], match[1], "match[%d]", i)
			}
		})
	}
}
