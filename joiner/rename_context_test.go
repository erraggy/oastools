package joiner

import (
	"testing"
	"text/template"
)

// ============================================================================
// Path Function Tests
// ============================================================================

func TestPathSegment(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		index    int
		expected string
	}{
		{"first segment", "/users/{id}/orders", 0, "users"},
		{"second segment", "/users/{id}/orders", 1, "orders"},
		{"negative index last", "/users/{id}/orders", -1, "orders"},
		{"negative index second to last", "/api/v1/users", -2, "v1"},
		{"skip parameters", "/users/{userId}/posts/{postId}", 1, "posts"},
		{"empty path", "", 0, ""},
		{"out of bounds", "/users", 5, ""},
		{"root only", "/", 0, ""},
		{"negative out of bounds", "/users", -5, ""},
		{"single segment", "/users", 0, "users"},
		{"single segment negative", "/users", -1, "users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathSegment(tt.path, tt.index)
			if got != tt.expected {
				t.Errorf("pathSegment(%q, %d) = %q, want %q", tt.path, tt.index, got, tt.expected)
			}
		})
	}
}

func TestPathResource(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"simple resource", "/users", "users"},
		{"nested resource", "/users/{id}/orders", "users"},
		{"api versioned", "/api/v1/users", "api"},
		{"empty", "", ""},
		{"parameter first", "/{version}/users", "users"},
		{"root only", "/", ""},
		{"only parameters", "/{a}/{b}/{c}", ""},
		{"parameter then resource", "/{version}/api/users", "api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathResource(tt.path)
			if got != tt.expected {
				t.Errorf("pathResource(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestPathLast(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"simple", "/users", "users"},
		{"nested", "/users/{id}/orders", "orders"},
		{"parameter last", "/users/{id}", "users"},
		{"empty", "", ""},
		{"root only", "/", ""},
		{"deeply nested", "/api/v1/users/{id}/orders/{orderId}/items", "items"},
		{"all parameters", "/{a}/{b}", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathLast(tt.path)
			if got != tt.expected {
				t.Errorf("pathLast(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestPathClean(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"simple", "/users", "users"},
		{"with parameter", "/users/{id}", "users_id"},
		{"nested", "/users/{id}/orders", "users_id_orders"},
		{"empty", "", ""},
		{"root only", "/", ""},
		{"with hyphen", "/user-profiles", "user_profiles"},
		{"with dot", "/api.v1/users", "api_v1_users"},
		{"trailing slash", "/users/", "users"},
		{"leading and trailing slash", "/users/", "users"},
		{"multiple parameters", "/{version}/users/{id}/orders/{orderId}", "version_users_id_orders_orderId"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathClean(tt.path)
			if got != tt.expected {
				t.Errorf("pathClean(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

// ============================================================================
// Tag Function Tests
// ============================================================================

func TestFirstTag(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{"single tag", []string{"Users"}, "Users"},
		{"multiple tags", []string{"Users", "Admin"}, "Users"},
		{"empty", []string{}, ""},
		{"nil", nil, ""},
		{"empty string first", []string{"", "Admin"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstTag(tt.tags)
			if got != tt.expected {
				t.Errorf("firstTag(%v) = %q, want %q", tt.tags, got, tt.expected)
			}
		})
	}
}

func TestJoinTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		sep      string
		expected string
	}{
		{"underscore sep", []string{"Users", "Admin"}, "_", "Users_Admin"},
		{"dash sep", []string{"A", "B", "C"}, "-", "A-B-C"},
		{"single tag", []string{"Users"}, "_", "Users"},
		{"empty", []string{}, "_", ""},
		{"nil tags", nil, "_", ""},
		{"empty separator", []string{"A", "B"}, "", "AB"},
		{"long separator", []string{"A", "B"}, "---", "A---B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinTags(tt.tags, tt.sep)
			if got != tt.expected {
				t.Errorf("joinTags(%v, %q) = %q, want %q", tt.tags, tt.sep, got, tt.expected)
			}
		})
	}
}

func TestHasTag(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		tag      string
		expected bool
	}{
		{"found", []string{"Users", "Admin"}, "Users", true},
		{"not found", []string{"Users", "Admin"}, "Other", false},
		{"empty tags", []string{}, "Users", false},
		{"case sensitive", []string{"Users"}, "users", false},
		{"nil tags", nil, "Users", false},
		{"empty string tag", []string{"Users", ""}, "", true},
		{"partial match", []string{"Users"}, "User", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasTag(tt.tags, tt.tag)
			if got != tt.expected {
				t.Errorf("hasTag(%v, %q) = %v, want %v", tt.tags, tt.tag, got, tt.expected)
			}
		})
	}
}

// ============================================================================
// Case Conversion Tests (via template functions)
// ============================================================================

func TestCaseConversions(t *testing.T) {
	funcs := renameFuncs()

	// Verify all case functions are registered
	requiredFuncs := []string{"pascalCase", "camelCase", "snakeCase", "kebabCase"}
	for _, name := range requiredFuncs {
		if _, ok := funcs[name]; !ok {
			t.Errorf("renameFuncs() missing required function %q", name)
		}
	}

	// Test that functions work via template execution
	tests := []struct {
		tmpl     string
		data     any
		expected string
	}{
		{`{{pascalCase .}}`, "user_profile", "UserProfile"},
		{`{{camelCase .}}`, "user_profile", "userProfile"},
		{`{{snakeCase .}}`, "UserProfile", "user_profile"},
		{`{{kebabCase .}}`, "UserProfile", "user-profile"},
		{`{{pascalCase .}}`, "api-client", "ApiClient"},
		{`{{camelCase .}}`, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.tmpl, func(t *testing.T) {
			tmpl, err := template.New("test").Funcs(funcs).Parse(tt.tmpl)
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			var buf []byte
			w := &testWriter{buf: &buf}
			err = tmpl.Execute(w, tt.data)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}

			got := string(buf)
			if got != tt.expected {
				t.Errorf("Template %q with %v = %q, want %q", tt.tmpl, tt.data, got, tt.expected)
			}
		})
	}
}

// testWriter is a simple io.Writer for template testing
type testWriter struct {
	buf *[]byte
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// ============================================================================
// Conditional Helper Tests
// ============================================================================

func TestDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		expected string
	}{
		{"non-empty value", "hello", "default", "hello"},
		{"empty value", "", "default", "default"},
		{"both empty", "", "", ""},
		{"whitespace value", "  ", "default", "  "},
		{"fallback empty", "hello", "", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultValue(tt.value, tt.fallback)
			if got != tt.expected {
				t.Errorf("defaultValue(%q, %q) = %q, want %q", tt.value, tt.fallback, got, tt.expected)
			}
		})
	}
}

func TestCoalesce(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{"first non-empty", []string{"", "hello", "world"}, "hello"},
		{"first is non-empty", []string{"hello", "world"}, "hello"},
		{"all empty", []string{"", "", ""}, ""},
		{"single value", []string{"hello"}, "hello"},
		{"empty slice", []string{}, ""},
		{"single empty value", []string{""}, ""},
		{"last non-empty", []string{"", "", "final"}, "final"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coalesce(tt.values...)
			if got != tt.expected {
				t.Errorf("coalesce(%v) = %q, want %q", tt.values, got, tt.expected)
			}
		})
	}
}

// ============================================================================
// Context Builder Tests
// ============================================================================

func TestBuildRenameContext_Basic(t *testing.T) {
	// No graph provided - only core fields should be populated
	ctx := buildRenameContext("MySchema", "/path/to/file.yaml", 2, nil, PolicyFirstEncountered)

	if ctx.Name != "MySchema" {
		t.Errorf("Name = %q, want %q", ctx.Name, "MySchema")
	}
	if ctx.Source != "file" {
		t.Errorf("Source = %q, want %q", ctx.Source, "file")
	}
	if ctx.Index != 2 {
		t.Errorf("Index = %d, want %d", ctx.Index, 2)
	}

	// Operation context should be empty
	if ctx.Path != "" {
		t.Errorf("Path = %q, want empty", ctx.Path)
	}
	if ctx.Method != "" {
		t.Errorf("Method = %q, want empty", ctx.Method)
	}
	if ctx.OperationID != "" {
		t.Errorf("OperationID = %q, want empty", ctx.OperationID)
	}
	if len(ctx.Tags) != 0 {
		t.Errorf("Tags = %v, want empty", ctx.Tags)
	}

	// Aggregate fields should be empty/zero
	if len(ctx.AllPaths) != 0 {
		t.Errorf("AllPaths = %v, want empty", ctx.AllPaths)
	}
	if ctx.RefCount != 0 {
		t.Errorf("RefCount = %d, want 0", ctx.RefCount)
	}
	if ctx.IsShared {
		t.Error("IsShared = true, want false")
	}
}

func TestBuildRenameContext_WithGraph(t *testing.T) {
	// Create a graph with schema that has operation lineage
	graph := &RefGraph{
		operationRefs: map[string][]OperationRef{
			"User": {
				{
					Path:        "/users/{id}",
					Method:      "get",
					OperationID: "getUser",
					Tags:        []string{"Users", "Public"},
					UsageType:   UsageTypeResponse,
					StatusCode:  "200",
					MediaType:   "application/json",
				},
				{
					Path:        "/users",
					Method:      "post",
					OperationID: "createUser",
					Tags:        []string{"Users"},
					UsageType:   UsageTypeRequest,
					MediaType:   "application/json",
				},
			},
		},
	}

	ctx := buildRenameContext("User", "api-spec.yaml", 0, graph, PolicyFirstEncountered)

	// Core fields
	if ctx.Name != "User" {
		t.Errorf("Name = %q, want %q", ctx.Name, "User")
	}
	if ctx.Source != "api_spec" {
		t.Errorf("Source = %q, want %q", ctx.Source, "api_spec")
	}
	if ctx.Index != 0 {
		t.Errorf("Index = %d, want %d", ctx.Index, 0)
	}

	// Operation context (from first/primary operation)
	if ctx.Path != "/users/{id}" {
		t.Errorf("Path = %q, want %q", ctx.Path, "/users/{id}")
	}
	if ctx.Method != "get" {
		t.Errorf("Method = %q, want %q", ctx.Method, "get")
	}
	if ctx.OperationID != "getUser" {
		t.Errorf("OperationID = %q, want %q", ctx.OperationID, "getUser")
	}
	if len(ctx.Tags) != 2 || ctx.Tags[0] != "Users" || ctx.Tags[1] != "Public" {
		t.Errorf("Tags = %v, want [Users Public]", ctx.Tags)
	}
	if ctx.UsageType != "response" {
		t.Errorf("UsageType = %q, want %q", ctx.UsageType, "response")
	}
	if ctx.StatusCode != "200" {
		t.Errorf("StatusCode = %q, want %q", ctx.StatusCode, "200")
	}
	if ctx.MediaType != "application/json" {
		t.Errorf("MediaType = %q, want %q", ctx.MediaType, "application/json")
	}

	// Aggregate fields
	if ctx.RefCount != 2 {
		t.Errorf("RefCount = %d, want %d", ctx.RefCount, 2)
	}
	if !ctx.IsShared {
		t.Error("IsShared = false, want true")
	}
	if len(ctx.AllPaths) != 2 {
		t.Errorf("AllPaths length = %d, want 2", len(ctx.AllPaths))
	}
	if len(ctx.AllMethods) != 2 {
		t.Errorf("AllMethods length = %d, want 2", len(ctx.AllMethods))
	}
	if len(ctx.AllOperationIDs) != 2 {
		t.Errorf("AllOperationIDs length = %d, want 2", len(ctx.AllOperationIDs))
	}
	if len(ctx.AllTags) != 2 {
		t.Errorf("AllTags length = %d, want 2 (Users, Public)", len(ctx.AllTags))
	}
	if ctx.PrimaryResource != "users" {
		t.Errorf("PrimaryResource = %q, want %q", ctx.PrimaryResource, "users")
	}
}

func TestBuildRenameContext_EmptyLineage(t *testing.T) {
	// Graph provided but schema has no operation refs
	graph := &RefGraph{
		operationRefs: map[string][]OperationRef{
			"OtherSchema": {
				{Path: "/other", Method: "get"},
			},
		},
	}

	ctx := buildRenameContext("UnreferencedSchema", "spec.yaml", 1, graph, PolicyFirstEncountered)

	// Only core fields should be populated
	if ctx.Name != "UnreferencedSchema" {
		t.Errorf("Name = %q, want %q", ctx.Name, "UnreferencedSchema")
	}
	if ctx.Source != "spec" {
		t.Errorf("Source = %q, want %q", ctx.Source, "spec")
	}
	if ctx.Index != 1 {
		t.Errorf("Index = %d, want %d", ctx.Index, 1)
	}

	// Operation context should be empty
	if ctx.Path != "" {
		t.Errorf("Path = %q, want empty", ctx.Path)
	}
	if ctx.RefCount != 0 {
		t.Errorf("RefCount = %d, want 0", ctx.RefCount)
	}
	if ctx.IsShared {
		t.Error("IsShared = true, want false")
	}
}

func TestBuildRenameContext_SourcePathSanitization(t *testing.T) {
	tests := []struct {
		name       string
		sourcePath string
		expected   string
	}{
		{"full path with yaml", "/path/to/file.yaml", "file"},
		{"json extension", "api-spec.json", "api_spec"},
		{"space in name", "my file.yaml", "my_file"},
		{"dot in name", "api.v2.spec.yaml", "api_v2_spec"},
		{"empty path", "", ""},
		{"no extension", "config", "config"},
		{"hyphen in name", "user-service.yaml", "user_service"},
		{"multiple dots", "spec.test.draft.yaml", "spec_test_draft"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := buildRenameContext("Schema", tt.sourcePath, 0, nil, PolicyFirstEncountered)
			if ctx.Source != tt.expected {
				t.Errorf("Source = %q, want %q", ctx.Source, tt.expected)
			}
		})
	}
}

// ============================================================================
// Primary Operation Policy Tests
// ============================================================================

func TestSelectPrimaryOperation_FirstEncountered(t *testing.T) {
	refs := []OperationRef{
		{Path: "/users", Method: "get", OperationID: "getUsers"},
		{Path: "/orders", Method: "post", OperationID: "createOrder"},
		{Path: "/api", Method: "get", OperationID: "getApi"},
	}

	result := selectPrimaryOperation(refs, PolicyFirstEncountered)

	if result.Path != "/users" {
		t.Errorf("Path = %q, want %q", result.Path, "/users")
	}
	if result.OperationID != "getUsers" {
		t.Errorf("OperationID = %q, want %q", result.OperationID, "getUsers")
	}
}

func TestSelectPrimaryOperation_Alphabetical(t *testing.T) {
	refs := []OperationRef{
		{Path: "/users", Method: "get", OperationID: "getUsers"},
		{Path: "/orders", Method: "post", OperationID: "createOrder"},
		{Path: "/api", Method: "get", OperationID: "getApi"},
	}

	result := selectPrimaryOperation(refs, PolicyAlphabetical)

	// /api + get = "/apiget" is alphabetically first
	if result.Path != "/api" {
		t.Errorf("Path = %q, want %q", result.Path, "/api")
	}
	if result.OperationID != "getApi" {
		t.Errorf("OperationID = %q, want %q", result.OperationID, "getApi")
	}
}

func TestSelectPrimaryOperation_MostSpecific(t *testing.T) {
	tests := []struct {
		name     string
		refs     []OperationRef
		wantPath string
		wantID   string
	}{
		{
			name: "prefer operationId",
			refs: []OperationRef{
				{Path: "/a", Method: "get", Tags: []string{"TagA"}},
				{Path: "/b", Method: "post", OperationID: "createB"},
				{Path: "/c", Method: "get"},
			},
			wantPath: "/b",
			wantID:   "createB",
		},
		{
			name: "fall back to tags",
			refs: []OperationRef{
				{Path: "/a", Method: "get"},
				{Path: "/b", Method: "post", Tags: []string{"TagB"}},
				{Path: "/c", Method: "get"},
			},
			wantPath: "/b",
			wantID:   "",
		},
		{
			name: "fall back to first",
			refs: []OperationRef{
				{Path: "/a", Method: "get"},
				{Path: "/b", Method: "post"},
				{Path: "/c", Method: "get"},
			},
			wantPath: "/a",
			wantID:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectPrimaryOperation(tt.refs, PolicyMostSpecific)
			if result.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", result.Path, tt.wantPath)
			}
			if result.OperationID != tt.wantID {
				t.Errorf("OperationID = %q, want %q", result.OperationID, tt.wantID)
			}
		})
	}
}

func TestSelectPrimaryOperation_EmptyRefs(t *testing.T) {
	result := selectPrimaryOperation([]OperationRef{}, PolicyFirstEncountered)
	if result.Path != "" {
		t.Errorf("Path = %q, want empty", result.Path)
	}

	result = selectPrimaryOperation(nil, PolicyAlphabetical)
	if result.Path != "" {
		t.Errorf("Path = %q, want empty", result.Path)
	}
}

func TestSelectPrimaryOperation_UnknownPolicy(t *testing.T) {
	refs := []OperationRef{
		{Path: "/users", Method: "get"},
	}

	// Unknown policy should fall back to first
	result := selectPrimaryOperation(refs, PrimaryOperationPolicy(999))
	if result.Path != "/users" {
		t.Errorf("Path = %q, want %q", result.Path, "/users")
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestSanitizeSourcePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"full path yaml", "/path/to/file.yaml", "file"},
		{"full path json", "/path/to/spec.json", "spec"},
		{"hyphen in name", "api-spec.yaml", "api_spec"},
		{"space in name", "my file.yaml", "my_file"},
		{"dot in name", "api.v2.yaml", "api_v2"},
		{"empty", "", ""},
		{"no extension", "config", "config"},
		{"double extension", "spec.test.yaml", "spec_test"},
		{"just extension", ".yaml", ""},
		// Note: Windows paths are not tested here as filepath.Base is platform-specific
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSourcePath(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeSourcePath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractPathSegments(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{"simple", "/users", []string{"users"}},
		{"nested", "/users/orders", []string{"users", "orders"}},
		{"with parameter", "/users/{id}", []string{"users"}},
		{"multiple parameters", "/users/{id}/orders/{orderId}", []string{"users", "orders"}},
		{"empty", "", nil},
		{"root only", "/", nil},
		{"only parameters", "/{a}/{b}", nil},
		{"mixed", "/api/v1/{version}/users/{id}/profile", []string{"api", "v1", "users", "profile"}},
		{"trailing slash", "/users/", []string{"users"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathSegments(tt.path)
			if len(got) != len(tt.expected) {
				t.Errorf("extractPathSegments(%q) = %v, want %v", tt.path, got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("extractPathSegments(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

// ============================================================================
// RefGraph Tests
// ============================================================================

func TestRefGraph_ResolveLineage(t *testing.T) {
	t.Run("nil graph", func(t *testing.T) {
		var g *RefGraph
		result := g.ResolveLineage("Schema")
		if result != nil {
			t.Errorf("ResolveLineage on nil graph = %v, want nil", result)
		}
	})

	t.Run("direct operation refs", func(t *testing.T) {
		g := &RefGraph{
			operationRefs: map[string][]OperationRef{
				"User": {
					{Path: "/users", Method: "get"},
					{Path: "/users/{id}", Method: "get"},
				},
			},
		}

		result := g.ResolveLineage("User")
		if len(result) != 2 {
			t.Errorf("ResolveLineage returned %d refs, want 2", len(result))
		}
	})

	t.Run("indirect refs through parent schema", func(t *testing.T) {
		g := &RefGraph{
			schemaRefs: map[string][]SchemaRef{
				"Address": {
					{FromSchema: "User", RefLocation: "properties.address"},
				},
			},
			operationRefs: map[string][]OperationRef{
				"User": {
					{Path: "/users", Method: "get"},
				},
			},
		}

		result := g.ResolveLineage("Address")
		if len(result) != 1 {
			t.Errorf("ResolveLineage returned %d refs, want 1", len(result))
		}
		if result[0].Path != "/users" {
			t.Errorf("ResolveLineage[0].Path = %q, want %q", result[0].Path, "/users")
		}
	})

	t.Run("caching", func(t *testing.T) {
		g := &RefGraph{
			operationRefs: map[string][]OperationRef{
				"User": {{Path: "/users", Method: "get"}},
			},
		}

		// First call
		result1 := g.ResolveLineage("User")
		// Second call should use cache
		result2 := g.ResolveLineage("User")

		if len(result1) != len(result2) {
			t.Error("Cached result differs from original")
		}
	})

	t.Run("cycle detection", func(t *testing.T) {
		// Create a cycle: A -> B -> A
		g := &RefGraph{
			schemaRefs: map[string][]SchemaRef{
				"A": {{FromSchema: "B", RefLocation: "properties.b"}},
				"B": {{FromSchema: "A", RefLocation: "properties.a"}},
			},
			operationRefs: map[string][]OperationRef{
				"A": {{Path: "/a", Method: "get"}},
			},
		}

		// Should not hang due to cycle detection
		result := g.ResolveLineage("A")
		if len(result) != 1 {
			t.Errorf("ResolveLineage with cycle returned %d refs, want 1", len(result))
		}
	})

	t.Run("no refs for schema", func(t *testing.T) {
		g := &RefGraph{
			operationRefs: map[string][]OperationRef{
				"Other": {{Path: "/other", Method: "get"}},
			},
		}

		result := g.ResolveLineage("Unknown")
		if len(result) != 0 {
			t.Errorf("ResolveLineage for unknown schema = %v, want empty", result)
		}
	})
}

// ============================================================================
// Template Function Registration Tests
// ============================================================================

func TestRenameFuncs_AllFunctionsRegistered(t *testing.T) {
	funcs := renameFuncs()

	expected := []string{
		// Path functions
		"pathSegment", "pathResource", "pathLast", "pathClean",
		// Tag functions
		"firstTag", "joinTags", "hasTag",
		// Case functions
		"pascalCase", "camelCase", "snakeCase", "kebabCase",
		// Conditional helpers
		"default", "coalesce",
	}

	for _, name := range expected {
		if _, ok := funcs[name]; !ok {
			t.Errorf("renameFuncs() missing function %q", name)
		}
	}
}

// ============================================================================
// RenameContext Field Tests
// ============================================================================

func TestRenameContext_SingleReference(t *testing.T) {
	graph := &RefGraph{
		operationRefs: map[string][]OperationRef{
			"Pet": {
				{
					Path:        "/pets/{petId}",
					Method:      "get",
					OperationID: "getPet",
					Tags:        []string{"Pets"},
					UsageType:   UsageTypeResponse,
					StatusCode:  "200",
					ParamName:   "",
					MediaType:   "application/json",
				},
			},
		},
	}

	ctx := buildRenameContext("Pet", "petstore.yaml", 0, graph, PolicyFirstEncountered)

	// Single reference should not be marked as shared
	if ctx.IsShared {
		t.Error("IsShared = true, want false for single reference")
	}
	if ctx.RefCount != 1 {
		t.Errorf("RefCount = %d, want 1", ctx.RefCount)
	}
}

func TestRenameContext_UsageTypes(t *testing.T) {
	tests := []struct {
		name      string
		usageType UsageType
		expected  UsageType
	}{
		{"request", UsageTypeRequest, UsageTypeRequest},
		{"response", UsageTypeResponse, UsageTypeResponse},
		{"parameter", UsageTypeParameter, UsageTypeParameter},
		{"header", UsageTypeHeader, UsageTypeHeader},
		{"callback", UsageTypeCallback, UsageTypeCallback},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := &RefGraph{
				operationRefs: map[string][]OperationRef{
					"Schema": {
						{Path: "/test", Method: "get", UsageType: tt.usageType},
					},
				},
			}

			ctx := buildRenameContext("Schema", "spec.yaml", 0, graph, PolicyFirstEncountered)
			if ctx.UsageType != tt.expected {
				t.Errorf("UsageType = %v, want %v", ctx.UsageType, tt.expected)
			}
		})
	}
}
