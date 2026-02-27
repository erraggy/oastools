package jsonpath

import (
	"bytes"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// suppressJSONPathLogger suppresses jsonpathLogger warnings for the duration of the test.
func suppressJSONPathLogger(t *testing.T) {
	t.Helper()
	old := jsonpathLogger
	jsonpathLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
	t.Cleanup(func() { jsonpathLogger = old })
}

// TestParse tests the JSONPath parser.
func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		segLen  int // Expected number of segments
	}{
		// Valid expressions
		{name: "root only", input: "$", wantErr: false, segLen: 1},
		{name: "simple child", input: "$.info", wantErr: false, segLen: 2},
		{name: "nested children", input: "$.info.title", wantErr: false, segLen: 3},
		{name: "bracket notation single quote", input: "$['info']", wantErr: false, segLen: 2},
		{name: "bracket notation double quote", input: "$[\"info\"]", wantErr: false, segLen: 2},
		{name: "path with slash", input: "$.paths['/users']", wantErr: false, segLen: 3},
		{name: "path with slash and method", input: "$.paths['/users'].get", wantErr: false, segLen: 4},
		{name: "wildcard", input: "$.paths.*", wantErr: false, segLen: 3},
		{name: "chained wildcards", input: "$.paths.*.*", wantErr: false, segLen: 4},
		{name: "wildcard then child", input: "$.paths.*.get", wantErr: false, segLen: 4},
		{name: "array index", input: "$.servers[0]", wantErr: false, segLen: 3},
		{name: "negative index", input: "$.servers[-1]", wantErr: false, segLen: 3},
		{name: "bracket wildcard", input: "$[*]", wantErr: false, segLen: 2},
		{name: "filter simple", input: "$.paths.*[?@.x-internal==true]", wantErr: false, segLen: 4},
		{name: "filter with string", input: "$.paths.*.get.parameters[?@.name=='filter']", wantErr: false, segLen: 6},
		{name: "filter with parens", input: "$.paths.*[?(@.x-internal==true)]", wantErr: false, segLen: 4},
		{name: "filter not equal", input: "$.components.schemas.*[?@.type!='object']", wantErr: false, segLen: 5},
		{name: "filter less than", input: "$.items[?@.count<10]", wantErr: false, segLen: 3},
		{name: "filter greater equal", input: "$.items[?@.priority>=5]", wantErr: false, segLen: 3},
		{name: "components schemas", input: "$.components.schemas.Pet", wantErr: false, segLen: 4},
		{name: "extension field", input: "$.info.x-custom-field", wantErr: false, segLen: 3},

		// Compound filters
		{name: "filter with AND", input: "$.items[?@.active==true && @.count>0]", wantErr: false, segLen: 3},
		{name: "filter with OR", input: "$.items[?@.status=='pending' || @.status=='active']", wantErr: false, segLen: 3},
		{name: "filter with multiple AND", input: "$.items[?@.a==1 && @.b==2 && @.c==3]", wantErr: false, segLen: 3},
		{name: "filter with parens AND", input: "$.items[?(@.x==1) && (@.y==2)]", wantErr: false, segLen: 3},

		// Recursive descent
		{name: "recursive descent field", input: "$..name", wantErr: false, segLen: 2},
		{name: "recursive descent wildcard", input: "$..*", wantErr: false, segLen: 2},
		{name: "recursive descent bracket", input: "$..[0]", wantErr: false, segLen: 2},
		{name: "recursive after child", input: "$.info..title", wantErr: false, segLen: 3},
		{name: "recursive with filter", input: "$..[?@.deprecated==true]", wantErr: false, segLen: 2},

		// Invalid expressions
		{name: "empty string", input: "", wantErr: true},
		{name: "no dollar", input: "info", wantErr: true},
		{name: "dot at start", input: ".info", wantErr: true},
		{name: "trailing dot", input: "$.info.", wantErr: true},
		{name: "unclosed bracket", input: "$['info", wantErr: true},
		{name: "unclosed filter", input: "$.paths[?@.foo", wantErr: true},
		{name: "invalid filter no field", input: "$.paths[?==true]", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := Parse(tt.input)

			if tt.wantErr {
				assert.Error(t, err, "Parse(%q) expected error, got nil", tt.input)
				return
			}

			require.NoError(t, err, "Parse(%q) unexpected error", tt.input)
			require.NotNil(t, path, "Parse(%q) returned nil path without error", tt.input)
			assert.Len(t, path.segments, tt.segLen, "Parse(%q) segment count", tt.input)
			assert.Equal(t, tt.input, path.String(), "Path.String()")
		})
	}
}

// TestGet tests the JSONPath Get method.
func TestGet(t *testing.T) {
	doc := map[string]any{
		"info": map[string]any{
			"title":   "Test API",
			"version": "1.0.0",
		},
		"paths": map[string]any{
			"/users": map[string]any{
				"x-internal": false, // Extension on path item level
				"get": map[string]any{
					"summary": "List users",
				},
				"post": map[string]any{
					"summary": "Create user",
				},
			},
			"/admin": map[string]any{
				"x-internal": true, // Extension on path item level
				"get": map[string]any{
					"summary": "Admin panel",
				},
			},
		},
		"servers": []any{
			map[string]any{"url": "https://api.example.com"},
			map[string]any{"url": "https://staging.example.com"},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"User": map[string]any{"type": "object"},
				"Pet":  map[string]any{"type": "object"},
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		wantLen  int
		checkVal func([]any) bool
	}{
		{
			name:    "root",
			path:    "$",
			wantLen: 1,
		},
		{
			name:    "simple child",
			path:    "$.info",
			wantLen: 1,
			checkVal: func(results []any) bool {
				m, ok := results[0].(map[string]any)
				return ok && m["title"] == "Test API"
			},
		},
		{
			name:    "nested child",
			path:    "$.info.title",
			wantLen: 1,
			checkVal: func(results []any) bool {
				return results[0] == "Test API"
			},
		},
		{
			name:    "bracket notation",
			path:    "$.paths['/users']",
			wantLen: 1,
		},
		{
			name:    "bracket and dot",
			path:    "$.paths['/users'].get",
			wantLen: 1,
			checkVal: func(results []any) bool {
				m, ok := results[0].(map[string]any)
				return ok && m["summary"] == "List users"
			},
		},
		{
			name:    "wildcard all paths",
			path:    "$.paths.*",
			wantLen: 2,
		},
		{
			name:    "wildcard all operations",
			path:    "$.paths.*.*",
			wantLen: 5, // /users: x-internal, get, post; /admin: x-internal, get
		},
		{
			name:    "wildcard then child",
			path:    "$.paths.*.get",
			wantLen: 2, // /users.get, /admin.get
		},
		{
			name:    "array index zero",
			path:    "$.servers[0]",
			wantLen: 1,
			checkVal: func(results []any) bool {
				m, ok := results[0].(map[string]any)
				return ok && m["url"] == "https://api.example.com"
			},
		},
		{
			name:    "array index negative",
			path:    "$.servers[-1]",
			wantLen: 1,
			checkVal: func(results []any) bool {
				m, ok := results[0].(map[string]any)
				return ok && m["url"] == "https://staging.example.com"
			},
		},
		{
			name:    "filter internal true",
			path:    "$.paths[?@.x-internal==true]",
			wantLen: 1, // /admin path item
		},
		{
			name:    "filter internal false",
			path:    "$.paths[?@.x-internal==false]",
			wantLen: 1, // /users path item
		},
		{
			name:    "non-existent path",
			path:    "$.nonexistent",
			wantLen: 0,
		},
		{
			name:    "deep non-existent",
			path:    "$.info.nonexistent.deep",
			wantLen: 0,
		},
		{
			name:    "component schema",
			path:    "$.components.schemas.User",
			wantLen: 1,
		},
		// Recursive descent tests
		{
			name:    "recursive descent summary",
			path:    "$..summary",
			wantLen: 3, // 3 summary fields: /users/get, /users/post, /admin/get
		},
		{
			name:    "recursive descent x-internal",
			path:    "$..x-internal",
			wantLen: 2, // /users and /admin path items
		},
		{
			name:    "recursive descent url",
			path:    "$..url",
			wantLen: 2, // 2 servers
			checkVal: func(results []any) bool {
				// Should find both server URLs
				hasApi := false
				hasStaging := false
				for _, r := range results {
					if s, ok := r.(string); ok {
						if s == "https://api.example.com" {
							hasApi = true
						}
						if s == "https://staging.example.com" {
							hasStaging = true
						}
					}
				}
				return hasApi && hasStaging
			},
		},
		{
			name:    "recursive descent type",
			path:    "$..type",
			wantLen: 2, // User.type and Pet.type in components/schemas
		},
		{
			name:    "recursive descent wildcard",
			path:    "$.paths..*",
			wantLen: 10, // All descendants: 2 path items, each with x-internal + operations
		},
		{
			name:    "recursive descent after child",
			path:    "$.paths..summary",
			wantLen: 3, // All summary fields under paths
		},
		{
			name:    "recursive descent nonexistent",
			path:    "$..nonexistent",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Parse(tt.path)
			require.NoError(t, err, "Parse(%q) error", tt.path)

			results := p.Get(doc)

			assert.Len(t, results, tt.wantLen, "Get(%q) result count", tt.path)

			if tt.checkVal != nil && len(results) > 0 {
				assert.True(t, tt.checkVal(results), "Get(%q) value check failed, got %v", tt.path, results)
			}
		})
	}
}

// TestCompoundFilters tests evaluation of compound filter expressions.
func TestCompoundFilters(t *testing.T) {
	doc := map[string]any{
		"items": []any{
			map[string]any{"name": "a", "active": true, "count": 5},
			map[string]any{"name": "b", "active": false, "count": 3},
			map[string]any{"name": "c", "active": true, "count": 0},
			map[string]any{"name": "d", "active": false, "count": 10},
			map[string]any{"name": "e", "status": "pending", "priority": 1},
			map[string]any{"name": "f", "status": "active", "priority": 2},
			map[string]any{"name": "g", "status": "completed", "priority": 3},
		},
	}

	tests := []struct {
		name    string
		path    string
		wantLen int
	}{
		{
			name:    "AND both true",
			path:    "$.items[?@.active==true && @.count>0]",
			wantLen: 1, // Only "a" has active=true AND count>0
		},
		{
			name:    "AND one false",
			path:    "$.items[?@.active==true && @.count==0]",
			wantLen: 1, // Only "c" has active=true AND count=0
		},
		{
			name:    "AND neither matches",
			path:    "$.items[?@.active==true && @.count>100]",
			wantLen: 0,
		},
		{
			name:    "OR either true",
			path:    "$.items[?@.status=='pending' || @.status=='active']",
			wantLen: 2, // "e" and "f"
		},
		{
			name:    "OR first true only",
			path:    "$.items[?@.name=='a' || @.name=='z']",
			wantLen: 1,
		},
		{
			name:    "OR second true only",
			path:    "$.items[?@.name=='z' || @.name=='b']",
			wantLen: 1,
		},
		{
			name:    "OR neither matches",
			path:    "$.items[?@.status=='unknown' || @.status=='deleted']",
			wantLen: 0,
		},
		{
			name:    "chained AND",
			path:    "$.items[?@.active==false && @.count>0 && @.count<15]",
			wantLen: 2, // "b" (count=3) and "d" (count=10)
		},
		{
			name:    "mixed priority filter",
			path:    "$.items[?@.priority>=2 && @.status!='completed']",
			wantLen: 1, // Only "f" has priority>=2 AND status!='completed'
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Parse(tt.path)
			require.NoError(t, err, "Parse(%q) error", tt.path)

			results := p.Get(doc)

			assert.Len(t, results, tt.wantLen, "Get(%q) result count", tt.path)
		})
	}
}

// TestSet tests the JSONPath Set method.
func TestSet(t *testing.T) {
	t.Run("set simple child", func(t *testing.T) {
		doc := map[string]any{
			"info": map[string]any{
				"title": "Old Title",
			},
		}

		p, _ := Parse("$.info.title")
		err := p.Set(doc, "New Title")

		require.NoError(t, err, "Set error")

		info := doc["info"].(map[string]any)
		assert.Equal(t, "New Title", info["title"])
	})

	t.Run("set array element", func(t *testing.T) {
		doc := map[string]any{
			"servers": []any{
				map[string]any{"url": "old"},
			},
		}

		p, _ := Parse("$.servers[0]")
		err := p.Set(doc, map[string]any{"url": "new"})

		require.NoError(t, err, "Set error")

		servers := doc["servers"].([]any)
		server := servers[0].(map[string]any)
		assert.Equal(t, "new", server["url"])
	})

	t.Run("set with wildcard", func(t *testing.T) {
		doc := map[string]any{
			"paths": map[string]any{
				"/a": map[string]any{"deprecated": false},
				"/b": map[string]any{"deprecated": false},
			},
		}

		p, _ := Parse("$.paths.*.deprecated")
		err := p.Set(doc, true)

		require.NoError(t, err, "Set error")

		paths := doc["paths"].(map[string]any)
		for _, pathItem := range paths {
			pi := pathItem.(map[string]any)
			assert.Equal(t, true, pi["deprecated"], "Set with wildcard did not update all values")
		}
	})

	t.Run("set on root fails", func(t *testing.T) {
		doc := map[string]any{}
		p, _ := Parse("$")
		err := p.Set(doc, "value")

		assert.Error(t, err, "Expected error when setting on root")
	})
}

// TestRemove tests the JSONPath Remove method.
func TestRemove(t *testing.T) {
	t.Run("remove simple child", func(t *testing.T) {
		doc := map[string]any{
			"info": map[string]any{
				"title":       "API",
				"description": "To remove",
			},
		}

		p, _ := Parse("$.info.description")
		_, err := p.Remove(doc)

		require.NoError(t, err, "Remove error")

		info := doc["info"].(map[string]any)
		assert.NotContains(t, info, "description", "Remove did not delete field")
		assert.Equal(t, "API", info["title"], "Remove deleted wrong field")
	})

	t.Run("remove with filter", func(t *testing.T) {
		doc := map[string]any{
			"operations": map[string]any{
				"public": map[string]any{
					"x-internal": false,
					"name":       "public",
				},
				"internal": map[string]any{
					"x-internal": true,
					"name":       "internal",
				},
			},
		}

		// Filter selects children of operations where x-internal==true
		p, _ := Parse("$.operations[?@.x-internal==true]")
		_, err := p.Remove(doc)

		require.NoError(t, err, "Remove error")

		ops := doc["operations"].(map[string]any)
		assert.NotContains(t, ops, "internal", "Remove with filter did not delete matching entry")
		assert.Contains(t, ops, "public", "Remove with filter deleted non-matching entry")
	})

	t.Run("remove non-existent path", func(t *testing.T) {
		doc := map[string]any{
			"info": map[string]any{},
		}

		p, _ := Parse("$.info.nonexistent")
		_, err := p.Remove(doc)

		assert.NoError(t, err, "Remove on non-existent path should not error")
	})
}

// TestModify tests the JSONPath Modify method.
func TestModify(t *testing.T) {
	t.Run("modify simple value", func(t *testing.T) {
		doc := map[string]any{
			"info": map[string]any{
				"title": "API",
			},
		}

		p, _ := Parse("$.info.title")
		err := p.Modify(doc, func(v any) any {
			return v.(string) + " v2"
		})

		require.NoError(t, err, "Modify error")

		info := doc["info"].(map[string]any)
		assert.Equal(t, "API v2", info["title"])
	})

	t.Run("modify with wildcard", func(t *testing.T) {
		doc := map[string]any{
			"paths": map[string]any{
				"/a": map[string]any{"x-rate-limit": 100},
				"/b": map[string]any{"x-rate-limit": 100},
			},
		}

		p, _ := Parse("$.paths.*")
		err := p.Modify(doc, func(v any) any {
			m := v.(map[string]any)
			m["x-modified"] = true
			return m
		})

		require.NoError(t, err, "Modify error")

		paths := doc["paths"].(map[string]any)
		for _, pathItem := range paths {
			pi := pathItem.(map[string]any)
			assert.Equal(t, true, pi["x-modified"], "Modify with wildcard did not update all values")
		}
	})

	t.Run("modify with filter", func(t *testing.T) {
		doc := map[string]any{
			"operations": map[string]any{
				"op1": map[string]any{"status": "active", "count": 0},
				"op2": map[string]any{"status": "inactive", "count": 0},
			},
		}

		// Filter selects children of operations where status=='active'
		p, _ := Parse("$.operations[?@.status=='active']")
		err := p.Modify(doc, func(v any) any {
			m := v.(map[string]any)
			m["count"] = 1
			return m
		})

		require.NoError(t, err, "Modify error")

		ops := doc["operations"].(map[string]any)
		op1 := ops["op1"].(map[string]any)
		op2 := ops["op2"].(map[string]any)

		assert.Equal(t, 1, op1["count"], "Modify with filter did not update matching entry")
		assert.Equal(t, 0, op2["count"], "Modify with filter updated non-matching entry")
	})
}

// TestFilterExpressions tests filter expression evaluation.
func TestFilterExpressions(t *testing.T) {
	doc := map[string]any{
		"items": []any{
			map[string]any{"name": "a", "value": 10, "active": true},
			map[string]any{"name": "b", "value": 20, "active": false},
			map[string]any{"name": "c", "value": 30, "active": true},
		},
	}

	tests := []struct {
		name    string
		path    string
		wantLen int
	}{
		{name: "equal string", path: "$.items[?@.name=='a']", wantLen: 1},
		{name: "equal number", path: "$.items[?@.value==20]", wantLen: 1},
		{name: "equal bool", path: "$.items[?@.active==true]", wantLen: 2},
		{name: "not equal", path: "$.items[?@.active!=true]", wantLen: 1},
		{name: "less than", path: "$.items[?@.value<25]", wantLen: 2},
		{name: "less equal", path: "$.items[?@.value<=20]", wantLen: 2},
		{name: "greater than", path: "$.items[?@.value>15]", wantLen: 2},
		{name: "greater equal", path: "$.items[?@.value>=20]", wantLen: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Parse(tt.path)
			require.NoError(t, err, "Parse error")

			results := p.Get(doc)
			assert.Len(t, results, tt.wantLen, "Get(%q) result count", tt.path)
		})
	}
}

// TestEdgeCases tests various edge cases.
func TestEdgeCases(t *testing.T) {
	t.Run("empty document", func(t *testing.T) {
		doc := map[string]any{}
		p, _ := Parse("$.info")
		results := p.Get(doc)
		assert.Empty(t, results, "Expected no results for empty document")
	})

	t.Run("nil document", func(t *testing.T) {
		p, _ := Parse("$.info")
		results := p.Get(nil)
		assert.Empty(t, results, "Expected empty results for nil document")
	})

	t.Run("special characters in key", func(t *testing.T) {
		doc := map[string]any{
			"paths": map[string]any{
				"/users/{id}": map[string]any{"get": "handler"},
			},
		}
		p, _ := Parse("$.paths['/users/{id}']")
		results := p.Get(doc)
		assert.Len(t, results, 1, "Expected 1 result for path with special chars")
	})

	t.Run("escaped quotes in string", func(t *testing.T) {
		p, err := Parse("$.paths['/test\\'s']")
		require.NoError(t, err, "Parse error for escaped quote")
		assert.NotNil(t, p, "Expected valid path for escaped quote")
	})

	t.Run("hyphenated field names", func(t *testing.T) {
		doc := map[string]any{
			"x-custom-extension": "value",
		}
		p, _ := Parse("$.x-custom-extension")
		results := p.Get(doc)
		require.Len(t, results, 1, "Expected 1 result for hyphenated field")
		assert.Equal(t, "value", results[0])
	})
}

// TestFilterExpr_String tests the String method of FilterExpr.
func TestFilterExpr_String(t *testing.T) {
	expr := &FilterExpr{
		Field:    "status",
		Operator: "==",
		Value:    "active",
	}

	assert.Equal(t, "@.status == active", expr.String())
}

// TestSetInParent tests setting values at nested paths.
func TestSetInParent(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		doc     any
		value   any
		wantErr bool
	}{
		{
			name:  "set in map",
			path:  "$.info.title",
			doc:   map[string]any{"info": map[string]any{"title": "old"}},
			value: "new",
		},
		{
			name:  "set array element",
			path:  "$.servers[0]",
			doc:   map[string]any{"servers": []any{"a", "b"}},
			value: "new",
		},
		{
			name:  "set via wildcard",
			path:  "$.items.*",
			doc:   map[string]any{"items": map[string]any{"a": 1, "b": 2}},
			value: 99,
		},
		{
			name:  "set with bracket notation",
			path:  "$.paths['/users']",
			doc:   map[string]any{"paths": map[string]any{"/users": "old"}},
			value: "new",
		},
		{
			name:    "set on nil parent",
			path:    "$.missing.child",
			doc:     map[string]any{},
			value:   "value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Parse(tt.path)
			require.NoError(t, err, "Parse error")

			err = p.Set(tt.doc, tt.value)
			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestRemoveFromParent tests removing values from nested paths.
func TestRemoveFromParent(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		doc     any
		wantErr bool
	}{
		{
			name: "remove from map",
			path: "$.info.title",
			doc:  map[string]any{"info": map[string]any{"title": "test", "version": "1.0"}},
		},
		{
			name: "remove array element",
			path: "$.servers[1]",
			doc:  map[string]any{"servers": []any{"a", "b", "c"}},
		},
		{
			name: "remove via wildcard",
			path: "$.items.*",
			doc:  map[string]any{"items": map[string]any{"a": 1, "b": 2}},
		},
		{
			name: "remove with bracket notation",
			path: "$.paths['/users']",
			doc:  map[string]any{"paths": map[string]any{"/users": "val", "/pets": "val2"}},
		},
		{
			name:    "remove from missing path",
			path:    "$.missing.child",
			doc:     map[string]any{},
			wantErr: false, // Remove on missing path is a no-op
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Parse(tt.path)
			require.NoError(t, err, "Parse error")

			_, err = p.Remove(tt.doc)
			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestModifyInParent tests modifying values at nested paths.
func TestModifyInParent(t *testing.T) {
	tests := []struct {
		name string
		path string
		doc  any
	}{
		{
			name: "modify in map",
			path: "$.info.title",
			doc:  map[string]any{"info": map[string]any{"title": "old"}},
		},
		{
			name: "modify array element",
			path: "$.servers[0]",
			doc:  map[string]any{"servers": []any{"a", "b"}},
		},
		{
			name: "modify via wildcard",
			path: "$.items.*",
			doc:  map[string]any{"items": map[string]any{"a": 1, "b": 2}},
		},
		{
			name: "modify with bracket notation",
			path: "$.paths['/users']",
			doc:  map[string]any{"paths": map[string]any{"/users": "old"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Parse(tt.path)
			require.NoError(t, err, "Parse error")

			err = p.Modify(tt.doc, func(v any) any {
				return "modified"
			})
			assert.NoError(t, err)
		})
	}
}

// TestFilterComparisons tests filter expressions with different operators.
func TestFilterComparisons(t *testing.T) {
	doc := map[string]any{
		"items": []any{
			map[string]any{"name": "a", "count": 5, "price": 10.5},
			map[string]any{"name": "b", "count": 10, "price": 20.0},
			map[string]any{"name": "c", "count": 15, "price": 5.5},
		},
	}

	tests := []struct {
		name     string
		path     string
		wantLen  int
		wantName string
	}{
		{name: "less than int", path: "$.items[?@.count<10]", wantLen: 1, wantName: "a"},
		{name: "less equal int", path: "$.items[?@.count<=10]", wantLen: 2},
		{name: "greater than int", path: "$.items[?@.count>10]", wantLen: 1, wantName: "c"},
		{name: "greater equal int", path: "$.items[?@.count>=10]", wantLen: 2},
		{name: "less than float", path: "$.items[?@.price<10]", wantLen: 1, wantName: "c"},
		{name: "greater than float", path: "$.items[?@.price>15]", wantLen: 1, wantName: "b"},
		{name: "not equal string", path: "$.items[?@.name!='a']", wantLen: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Parse(tt.path)
			require.NoError(t, err, "Parse error")

			results := p.Get(doc)
			assert.Len(t, results, tt.wantLen, "Get(%q) result count", tt.path)
			if tt.wantName != "" && len(results) > 0 {
				if m, ok := results[0].(map[string]any); ok {
					assert.Equal(t, tt.wantName, m["name"], "first match name")
				}
			}
		})
	}
}

// TestParseQuotedStrings tests parsing various quoted string formats.
func TestParseQuotedStrings(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "single quoted", path: "$.paths['/api/v1']", wantErr: false},
		{name: "double quoted", path: `$.paths["/api/v1"]`, wantErr: false},
		{name: "escaped single", path: "$.paths['it\\'s']", wantErr: false},
		{name: "escaped double", path: `$.paths["say \"hello\""]`, wantErr: false},
		{name: "with special chars", path: "$.paths['/users/{id}/profile']", wantErr: false},
		{name: "empty string", path: "$.paths['']", wantErr: false},
		{name: "unclosed single", path: "$.paths['test", wantErr: true},
		{name: "unclosed double", path: `$.paths["test`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.path)
			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestParseNumbers tests parsing numeric indices and values.
func TestParseNumbers(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "positive index", path: "$.arr[0]", wantErr: false},
		{name: "larger index", path: "$.arr[123]", wantErr: false},
		{name: "negative index", path: "$.arr[-1]", wantErr: false},
		{name: "filter with int", path: "$.items[?@.count==42]", wantErr: false},
		{name: "filter with negative", path: "$.items[?@.val==-5]", wantErr: false},
		{name: "filter with float", path: "$.items[?@.price==19.99]", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.path)
			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNormalizeValue tests value normalization for comparisons.
func TestNormalizeValue(t *testing.T) {
	doc := map[string]any{
		"items": []any{
			map[string]any{"id": int64(1), "val": float32(1.5)},
			map[string]any{"id": int64(2), "val": float32(2.5)},
		},
	}

	// Test that int64 compares correctly
	p, _ := Parse("$.items[?@.id==1]")
	results := p.Get(doc)
	assert.Len(t, results, 1, "int64 comparison")

	// Test float32 comparison
	p2, _ := Parse("$.items[?@.val>2.0]")
	results2 := p2.Get(doc)
	assert.Len(t, results2, 1, "float32 comparison")
}

// TestPathString tests the String method of Path.
func TestPathString(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"$"},
		{"$.info"},
		{"$.paths.*"},
		{"$.servers[0]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, _ := Parse(tt.input)
			assert.Equal(t, tt.input, p.String())
		})
	}
}

// TestRecursiveDescentDepthLimit verifies that recursiveDescend stops at
// maxRecursionDepth and does not stack overflow on deeply nested structures.
func TestRecursiveDescentDepthLimit(t *testing.T) {
	suppressJSONPathLogger(t)

	// Build a deeply nested structure (600 levels, exceeding the 500 cap).
	var node any = "leaf"
	for range 600 {
		node = map[string]any{"nested": node}
	}

	// Use a child segment that matches the "nested" key at every level.
	child := ChildSegment{Key: "nested"}
	results := recursiveDescend(node, child, 0)

	// Should not panic or stack overflow; results are capped by depth limit.
	assert.LessOrEqual(t, len(results), 501)
	assert.Greater(t, len(results), 0, "expected some results before hitting depth cap")
}

// TestRecursiveDescentDepthLimit_WarningEmitted verifies that a slog.Warn
// is actually emitted when the depth limit is hit during recursive descent.
func TestRecursiveDescentDepthLimit_WarningEmitted(t *testing.T) {
	var buf bytes.Buffer
	old := jsonpathLogger
	jsonpathLogger = slog.New(slog.NewTextHandler(&buf, nil))
	t.Cleanup(func() { jsonpathLogger = old })

	var node any = "leaf"
	for range 600 {
		node = map[string]any{"nested": node}
	}

	child := ChildSegment{Key: "nested"}
	recursiveDescend(node, child, 0)

	assert.Contains(t, buf.String(), "truncated at depth limit")
}

// TestRecursiveDescentDepthLimit_nilChild verifies the depth cap when
// recursiveDescend delegates to collectAllDescendants (nil child segment).
func TestRecursiveDescentDepthLimit_nilChild(t *testing.T) {
	suppressJSONPathLogger(t)

	var node any = "leaf"
	for range 600 {
		node = map[string]any{"nested": node}
	}

	results := recursiveDescend(node, nil, 0)

	assert.LessOrEqual(t, len(results), 501)
	assert.Greater(t, len(results), 0, "expected some results before hitting depth cap")
}

// TestCollectAllDescendantsDepthLimit verifies that collectAllDescendants
// stops at maxRecursionDepth on deeply nested structures.
func TestCollectAllDescendantsDepthLimit(t *testing.T) {
	suppressJSONPathLogger(t)

	var node any = "leaf"
	for range 600 {
		node = map[string]any{"nested": node}
	}

	var results []any
	collectAllDescendants(node, &results, 0)

	// Each level adds one value; depth cap at 500 means at most ~501 entries.
	assert.LessOrEqual(t, len(results), 501)
	assert.Greater(t, len(results), 0, "expected some results before hitting depth cap")
}
