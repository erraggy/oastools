package joiner

import (
	"reflect"
	"testing"

	"github.com/erraggy/oastools/parser"
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
				if copied != nil {
					t.Error("Expected nil copy for nil input")
				}
				return
			}

			// Verify values are equal
			if !reflect.DeepEqual(copied, tt.info) {
				t.Error("Copied info does not match original")
			}

			// Verify it's a different pointer
			if copied == tt.info {
				t.Error("Copy should be a different pointer")
			}

			// Verify nested pointers are also copied
			if tt.info.Contact != nil && copied.Contact == tt.info.Contact {
				t.Error("Contact should be copied to a different pointer")
			}
			if tt.info.License != nil && copied.License == tt.info.License {
				t.Error("License should be copied to a different pointer")
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
				if copied != nil {
					t.Error("Expected nil copy for nil input")
				}
				return
			}

			// Verify length matches
			if len(copied) != len(tt.servers) {
				t.Errorf("Length mismatch: got %d, want %d", len(copied), len(tt.servers))
			}

			// Verify each server is deeply copied
			for i, server := range tt.servers {
				if server == nil {
					if copied[i] != nil {
						t.Errorf("Server %d should be nil", i)
					}
					continue
				}

				if copied[i] == server {
					t.Errorf("Server %d should be a different pointer", i)
				}

				if !reflect.DeepEqual(copied[i], server) {
					t.Errorf("Server %d content mismatch", i)
				}

				// Verify variables are deeply copied
				if server.Variables != nil {
					// Maps can't be compared directly, check they have same content
					if len(copied[i].Variables) != len(server.Variables) {
						t.Errorf("Server %d variables length mismatch", i)
					}

					for k, v := range server.Variables {
						copiedVar := copied[i].Variables[k]

						// Verify enum slice is copied
						if len(v.Enum) > 0 {
							if &copiedVar.Enum[0] == &v.Enum[0] {
								t.Errorf("Server %d variable %s enum should be copied", i, k)
							}
						}

						// Verify extra map is copied (check by content since maps can't be compared)
						if len(v.Extra) > 0 {
							if len(copiedVar.Extra) != len(v.Extra) {
								t.Errorf("Server %d variable %s extra length mismatch", i, k)
							}
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
				if copied != nil {
					t.Error("Expected nil copy for nil input")
				}
				return
			}

			if len(copied) != len(tt.reqs) {
				t.Errorf("Length mismatch: got %d, want %d", len(copied), len(tt.reqs))
			}

			for i, req := range tt.reqs {
				if len(copied[i]) != len(req) {
					t.Errorf("Requirement %d size mismatch", i)
				}

				for scheme, scopes := range req {
					copiedScopes, ok := copied[i][scheme]
					if !ok {
						t.Errorf("Requirement %d missing scheme %s", i, scheme)
						continue
					}

					if !reflect.DeepEqual(copiedScopes, scopes) {
						t.Errorf("Requirement %d scheme %s scopes mismatch", i, scheme)
					}

					// Verify scopes slice is copied (not same pointer)
					if len(scopes) > 0 && len(copiedScopes) > 0 {
						if &copiedScopes[0] == &scopes[0] {
							t.Errorf("Requirement %d scheme %s scopes should be copied", i, scheme)
						}
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

			if len(result) != tt.wantLen {
				t.Errorf("Length mismatch: got %d, want %d", len(result), tt.wantLen)
			}

			// Verify tag names match expected order
			for i, wantName := range tt.wantNames {
				if i >= len(result) {
					break
				}
				gotName := ""
				if result[i] != nil {
					gotName = result[i].Name
				}
				if gotName != wantName {
					t.Errorf("Tag %d name mismatch: got %s, want %s", i, gotName, wantName)
				}
			}
		})
	}
}
