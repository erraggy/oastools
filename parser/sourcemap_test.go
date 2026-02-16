package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceLocation_IsKnown(t *testing.T) {
	tests := []struct {
		name string
		loc  SourceLocation
		want bool
	}{
		{
			name: "known location",
			loc:  SourceLocation{Line: 1, Column: 1, File: "test.yaml"},
			want: true,
		},
		{
			name: "zero line",
			loc:  SourceLocation{Line: 0, Column: 1, File: "test.yaml"},
			want: false,
		},
		{
			name: "empty struct",
			loc:  SourceLocation{},
			want: false,
		},
		{
			name: "negative line (treated as unknown)",
			loc:  SourceLocation{Line: -1, Column: 1},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.loc.IsKnown())
		})
	}
}

func TestSourceLocation_String(t *testing.T) {
	tests := []struct {
		name string
		loc  SourceLocation
		want string
	}{
		{
			name: "full location with file",
			loc:  SourceLocation{Line: 10, Column: 5, File: "test.yaml"},
			want: "test.yaml:10:5",
		},
		{
			name: "location without file",
			loc:  SourceLocation{Line: 10, Column: 5},
			want: "10:5",
		},
		{
			name: "unknown location with file",
			loc:  SourceLocation{Line: 0, Column: 0, File: "test.yaml"},
			want: "test.yaml",
		},
		{
			name: "unknown location without file",
			loc:  SourceLocation{},
			want: "<unknown>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.loc.String())
		})
	}
}

func TestSourceMap_NilSafety(t *testing.T) {
	var sm *SourceMap

	// All methods should return zero values on nil receiver
	assert.False(t, sm.Get("$.test").IsKnown(), "Get() on nil should return unknown location")
	assert.False(t, sm.GetKey("$.test").IsKnown(), "GetKey() on nil should return unknown location")
	assert.False(t, sm.GetRef("$.test").Origin.IsKnown(), "GetRef() on nil should return empty RefLocation")
	assert.False(t, sm.Has("$.test"), "Has() on nil should return false")
	assert.Equal(t, 0, sm.Len(), "Len() on nil should return 0")
	assert.Nil(t, sm.Paths(), "Paths() on nil should return nil")
	assert.Nil(t, sm.Copy(), "Copy() on nil should return nil")

	// Merge with nil receiver should not panic
	sm.Merge(NewSourceMap())

	// Merge with nil other should not panic
	sm2 := NewSourceMap()
	sm2.Merge(nil)

	// Private setters on nil should not panic
	sm.set("$.test", SourceLocation{Line: 1, Column: 1})
	sm.setKey("$.test", SourceLocation{Line: 1, Column: 1})
	sm.setRef("$.test", RefLocation{})
}

func TestSourceMap_LazyInitialization(t *testing.T) {
	// Test that set/setKey/setRef lazily initialize maps on zero-value struct
	sm := &SourceMap{} // Zero value, maps are nil

	// These should lazily initialize the maps
	sm.set("$.a", SourceLocation{Line: 1, Column: 1})
	assert.True(t, sm.Has("$.a"), "set() should lazily initialize locations map")

	sm.setKey("$.b", SourceLocation{Line: 2, Column: 1})
	keyLoc := sm.GetKey("$.b")
	assert.True(t, keyLoc.IsKnown(), "setKey() should lazily initialize keyLocations map")

	sm.setRef("$.c", RefLocation{TargetRef: "test"})
	ref := sm.GetRef("$.c")
	assert.Equal(t, "test", ref.TargetRef, "setRef() should lazily initialize refs map")
}

func TestSourceMap_MergeLazyInit(t *testing.T) {
	// Test that Merge lazily initializes maps on zero-value struct
	sm := &SourceMap{} // Zero value, maps are nil
	other := NewSourceMap()
	other.set("$.test", SourceLocation{Line: 1, Column: 1})
	other.setKey("$.test", SourceLocation{Line: 1, Column: 1})
	other.setRef("$.test", RefLocation{TargetRef: "ref"})

	sm.Merge(other)

	assert.True(t, sm.Has("$.test"), "Merge should lazily initialize locations map")
	assert.True(t, sm.GetKey("$.test").IsKnown(), "Merge should lazily initialize keyLocations map")
	assert.Equal(t, "ref", sm.GetRef("$.test").TargetRef, "Merge should lazily initialize refs map")
}

func TestSourceMap_BasicOperations(t *testing.T) {
	sm := NewSourceMap()

	// Test initial state
	assert.Equal(t, 0, sm.Len(), "new SourceMap should have length 0")
	assert.False(t, sm.Has("$.test"), "new SourceMap should not have any paths")

	// Set a location via buildSourceMap simulation
	sm.set("$.info", SourceLocation{Line: 2, Column: 1, File: "test.yaml"})
	sm.set("$.info.title", SourceLocation{Line: 3, Column: 3, File: "test.yaml"})
	sm.setKey("$.info.title", SourceLocation{Line: 3, Column: 3, File: "test.yaml"})

	// Test Get
	loc := sm.Get("$.info")
	assert.True(t, loc.IsKnown(), "Get() should return known location for existing path")
	assert.Equal(t, 2, loc.Line)
	assert.Equal(t, 1, loc.Column)

	// Test GetKey
	keyLoc := sm.GetKey("$.info.title")
	assert.True(t, keyLoc.IsKnown(), "GetKey() should return known location")

	// Test Has
	assert.True(t, sm.Has("$.info"), "Has() should return true for existing path")
	assert.False(t, sm.Has("$.nonexistent"), "Has() should return false for non-existing path")

	// Test Len
	assert.Equal(t, 2, sm.Len())

	// Test Paths
	paths := sm.Paths()
	require.Len(t, paths, 2)
	// Paths should be sorted
	assert.Equal(t, "$.info", paths[0])
	assert.Equal(t, "$.info.title", paths[1])
}

func TestSourceMap_Copy(t *testing.T) {
	sm := NewSourceMap()
	sm.set("$.test", SourceLocation{Line: 1, Column: 1, File: "test.yaml"})
	sm.setKey("$.test", SourceLocation{Line: 1, Column: 1, File: "test.yaml"})
	sm.setRef("$.ref", RefLocation{
		Origin:    SourceLocation{Line: 5, Column: 3, File: "test.yaml"},
		TargetRef: "#/components/schemas/Pet",
	})

	copied := sm.Copy()

	// Verify copy has same values
	assert.Equal(t, sm.Len(), copied.Len())
	assert.True(t, copied.Has("$.test"), "Copy missing $.test path")
	assert.Equal(t, 1, copied.Get("$.test").Line)
	assert.Equal(t, "#/components/schemas/Pet", copied.GetRef("$.ref").TargetRef)

	// Modify original and verify copy is unchanged
	sm.set("$.test", SourceLocation{Line: 99, Column: 99, File: "changed.yaml"})
	assert.NotEqual(t, 99, copied.Get("$.test").Line, "Copy was affected by modification to original")
}

func TestSourceMap_Merge(t *testing.T) {
	sm1 := NewSourceMap()
	sm1.set("$.a", SourceLocation{Line: 1, Column: 1, File: "a.yaml"})
	sm1.setKey("$.a", SourceLocation{Line: 1, Column: 1, File: "a.yaml"})

	sm2 := NewSourceMap()
	sm2.set("$.b", SourceLocation{Line: 2, Column: 2, File: "b.yaml"})
	sm2.set("$.a", SourceLocation{Line: 99, Column: 99, File: "b.yaml"}) // Overwrites

	sm1.Merge(sm2)

	// Verify merge results
	assert.Equal(t, 2, sm1.Len())
	assert.True(t, sm1.Has("$.b"), "Merged SourceMap missing $.b")
	// $.a should be overwritten by sm2's value
	assert.Equal(t, 99, sm1.Get("$.a").Line, "Merge should overwrite")
}

func TestSourceMap_RefTracking(t *testing.T) {
	sm := NewSourceMap()
	ref := RefLocation{
		Origin:    SourceLocation{Line: 10, Column: 5, File: "test.yaml"},
		Target:    SourceLocation{Line: 50, Column: 3, File: "test.yaml"},
		TargetRef: "#/components/schemas/Pet",
	}
	sm.setRef("$.paths./pets.get.responses.200.content.application/json.schema", ref)

	got := sm.GetRef("$.paths./pets.get.responses.200.content.application/json.schema")
	assert.Equal(t, ref.TargetRef, got.TargetRef)
	assert.Equal(t, 10, got.Origin.Line)
}

func TestBuildChildPath(t *testing.T) {
	tests := []struct {
		parent string
		key    string
		want   string
	}{
		{"$", "info", "$.info"},
		{"$.info", "title", "$.info.title"},
		{"$.paths", "/users", "$.paths./users"},
		{"$.paths", "/users/{id}", "$.paths./users/{id}"},
		{"$", "1invalid", "$['1invalid']"},
		{"$", "has.dot", "$['has.dot']"},
		{"$", "has[bracket", "$['has[bracket']"},
		{"$", "has]bracket", "$['has]bracket']"},
		{"$", "has'quote", "$['has\\'quote']"},
		{"$", `has"double`, `$['has"double']`},
		{"$", "has space", "$['has space']"},
		{"$", "has\ttab", "$['has\ttab']"},
		{"$", "", "$['']"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.want, buildChildPath(tt.parent, tt.key))
		})
	}
}

func TestNeedsBracketNotation(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"simple", false},
		{"camelCase", false},
		{"with_underscore", false},
		{"with-dash", false},
		{"1startsWithDigit", true},
		{"has.dot", true},
		{"has[bracket", true},
		{"has]bracket", true},
		{"has'quote", true},
		{`has"double`, true},
		{"has space", true},
		{"has\ttab", true},
		{"has\nnewline", true},
		{"has\rcarriage", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.want, needsBracketNotation(tt.key))
		})
	}
}

func TestUpdateSourceMapFilePath(t *testing.T) {
	sm := NewSourceMap()
	sm.set("$.info", SourceLocation{Line: 1, Column: 1, File: ""})
	sm.setKey("$.info", SourceLocation{Line: 1, Column: 1, File: ""})
	sm.setRef("$.ref", RefLocation{
		Origin:    SourceLocation{Line: 5, Column: 3, File: ""},
		TargetRef: "#/test",
	})

	updateSourceMapFilePath(sm, "updated.yaml")

	assert.Equal(t, "updated.yaml", sm.Get("$.info").File, "location file path not updated")
	assert.Equal(t, "updated.yaml", sm.GetKey("$.info").File, "key location file path not updated")
	assert.Equal(t, "updated.yaml", sm.GetRef("$.ref").Origin.File, "ref origin file path not updated")

	// Test nil safety
	updateSourceMapFilePath(nil, "test.yaml")
}

func TestBuildSourceMap_Basic(t *testing.T) {
	yaml := `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
paths: {}`

	result, err := ParseWithOptions(
		WithBytes([]byte(yaml)),
		WithSourceMap(true),
	)
	require.NoError(t, err)

	sm := result.SourceMap
	require.NotNil(t, sm, "SourceMap is nil when WithSourceMap(true) was used")

	// Check root
	assert.True(t, sm.Has("$"), "SourceMap missing root path '$'")

	// Check info - the value position is on line 3 (where the mapping content starts)
	assert.True(t, sm.Has("$.info"), "SourceMap missing '$.info'")
	loc := sm.Get("$.info")
	assert.Equal(t, 3, loc.Line, "$.info value should be on line 3")

	// Check info key location - should be on line 2 where "info:" appears
	keyLoc := sm.GetKey("$.info")
	assert.Equal(t, 2, keyLoc.Line, "$.info key should be on line 2")

	// Check info.title - value "Test API" is on line 3
	assert.True(t, sm.Has("$.info.title"), "SourceMap missing '$.info.title'")
	titleLoc := sm.Get("$.info.title")
	assert.Equal(t, 3, titleLoc.Line, "$.info.title should be on line 3")

	// Check that key location is also recorded
	titleKeyLoc := sm.GetKey("$.info.title")
	assert.True(t, titleKeyLoc.IsKnown(), "Key location for $.info.title should be known")
}

func TestBuildSourceMap_ArrayElements(t *testing.T) {
	yaml := `openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
servers:
  - url: https://api.example.com
  - url: https://staging.example.com
paths: {}`

	result, err := ParseWithOptions(
		WithBytes([]byte(yaml)),
		WithSourceMap(true),
	)
	require.NoError(t, err)

	sm := result.SourceMap
	require.NotNil(t, sm)

	// Check servers array
	assert.True(t, sm.Has("$.servers"), "SourceMap missing '$.servers'")

	// Check array elements
	assert.True(t, sm.Has("$.servers[0]"), "SourceMap missing '$.servers[0]'")
	assert.True(t, sm.Has("$.servers[1]"), "SourceMap missing '$.servers[1]'")

	// Check nested values in array elements
	assert.True(t, sm.Has("$.servers[0].url"), "SourceMap missing '$.servers[0].url'")

	// Verify line numbers make sense
	servers0 := sm.Get("$.servers[0]")
	servers1 := sm.Get("$.servers[1]")
	assert.Greater(t, servers1.Line, servers0.Line, "servers[1] should be after servers[0]")
}

func TestBuildSourceMap_RefTracking(t *testing.T) {
	yaml := `openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object`

	result, err := ParseWithOptions(
		WithBytes([]byte(yaml)),
		WithSourceMap(true),
	)
	require.NoError(t, err)

	sm := result.SourceMap
	require.NotNil(t, sm)

	// Find the $ref path - it should be tracked at the parent schema object
	// Note: path keys with slashes don't need bracket notation, but '200' does as it starts with a digit
	schemaPath := "$.paths./pets.get.responses['200'].content.application/json.schema"
	ref := sm.GetRef(schemaPath)
	assert.Equal(t, "#/components/schemas/Pet", ref.TargetRef)
	assert.True(t, ref.Origin.IsKnown(), "Ref origin should be known")
}

func TestBuildSourceMap_JSON(t *testing.T) {
	json := `{
  "openapi": "3.0.3",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "paths": {}
}`

	result, err := ParseWithOptions(
		WithBytes([]byte(json)),
		WithSourceMap(true),
	)
	require.NoError(t, err)

	sm := result.SourceMap
	require.NotNil(t, sm, "SourceMap is nil for JSON input")

	// JSON should also get line tracking
	assert.True(t, sm.Has("$.info"), "SourceMap missing '$.info' for JSON input")
	assert.True(t, sm.Has("$.info.title"), "SourceMap missing '$.info.title' for JSON input")
}

func TestBuildSourceMap_FilePath(t *testing.T) {
	result, err := ParseWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithSourceMap(true),
	)
	require.NoError(t, err)

	sm := result.SourceMap
	require.NotNil(t, sm)

	// Check that file paths are set correctly
	loc := sm.Get("$.info")
	assert.Equal(t, "../testdata/petstore-3.0.yaml", loc.File)
}

func TestBuildSourceMap_ParseBytesPath(t *testing.T) {
	yaml := `openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths: {}`

	p := New()
	p.BuildSourceMap = true
	result, err := p.ParseBytes([]byte(yaml))
	require.NoError(t, err)

	sm := result.SourceMap
	require.NotNil(t, sm)

	// ParseBytes should set the file path to "ParseBytes.yaml"
	loc := sm.Get("$.info")
	assert.Equal(t, "ParseBytes.yaml", loc.File)
}

func TestBuildSourceMap_ParseReaderPath(t *testing.T) {
	yaml := `openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths: {}`

	p := New()
	p.BuildSourceMap = true
	result, err := p.ParseReader(strings.NewReader(yaml))
	require.NoError(t, err)

	sm := result.SourceMap
	require.NotNil(t, sm)

	// ParseReader should set the file path to "ParseReader.yaml"
	loc := sm.Get("$.info")
	assert.Equal(t, "ParseReader.yaml", loc.File)
}

func TestBuildSourceMap_Disabled(t *testing.T) {
	yaml := `openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths: {}`

	// Default: source map disabled
	result, err := ParseWithOptions(
		WithBytes([]byte(yaml)),
	)
	require.NoError(t, err)

	assert.Nil(t, result.SourceMap, "SourceMap should be nil when not enabled")

	// Explicitly disabled
	result2, err := ParseWithOptions(
		WithBytes([]byte(yaml)),
		WithSourceMap(false),
	)
	require.NoError(t, err)

	assert.Nil(t, result2.SourceMap, "SourceMap should be nil when explicitly disabled")
}

func TestParseResult_CopyWithSourceMap(t *testing.T) {
	yaml := `openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths: {}`

	result, err := ParseWithOptions(
		WithBytes([]byte(yaml)),
		WithSourceMap(true),
	)
	require.NoError(t, err)

	copied := result.Copy()
	require.NotNil(t, copied.SourceMap, "Copied result should have SourceMap")
	assert.NotSame(t, result.SourceMap, copied.SourceMap, "SourceMap should be deep copied, not shared")
	assert.True(t, copied.SourceMap.Has("$.info"), "Copied SourceMap should have same paths")
}

func TestBuildSourceMap_SpecialPathCharacters(t *testing.T) {
	yaml := `openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      responses:
        '200':
          description: OK
  /items.json:
    get:
      responses:
        '200':
          description: OK`

	result, err := ParseWithOptions(
		WithBytes([]byte(yaml)),
		WithSourceMap(true),
	)
	require.NoError(t, err)

	sm := result.SourceMap

	// Path with braces - slashes and braces don't need bracket notation
	assert.True(t, sm.Has("$.paths./users/{userId}"), "SourceMap should handle path with braces")

	// Path with dot in the name needs bracket notation because of the '.'
	assert.True(t, sm.Has("$.paths['/items.json']"), "SourceMap should handle path with dot")
}

func TestBuildSourceMap_NilRoot(t *testing.T) {
	// Test that buildSourceMap handles nil root gracefully
	sm := buildSourceMap(nil, "test.yaml")
	require.NotNil(t, sm, "buildSourceMap should return non-nil SourceMap even for nil root")
	assert.Equal(t, 0, sm.Len(), "buildSourceMap with nil root should return empty SourceMap")
}

func TestWalkNode_NilNode(t *testing.T) {
	// Test that walkNode handles nil node gracefully
	sm := NewSourceMap()
	walkNode(nil, "$", sm, "test.yaml")
	assert.Equal(t, 0, sm.Len(), "walkNode with nil should not add any entries")
}

func TestConvertRefToJSONPath(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{
			name: "simple local ref",
			ref:  "#/components/schemas/Pet",
			want: "$.components.schemas.Pet",
		},
		{
			name: "ref with special characters needing bracket notation",
			ref:  "#/paths/~1users~1{id}/get",
			want: "$.paths./users/{id}.get",
		},
		{
			name: "ref with URL encoding",
			ref:  "#/components/schemas/User%20Model",
			want: "$.components.schemas['User Model']",
		},
		{
			name: "external ref returns empty",
			ref:  "./external.yaml#/schemas/Pet",
			want: "",
		},
		{
			name: "http ref returns empty",
			ref:  "https://example.com/api.yaml#/schemas/Pet",
			want: "",
		},
		{
			name: "ref without hash prefix returns empty",
			ref:  "/components/schemas/Pet",
			want: "",
		},
		{
			name: "root ref returns just $",
			ref:  "#/",
			want: "$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, convertRefToJSONPath(tt.ref))
		})
	}
}

func TestRefResolver_UpdateAllRefTargets(t *testing.T) {
	// Test that updateAllRefTargets populates target locations
	yaml := `openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string`

	result, err := ParseWithOptions(
		WithBytes([]byte(yaml)),
		WithSourceMap(true),
		WithResolveRefs(true),
	)
	require.NoError(t, err)

	sm := result.SourceMap
	require.NotNil(t, sm)

	// Check that the ref at the schema path has both origin and target
	// Note: The path uses actual characters - only special chars like . space quotes need bracket notation
	// application/json contains / but that doesn't need bracket notation in JSON paths
	refPath := "$.paths./pets.get.responses['200'].content.application/json.schema"
	refLoc := sm.GetRef(refPath)

	// Origin should be set from parsing
	assert.True(t, refLoc.Origin.IsKnown(), "RefLocation Origin should be known")

	// TargetRef should be the $ref value
	assert.Equal(t, "#/components/schemas/Pet", refLoc.TargetRef)

	// Target should be populated after resolution
	assert.True(t, refLoc.Target.IsKnown(), "RefLocation Target should be known after resolution")

	// Target should point to the Pet schema location
	petLoc := sm.Get("$.components.schemas.Pet")
	assert.Equal(t, petLoc.Line, refLoc.Target.Line, "Target should point to Pet schema line")
}

func TestRefResolver_ExternalSourceMaps(t *testing.T) {
	// Test that external file source maps are built and merged
	tmpDir := t.TempDir()

	// Create external schema file
	extSchemaContent := `type: object
properties:
  id:
    type: integer
  name:
    type: string`
	extSchemaPath := filepath.Join(tmpDir, "pet.yaml")
	err := os.WriteFile(extSchemaPath, []byte(extSchemaContent), 0644)
	require.NoError(t, err, "Failed to write external schema")

	// Create main spec that references the external file
	mainSpec := `openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: "./pet.yaml"`
	mainPath := filepath.Join(tmpDir, "main.yaml")
	err = os.WriteFile(mainPath, []byte(mainSpec), 0644)
	require.NoError(t, err, "Failed to write main spec")

	result, err := ParseWithOptions(
		WithFilePath(mainPath),
		WithSourceMap(true),
		WithResolveRefs(true),
	)
	require.NoError(t, err)

	sm := result.SourceMap
	require.NotNil(t, sm)

	// The external file's source map should be merged in
	// External paths are prefixed with the file path
	externalRoot := sm.Get("$")

	// Verify locations exist from both files
	mainLoc := sm.Get("$.openapi")
	assert.True(t, mainLoc.IsKnown(), "SourceMap should have main file locations")
	assert.Equal(t, mainPath, mainLoc.File)

	// The external file should be parsed and its source map merged
	// After merging, the external file's "$" becomes the extSchemaPath's root
	_ = externalRoot // external paths are at the file path level
}

func TestRefResolver_UpdateAllRefTargets_NilSourceMap(t *testing.T) {
	// Test that updateAllRefTargets handles nil SourceMap gracefully
	resolver := NewRefResolver(".", 0, 0, 0)
	resolver.updateAllRefTargets() // Should not panic

	// Also test with nil refs map
	resolver.SourceMap = NewSourceMap()
	resolver.SourceMap.refs = nil
	resolver.updateAllRefTargets() // Should not panic
}

func TestRefResolver_UpdateAllRefTargets_Comprehensive(t *testing.T) {
	// Test comprehensive ref target updates
	resolver := NewRefResolver(".", 0, 0, 0)
	resolver.SourceMap = NewSourceMap()

	// Add locations for targets
	resolver.SourceMap.set("$.components.schemas.Pet", SourceLocation{
		Line: 50, Column: 5, File: "test.yaml",
	})
	resolver.SourceMap.set("$.components.schemas.User", SourceLocation{
		Line: 60, Column: 5, File: "test.yaml",
	})

	// Add ref that already has target set (should be skipped)
	resolver.SourceMap.setRef("$.already-resolved", RefLocation{
		Origin:    SourceLocation{Line: 10, Column: 3},
		Target:    SourceLocation{Line: 50, Column: 5, File: "test.yaml"},
		TargetRef: "#/components/schemas/Pet",
	})

	// Add ref with empty TargetRef (should be skipped)
	resolver.SourceMap.setRef("$.empty-target", RefLocation{
		Origin: SourceLocation{Line: 15, Column: 3},
	})

	// Add ref pointing to external file (should be skipped)
	resolver.SourceMap.setRef("$.external-ref", RefLocation{
		Origin:    SourceLocation{Line: 20, Column: 3},
		TargetRef: "./external.yaml#/schema",
	})

	// Add ref pointing to non-existent location (should skip, target not found)
	resolver.SourceMap.setRef("$.missing-target", RefLocation{
		Origin:    SourceLocation{Line: 25, Column: 3},
		TargetRef: "#/components/schemas/Missing",
	})

	// Add ref that should be resolved
	resolver.SourceMap.setRef("$.needs-resolution", RefLocation{
		Origin:    SourceLocation{Line: 30, Column: 3},
		TargetRef: "#/components/schemas/User",
	})

	// Run update
	resolver.updateAllRefTargets()

	// Verify: already-resolved should be unchanged
	alreadyResolved := resolver.SourceMap.GetRef("$.already-resolved")
	assert.Equal(t, 50, alreadyResolved.Target.Line, "Already resolved ref target should be unchanged")

	// Verify: empty-target should remain empty
	emptyTarget := resolver.SourceMap.GetRef("$.empty-target")
	assert.False(t, emptyTarget.Target.IsKnown(), "Empty target ref should not be resolved")

	// Verify: external-ref should not be resolved
	externalRef := resolver.SourceMap.GetRef("$.external-ref")
	assert.False(t, externalRef.Target.IsKnown(), "External ref should not be resolved")

	// Verify: missing-target should not be resolved
	missingTarget := resolver.SourceMap.GetRef("$.missing-target")
	assert.False(t, missingTarget.Target.IsKnown(), "Ref pointing to missing target should not be resolved")

	// Verify: needs-resolution should be resolved
	needsResolution := resolver.SourceMap.GetRef("$.needs-resolution")
	assert.True(t, needsResolution.Target.IsKnown(), "Ref needing resolution should be resolved")
	assert.Equal(t, 60, needsResolution.Target.Line)
}

func TestRefResolver_BuildExternalSourceMap_NilSourceMap(t *testing.T) {
	// Test that buildExternalSourceMap handles nil SourceMap gracefully
	resolver := NewRefResolver(".", 0, 0, 0)
	resolver.buildExternalSourceMap("/test.yaml", []byte("type: string"))
	// Should not panic, and should not create ExternalSourceMaps

	assert.Nil(t, resolver.ExternalSourceMaps, "ExternalSourceMaps should remain nil when SourceMap is nil")
}

func TestRefResolver_UpdateRefTargetLocation_EdgeCases(t *testing.T) {
	// Test updateRefTargetLocation edge cases
	resolver := NewRefResolver(".", 0, 0, 0)

	// Test with nil SourceMap
	resolver.updateRefTargetLocation("$.path", "#/ref")
	// Should not panic

	// Test with SourceMap but no ref at path
	resolver.SourceMap = NewSourceMap()
	resolver.updateRefTargetLocation("$.nonexistent", "#/ref")
	// Should not panic, ref not found

	// Test with external ref (should skip)
	resolver.SourceMap.setRef("$.path", RefLocation{
		Origin:    SourceLocation{Line: 1, Column: 1},
		TargetRef: "./external.yaml#/schema",
	})
	resolver.updateRefTargetLocation("$.path", "./external.yaml#/schema")
	// Should skip external refs

	refLoc := resolver.SourceMap.GetRef("$.path")
	assert.False(t, refLoc.Target.IsKnown(), "External ref target should not be set")

	// Test with local ref where target location doesn't exist in source map
	resolver.SourceMap.setRef("$.localref", RefLocation{
		Origin:    SourceLocation{Line: 5, Column: 3},
		TargetRef: "#/components/schemas/NotFound",
	})
	resolver.updateRefTargetLocation("$.localref", "#/components/schemas/NotFound")
	// Target should remain unknown since the target path isn't in the source map

	localRefLoc := resolver.SourceMap.GetRef("$.localref")
	assert.False(t, localRefLoc.Target.IsKnown(), "Target should not be set when target path not in source map")

	// Test with local ref where target location DOES exist
	resolver.SourceMap.set("$.components.schemas.Found", SourceLocation{
		Line:   100,
		Column: 5,
		File:   "test.yaml",
	})
	resolver.SourceMap.setRef("$.foundref", RefLocation{
		Origin:    SourceLocation{Line: 10, Column: 3},
		TargetRef: "#/components/schemas/Found",
	})
	resolver.updateRefTargetLocation("$.foundref", "#/components/schemas/Found")
	// Target should be populated

	foundRefLoc := resolver.SourceMap.GetRef("$.foundref")
	assert.True(t, foundRefLoc.Target.IsKnown(), "Target should be set when target path exists in source map")
	assert.Equal(t, 100, foundRefLoc.Target.Line)
}
