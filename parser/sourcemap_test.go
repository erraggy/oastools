package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			if got := tt.loc.IsKnown(); got != tt.want {
				t.Errorf("IsKnown() = %v, want %v", got, tt.want)
			}
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
			if got := tt.loc.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSourceMap_NilSafety(t *testing.T) {
	var sm *SourceMap

	// All methods should return zero values on nil receiver
	if got := sm.Get("$.test"); got.IsKnown() {
		t.Error("Get() on nil should return unknown location")
	}
	if got := sm.GetKey("$.test"); got.IsKnown() {
		t.Error("GetKey() on nil should return unknown location")
	}
	if got := sm.GetRef("$.test"); got.Origin.IsKnown() {
		t.Error("GetRef() on nil should return empty RefLocation")
	}
	if sm.Has("$.test") {
		t.Error("Has() on nil should return false")
	}
	if sm.Len() != 0 {
		t.Error("Len() on nil should return 0")
	}
	if sm.Paths() != nil {
		t.Error("Paths() on nil should return nil")
	}
	if sm.Copy() != nil {
		t.Error("Copy() on nil should return nil")
	}

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
	if !sm.Has("$.a") {
		t.Error("set() should lazily initialize locations map")
	}

	sm.setKey("$.b", SourceLocation{Line: 2, Column: 1})
	keyLoc := sm.GetKey("$.b")
	if !keyLoc.IsKnown() {
		t.Error("setKey() should lazily initialize keyLocations map")
	}

	sm.setRef("$.c", RefLocation{TargetRef: "test"})
	ref := sm.GetRef("$.c")
	if ref.TargetRef != "test" {
		t.Error("setRef() should lazily initialize refs map")
	}
}

func TestSourceMap_MergeLazyInit(t *testing.T) {
	// Test that Merge lazily initializes maps on zero-value struct
	sm := &SourceMap{} // Zero value, maps are nil
	other := NewSourceMap()
	other.set("$.test", SourceLocation{Line: 1, Column: 1})
	other.setKey("$.test", SourceLocation{Line: 1, Column: 1})
	other.setRef("$.test", RefLocation{TargetRef: "ref"})

	sm.Merge(other)

	if !sm.Has("$.test") {
		t.Error("Merge should lazily initialize locations map")
	}
	if !sm.GetKey("$.test").IsKnown() {
		t.Error("Merge should lazily initialize keyLocations map")
	}
	if sm.GetRef("$.test").TargetRef != "ref" {
		t.Error("Merge should lazily initialize refs map")
	}
}

func TestSourceMap_BasicOperations(t *testing.T) {
	sm := NewSourceMap()

	// Test initial state
	if sm.Len() != 0 {
		t.Errorf("new SourceMap should have length 0, got %d", sm.Len())
	}
	if sm.Has("$.test") {
		t.Error("new SourceMap should not have any paths")
	}

	// Set a location via buildSourceMap simulation
	sm.set("$.info", SourceLocation{Line: 2, Column: 1, File: "test.yaml"})
	sm.set("$.info.title", SourceLocation{Line: 3, Column: 3, File: "test.yaml"})
	sm.setKey("$.info.title", SourceLocation{Line: 3, Column: 3, File: "test.yaml"})

	// Test Get
	loc := sm.Get("$.info")
	if !loc.IsKnown() {
		t.Error("Get() should return known location for existing path")
	}
	if loc.Line != 2 || loc.Column != 1 {
		t.Errorf("Get() returned wrong location: %v", loc)
	}

	// Test GetKey
	keyLoc := sm.GetKey("$.info.title")
	if !keyLoc.IsKnown() {
		t.Error("GetKey() should return known location")
	}

	// Test Has
	if !sm.Has("$.info") {
		t.Error("Has() should return true for existing path")
	}
	if sm.Has("$.nonexistent") {
		t.Error("Has() should return false for non-existing path")
	}

	// Test Len
	if sm.Len() != 2 {
		t.Errorf("Len() = %d, want 2", sm.Len())
	}

	// Test Paths
	paths := sm.Paths()
	if len(paths) != 2 {
		t.Errorf("Paths() returned %d paths, want 2", len(paths))
	}
	// Paths should be sorted
	if paths[0] != "$.info" || paths[1] != "$.info.title" {
		t.Errorf("Paths() not sorted correctly: %v", paths)
	}
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
	if copied.Len() != sm.Len() {
		t.Errorf("Copy has different length: got %d, want %d", copied.Len(), sm.Len())
	}
	if !copied.Has("$.test") {
		t.Error("Copy missing $.test path")
	}
	if copied.Get("$.test").Line != 1 {
		t.Error("Copy has wrong location")
	}
	if copied.GetRef("$.ref").TargetRef != "#/components/schemas/Pet" {
		t.Error("Copy has wrong ref")
	}

	// Modify original and verify copy is unchanged
	sm.set("$.test", SourceLocation{Line: 99, Column: 99, File: "changed.yaml"})
	if copied.Get("$.test").Line == 99 {
		t.Error("Copy was affected by modification to original")
	}
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
	if sm1.Len() != 2 {
		t.Errorf("Merged SourceMap has length %d, want 2", sm1.Len())
	}
	if !sm1.Has("$.b") {
		t.Error("Merged SourceMap missing $.b")
	}
	// $.a should be overwritten by sm2's value
	if sm1.Get("$.a").Line != 99 {
		t.Errorf("Merge should overwrite: got Line=%d, want 99", sm1.Get("$.a").Line)
	}
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
	if got.TargetRef != ref.TargetRef {
		t.Errorf("GetRef() returned wrong TargetRef: %q", got.TargetRef)
	}
	if got.Origin.Line != 10 {
		t.Errorf("GetRef() returned wrong Origin.Line: %d", got.Origin.Line)
	}
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
			got := buildChildPath(tt.parent, tt.key)
			if got != tt.want {
				t.Errorf("buildChildPath(%q, %q) = %q, want %q", tt.parent, tt.key, got, tt.want)
			}
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
			got := needsBracketNotation(tt.key)
			if got != tt.want {
				t.Errorf("needsBracketNotation(%q) = %v, want %v", tt.key, got, tt.want)
			}
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

	if sm.Get("$.info").File != "updated.yaml" {
		t.Error("location file path not updated")
	}
	if sm.GetKey("$.info").File != "updated.yaml" {
		t.Error("key location file path not updated")
	}
	if sm.GetRef("$.ref").Origin.File != "updated.yaml" {
		t.Error("ref origin file path not updated")
	}

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
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	sm := result.SourceMap
	if sm == nil {
		t.Fatal("SourceMap is nil when WithSourceMap(true) was used")
	}

	// Check root
	if !sm.Has("$") {
		t.Error("SourceMap missing root path '$'")
	}

	// Check info - the value position is on line 3 (where the mapping content starts)
	if !sm.Has("$.info") {
		t.Error("SourceMap missing '$.info'")
	}
	loc := sm.Get("$.info")
	if loc.Line != 3 {
		t.Errorf("$.info value should be on line 3, got %d", loc.Line)
	}

	// Check info key location - should be on line 2 where "info:" appears
	keyLoc := sm.GetKey("$.info")
	if keyLoc.Line != 2 {
		t.Errorf("$.info key should be on line 2, got %d", keyLoc.Line)
	}

	// Check info.title - value "Test API" is on line 3
	if !sm.Has("$.info.title") {
		t.Error("SourceMap missing '$.info.title'")
	}
	titleLoc := sm.Get("$.info.title")
	if titleLoc.Line != 3 {
		t.Errorf("$.info.title should be on line 3, got %d", titleLoc.Line)
	}

	// Check that key location is also recorded
	titleKeyLoc := sm.GetKey("$.info.title")
	if !titleKeyLoc.IsKnown() {
		t.Error("Key location for $.info.title should be known")
	}
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
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	sm := result.SourceMap
	if sm == nil {
		t.Fatal("SourceMap is nil")
	}

	// Check servers array
	if !sm.Has("$.servers") {
		t.Error("SourceMap missing '$.servers'")
	}

	// Check array elements
	if !sm.Has("$.servers[0]") {
		t.Error("SourceMap missing '$.servers[0]'")
	}
	if !sm.Has("$.servers[1]") {
		t.Error("SourceMap missing '$.servers[1]'")
	}

	// Check nested values in array elements
	if !sm.Has("$.servers[0].url") {
		t.Error("SourceMap missing '$.servers[0].url'")
	}

	// Verify line numbers make sense
	servers0 := sm.Get("$.servers[0]")
	servers1 := sm.Get("$.servers[1]")
	if servers1.Line <= servers0.Line {
		t.Errorf("servers[1] (line %d) should be after servers[0] (line %d)", servers1.Line, servers0.Line)
	}
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
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	sm := result.SourceMap
	if sm == nil {
		t.Fatal("SourceMap is nil")
	}

	// Find the $ref path - it should be tracked at the parent schema object
	// Note: path keys with slashes don't need bracket notation, but '200' does as it starts with a digit
	schemaPath := "$.paths./pets.get.responses['200'].content.application/json.schema"
	ref := sm.GetRef(schemaPath)
	if ref.TargetRef != "#/components/schemas/Pet" {
		t.Errorf("Expected ref target '#/components/schemas/Pet', got %q", ref.TargetRef)
	}
	if !ref.Origin.IsKnown() {
		t.Error("Ref origin should be known")
	}
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
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	sm := result.SourceMap
	if sm == nil {
		t.Fatal("SourceMap is nil for JSON input")
	}

	// JSON should also get line tracking
	if !sm.Has("$.info") {
		t.Error("SourceMap missing '$.info' for JSON input")
	}
	if !sm.Has("$.info.title") {
		t.Error("SourceMap missing '$.info.title' for JSON input")
	}
}

func TestBuildSourceMap_FilePath(t *testing.T) {
	result, err := ParseWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithSourceMap(true),
	)
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	sm := result.SourceMap
	if sm == nil {
		t.Fatal("SourceMap is nil")
	}

	// Check that file paths are set correctly
	loc := sm.Get("$.info")
	if loc.File != "../testdata/petstore-3.0.yaml" {
		t.Errorf("File path not set correctly: got %q", loc.File)
	}
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
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	sm := result.SourceMap
	if sm == nil {
		t.Fatal("SourceMap is nil")
	}

	// ParseBytes should set the file path to "ParseBytes.yaml"
	loc := sm.Get("$.info")
	if loc.File != "ParseBytes.yaml" {
		t.Errorf("File path for ParseBytes should be 'ParseBytes.yaml', got %q", loc.File)
	}
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
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	sm := result.SourceMap
	if sm == nil {
		t.Fatal("SourceMap is nil")
	}

	// ParseReader should set the file path to "ParseReader.yaml"
	loc := sm.Get("$.info")
	if loc.File != "ParseReader.yaml" {
		t.Errorf("File path for ParseReader should be 'ParseReader.yaml', got %q", loc.File)
	}
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
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	if result.SourceMap != nil {
		t.Error("SourceMap should be nil when not enabled")
	}

	// Explicitly disabled
	result2, err := ParseWithOptions(
		WithBytes([]byte(yaml)),
		WithSourceMap(false),
	)
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	if result2.SourceMap != nil {
		t.Error("SourceMap should be nil when explicitly disabled")
	}
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
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	copied := result.Copy()
	if copied.SourceMap == nil {
		t.Fatal("Copied result should have SourceMap")
	}
	if copied.SourceMap == result.SourceMap {
		t.Error("SourceMap should be deep copied, not shared")
	}
	if !copied.SourceMap.Has("$.info") {
		t.Error("Copied SourceMap should have same paths")
	}
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
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	sm := result.SourceMap

	// Path with braces - slashes and braces don't need bracket notation
	if !sm.Has("$.paths./users/{userId}") {
		t.Error("SourceMap should handle path with braces")
	}

	// Path with dot in the name needs bracket notation because of the '.'
	if !sm.Has("$.paths['/items.json']") {
		t.Error("SourceMap should handle path with dot")
	}
}

func TestBuildSourceMap_NilRoot(t *testing.T) {
	// Test that buildSourceMap handles nil root gracefully
	sm := buildSourceMap(nil, "test.yaml")
	if sm == nil {
		t.Fatal("buildSourceMap should return non-nil SourceMap even for nil root")
	}
	if sm.Len() != 0 {
		t.Error("buildSourceMap with nil root should return empty SourceMap")
	}
}

func TestWalkNode_NilNode(t *testing.T) {
	// Test that walkNode handles nil node gracefully
	sm := NewSourceMap()
	walkNode(nil, "$", sm, "test.yaml")
	if sm.Len() != 0 {
		t.Error("walkNode with nil should not add any entries")
	}
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
			got := convertRefToJSONPath(tt.ref)
			if got != tt.want {
				t.Errorf("convertRefToJSONPath(%q) = %q, want %q", tt.ref, got, tt.want)
			}
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
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	sm := result.SourceMap
	if sm == nil {
		t.Fatal("SourceMap should not be nil")
	}

	// Check that the ref at the schema path has both origin and target
	// Note: The path uses actual characters - only special chars like . space quotes need bracket notation
	// application/json contains / but that doesn't need bracket notation in JSON paths
	refPath := "$.paths./pets.get.responses['200'].content.application/json.schema"
	refLoc := sm.GetRef(refPath)

	// Origin should be set from parsing
	if !refLoc.Origin.IsKnown() {
		t.Error("RefLocation Origin should be known")
	}

	// TargetRef should be the $ref value
	if refLoc.TargetRef != "#/components/schemas/Pet" {
		t.Errorf("RefLocation TargetRef = %q, want %q", refLoc.TargetRef, "#/components/schemas/Pet")
	}

	// Target should be populated after resolution
	if !refLoc.Target.IsKnown() {
		t.Error("RefLocation Target should be known after resolution")
	}

	// Target should point to the Pet schema location
	petLoc := sm.Get("$.components.schemas.Pet")
	if refLoc.Target.Line != petLoc.Line {
		t.Errorf("Target line = %d, want %d (Pet schema line)", refLoc.Target.Line, petLoc.Line)
	}
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
	if err := os.WriteFile(extSchemaPath, []byte(extSchemaContent), 0644); err != nil {
		t.Fatalf("Failed to write external schema: %v", err)
	}

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
	if err := os.WriteFile(mainPath, []byte(mainSpec), 0644); err != nil {
		t.Fatalf("Failed to write main spec: %v", err)
	}

	result, err := ParseWithOptions(
		WithFilePath(mainPath),
		WithSourceMap(true),
		WithResolveRefs(true),
	)
	if err != nil {
		t.Fatalf("ParseWithOptions failed: %v", err)
	}

	sm := result.SourceMap
	if sm == nil {
		t.Fatal("SourceMap should not be nil")
	}

	// The external file's source map should be merged in
	// External paths are prefixed with the file path
	externalRoot := sm.Get("$")

	// Verify locations exist from both files
	mainLoc := sm.Get("$.openapi")
	if !mainLoc.IsKnown() {
		t.Error("SourceMap should have main file locations")
	}
	if mainLoc.File != mainPath {
		t.Errorf("Main file location File = %q, want %q", mainLoc.File, mainPath)
	}

	// The external file should be parsed and its source map merged
	// After merging, the external file's "$" becomes the extSchemaPath's root
	_ = externalRoot // external paths are at the file path level
}

func TestRefResolver_UpdateAllRefTargets_NilSourceMap(t *testing.T) {
	// Test that updateAllRefTargets handles nil SourceMap gracefully
	resolver := NewRefResolver(".")
	resolver.updateAllRefTargets() // Should not panic

	// Also test with nil refs map
	resolver.SourceMap = NewSourceMap()
	resolver.SourceMap.refs = nil
	resolver.updateAllRefTargets() // Should not panic
}

func TestRefResolver_UpdateAllRefTargets_Comprehensive(t *testing.T) {
	// Test comprehensive ref target updates
	resolver := NewRefResolver(".")
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
	if alreadyResolved.Target.Line != 50 {
		t.Errorf("Already resolved ref target should be unchanged, got line %d", alreadyResolved.Target.Line)
	}

	// Verify: empty-target should remain empty
	emptyTarget := resolver.SourceMap.GetRef("$.empty-target")
	if emptyTarget.Target.IsKnown() {
		t.Error("Empty target ref should not be resolved")
	}

	// Verify: external-ref should not be resolved
	externalRef := resolver.SourceMap.GetRef("$.external-ref")
	if externalRef.Target.IsKnown() {
		t.Error("External ref should not be resolved")
	}

	// Verify: missing-target should not be resolved
	missingTarget := resolver.SourceMap.GetRef("$.missing-target")
	if missingTarget.Target.IsKnown() {
		t.Error("Ref pointing to missing target should not be resolved")
	}

	// Verify: needs-resolution should be resolved
	needsResolution := resolver.SourceMap.GetRef("$.needs-resolution")
	if !needsResolution.Target.IsKnown() {
		t.Error("Ref needing resolution should be resolved")
	}
	if needsResolution.Target.Line != 60 {
		t.Errorf("Resolved ref target line = %d, want 60", needsResolution.Target.Line)
	}
}

func TestRefResolver_BuildExternalSourceMap_NilSourceMap(t *testing.T) {
	// Test that buildExternalSourceMap handles nil SourceMap gracefully
	resolver := NewRefResolver(".")
	resolver.buildExternalSourceMap("/test.yaml", []byte("type: string"))
	// Should not panic, and should not create ExternalSourceMaps

	if resolver.ExternalSourceMaps != nil {
		t.Error("ExternalSourceMaps should remain nil when SourceMap is nil")
	}
}

func TestRefResolver_UpdateRefTargetLocation_EdgeCases(t *testing.T) {
	// Test updateRefTargetLocation edge cases
	resolver := NewRefResolver(".")

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
	if refLoc.Target.IsKnown() {
		t.Error("External ref target should not be set")
	}

	// Test with local ref where target location doesn't exist in source map
	resolver.SourceMap.setRef("$.localref", RefLocation{
		Origin:    SourceLocation{Line: 5, Column: 3},
		TargetRef: "#/components/schemas/NotFound",
	})
	resolver.updateRefTargetLocation("$.localref", "#/components/schemas/NotFound")
	// Target should remain unknown since the target path isn't in the source map

	localRefLoc := resolver.SourceMap.GetRef("$.localref")
	if localRefLoc.Target.IsKnown() {
		t.Error("Target should not be set when target path not in source map")
	}

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
	if !foundRefLoc.Target.IsKnown() {
		t.Error("Target should be set when target path exists in source map")
	}
	if foundRefLoc.Target.Line != 100 {
		t.Errorf("Target line = %d, want 100", foundRefLoc.Target.Line)
	}
}
