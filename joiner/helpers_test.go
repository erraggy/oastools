package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyInfo(t *testing.T) {
	tests := []struct {
		name string
		info *parser.Info
	}{
		{
			name: "nil info",
			info: nil,
		},
		{
			name: "basic info",
			info: &parser.Info{
				Title:       "Test API",
				Version:     "1.0.0",
				Description: "Test description",
			},
		},
		{
			name: "info with contact",
			info: &parser.Info{
				Title:   "Test API",
				Version: "1.0.0",
				Contact: &parser.Contact{
					Name:  "API Team",
					Email: "api@example.com",
					URL:   "https://example.com",
				},
			},
		},
		{
			name: "info with license",
			info: &parser.Info{
				Title:   "Test API",
				Version: "1.0.0",
				License: &parser.License{
					Name: "MIT",
					URL:  "https://opensource.org/licenses/MIT",
				},
			},
		},
		{
			name: "info with contact and license",
			info: &parser.Info{
				Title:   "Test API",
				Version: "1.0.0",
				Contact: &parser.Contact{
					Name:  "API Team",
					Email: "api@example.com",
				},
				License: &parser.License{
					Name: "Apache 2.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copied := copyInfo(tt.info)

			if tt.info == nil {
				assert.Nil(t, copied, "Expected nil copy for nil input")
				return
			}

			// Verify values are equal
			assert.Equal(t, tt.info, copied, "Copied info does not match original")

			// Verify it's a different pointer
			assert.False(t, copied == tt.info, "Copy should be a different pointer")

			// Verify nested pointers are also copied
			if tt.info.Contact != nil {
				assert.False(t, copied.Contact == tt.info.Contact, "Contact should be copied to a different pointer")
			}
			if tt.info.License != nil {
				assert.False(t, copied.License == tt.info.License, "License should be copied to a different pointer")
			}
		})
	}
}

func TestCopyServers(t *testing.T) {
	tests := []struct {
		name    string
		servers []*parser.Server
	}{
		{
			name:    "nil servers",
			servers: nil,
		},
		{
			name:    "empty servers",
			servers: []*parser.Server{},
		},
		{
			name: "single server without variables",
			servers: []*parser.Server{
				{
					URL:         "https://api.example.com",
					Description: "Production server",
				},
			},
		},
		{
			name: "server with variables",
			servers: []*parser.Server{
				{
					URL:         "https://{environment}.example.com",
					Description: "Configurable server",
					Variables: map[string]parser.ServerVariable{
						"environment": {
							Default:     "api",
							Description: "Environment name",
							Enum:        []string{"api", "staging", "dev"},
							Extra: map[string]any{
								"x-custom": "value",
							},
						},
					},
				},
			},
		},
		{
			name: "multiple servers with mixed config",
			servers: []*parser.Server{
				{
					URL: "https://api.example.com",
				},
				{
					URL: "https://{env}.example.com",
					Variables: map[string]parser.ServerVariable{
						"env": {Default: "prod"},
					},
				},
			},
		},
		{
			name: "servers with nil element",
			servers: []*parser.Server{
				{URL: "https://api.example.com"},
				nil,
				{URL: "https://api2.example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copied := copyServers(tt.servers)

			if tt.servers == nil {
				assert.Nil(t, copied, "Expected nil copy for nil input")
				return
			}

			// Verify length matches
			require.Equal(t, len(tt.servers), len(copied), "Length mismatch")

			// Verify each server is deeply copied
			for i, server := range tt.servers {
				if server == nil {
					assert.Nil(t, copied[i], "Server %d should be nil", i)
					continue
				}

				assert.False(t, copied[i] == server, "Server %d should be a different pointer", i)

				assert.Equal(t, server, copied[i], "Server %d content mismatch", i)

				// Verify variables are deeply copied
				if server.Variables != nil {
					// Maps can't be compared directly, check they have same content
					assert.Equal(t, len(server.Variables), len(copied[i].Variables), "Server %d variables length mismatch", i)

					for k, v := range server.Variables {
						copiedVar := copied[i].Variables[k]

						// Verify enum slice is copied
						if len(v.Enum) > 0 {
							assert.False(t, &copiedVar.Enum[0] == &v.Enum[0], "Server %d variable %s enum should be copied", i, k)
						}

						// Verify extra map is copied (check by content since maps can't be compared)
						if len(v.Extra) > 0 {
							assert.Equal(t, len(v.Extra), len(copiedVar.Extra), "Server %d variable %s extra length mismatch", i, k)
						}
					}
				}
			}
		})
	}
}

func TestCopySecurityRequirements(t *testing.T) {
	tests := []struct {
		name string
		reqs []parser.SecurityRequirement
	}{
		{
			name: "nil requirements",
			reqs: nil,
		},
		{
			name: "empty requirements",
			reqs: []parser.SecurityRequirement{},
		},
		{
			name: "single requirement without scopes",
			reqs: []parser.SecurityRequirement{
				{"api_key": []string{}},
			},
		},
		{
			name: "requirement with scopes",
			reqs: []parser.SecurityRequirement{
				{"oauth2": []string{"read:users", "write:users"}},
			},
		},
		{
			name: "multiple requirements",
			reqs: []parser.SecurityRequirement{
				{"api_key": []string{}},
				{"oauth2": []string{"read:users"}},
			},
		},
		{
			name: "requirement with multiple schemes",
			reqs: []parser.SecurityRequirement{
				{
					"api_key": []string{},
					"oauth2":  []string{"read:users", "write:users"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copied := copySecurityRequirements(tt.reqs)

			if tt.reqs == nil {
				assert.Nil(t, copied, "Expected nil copy for nil input")
				return
			}

			require.Equal(t, len(tt.reqs), len(copied), "Length mismatch")

			for i, req := range tt.reqs {
				assert.Equal(t, len(req), len(copied[i]), "Requirement %d size mismatch", i)

				for scheme, scopes := range req {
					copiedScopes, ok := copied[i][scheme]
					require.True(t, ok, "Requirement %d missing scheme %s", i, scheme)

					assert.Equal(t, scopes, copiedScopes, "Requirement %d scheme %s scopes mismatch", i, scheme)

					// Verify scopes slice is copied (not same pointer)
					if len(scopes) > 0 && len(copiedScopes) > 0 {
						assert.False(t, &copiedScopes[0] == &scopes[0], "Requirement %d scheme %s scopes should be copied", i, scheme)
					}
				}
			}
		})
	}
}

func TestMergeTags(t *testing.T) {
	tests := []struct {
		name            string
		existing        []*parser.Tag
		new             []*parser.Tag
		deduplicateTags bool
		wantLen         int
		wantNames       []string
	}{
		{
			name: "no deduplication - append all",
			existing: []*parser.Tag{
				{Name: "users"},
				{Name: "products"},
			},
			new: []*parser.Tag{
				{Name: "users"}, // duplicate
				{Name: "orders"},
			},
			deduplicateTags: false,
			wantLen:         4,
			wantNames:       []string{"users", "products", "users", "orders"},
		},
		{
			name: "with deduplication - skip duplicates",
			existing: []*parser.Tag{
				{Name: "users"},
				{Name: "products"},
			},
			new: []*parser.Tag{
				{Name: "users"}, // duplicate - skip
				{Name: "orders"},
			},
			deduplicateTags: true,
			wantLen:         3,
			wantNames:       []string{"users", "products", "orders"},
		},
		{
			name:            "deduplication with empty existing",
			existing:        []*parser.Tag{},
			new:             []*parser.Tag{{Name: "users"}},
			deduplicateTags: true,
			wantLen:         1,
			wantNames:       []string{"users"},
		},
		{
			name:            "deduplication with all new duplicates",
			existing:        []*parser.Tag{{Name: "users"}},
			new:             []*parser.Tag{{Name: "users"}},
			deduplicateTags: true,
			wantLen:         1,
			wantNames:       []string{"users"},
		},
		{
			name: "deduplication with nil tags",
			existing: []*parser.Tag{
				{Name: "users"},
				nil,
			},
			new: []*parser.Tag{
				nil,
				{Name: "products"},
			},
			deduplicateTags: true,
			wantLen:         3,
			wantNames:       []string{"users", "", "products"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &Joiner{
				config: JoinerConfig{
					DeduplicateTags: tt.deduplicateTags,
				},
			}

			result := j.mergeTags(tt.existing, tt.new)

			assert.Equal(t, tt.wantLen, len(result), "Length mismatch")

			// Verify tag names match expected order
			for i, wantName := range tt.wantNames {
				if i >= len(result) {
					break
				}
				gotName := ""
				if result[i] != nil {
					gotName = result[i].Name
				}
				assert.Equal(t, wantName, gotName, "Tag %d name mismatch", i)
			}
		})
	}
}
