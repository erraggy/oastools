package generator

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

func TestNewFileSplitter(t *testing.T) {
	fs := NewFileSplitter()

	assert.Equal(t, 2000, fs.MaxLinesPerFile)
	assert.Equal(t, 200, fs.MaxTypesPerFile)
	assert.Equal(t, 100, fs.MaxOperationsPerFile)
	assert.True(t, fs.SplitByTag)
	assert.True(t, fs.SplitByPathPrefix)
}

func TestFileSplitter_NeedsSplit(t *testing.T) {
	tests := []struct {
		name       string
		fs         *FileSplitter
		operations int
		types      int
		lines      int
		want       bool
	}{
		{
			name:       "no split needed - all below limits",
			fs:         &FileSplitter{MaxOperationsPerFile: 100, MaxTypesPerFile: 200, MaxLinesPerFile: 2000},
			operations: 50,
			types:      100,
			lines:      1000,
			want:       false,
		},
		{
			name:       "split needed - operations exceed limit",
			fs:         &FileSplitter{MaxOperationsPerFile: 100, MaxTypesPerFile: 200, MaxLinesPerFile: 2000},
			operations: 150,
			types:      50,
			lines:      1000,
			want:       true,
		},
		{
			name:       "split needed - types exceed limit",
			fs:         &FileSplitter{MaxOperationsPerFile: 100, MaxTypesPerFile: 200, MaxLinesPerFile: 2000},
			operations: 50,
			types:      250,
			lines:      1000,
			want:       true,
		},
		{
			name:       "split needed - lines exceed limit",
			fs:         &FileSplitter{MaxOperationsPerFile: 100, MaxTypesPerFile: 200, MaxLinesPerFile: 2000},
			operations: 50,
			types:      100,
			lines:      2500,
			want:       true,
		},
		{
			name:       "no split - limits disabled (0)",
			fs:         &FileSplitter{MaxOperationsPerFile: 0, MaxTypesPerFile: 0, MaxLinesPerFile: 0},
			operations: 1000,
			types:      1000,
			lines:      50000,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fs.needsSplit(tt.operations, tt.types, tt.lines)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileSplitter_SanitizeGroupName(t *testing.T) {
	fs := NewFileSplitter()

	tests := []struct {
		input string
		want  string
	}{
		{"users", "users"},
		{"User Management", "user_management"},
		{"user-management", "user_management"},
		{"User_Management", "user_management"},
		{"API v2", "api_v2"},
		{"123numbers", "123numbers"},
		{"special!@#chars", "specialchars"},
		{"multiple___underscores", "multiple_underscores"},
		{"_leading_trailing_", "leading_trailing"},
		{"", "misc"},
		{"___", "misc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := fs.sanitizeGroupName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileSplitter_ExtractPathPrefix(t *testing.T) {
	fs := NewFileSplitter()

	tests := []struct {
		path string
		want string
	}{
		{"/users", "users"},
		{"/users/123", "users"},
		{"/api/v1/users", "api"},
		{"/{userId}/profile", "default"},
		{"/", "default"},
		{"", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := fs.extractPathPrefix(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileSplitter_ExtractRefName(t *testing.T) {
	fs := NewFileSplitter()

	tests := []struct {
		ref  string
		want string
	}{
		{"#/components/schemas/User", "User"},
		{"#/components/schemas/UserResponse", "UserResponse"},
		{"#/definitions/User", "User"},
		{"#/definitions/Pet", "Pet"},
		{"User", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got := fs.extractRefName(tt.ref)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileSplitter_GroupByTag(t *testing.T) {
	fs := NewFileSplitter()

	operations := []*OperationInfo{
		{OperationID: "listUsers", Tags: []string{"users"}},
		{OperationID: "getUser", Tags: []string{"users"}},
		{OperationID: "listPets", Tags: []string{"pets"}},
		{OperationID: "getPet", Tags: []string{"pets", "animals"}},
		{OperationID: "getHealth", Tags: nil},
	}

	groups := fs.groupByTag(operations)

	assert.Len(t, groups, 3)
	assert.Len(t, groups["users"], 2)
	assert.Len(t, groups["pets"], 2)
	assert.Len(t, groups["default"], 1)
}

func TestFileSplitter_GroupByPathPrefix(t *testing.T) {
	fs := NewFileSplitter()

	operations := []*OperationInfo{
		{OperationID: "listUsers", Path: "/users"},
		{OperationID: "getUser", Path: "/users/{id}"},
		{OperationID: "listPets", Path: "/pets"},
		{OperationID: "getPet", Path: "/pets/{id}"},
		{OperationID: "getHealth", Path: "/health"},
	}

	groups := fs.groupByPathPrefix(operations)

	assert.Len(t, groups, 3)
	assert.Len(t, groups["users"], 2)
	assert.Len(t, groups["pets"], 2)
	assert.Len(t, groups["health"], 1)
}

func TestFileSplitter_AnalyzeOAS3_NoSplit(t *testing.T) {
	fs := NewFileSplitter()
	fs.MaxOperationsPerFile = 100
	fs.MaxTypesPerFile = 200
	fs.MaxLinesPerFile = 2000

	doc := &parser.OAS3Document{
		Paths: map[string]*parser.PathItem{
			"/users": {
				Get: &parser.Operation{OperationID: "listUsers"},
			},
			"/pets": {
				Get: &parser.Operation{OperationID: "listPets"},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"User": {Type: "object"},
				"Pet":  {Type: "object"},
			},
		},
	}

	plan := fs.AnalyzeOAS3(doc)

	assert.False(t, plan.NeedsSplit)
	assert.Equal(t, 2, plan.TotalOperations)
	assert.Equal(t, 2, plan.TotalTypes)
	assert.Len(t, plan.Groups, 1)
}

func TestFileSplitter_AnalyzeOAS3_WithSplit(t *testing.T) {
	fs := NewFileSplitter()
	fs.MaxOperationsPerFile = 2 // Force split
	fs.MaxTypesPerFile = 200
	fs.MaxLinesPerFile = 2000

	doc := &parser.OAS3Document{
		Paths: map[string]*parser.PathItem{
			"/users": {
				Get:  &parser.Operation{OperationID: "listUsers", Tags: []string{"users"}},
				Post: &parser.Operation{OperationID: "createUser", Tags: []string{"users"}},
			},
			"/pets": {
				Get:  &parser.Operation{OperationID: "listPets", Tags: []string{"pets"}},
				Post: &parser.Operation{OperationID: "createPet", Tags: []string{"pets"}},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"User": {Type: "object"},
				"Pet":  {Type: "object"},
			},
		},
	}

	plan := fs.AnalyzeOAS3(doc)

	assert.True(t, plan.NeedsSplit)
	assert.Equal(t, 4, plan.TotalOperations)
	assert.GreaterOrEqual(t, len(plan.Groups), 2)
}

func TestFileSplitter_AnalyzeOAS2_NoSplit(t *testing.T) {
	fs := NewFileSplitter()
	fs.MaxOperationsPerFile = 100
	fs.MaxTypesPerFile = 200
	fs.MaxLinesPerFile = 2000

	doc := &parser.OAS2Document{
		Paths: map[string]*parser.PathItem{
			"/users": {
				Get: &parser.Operation{OperationID: "listUsers"},
			},
		},
		Definitions: map[string]*parser.Schema{
			"User": {Type: "object"},
		},
	}

	plan := fs.AnalyzeOAS2(doc)

	assert.False(t, plan.NeedsSplit)
	assert.Equal(t, 1, plan.TotalOperations)
	assert.Equal(t, 1, plan.TotalTypes)
}

func TestFileSplitter_GroupAlphabetically(t *testing.T) {
	fs := NewFileSplitter()

	operations := []*OperationInfo{
		{OperationID: "aOperation"},
		{OperationID: "bOperation"},
		{OperationID: "cOperation"},
		{OperationID: "dOperation"},
		{OperationID: "eOperation"},
	}

	groups := fs.groupAlphabetically(operations, 2)

	assert.Len(t, groups, 3)

	// Verify all operations are in a group
	totalOps := 0
	for _, ops := range groups {
		totalOps += len(ops)
	}
	assert.Equal(t, 5, totalOps)
}

func TestFileSplitter_EmptyDocument(t *testing.T) {
	fs := NewFileSplitter()

	// Test OAS 3.x empty document
	doc3 := &parser.OAS3Document{}
	plan3 := fs.AnalyzeOAS3(doc3)

	assert.False(t, plan3.NeedsSplit)
	assert.Equal(t, 0, plan3.TotalOperations)

	// Test OAS 2.0 empty document
	doc2 := &parser.OAS2Document{}
	plan2 := fs.AnalyzeOAS2(doc2)

	assert.False(t, plan2.NeedsSplit)
	assert.Equal(t, 0, plan2.TotalOperations)
}

func TestGroupNameToTypeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "Users"},
		{"user_management", "UserManagement"},
		{"api-v2", "ApiV2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := GroupNameToTypeName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
