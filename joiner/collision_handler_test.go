package joiner

import (
	"testing"

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
