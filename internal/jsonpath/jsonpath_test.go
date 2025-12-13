package jsonpath

import (
	"testing"
)

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
				if err == nil {
					t.Errorf("Parse(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
				return
			}

			if path == nil {
				t.Errorf("Parse(%q) returned nil path without error", tt.input)
				return
			}

			if len(path.segments) != tt.segLen {
				t.Errorf("Parse(%q) got %d segments, want %d", tt.input, len(path.segments), tt.segLen)
			}

			if path.String() != tt.input {
				t.Errorf("Path.String() = %q, want %q", path.String(), tt.input)
			}
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
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.path, err)
			}

			results := p.Get(doc)

			if len(results) != tt.wantLen {
				t.Errorf("Get(%q) returned %d results, want %d", tt.path, len(results), tt.wantLen)
			}

			if tt.checkVal != nil && len(results) > 0 {
				if !tt.checkVal(results) {
					t.Errorf("Get(%q) value check failed, got %v", tt.path, results)
				}
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
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.path, err)
			}

			results := p.Get(doc)

			if len(results) != tt.wantLen {
				t.Errorf("Get(%q) returned %d results, want %d", tt.path, len(results), tt.wantLen)
				for i, r := range results {
					t.Logf("  result[%d]: %v", i, r)
				}
			}
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

		if err != nil {
			t.Fatalf("Set error: %v", err)
		}

		info := doc["info"].(map[string]any)
		if info["title"] != "New Title" {
			t.Errorf("Set did not update value, got %v", info["title"])
		}
	})

	t.Run("set array element", func(t *testing.T) {
		doc := map[string]any{
			"servers": []any{
				map[string]any{"url": "old"},
			},
		}

		p, _ := Parse("$.servers[0]")
		err := p.Set(doc, map[string]any{"url": "new"})

		if err != nil {
			t.Fatalf("Set error: %v", err)
		}

		servers := doc["servers"].([]any)
		server := servers[0].(map[string]any)
		if server["url"] != "new" {
			t.Errorf("Set did not update array element, got %v", server["url"])
		}
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

		if err != nil {
			t.Fatalf("Set error: %v", err)
		}

		paths := doc["paths"].(map[string]any)
		for _, pathItem := range paths {
			pi := pathItem.(map[string]any)
			if pi["deprecated"] != true {
				t.Errorf("Set with wildcard did not update all values")
			}
		}
	})

	t.Run("set on root fails", func(t *testing.T) {
		doc := map[string]any{}
		p, _ := Parse("$")
		err := p.Set(doc, "value")

		if err == nil {
			t.Error("Expected error when setting on root")
		}
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

		if err != nil {
			t.Fatalf("Remove error: %v", err)
		}

		info := doc["info"].(map[string]any)
		if _, exists := info["description"]; exists {
			t.Error("Remove did not delete field")
		}
		if info["title"] != "API" {
			t.Error("Remove deleted wrong field")
		}
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

		if err != nil {
			t.Fatalf("Remove error: %v", err)
		}

		ops := doc["operations"].(map[string]any)
		if _, exists := ops["internal"]; exists {
			t.Error("Remove with filter did not delete matching entry")
		}
		if _, exists := ops["public"]; !exists {
			t.Error("Remove with filter deleted non-matching entry")
		}
	})

	t.Run("remove non-existent path", func(t *testing.T) {
		doc := map[string]any{
			"info": map[string]any{},
		}

		p, _ := Parse("$.info.nonexistent")
		_, err := p.Remove(doc)

		if err != nil {
			t.Errorf("Remove on non-existent path should not error, got: %v", err)
		}
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

		if err != nil {
			t.Fatalf("Modify error: %v", err)
		}

		info := doc["info"].(map[string]any)
		if info["title"] != "API v2" {
			t.Errorf("Modify did not transform value, got %v", info["title"])
		}
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

		if err != nil {
			t.Fatalf("Modify error: %v", err)
		}

		paths := doc["paths"].(map[string]any)
		for _, pathItem := range paths {
			pi := pathItem.(map[string]any)
			if pi["x-modified"] != true {
				t.Error("Modify with wildcard did not update all values")
			}
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

		if err != nil {
			t.Fatalf("Modify error: %v", err)
		}

		ops := doc["operations"].(map[string]any)
		op1 := ops["op1"].(map[string]any)
		op2 := ops["op2"].(map[string]any)

		if op1["count"] != 1 {
			t.Error("Modify with filter did not update matching entry")
		}
		if op2["count"] != 0 {
			t.Error("Modify with filter updated non-matching entry")
		}
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
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			results := p.Get(doc)
			if len(results) != tt.wantLen {
				t.Errorf("Get(%q) returned %d results, want %d", tt.path, len(results), tt.wantLen)
			}
		})
	}
}

// TestEdgeCases tests various edge cases.
func TestEdgeCases(t *testing.T) {
	t.Run("empty document", func(t *testing.T) {
		doc := map[string]any{}
		p, _ := Parse("$.info")
		results := p.Get(doc)
		if len(results) != 0 {
			t.Error("Expected no results for empty document")
		}
	})

	t.Run("nil document", func(t *testing.T) {
		p, _ := Parse("$.info")
		results := p.Get(nil)
		if len(results) != 0 {
			t.Error("Expected empty results for nil document")
		}
	})

	t.Run("special characters in key", func(t *testing.T) {
		doc := map[string]any{
			"paths": map[string]any{
				"/users/{id}": map[string]any{"get": "handler"},
			},
		}
		p, _ := Parse("$.paths['/users/{id}']")
		results := p.Get(doc)
		if len(results) != 1 {
			t.Error("Expected 1 result for path with special chars")
		}
	})

	t.Run("escaped quotes in string", func(t *testing.T) {
		p, err := Parse("$.paths['/test\\'s']")
		if err != nil {
			t.Fatalf("Parse error for escaped quote: %v", err)
		}
		if p == nil {
			t.Error("Expected valid path for escaped quote")
		}
	})

	t.Run("hyphenated field names", func(t *testing.T) {
		doc := map[string]any{
			"x-custom-extension": "value",
		}
		p, _ := Parse("$.x-custom-extension")
		results := p.Get(doc)
		if len(results) != 1 || results[0] != "value" {
			t.Error("Expected to access hyphenated field")
		}
	})
}

// TestFilterExpr_String tests the String method of FilterExpr.
func TestFilterExpr_String(t *testing.T) {
	expr := &FilterExpr{
		Field:    "status",
		Operator: "==",
		Value:    "active",
	}

	expected := "@.status == active"
	if expr.String() != expected {
		t.Errorf("FilterExpr.String() = %q, want %q", expr.String(), expected)
	}
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
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			err = p.Set(tt.doc, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
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
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			_, err = p.Remove(tt.doc)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
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
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			err = p.Modify(tt.doc, func(v any) any {
				return "modified"
			})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
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
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			results := p.Get(doc)
			if len(results) != tt.wantLen {
				t.Errorf("Get(%q) returned %d results, want %d", tt.path, len(results), tt.wantLen)
			}
			if tt.wantName != "" && len(results) > 0 {
				if m, ok := results[0].(map[string]any); ok {
					if m["name"] != tt.wantName {
						t.Errorf("Expected first match to have name %q, got %v", tt.wantName, m["name"])
					}
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
			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
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
			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
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
	if len(results) != 1 {
		t.Errorf("Expected 1 result for int64 comparison, got %d", len(results))
	}

	// Test float32 comparison
	p2, _ := Parse("$.items[?@.val>2.0]")
	results2 := p2.Get(doc)
	if len(results2) != 1 {
		t.Errorf("Expected 1 result for float32 comparison, got %d", len(results2))
	}
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
			if p.String() != tt.input {
				t.Errorf("String() = %q, want %q", p.String(), tt.input)
			}
		})
	}
}
