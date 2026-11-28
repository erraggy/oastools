package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestServerVariableUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantDefault    string
		wantDefaultSet bool
		wantHasDefault bool
		wantEnum       []string
	}{
		{
			name: "standard default",
			input: `
default: production
enum: [production, staging]
description: Environment
`,
			wantDefault:    "production",
			wantDefaultSet: true,
			wantHasDefault: true,
			wantEnum:       []string{"production", "staging"},
		},
		{
			name: "empty string default",
			input: `
default: ""
description: Optional prefix
`,
			wantDefault:    "",
			wantDefaultSet: true,
			wantHasDefault: true,
		},
		{
			name: "empty string default with enum",
			input: `
default: ""
enum: ["", /v2]
`,
			wantDefault:    "",
			wantDefaultSet: true,
			wantHasDefault: true,
			wantEnum:       []string{"", "/v2"},
		},
		{
			name: "missing default",
			input: `
description: Missing default field
enum: [a, b]
`,
			wantDefault:    "",
			wantDefaultSet: false,
			wantHasDefault: false,
			wantEnum:       []string{"a", "b"},
		},
		{
			name: "default through anchor",
			input: `
anchor: &varDefault
  default: ""
  description: From anchor
var:
  *varDefault
`,
			wantDefault:    "",
			wantDefaultSet: true,
			wantHasDefault: true,
		},
		{
			name: "default through merge key",
			input: `
anchor: &baseVar
  default: ""
var:
  <<: *baseVar
  description: Extended
`,
			wantDefault:    "",
			wantDefaultSet: true,
			wantHasDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sv ServerVariable
			if tt.name == "default through anchor" || tt.name == "default through merge key" {
				var root map[string]ServerVariable
				err := yaml.Unmarshal([]byte(tt.input), &root)
				require.NoError(t, err)
				sv = root["var"]
			} else {
				err := yaml.Unmarshal([]byte(tt.input), &sv)
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantDefault, sv.Default)
			assert.Equal(t, tt.wantDefaultSet, sv.DefaultSet)
			assert.Equal(t, tt.wantHasDefault, sv.HasDefault())
			if tt.wantEnum != nil {
				assert.Equal(t, tt.wantEnum, sv.Enum)
			}
		})
	}
}
