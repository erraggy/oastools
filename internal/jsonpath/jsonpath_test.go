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
