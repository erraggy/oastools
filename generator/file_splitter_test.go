package generator

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestNewFileSplitter(t *testing.T) {
	fs := NewFileSplitter()

	if fs.MaxLinesPerFile != 2000 {
		t.Errorf("expected MaxLinesPerFile = 2000, got %d", fs.MaxLinesPerFile)
	}
	if fs.MaxTypesPerFile != 200 {
		t.Errorf("expected MaxTypesPerFile = 200, got %d", fs.MaxTypesPerFile)
	}
	if fs.MaxOperationsPerFile != 100 {
		t.Errorf("expected MaxOperationsPerFile = 100, got %d", fs.MaxOperationsPerFile)
	}
	if !fs.SplitByTag {
		t.Error("expected SplitByTag = true")
	}
	if !fs.SplitByPathPrefix {
		t.Error("expected SplitByPathPrefix = true")
	}
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
			if got != tt.want {
				t.Errorf("needsSplit() = %v, want %v", got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("sanitizeGroupName(%q) = %q, want %q", tt.input, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("extractPathPrefix(%q) = %q, want %q", tt.path, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("extractRefName(%q) = %q, want %q", tt.ref, got, tt.want)
			}
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

	if len(groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(groups))
	}

	if len(groups["users"]) != 2 {
		t.Errorf("expected 2 operations in 'users' group, got %d", len(groups["users"]))
	}

	if len(groups["pets"]) != 2 {
		t.Errorf("expected 2 operations in 'pets' group, got %d", len(groups["pets"]))
	}

	if len(groups["default"]) != 1 {
		t.Errorf("expected 1 operation in 'default' group, got %d", len(groups["default"]))
	}
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

	if len(groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(groups))
	}

	if len(groups["users"]) != 2 {
		t.Errorf("expected 2 operations in 'users' group, got %d", len(groups["users"]))
	}

	if len(groups["pets"]) != 2 {
		t.Errorf("expected 2 operations in 'pets' group, got %d", len(groups["pets"]))
	}

	if len(groups["health"]) != 1 {
		t.Errorf("expected 1 operation in 'health' group, got %d", len(groups["health"]))
	}
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

	if plan.NeedsSplit {
		t.Error("expected NeedsSplit = false")
	}

	if plan.TotalOperations != 2 {
		t.Errorf("expected TotalOperations = 2, got %d", plan.TotalOperations)
	}

	if plan.TotalTypes != 2 {
		t.Errorf("expected TotalTypes = 2, got %d", plan.TotalTypes)
	}

	if len(plan.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(plan.Groups))
	}
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

	if !plan.NeedsSplit {
		t.Error("expected NeedsSplit = true")
	}

	if plan.TotalOperations != 4 {
		t.Errorf("expected TotalOperations = 4, got %d", plan.TotalOperations)
	}

	if len(plan.Groups) < 2 {
		t.Errorf("expected at least 2 groups, got %d", len(plan.Groups))
	}
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

	if plan.NeedsSplit {
		t.Error("expected NeedsSplit = false")
	}

	if plan.TotalOperations != 1 {
		t.Errorf("expected TotalOperations = 1, got %d", plan.TotalOperations)
	}

	if plan.TotalTypes != 1 {
		t.Errorf("expected TotalTypes = 1, got %d", plan.TotalTypes)
	}
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

	if len(groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(groups))
	}

	// Verify all operations are in a group
	totalOps := 0
	for _, ops := range groups {
		totalOps += len(ops)
	}
	if totalOps != 5 {
		t.Errorf("expected 5 total operations, got %d", totalOps)
	}
}

func TestFileSplitter_EmptyDocument(t *testing.T) {
	fs := NewFileSplitter()

	// Test OAS 3.x empty document
	doc3 := &parser.OAS3Document{}
	plan3 := fs.AnalyzeOAS3(doc3)

	if plan3.NeedsSplit {
		t.Error("expected NeedsSplit = false for empty document")
	}
	if plan3.TotalOperations != 0 {
		t.Errorf("expected TotalOperations = 0, got %d", plan3.TotalOperations)
	}

	// Test OAS 2.0 empty document
	doc2 := &parser.OAS2Document{}
	plan2 := fs.AnalyzeOAS2(doc2)

	if plan2.NeedsSplit {
		t.Error("expected NeedsSplit = false for empty document")
	}
	if plan2.TotalOperations != 0 {
		t.Errorf("expected TotalOperations = 0, got %d", plan2.TotalOperations)
	}
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
			if got != tt.want {
				t.Errorf("GroupNameToTypeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
