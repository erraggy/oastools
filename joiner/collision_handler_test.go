package joiner

import (
	"fmt"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

func TestCollisionType_Constants(t *testing.T) {
	// Verify all collision types are defined
	types := []CollisionType{
		CollisionTypeSchema,
		CollisionTypePath,
		CollisionTypeWebhook,
		CollisionTypeResponse,
		CollisionTypeParameter,
		CollisionTypeExample,
		CollisionTypeRequestBody,
		CollisionTypeHeader,
		CollisionTypeSecurityScheme,
		CollisionTypeLink,
		CollisionTypeCallback,
	}

	assert.Len(t, types, 11, "should have 11 collision types")

	// Verify each has a non-empty string value
	for _, ct := range types {
		assert.NotEmpty(t, string(ct), "collision type should have non-empty value")
	}
}

func TestResolutionAction_Constants(t *testing.T) {
	// Verify resolution actions are defined with correct iota values
	assert.Equal(t, ResolutionAction(0), ResolutionContinue)
	assert.Equal(t, ResolutionAction(1), ResolutionAcceptLeft)
	assert.Equal(t, ResolutionAction(2), ResolutionAcceptRight)
	assert.Equal(t, ResolutionAction(3), ResolutionRename)
	assert.Equal(t, ResolutionAction(4), ResolutionDeduplicate)
	assert.Equal(t, ResolutionAction(5), ResolutionFail)
	assert.Equal(t, ResolutionAction(6), ResolutionCustom)
}

func TestResolutionAction_String(t *testing.T) {
	tests := []struct {
		action   ResolutionAction
		expected string
	}{
		{ResolutionContinue, "continue"},
		{ResolutionAcceptLeft, "accept-left"},
		{ResolutionAcceptRight, "accept-right"},
		{ResolutionRename, "rename"},
		{ResolutionDeduplicate, "deduplicate"},
		{ResolutionFail, "fail"},
		{ResolutionCustom, "custom"},
		{ResolutionAction(99), "unknown"}, // Unknown value
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.action.String())
		})
	}
}

func TestCollisionContext_Fields(t *testing.T) {
	ctx := CollisionContext{
		Type:               CollisionTypeSchema,
		Name:               "User",
		JSONPath:           "$.components.schemas.User",
		LeftSource:         "base.yaml",
		LeftValue:          "left-schema",
		RightSource:        "overlay.yaml",
		RightValue:         "right-schema",
		ConfiguredStrategy: StrategyAcceptLeft,
	}

	assert.Equal(t, CollisionTypeSchema, ctx.Type)
	assert.Equal(t, "User", ctx.Name)
	assert.Equal(t, "$.components.schemas.User", ctx.JSONPath)
	assert.Equal(t, "base.yaml", ctx.LeftSource)
	assert.Equal(t, "left-schema", ctx.LeftValue)
	assert.Equal(t, "overlay.yaml", ctx.RightSource)
	assert.Equal(t, "right-schema", ctx.RightValue)
	assert.Equal(t, StrategyAcceptLeft, ctx.ConfiguredStrategy)
}

func TestCollisionResolution_Fields(t *testing.T) {
	res := CollisionResolution{
		Action:      ResolutionCustom,
		CustomValue: "merged-value",
		Message:     "custom merge applied",
	}

	assert.Equal(t, ResolutionCustom, res.Action)
	assert.Equal(t, "merged-value", res.CustomValue)
	assert.Equal(t, "custom merge applied", res.Message)
}

func TestCollisionHandler_FunctionType(t *testing.T) {
	// Verify CollisionHandler can be used as a function type
	var handler CollisionHandler = func(ctx CollisionContext) (CollisionResolution, error) {
		return CollisionResolution{
			Action:  ResolutionAcceptLeft,
			Message: "handled " + ctx.Name,
		}, nil
	}

	// Call the handler
	ctx := CollisionContext{
		Type:       CollisionTypeSchema,
		Name:       "TestSchema",
		LeftSource: "a.yaml",
	}
	res, err := handler(ctx)

	assert.NoError(t, err)
	assert.Equal(t, ResolutionAcceptLeft, res.Action)
	assert.Equal(t, "handled TestSchema", res.Message)
}

func TestSourceLocation_Fields(t *testing.T) {
	loc := SourceLocation{
		Line:   42,
		Column: 10,
	}

	assert.Equal(t, 42, loc.Line)
	assert.Equal(t, 10, loc.Column)
}

func TestCollisionContext_WithLocations(t *testing.T) {
	leftLoc := &SourceLocation{Line: 10, Column: 5}
	rightLoc := &SourceLocation{Line: 20, Column: 3}
	renameCtx := &RenameContext{Name: "User", Source: "api"}

	ctx := CollisionContext{
		Type:               CollisionTypeSchema,
		Name:               "User",
		JSONPath:           "$.components.schemas.User",
		LeftSource:         "base.yaml",
		LeftLocation:       leftLoc,
		LeftValue:          map[string]any{"type": "object"},
		RightSource:        "overlay.yaml",
		RightLocation:      rightLoc,
		RightValue:         map[string]any{"type": "object", "description": "A user"},
		RenameInfo:         renameCtx,
		ConfiguredStrategy: StrategyRenameRight,
	}

	assert.Equal(t, 10, ctx.LeftLocation.Line)
	assert.Equal(t, 5, ctx.LeftLocation.Column)
	assert.Equal(t, 20, ctx.RightLocation.Line)
	assert.Equal(t, 3, ctx.RightLocation.Column)
	assert.Equal(t, "User", ctx.RenameInfo.Name)
	assert.Equal(t, "api", ctx.RenameInfo.Source)
}

func TestResolutionHelpers(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() CollisionResolution
		expected CollisionResolution
	}{
		{
			name: "ContinueWithStrategy",
			fn:   ContinueWithStrategy,
			expected: CollisionResolution{
				Action: ResolutionContinue,
			},
		},
		{
			name: "AcceptLeft",
			fn:   AcceptLeft,
			expected: CollisionResolution{
				Action: ResolutionAcceptLeft,
			},
		},
		{
			name: "AcceptRight",
			fn:   AcceptRight,
			expected: CollisionResolution{
				Action: ResolutionAcceptRight,
			},
		},
		{
			name: "Rename",
			fn:   Rename,
			expected: CollisionResolution{
				Action: ResolutionRename,
			},
		},
		{
			name: "Deduplicate",
			fn:   Deduplicate,
			expected: CollisionResolution{
				Action: ResolutionDeduplicate,
			},
		},
		{
			name: "Fail",
			fn:   Fail,
			expected: CollisionResolution{
				Action: ResolutionFail,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			assert.Equal(t, tt.expected.Action, got.Action)
			assert.Empty(t, got.Message)
			assert.Nil(t, got.CustomValue)
		})
	}
}

func TestResolutionHelpersWithMessage(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) CollisionResolution
		message  string
		expected ResolutionAction
	}{
		{"ContinueWithStrategyWithMessage", ContinueWithStrategyWithMessage, "observed collision", ResolutionContinue},
		{"AcceptLeftWithMessage", AcceptLeftWithMessage, "kept base", ResolutionAcceptLeft},
		{"AcceptRightWithMessage", AcceptRightWithMessage, "overlay wins", ResolutionAcceptRight},
		{"RenameWithMessage", RenameWithMessage, "renamed to avoid conflict", ResolutionRename},
		{"DeduplicateWithMessage", DeduplicateWithMessage, "schemas are equivalent", ResolutionDeduplicate},
		{"FailWithMessage", FailWithMessage, "intentional failure", ResolutionFail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.message)
			assert.Equal(t, tt.expected, got.Action)
			assert.Equal(t, tt.message, got.Message)
		})
	}
}

func TestUseCustomValue(t *testing.T) {
	customSchema := map[string]string{"type": "merged"}

	got := UseCustomValue(customSchema)
	assert.Equal(t, ResolutionCustom, got.Action)
	assert.Equal(t, customSchema, got.CustomValue)
	assert.Empty(t, got.Message)

	gotWithMsg := UseCustomValueWithMessage(customSchema, "custom merge")
	assert.Equal(t, ResolutionCustom, gotWithMsg.Action)
	assert.Equal(t, customSchema, gotWithMsg.CustomValue)
	assert.Equal(t, "custom merge", gotWithMsg.Message)
}

// createTestOAS3Doc creates a test OAS 3.0 document with the given schemas.
// Each schema map entry creates a schema with that name and description.
func createTestOAS3Doc(sourcePath string, schemas map[string]string) parser.ParseResult {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(parser.Paths),
		Components: &parser.Components{
			Schemas: make(map[string]*parser.Schema),
		},
	}
	for name, desc := range schemas {
		doc.Components.Schemas[name] = &parser.Schema{
			Type:        "object",
			Description: desc,
		}
	}
	return parser.ParseResult{
		SourcePath: sourcePath,
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}
}

func TestCollisionHandler_InvokedOnSchemaCollision(t *testing.T) {
	var receivedCollision CollisionContext
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		receivedCollision = collision
		return AcceptLeft(), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	assert.Equal(t, CollisionTypeSchema, receivedCollision.Type)
	assert.Equal(t, "User", receivedCollision.Name)
	assert.Equal(t, "$.components.schemas.User", receivedCollision.JSONPath)
	assert.Equal(t, "base.yaml", receivedCollision.LeftSource)
	assert.Equal(t, "overlay.yaml", receivedCollision.RightSource)
	assert.NotNil(t, receivedCollision.LeftValue)
	assert.NotNil(t, receivedCollision.RightValue)
	assert.Equal(t, StrategyAcceptLeft, receivedCollision.ConfiguredStrategy)

	// Verify the resolution was applied (left was kept)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "base-user", oas3Doc.Components.Schemas["User"].Description)
}

func TestCollisionHandler_AcceptRightResolution(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return AcceptRight(), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft), // Would keep left, but handler overrides
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "overlay-user", oas3Doc.Components.Schemas["User"].Description)
}

func TestCollisionHandler_ContinueWithStrategy(t *testing.T) {
	handlerCalled := false
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		handlerCalled = true
		return ContinueWithStrategy(), nil // Defer to configured strategy
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithSchemaStrategy(StrategyAcceptRight), // Should take effect for schemas
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "overlay-user", oas3Doc.Components.Schemas["User"].Description)
}

func TestCollisionHandler_ErrorFallsBackToStrategy(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return CollisionResolution{}, fmt.Errorf("simulated handler error")
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft), // Fallback
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err, "join should succeed despite handler error")

	// Verify fallback to AcceptLeft occurred
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "base-user", oas3Doc.Components.Schemas["User"].Description)

	// Verify warning was recorded
	var foundWarning bool
	for _, warn := range result.StructuredWarnings {
		if warn.Category == WarnHandlerError {
			foundWarning = true
			assert.Contains(t, warn.Message, "simulated handler error")
		}
	}
	assert.True(t, foundWarning, "should have handler error warning")
}

func TestCollisionHandler_FailResolution(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return FailWithMessage("intentional failure from handler"), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "intentional failure from handler")
}

func TestCollisionHandler_RenameResolution(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return Rename(), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)

	// Original should be kept
	assert.Equal(t, "base-user", oas3Doc.Components.Schemas["User"].Description)

	// Renamed schema should exist
	var foundRenamed bool
	for name, schema := range oas3Doc.Components.Schemas {
		if name != "User" && schema.Description == "overlay-user" {
			foundRenamed = true
			break
		}
	}
	assert.True(t, foundRenamed, "should have a renamed schema with overlay-user description")
}

func TestCollisionHandler_DeduplicateResolution(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return Deduplicate(), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)

	// Only one User schema should exist (deduplicated keeps left)
	assert.Len(t, oas3Doc.Components.Schemas, 1)
	assert.Equal(t, "base-user", oas3Doc.Components.Schemas["User"].Description)

	// Should have a dedup warning
	var foundDedup bool
	for _, warn := range result.StructuredWarnings {
		if warn.Category == WarnSchemaDeduplicated {
			foundDedup = true
			break
		}
	}
	assert.True(t, foundDedup, "should have schema deduplicated warning")
}

func TestCollisionHandler_NotInvokedForNonCollision(t *testing.T) {
	handlerCalled := false
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		handlerCalled = true
		return AcceptLeft(), nil
	}

	// Different schema names, no collision
	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"Pet": "overlay-pet"})

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	assert.False(t, handlerCalled, "handler should not be called when there's no collision")
}

func TestCollisionHandler_WithMessageResolution(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return AcceptLeftWithMessage("keeping base schema due to policy"), nil
	}

	base := createTestOAS3Doc("base.yaml", map[string]string{"User": "base-user"})
	overlay := createTestOAS3Doc("overlay.yaml", map[string]string{"User": "overlay-user"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)

	// Should have a handler resolution warning with the message
	var foundResolution bool
	for _, warn := range result.StructuredWarnings {
		if warn.Category == WarnHandlerResolution {
			foundResolution = true
			assert.Contains(t, warn.Message, "keeping base schema due to policy")
			break
		}
	}
	assert.True(t, foundResolution, "should have handler resolution warning with message")
}

// createTestOAS3DocWithPaths creates a test OAS 3.0 document with the given schemas and paths.
// Each schema map entry creates a schema with that name and description.
// Each path creates a PathItem with a GET operation.
func createTestOAS3DocWithPaths(sourcePath string, schemas map[string]string, paths []string) parser.ParseResult {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(parser.Paths),
		Components: &parser.Components{
			Schemas: make(map[string]*parser.Schema),
		},
	}
	for name, desc := range schemas {
		doc.Components.Schemas[name] = &parser.Schema{
			Type:        "object",
			Description: desc,
		}
	}
	for _, path := range paths {
		doc.Paths[path] = &parser.PathItem{
			Get: &parser.Operation{
				OperationID: "get" + path,
				Responses:   &parser.Responses{},
			},
		}
	}
	return parser.ParseResult{
		SourcePath: sourcePath,
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}
}

func TestCollisionHandler_PathCollision(t *testing.T) {
	var receivedCollision CollisionContext
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		receivedCollision = collision
		return AcceptRight(), nil
	}

	base := createTestOAS3DocWithPaths("base.yaml", nil, []string{"/users"})
	overlay := createTestOAS3DocWithPaths("overlay.yaml", nil, []string{"/users"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	assert.Equal(t, CollisionTypePath, receivedCollision.Type)
	assert.Equal(t, "/users", receivedCollision.Name)
	assert.Contains(t, receivedCollision.JSONPath, "paths")
	assert.Equal(t, "base.yaml", receivedCollision.LeftSource)
	assert.Equal(t, "overlay.yaml", receivedCollision.RightSource)
	assert.NotNil(t, receivedCollision.LeftValue)
	assert.NotNil(t, receivedCollision.RightValue)

	// Verify overlay path was used (AcceptRight)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "get/users", oas3Doc.Paths["/users"].Get.OperationID)
}

func TestCollisionHandler_PathAcceptLeft(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return AcceptLeft(), nil
	}

	base := createTestOAS3DocWithPaths("base.yaml", nil, []string{"/users"})
	overlay := createTestOAS3DocWithPaths("overlay.yaml", nil, []string{"/users"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithPathStrategy(StrategyAcceptRight), // Would keep right, but handler overrides
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	// Handler said AcceptLeft, so base path should be kept
	assert.Equal(t, "get/users", oas3Doc.Paths["/users"].Get.OperationID)
}

func TestCollisionHandler_PathContinueWithStrategy(t *testing.T) {
	handlerCalled := false
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		handlerCalled = true
		return ContinueWithStrategy(), nil // Defer to configured strategy
	}

	base := createTestOAS3DocWithPaths("base.yaml", nil, []string{"/users"})
	overlay := createTestOAS3DocWithPaths("overlay.yaml", nil, []string{"/users"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithPathStrategy(StrategyAcceptRight), // Should take effect
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "get/users", oas3Doc.Paths["/users"].Get.OperationID)
}

func TestCollisionHandler_PathFailResolution(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return FailWithMessage("path collision not allowed by policy"), nil
	}

	base := createTestOAS3DocWithPaths("base.yaml", nil, []string{"/users"})
	overlay := createTestOAS3DocWithPaths("overlay.yaml", nil, []string{"/users"})

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithPathStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path collision not allowed by policy")
}

func TestCollisionHandler_PathHandlerErrorFallback(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return CollisionResolution{}, fmt.Errorf("simulated path handler error")
	}

	base := createTestOAS3DocWithPaths("base.yaml", nil, []string{"/users"})
	overlay := createTestOAS3DocWithPaths("overlay.yaml", nil, []string{"/users"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithPathStrategy(StrategyAcceptLeft), // Fallback
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err, "join should succeed despite handler error")

	// Verify fallback to AcceptLeft occurred (base path kept)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "get/users", oas3Doc.Paths["/users"].Get.OperationID)

	// Verify warning was recorded
	var foundWarning bool
	for _, warn := range result.StructuredWarnings {
		if warn.Category == WarnHandlerError {
			foundWarning = true
			assert.Contains(t, warn.Message, "simulated path handler error")
		}
	}
	assert.True(t, foundWarning, "should have handler error warning")
}

func TestCollisionHandler_TypeFiltering(t *testing.T) {
	schemaCallCount := 0
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		schemaCallCount++
		return ContinueWithStrategy(), nil
	}

	// Create docs with both schema and path collisions
	base := createTestOAS3DocWithPaths("base.yaml",
		map[string]string{"User": "base-user"},
		[]string{"/users"},
	)
	overlay := createTestOAS3DocWithPaths("overlay.yaml",
		map[string]string{"User": "overlay-user"},
		[]string{"/users"},
	)

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithPathStrategy(StrategyAcceptLeft),                  // Need non-fail strategy for paths
		WithCollisionHandlerFor(handler, CollisionTypeSchema), // Only schemas
	)

	assert.NoError(t, err)
	assert.Equal(t, 1, schemaCallCount, "should only call for schema collision, not path")
}

func TestCollisionHandler_TypeFilteringPathsOnly(t *testing.T) {
	pathCallCount := 0
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		pathCallCount++
		return ContinueWithStrategy(), nil
	}

	// Create docs with both schema and path collisions
	base := createTestOAS3DocWithPaths("base.yaml",
		map[string]string{"User": "base-user"},
		[]string{"/users"},
	)
	overlay := createTestOAS3DocWithPaths("overlay.yaml",
		map[string]string{"User": "overlay-user"},
		[]string{"/users"},
	)

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithPathStrategy(StrategyAcceptLeft),                // Need non-fail strategy for paths
		WithCollisionHandlerFor(handler, CollisionTypePath), // Only paths
	)

	assert.NoError(t, err)
	assert.Equal(t, 1, pathCallCount, "should only call for path collision, not schema")
}

func TestCollisionHandler_MultipleTypeFiltering(t *testing.T) {
	callCount := 0
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		callCount++
		return ContinueWithStrategy(), nil
	}

	// Create docs with both schema and path collisions
	base := createTestOAS3DocWithPaths("base.yaml",
		map[string]string{"User": "base-user"},
		[]string{"/users"},
	)
	overlay := createTestOAS3DocWithPaths("overlay.yaml",
		map[string]string{"User": "overlay-user"},
		[]string{"/users"},
	)

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithPathStrategy(StrategyAcceptLeft),                                     // Need non-fail strategy for paths
		WithCollisionHandlerFor(handler, CollisionTypeSchema, CollisionTypePath), // Both
	)

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount, "should call for both schema and path collisions")
}

func TestCollisionHandler_PathDeduplicateResolution(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return Deduplicate(), nil
	}

	base := createTestOAS3DocWithPaths("base.yaml", nil, []string{"/users"})
	overlay := createTestOAS3DocWithPaths("overlay.yaml", nil, []string{"/users"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithPathStrategy(StrategyAcceptRight), // Would take right, but handler says deduplicate
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)

	// Deduplicate keeps left (base)
	oas3Doc, _ := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "get/users", oas3Doc.Paths["/users"].Get.OperationID)

	// Should have a warning about deduplication
	var foundDedup bool
	for _, warn := range result.StructuredWarnings {
		if warn.Category == WarnPathCollision && warn.Message != "" {
			if assert.Contains(t, warn.Message, "deduplicated") {
				foundDedup = true
			}
		}
	}
	assert.True(t, foundDedup, "should have path deduplicated warning")
}

func TestCollisionHandler_PathRenameNotSupported(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return Rename(), nil
	}

	base := createTestOAS3DocWithPaths("base.yaml", nil, []string{"/users"})
	overlay := createTestOAS3DocWithPaths("overlay.yaml", nil, []string{"/users"})

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithPathStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ResolutionRename not supported for paths")
}

func TestCollisionHandler_PathCustomNotSupported(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return UseCustomValue(&parser.PathItem{}), nil
	}

	base := createTestOAS3DocWithPaths("base.yaml", nil, []string{"/users"})
	overlay := createTestOAS3DocWithPaths("overlay.yaml", nil, []string{"/users"})

	_, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithPathStrategy(StrategyAcceptLeft),
		WithCollisionHandler(handler),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ResolutionCustom for paths not yet implemented")
}

func TestCollisionHandler_PathWithMessage(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return AcceptLeftWithMessage("keeping base path due to policy"), nil
	}

	base := createTestOAS3DocWithPaths("base.yaml", nil, []string{"/users"})
	overlay := createTestOAS3DocWithPaths("overlay.yaml", nil, []string{"/users"})

	result, err := JoinWithOptions(
		WithParsed(base, overlay),
		WithPathStrategy(StrategyAcceptRight), // Would take right, but handler overrides
		WithCollisionHandler(handler),
	)

	assert.NoError(t, err)

	// Should have a handler resolution warning with the message
	var foundResolution bool
	for _, warn := range result.StructuredWarnings {
		if warn.Category == WarnHandlerResolution {
			foundResolution = true
			assert.Contains(t, warn.Message, "keeping base path due to policy")
			break
		}
	}
	assert.True(t, foundResolution, "should have handler resolution warning with message")
}
