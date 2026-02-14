package joiner

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, tt.expected, got)
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
			assert.Equal(t, tt.expected, got)
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
			assert.Equal(t, tt.expected, got)
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
			assert.Equal(t, tt.expected, got)
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
			assert.Equal(t, tt.expected, got)
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
			assert.Equal(t, tt.expected, got)
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
			assert.Equal(t, tt.expected, got)
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
		assert.Contains(t, funcs, name, "renameFuncs() missing required function %q", name)
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
			require.NoError(t, err)

			var buf []byte
			w := &testWriter{buf: &buf}
			err = tmpl.Execute(w, tt.data)
			require.NoError(t, err)

			got := string(buf)
			assert.Equal(t, tt.expected, got)
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
			assert.Equal(t, tt.expected, got)
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
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ============================================================================
// Context Builder Tests
// ============================================================================

func TestBuildRenameContext_Basic(t *testing.T) {
	// No graph provided - only core fields should be populated
	ctx := buildRenameContext("MySchema", "/path/to/file.yaml", 2, nil, PolicyFirstEncountered)

	assert.Equal(t, "MySchema", ctx.Name)
	assert.Equal(t, "file", ctx.Source)
	assert.Equal(t, 2, ctx.Index)

	// Operation context should be empty
	assert.Empty(t, ctx.Path)
	assert.Empty(t, ctx.Method)
	assert.Empty(t, ctx.OperationID)
	assert.Empty(t, ctx.Tags)

	// Aggregate fields should be empty/zero
	assert.Empty(t, ctx.AllPaths)
	assert.Equal(t, 0, ctx.RefCount)
	assert.False(t, ctx.IsShared)
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
	assert.Equal(t, "User", ctx.Name)
	assert.Equal(t, "api_spec", ctx.Source)
	assert.Equal(t, 0, ctx.Index)

	// Operation context (from first/primary operation)
	assert.Equal(t, "/users/{id}", ctx.Path)
	assert.Equal(t, "get", ctx.Method)
	assert.Equal(t, "getUser", ctx.OperationID)
	assert.Equal(t, []string{"Users", "Public"}, ctx.Tags)
	assert.Equal(t, UsageType("response"), ctx.UsageType)
	assert.Equal(t, "200", ctx.StatusCode)
	assert.Equal(t, "application/json", ctx.MediaType)

	// Aggregate fields
	assert.Equal(t, 2, ctx.RefCount)
	assert.True(t, ctx.IsShared)
	assert.Len(t, ctx.AllPaths, 2)
	assert.Len(t, ctx.AllMethods, 2)
	assert.Len(t, ctx.AllOperationIDs, 2)
	assert.Len(t, ctx.AllTags, 2)
	assert.Equal(t, "users", ctx.PrimaryResource)
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
	assert.Equal(t, "UnreferencedSchema", ctx.Name)
	assert.Equal(t, "spec", ctx.Source)
	assert.Equal(t, 1, ctx.Index)

	// Operation context should be empty
	assert.Empty(t, ctx.Path)
	assert.Equal(t, 0, ctx.RefCount)
	assert.False(t, ctx.IsShared)
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
			assert.Equal(t, tt.expected, ctx.Source)
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

	assert.Equal(t, "/users", result.Path)
	assert.Equal(t, "getUsers", result.OperationID)
}

func TestSelectPrimaryOperation_Alphabetical(t *testing.T) {
	refs := []OperationRef{
		{Path: "/users", Method: "get", OperationID: "getUsers"},
		{Path: "/orders", Method: "post", OperationID: "createOrder"},
		{Path: "/api", Method: "get", OperationID: "getApi"},
	}

	result := selectPrimaryOperation(refs, PolicyAlphabetical)

	// /api + get = "/apiget" is alphabetically first
	assert.Equal(t, "/api", result.Path)
	assert.Equal(t, "getApi", result.OperationID)
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
			assert.Equal(t, tt.wantPath, result.Path)
			assert.Equal(t, tt.wantID, result.OperationID)
		})
	}
}

func TestSelectPrimaryOperation_EmptyRefs(t *testing.T) {
	result := selectPrimaryOperation([]OperationRef{}, PolicyFirstEncountered)
	assert.Empty(t, result.Path)

	result = selectPrimaryOperation(nil, PolicyAlphabetical)
	assert.Empty(t, result.Path)
}

func TestSelectPrimaryOperation_UnknownPolicy(t *testing.T) {
	refs := []OperationRef{
		{Path: "/users", Method: "get"},
	}

	// Unknown policy should fall back to first
	result := selectPrimaryOperation(refs, PrimaryOperationPolicy(999))
	assert.Equal(t, "/users", result.Path)
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
			assert.Equal(t, tt.expected, got)
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
			assert.Equal(t, tt.expected, got)
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
		assert.Nil(t, result)
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
		assert.Len(t, result, 2)
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
		require.Len(t, result, 1)
		assert.Equal(t, "/users", result[0].Path)
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

		assert.Equal(t, len(result1), len(result2), "Cached result differs from original")
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
		assert.Len(t, result, 1)
	})

	t.Run("no refs for schema", func(t *testing.T) {
		g := &RefGraph{
			operationRefs: map[string][]OperationRef{
				"Other": {{Path: "/other", Method: "get"}},
			},
		}

		result := g.ResolveLineage("Unknown")
		assert.Empty(t, result)
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
		assert.Contains(t, funcs, name, "renameFuncs() missing function %q", name)
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
	assert.False(t, ctx.IsShared, "IsShared = true, want false for single reference")
	assert.Equal(t, 1, ctx.RefCount)
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
			assert.Equal(t, tt.expected, ctx.UsageType)
		})
	}
}
